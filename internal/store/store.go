// SPDX-License-Identifier: GPL-3.0-or-later

// Package store is the data layer: database/sql + hand-written portable SQL
// behind repository methods, no ORM. It supports SQLite (modernc, pure Go) and
// PostgreSQL (pgx stdlib). All SQL is written with '?' placeholders; for
// PostgreSQL they are rebound to $1, $2, ... at exec time.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

//go:embed schema_sqlite.sql
var schemaSQLite string

//go:embed schema_pg.sql
var schemaPostgres string

// Store wraps the connection pool and remembers the dialect for rebinding.
type Store struct {
	db     *sql.DB
	driver config.Driver
}

// Open connects to the configured database, applies the schema, and seeds
// default permission profiles.
func Open(ctx context.Context, cfg *config.Config) (*Store, error) {
	var driverName string
	switch cfg.DBDriver {
	case config.DriverSQLite:
		driverName = "sqlite"
	case config.DriverPostgres:
		driverName = "pgx"
	default:
		return nil, fmt.Errorf("unknown db driver %q", cfg.DBDriver)
	}

	dsn := cfg.DBDSN
	if cfg.DBDriver == config.DriverSQLite {
		// Enable FK enforcement and a sane busy timeout for the file engine.
		dsn = sqliteDSN(dsn)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", driverName, err)
	}
	if cfg.DBDriver == config.DriverSQLite {
		// modernc/sqlite is safe for concurrent use but the file engine
		// serialises writes; a single connection avoids "database is locked".
		db.SetMaxOpenConns(1)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	s := &Store{db: db, driver: cfg.DBDriver}
	if err := s.applySchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.seedProfiles(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the pool.
func (s *Store) Close() error { return s.db.Close() }

func sqliteDSN(path string) string {
	if strings.Contains(path, "?") {
		return path + "&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}
	return path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
}

func (s *Store) applySchema(ctx context.Context) error {
	schema := schemaSQLite
	if s.driver == config.DriverPostgres {
		schema = schemaPostgres
	}
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

// rebind converts '?' placeholders to the dialect's form. SQLite keeps '?';
// PostgreSQL needs positional $N. Question marks inside string literals are not
// expected in our SQL, so a simple scan is sufficient.
func (s *Store) rebind(query string) string {
	if s.driver != config.DriverPostgres {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func (s *Store) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, s.rebind(query), args...)
}

func (s *Store) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, s.rebind(query), args...)
}

func (s *Store) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, s.rebind(query), args...)
}

// nowMS returns the current time as epoch milliseconds (our portable time unit).
func nowMS() int64 { return time.Now().UnixMilli() }

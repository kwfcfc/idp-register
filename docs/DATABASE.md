<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Database Policy

idp-register supports fresh deployments on either SQLite or PostgreSQL. The goal is
functional consistency: the same Go repository methods, public/admin flows, token
semantics, and application state machine should behave the same on both engines.
PostgreSQL is the scale/concurrency option; SQLite is the simple single-file option.

## Supported Versions

Current development baseline:

| Backend | Supported baseline | Notes |
|---|---|---|
| SQLite | `modernc.org/sqlite v1.52.0` embedding SQLite `3.53.2` | The app uses the pure-Go driver; no system SQLite install is required. |
| PostgreSQL | PostgreSQL `16` via `postgres:16-alpine` | Accessed through `github.com/jackc/pgx/v5 v5.10.0`. Older PG versions may work but are not declared supported until tested. |

The local M3 smoke-test harness uses SQLite through `SQLITE_PATH=idp-register-dev.db`.
The root `docker-compose.yml` documents the planned PostgreSQL deployment shape with
`postgres:16-alpine`.

## Fresh Deployments

For a new installation, choose exactly one database backend:

- **SQLite**: set `SQLITE_PATH`; the application creates the schema in that file.
- **PostgreSQL**: set `DATABASE_URL`; the application creates the schema in that database.

The application currently applies embedded schema files at startup:

- `internal/store/schema_sqlite.sql`
- `internal/store/schema_pg.sql`

The matching operator-facing initial SQL files are:

- `migrations/sqlite/001_init.sql`
- `migrations/postgres/001_init.sql`

Until a real upgrade migrator exists, keep the embedded schema files and these initial SQL
files in lock-step.

## Upgrade Policy

There is no automatic old-database upgrade framework yet. Before a stable release, a
breaking schema change may require a backup and database rebuild.

When a future release introduces a breaking schema change, that release must explicitly
choose one of these paths:

- provide an automatic in-place upgrade for existing deployments;
- provide a documented manual upgrade path;
- declare the change as requiring export, rebuild, and import.

Do not silently change persisted schema semantics without a release note or schema
document update.

## Backend Switching

Automatic SQLite-to-PostgreSQL or PostgreSQL-to-SQLite conversion is not a project goal.
Operators who switch backends should treat it as a manual data migration:

1. Stop idp-register.
2. Back up the source database.
3. Create the target database with the matching initial schema.
4. Copy tables in dependency order:
   `permission_profiles`, `registration_tokens`, `applications`, `admin_sessions`,
   `audit_log`.
5. Preserve integer epoch-millisecond timestamps exactly.
6. Preserve JSON-in-`TEXT` fields as JSON text, not as PostgreSQL arrays or JSONB.
7. Preserve invite counters exactly: `pending`, `completed`, and `uses_allowed`.
8. Run foreign-key checks on the target engine.

Session rows may be discarded during a backend switch; admins can log in again.

## Operational Notes

- SQLite is simplest for small self-hosted deployments. The app limits SQLite to one open
  connection to avoid file-lock contention.
- PostgreSQL is preferred when the service needs external backups, stronger operational
  tooling, or higher write concurrency.
- Both engines must preserve account-enumeration behavior and invite-code semantics.

// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"context"
	"database/sql"
	"errors"
)

// CreateSession stores an opaque admin session keyed by the SHA-256 digest of
// the cookie value (the raw value never touches the database).
func (s *Store) CreateSession(ctx context.Context, id, tokenDigest string, user AdminUser, expiresAt int64) error {
	now := nowMS()
	_, err := s.exec(ctx,
		`INSERT INTO admin_sessions
		   (id, token_digest, subject, email, display_name, groups, expires_at, created_at, last_seen_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tokenDigest, user.Sub, user.Email, user.DisplayName, jsonArray(user.Groups), expiresAt, now, now)
	return err
}

// LoadSession validates a session by token digest, refreshes last_seen_at, and
// returns the admin identity. Returns ErrNotFound if absent or expired.
func (s *Store) LoadSession(ctx context.Context, tokenDigest string, now int64) (*AdminUser, error) {
	res, err := s.exec(ctx,
		`UPDATE admin_sessions SET last_seen_at = ?
		  WHERE token_digest = ? AND expires_at > ?`, now, tokenDigest, now)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}

	var u AdminUser
	var groups string
	err = s.queryRow(ctx,
		`SELECT subject, email, display_name, groups FROM admin_sessions WHERE token_digest = ?`, tokenDigest).
		Scan(&u.Sub, &u.Email, &u.DisplayName, &groups)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Groups = parseJSONArray(groups)
	return &u, nil
}

// DeleteSession removes a session by token digest (logout).
func (s *Store) DeleteSession(ctx context.Context, tokenDigest string) error {
	_, err := s.exec(ctx, `DELETE FROM admin_sessions WHERE token_digest = ?`, tokenDigest)
	return err
}

// PurgeExpiredSessions deletes sessions past their expiry.
func (s *Store) PurgeExpiredSessions(ctx context.Context, now int64) error {
	_, err := s.exec(ctx, `DELETE FROM admin_sessions WHERE expires_at <= ?`, now)
	return err
}

// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"context"
	"database/sql"
	"errors"
)

// CreateToken inserts a new plaintext registration token. The caller mints the
// token string and id (see internal/token).
func (s *Store) CreateToken(ctx context.Context, t *RegistrationToken) error {
	if t.CreatedAt == 0 {
		t.CreatedAt = nowMS()
	}
	active := 0
	if t.Active {
		active = 1
	}
	_, err := s.exec(ctx,
		`INSERT INTO registration_tokens
		   (id, token, uses_allowed, pending, completed, expiry_time, active,
		    email_constraint, profile_id, note, created_by_sub, created_by_email, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Token, t.UsesAllowed, t.Pending, t.Completed, t.ExpiryTime, active,
		t.EmailConstraint, t.ProfileID, t.Note, t.CreatedBySub, t.CreatedByEmail, t.CreatedAt)
	return err
}

const tokenColumns = `id, token, uses_allowed, pending, completed, expiry_time, active,
	email_constraint, profile_id, note, created_by_sub, created_by_email, created_at`

func scanToken(sc interface{ Scan(...any) error }) (*RegistrationToken, error) {
	var t RegistrationToken
	var active int
	if err := sc.Scan(&t.ID, &t.Token, &t.UsesAllowed, &t.Pending, &t.Completed,
		&t.ExpiryTime, &active, &t.EmailConstraint, &t.ProfileID, &t.Note,
		&t.CreatedBySub, &t.CreatedByEmail, &t.CreatedAt); err != nil {
		return nil, err
	}
	t.Active = active != 0
	return &t, nil
}

// GetTokenByValue looks up a token by its plaintext value.
func (s *Store) GetTokenByValue(ctx context.Context, token string) (*RegistrationToken, error) {
	row := s.queryRow(ctx, `SELECT `+tokenColumns+` FROM registration_tokens WHERE token = ?`, token)
	t, err := scanToken(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// GetTokenByID looks up a token by id.
func (s *Store) GetTokenByID(ctx context.Context, id string) (*RegistrationToken, error) {
	row := s.queryRow(ctx, `SELECT `+tokenColumns+` FROM registration_tokens WHERE id = ?`, id)
	t, err := scanToken(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// ListTokens returns all tokens, newest first. validOnly filters to currently
// usable tokens (mirrors Synapse's ?valid= query parameter).
func (s *Store) ListTokens(ctx context.Context, validOnly bool, now int64) ([]RegistrationToken, error) {
	q := `SELECT ` + tokenColumns + ` FROM registration_tokens`
	var args []any
	if validOnly {
		q += ` WHERE active = 1
		         AND (expiry_time IS NULL OR expiry_time > ?)
		         AND (uses_allowed IS NULL OR pending + completed < uses_allowed)`
		args = append(args, now)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := s.query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RegistrationToken
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// ReserveToken atomically reserves one use of a token (pending++) iff it is
// still valid. Returns true on success. This is the load-bearing concurrency
// guard (docs/ARCHITECTURE.md): success is defined by RowsAffected == 1.
func (s *Store) ReserveToken(ctx context.Context, token string, now int64) (bool, error) {
	res, err := s.exec(ctx,
		`UPDATE registration_tokens SET pending = pending + 1
		  WHERE token = ?
		    AND active = 1
		    AND (expiry_time IS NULL OR expiry_time > ?)
		    AND (uses_allowed IS NULL OR pending + completed < uses_allowed)`,
		token, now)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// CompleteTokenUse converts a reservation into a completed registration
// (pending--, completed++). Called when activation is confirmed.
func (s *Store) CompleteTokenUse(ctx context.Context, tokenID string) error {
	_, err := s.exec(ctx,
		`UPDATE registration_tokens
		    SET pending = CASE WHEN pending > 0 THEN pending - 1 ELSE 0 END,
		        completed = completed + 1
		  WHERE id = ?`, tokenID)
	return err
}

// ReleaseTokenReservation returns a reserved slot to the pool (pending--).
// Called when provisioning fails permanently or an activation link expires.
func (s *Store) ReleaseTokenReservation(ctx context.Context, tokenID string) error {
	_, err := s.exec(ctx,
		`UPDATE registration_tokens
		    SET pending = CASE WHEN pending > 0 THEN pending - 1 ELSE 0 END
		  WHERE id = ?`, tokenID)
	return err
}

// SetTokenActive flips the manual disable switch.
func (s *Store) SetTokenActive(ctx context.Context, id string, active bool) (bool, error) {
	v := 0
	if active {
		v = 1
	}
	res, err := s.exec(ctx, `UPDATE registration_tokens SET active = ? WHERE id = ?`, v, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// DeleteToken removes a token by id.
func (s *Store) DeleteToken(ctx context.Context, id string) (bool, error) {
	res, err := s.exec(ctx, `DELETE FROM registration_tokens WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"context"
	"database/sql"
	"errors"
)

const appColumns = `id, token_id, email, username, review_text, requested_services, status,
	captcha_provider, submitted_ip, approved_profile_id, provider_user_id, provisioning_error,
	decision_note, reviewed_at, reviewed_by_sub, reviewed_by_email, created_at, updated_at`

func scanApplication(sc interface{ Scan(...any) error }) (*Application, error) {
	var a Application
	var services string
	if err := sc.Scan(&a.ID, &a.TokenID, &a.Email, &a.Username, &a.ReviewText, &services, &a.Status,
		&a.CaptchaProvider, &a.SubmittedIP, &a.ApprovedProfileID, &a.ProviderUserID, &a.ProvisioningError,
		&a.DecisionNote, &a.ReviewedAt, &a.ReviewedBySub, &a.ReviewedByEmail, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.RequestedServices = parseJSONArray(services)
	return &a, nil
}

// CreateApplication inserts a new submission. The caller fills id, normalized
// fields and timestamps. The active-email/username partial unique indexes will
// reject a duplicate live application (handled as a uniform response upstream).
func (s *Store) CreateApplication(ctx context.Context, a *Application, emailNorm, usernameNorm string) error {
	now := nowMS()
	if a.CreatedAt == 0 {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	_, err := s.exec(ctx,
		`INSERT INTO applications
		   (id, token_id, email, email_normalized, username, username_normalized,
		    review_text, requested_services, status, captcha_provider, submitted_ip,
		    approved_profile_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.TokenID, a.Email, emailNorm, a.Username, usernameNorm,
		a.ReviewText, jsonArray(a.RequestedServices), a.Status, a.CaptchaProvider, a.SubmittedIP,
		a.ApprovedProfileID, a.CreatedAt, a.UpdatedAt)
	return err
}

// GetApplication returns a single application by id, or ErrNotFound.
func (s *Store) GetApplication(ctx context.Context, id string) (*Application, error) {
	row := s.queryRow(ctx, `SELECT `+appColumns+` FROM applications WHERE id = ?`, id)
	a, err := scanApplication(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

// ListApplications returns applications, newest first, optionally filtered by status.
func (s *Store) ListApplications(ctx context.Context, status string) ([]Application, error) {
	q := `SELECT ` + appColumns + ` FROM applications`
	var args []any
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := s.query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Application
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// ClaimForProvisioning atomically moves an application from pending or
// provisioning_failed into provisioning, recording the approving admin and the
// chosen profile. Returns true if this caller won the claim. The actual IdP
// group list is resolved separately from the profile (the public form never
// supplies groups — invariant #4).
func (s *Store) ClaimForProvisioning(ctx context.Context, id, profileID, note string, actor AdminUser) (bool, error) {
	res, err := s.exec(ctx,
		`UPDATE applications
		    SET status = 'provisioning',
		        approved_profile_id = ?,
		        decision_note = ?,
		        reviewed_at = ?,
		        reviewed_by_sub = ?,
		        reviewed_by_email = ?,
		        provisioning_error = NULL,
		        updated_at = ?
		  WHERE id = ?
		    AND status IN ('pending', 'provisioning_failed')`,
		profileID, nullIfEmpty(note), nowMS(), actor.Sub, actor.Email, nowMS(), id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// MarkApproved records a successful provisioning.
func (s *Store) MarkApproved(ctx context.Context, id, providerUserID string) error {
	_, err := s.exec(ctx,
		`UPDATE applications
		    SET status = 'approved', provider_user_id = ?, provisioning_error = NULL, updated_at = ?
		  WHERE id = ?`,
		providerUserID, nowMS(), id)
	return err
}

// MarkProvisioningFailed records a provisioning failure for safe retry.
func (s *Store) MarkProvisioningFailed(ctx context.Context, id, message string, providerUserID *string) error {
	if len(message) > 4000 {
		message = message[:4000]
	}
	_, err := s.exec(ctx,
		`UPDATE applications
		    SET status = 'provisioning_failed',
		        provisioning_error = ?,
		        provider_user_id = COALESCE(?, provider_user_id),
		        updated_at = ?
		  WHERE id = ?`,
		message, providerUserID, nowMS(), id)
	return err
}

// RecoverStaleProvisioning sweeps applications stuck in 'provisioning' into
// 'provisioning_failed'. Provisioning runs synchronously inside a request, so
// after a restart any row still in 'provisioning' was interrupted mid-flight
// and would otherwise be unreachable (neither claimable nor decidable). Moving
// it to provisioning_failed re-enters the normal retry/reject recovery path;
// a held invite reservation stays held, exactly as for any other failure.
// Returns the number of recovered applications.
func (s *Store) RecoverStaleProvisioning(ctx context.Context, message string) (int64, error) {
	res, err := s.exec(ctx,
		`UPDATE applications
		    SET status = 'provisioning_failed', provisioning_error = ?, updated_at = ?
		  WHERE status = 'provisioning'`,
		message, nowMS())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Decide rejects an application. Returns true if a row in a decidable state was
// updated.
func (s *Store) Decide(ctx context.Context, id, status, note string, actor AdminUser) (bool, error) {
	res, err := s.exec(ctx,
		`UPDATE applications
		    SET status = ?, decision_note = ?, reviewed_at = ?,
		        reviewed_by_sub = ?, reviewed_by_email = ?, updated_at = ?
		  WHERE id = ? AND status IN ('pending', 'provisioning_failed')`,
		status, nullIfEmpty(note), nowMS(), actor.Sub, actor.Email, nowMS(), id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

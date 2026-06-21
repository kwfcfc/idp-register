// SPDX-License-Identifier: GPL-3.0-or-later

package store

import "context"

// InsertAudit appends one entry to the audit log. details must be a JSON
// object string (the caller marshals). id is app-generated.
func (s *Store) InsertAudit(ctx context.Context, e *AuditEntry) error {
	if e.CreatedAt == 0 {
		e.CreatedAt = nowMS()
	}
	if e.Details == "" {
		e.Details = "{}"
	}
	_, err := s.exec(ctx,
		`INSERT INTO audit_log
		   (id, actor_sub, actor_email, action, target_type, target_id, details, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.ActorSub, e.ActorEmail, e.Action, e.TargetType, e.TargetID, e.Details, e.CreatedAt)
	return err
}

// ListAudit returns the most recent audit entries, newest first.
func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.query(ctx,
		`SELECT id, actor_sub, actor_email, action, target_type, target_id, details, created_at
		   FROM audit_log ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorSub, &e.ActorEmail, &e.Action,
			&e.TargetType, &e.TargetID, &e.Details, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

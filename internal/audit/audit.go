// SPDX-License-Identifier: GPL-3.0-or-later

// Package audit records admin actions. It is a thin convenience wrapper over
// store.InsertAudit that generates the id and marshals details to JSON.
//
// IMPORTANT: never pass a plaintext invite code in details (invariant #3).
package audit

import (
	"context"
	"encoding/json"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/store"

	"github.com/google/uuid"
)

// Logger writes audit entries.
type Logger struct {
	store *store.Store
}

// New creates an audit Logger.
func New(s *store.Store) *Logger { return &Logger{store: s} }

// Record appends one audit entry. details may be nil. Failures are returned so
// callers can decide whether to treat auditing as best-effort.
func (l *Logger) Record(ctx context.Context, actor store.AdminUser, action, targetType, targetID string, details map[string]any) error {
	raw := "{}"
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			raw = string(b)
		}
	}
	return l.store.InsertAudit(ctx, &store.AuditEntry{
		ID:         uuid.NewString(),
		ActorSub:   actor.Sub,
		ActorEmail: actor.Email,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Details:    raw,
	})
}

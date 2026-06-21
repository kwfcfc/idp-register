// SPDX-License-Identifier: GPL-3.0-or-later

// Package token mints and manages plaintext, Synapse-aligned registration
// codes. Semantics mirror Synapse registration tokens: uses_allowed / pending /
// completed / expiry_time. Codes are NEVER logged (invariant #3).
package token

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/audit"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/store"

	"github.com/google/uuid"
)

// Synapse token rules: charset [A-Za-z0-9._~-], default length 16, max 64.
const (
	charset       = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._~-"
	defaultLength = 16
	maxLength     = 64
)

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{1,64}$`)

// ErrInvalidToken is returned for a malformed admin-supplied token string.
var ErrInvalidToken = errors.New("token must be 1-64 chars from [A-Za-z0-9._~-]")

// Service owns invite-code lifecycle.
type Service struct {
	store *store.Store
	audit *audit.Logger
}

// New constructs the token Service.
func New(s *store.Store, a *audit.Logger) *Service { return &Service{store: s, audit: a} }

// MintParams describes a token to create. A blank Token is randomly generated.
type MintParams struct {
	Token           string
	UsesAllowed     *int64  // nil = unlimited
	ExpiryTime      *int64  // epoch ms; nil = never
	EmailConstraint *string // lowercased binding to one email
	ProfileID       *string
	Note            string
}

// Mint creates a token. The plaintext value lives only on the returned struct
// and in the database; it is deliberately excluded from the audit details.
func (s *Service) Mint(ctx context.Context, p MintParams, actor store.AdminUser) (*store.RegistrationToken, error) {
	tok := p.Token
	if tok == "" {
		var err error
		if tok, err = generate(defaultLength); err != nil {
			return nil, err
		}
	} else if !tokenPattern.MatchString(tok) {
		return nil, ErrInvalidToken
	}

	t := &store.RegistrationToken{
		ID:              uuid.NewString(),
		Token:           tok,
		UsesAllowed:     p.UsesAllowed,
		ExpiryTime:      p.ExpiryTime,
		Active:          true,
		EmailConstraint: p.EmailConstraint,
		ProfileID:       p.ProfileID,
		Note:            p.Note,
		CreatedBySub:    actor.Sub,
		CreatedByEmail:  actor.Email,
	}
	if err := s.store.CreateToken(ctx, t); err != nil {
		return nil, err
	}

	_ = s.audit.Record(ctx, actor, "token.create", "registration_token", t.ID, map[string]any{
		"usesAllowed":     p.UsesAllowed,
		"expiryTime":      p.ExpiryTime,
		"profileId":       p.ProfileID,
		"emailConstraint": p.EmailConstraint, // an email is not a secret like the code is
	})
	return t, nil
}

// List returns tokens; validOnly mirrors Synapse's ?valid= filter.
func (s *Service) List(ctx context.Context, validOnly bool, now int64) ([]store.RegistrationToken, error) {
	return s.store.ListTokens(ctx, validOnly, now)
}

// SetActive toggles the manual disable switch.
func (s *Service) SetActive(ctx context.Context, id string, active bool, actor store.AdminUser) (bool, error) {
	ok, err := s.store.SetTokenActive(ctx, id, active)
	if err != nil || !ok {
		return ok, err
	}
	action := "token.disable"
	if active {
		action = "token.enable"
	}
	_ = s.audit.Record(ctx, actor, action, "registration_token", id, nil)
	return true, nil
}

// Delete removes a token.
func (s *Service) Delete(ctx context.Context, id string, actor store.AdminUser) (bool, error) {
	ok, err := s.store.DeleteToken(ctx, id)
	if err != nil || !ok {
		return ok, err
	}
	_ = s.audit.Record(ctx, actor, "token.delete", "registration_token", id, nil)
	return true, nil
}

// generate returns a random token of the given length over the Synapse charset.
// The modulo over a 66-char set introduces negligible bias for a shareable
// invite code (entropy stays well above brute-force concerns at length 16).
func generate(length int) (string, error) {
	if length <= 0 || length > maxLength {
		length = defaultLength
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	for i := range buf {
		buf[i] = charset[int(buf[i])%len(charset)]
	}
	return string(buf), nil
}

// SPDX-License-Identifier: GPL-3.0-or-later

// Package application implements the registration state machine: public
// submission (with optional auto-approval via a valid invite code), admin
// review decisions, and provisioning into the target IdP.
//
// Key invariants enforced here:
//   - the public form never selects IdP groups; groups are resolved server-side
//     from a permission profile at provisioning time (invariant #4);
//   - submission responses are uniform — callers get "received" regardless of
//     whether the email/code exists (anti-enumeration, invariant #5);
//   - one user-facing email: provisioning lets the IdP send the activation mail.
package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/audit"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/provisioner"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/store"

	"github.com/google/uuid"
)

// systemActor attributes auto-approval actions in the audit log.
var systemActor = store.AdminUser{Sub: "system", Email: "auto@idp-register", DisplayName: "auto-approval"}

func nowMS() int64 { return time.Now().UnixMilli() }

// Service orchestrates applications.
type Service struct {
	store *store.Store
	prov  provisioner.Provisioner
	audit *audit.Logger
}

// New constructs the application Service.
func New(s *store.Store, p provisioner.Provisioner, a *audit.Logger) *Service {
	return &Service{store: s, prov: p, audit: a}
}

// SubmitInput is the public form payload (already CAPTCHA-verified upstream).
type SubmitInput struct {
	Email           string
	Username        string
	ReviewText      string
	InviteCode      string // optional plaintext; never logged
	CaptchaProvider string
	SubmittedIP     string
}

// SubmitResult is intentionally minimal so the HTTP layer can return a uniform
// response. AutoApproved is for internal logging/metrics, not for the user.
type SubmitResult struct {
	ApplicationID string
	AutoApproved  bool
}

// Submit records a registration application. If a valid invite code with spare
// capacity is supplied, the code's use is reserved and provisioning is attempted
// immediately (auto-approval); otherwise the application waits for admin review.
//
// A duplicate live application (same email/username) is reported via
// store's unique indexes; callers should still return the uniform response.
func (s *Service) Submit(ctx context.Context, in SubmitInput) (*SubmitResult, error) {
	emailNorm := strings.ToLower(strings.TrimSpace(in.Email))
	usernameNorm := strings.ToLower(strings.TrimSpace(in.Username))

	app := &store.Application{
		ID:         uuid.NewString(),
		Email:      strings.TrimSpace(in.Email),
		Username:   strings.TrimSpace(in.Username),
		ReviewText: strings.TrimSpace(in.ReviewText),
		Status:     store.StatusPending,
	}
	if in.CaptchaProvider != "" {
		app.CaptchaProvider = &in.CaptchaProvider
	}
	if in.SubmittedIP != "" {
		app.SubmittedIP = &in.SubmittedIP
	}

	// Try the invite code. Any failure here silently falls back to manual
	// review so we never leak code validity (anti-enumeration).
	var reservedToken *store.RegistrationToken
	if code := strings.TrimSpace(in.InviteCode); code != "" {
		if t := s.tryReserve(ctx, code, emailNorm); t != nil {
			reservedToken = t
			app.TokenID = &t.ID
			app.Status = store.StatusProvisioning
			if t.ProfileID != nil {
				app.ApprovedProfileID = t.ProfileID
			}
		}
	}

	if err := s.store.CreateApplication(ctx, app, emailNorm, usernameNorm); err != nil {
		// Creation failed (e.g. duplicate live application). Release any slot
		// we reserved so a transient duplicate doesn't burn a use.
		if reservedToken != nil {
			_ = s.store.ReleaseTokenReservation(ctx, reservedToken.ID)
		}
		return nil, err
	}

	if reservedToken == nil {
		return &SubmitResult{ApplicationID: app.ID, AutoApproved: false}, nil
	}

	// Auto-approval path: provision now.
	groups := s.groupsForProfile(ctx, app.ApprovedProfileID)
	if err := s.provision(ctx, app, groups); err != nil {
		// Leave the application in provisioning_failed (provision() set that)
		// for admin retry; the reservation stays held.
		return &SubmitResult{ApplicationID: app.ID, AutoApproved: true}, nil
	}
	return &SubmitResult{ApplicationID: app.ID, AutoApproved: true}, nil
}

// tryReserve validates an invite code and atomically reserves one use. Returns
// the token on success, nil on any failure (caller falls back to review).
func (s *Service) tryReserve(ctx context.Context, code, emailNorm string) *store.RegistrationToken {
	t, err := s.store.GetTokenByValue(ctx, code)
	if err != nil {
		return nil
	}
	// Enforce an email-bound code before consuming a use.
	if t.EmailConstraint != nil && !strings.EqualFold(*t.EmailConstraint, emailNorm) {
		return nil
	}
	ok, err := s.store.ReserveToken(ctx, code, nowMS())
	if err != nil || !ok {
		return nil
	}
	return t
}

// List returns applications, optionally filtered by status.
func (s *Service) List(ctx context.Context, status string) ([]store.Application, error) {
	return s.store.ListApplications(ctx, status)
}

// Get returns one application.
func (s *Service) Get(ctx context.Context, id string) (*store.Application, error) {
	return s.store.GetApplication(ctx, id)
}

// Approve claims a pending/failed application and provisions it under the given
// profile. Groups are resolved from the profile server-side.
func (s *Service) Approve(ctx context.Context, id, profileID, note string, actor store.AdminUser) error {
	if profileID == "" {
		return errors.New("a permission profile is required to approve")
	}
	prof, err := s.store.GetProfile(ctx, profileID)
	if err != nil {
		return err
	}
	won, err := s.store.ClaimForProvisioning(ctx, id, profileID, note, actor)
	if err != nil {
		return err
	}
	if !won {
		return errors.New("application is no longer available for provisioning")
	}
	_ = s.audit.Record(ctx, actor, "application.provision.start", "application", id, map[string]any{"profileId": profileID})

	app, err := s.store.GetApplication(ctx, id)
	if err != nil {
		return err
	}
	return s.provision(ctx, app, prof.Groups)
}

// Decide rejects or requests changes. A held invite reservation (if any) is
// released so the use returns to the pool.
func (s *Service) Decide(ctx context.Context, id, status, note string, actor store.AdminUser) (bool, error) {
	if status != store.StatusRejected && status != store.StatusNeedsChanges {
		return false, errors.New("invalid decision status")
	}
	app, err := s.store.GetApplication(ctx, id)
	if err != nil {
		return false, err
	}
	ok, err := s.store.Decide(ctx, id, status, note, actor)
	if err != nil || !ok {
		return ok, err
	}
	if app.TokenID != nil {
		_ = s.store.ReleaseTokenReservation(ctx, *app.TokenID)
	}
	_ = s.audit.Record(ctx, actor, "application."+status, "application", id, map[string]any{"note": note})
	return true, nil
}

// provision performs the IdP-side work and advances the state machine. On any
// failure the application is marked provisioning_failed (safe to retry) and the
// invite reservation is intentionally kept.
func (s *Service) provision(ctx context.Context, app *store.Application, groups []string) error {
	actor := systemActor

	// Refuse to silently collide with an existing IdP account.
	if existing, found, err := s.prov.FindUserByEmail(ctx, app.Email); err != nil {
		return s.failProvision(ctx, app.ID, "lookup failed: "+err.Error(), nil)
	} else if found {
		uid := existing.ID
		return s.failProvision(ctx, app.ID, "a user with this email already exists in the target IdP", &uid)
	}

	userID, err := s.prov.CreateUser(ctx, provisioner.NewUser{
		Email:    app.Email,
		Username: app.Username,
		Groups:   groups,
	})
	if err != nil {
		var uid *string
		if userID != "" {
			uid = &userID
		}
		return s.failProvision(ctx, app.ID, "create user failed: "+err.Error(), uid)
	}

	if _, err := s.prov.InitCredentials(ctx, userID, app.Email); err != nil {
		return s.failProvision(ctx, app.ID, "send activation failed: "+err.Error(), &userID)
	}

	if err := s.store.MarkApproved(ctx, app.ID, userID); err != nil {
		return err
	}
	if app.TokenID != nil {
		_ = s.store.CompleteTokenUse(ctx, *app.TokenID)
	}
	_ = s.audit.Record(ctx, actor, "application.approve", "application", app.ID, map[string]any{"providerUserId": userID})
	return nil
}

func (s *Service) failProvision(ctx context.Context, id, msg string, providerUserID *string) error {
	_ = s.store.MarkProvisioningFailed(ctx, id, msg, providerUserID)
	_ = s.audit.Record(ctx, systemActor, "application.provision.fail", "application", id, map[string]any{"message": msg})
	return errors.New(msg)
}

func (s *Service) groupsForProfile(ctx context.Context, profileID *string) []string {
	if profileID == nil {
		return nil
	}
	prof, err := s.store.GetProfile(ctx, *profileID)
	if err != nil {
		return nil
	}
	return prof.Groups
}

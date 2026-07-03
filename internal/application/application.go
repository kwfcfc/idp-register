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
	"sort"
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
	store    *store.Store
	prov     provisioner.Provisioner
	audit    *audit.Logger
	denylist map[string]bool // group names never assignable to a profile
}

// New constructs the application Service. groupDenylist names IdP groups that
// must never appear in a permission profile (infra/admin groups).
func New(s *store.Store, p provisioner.Provisioner, a *audit.Logger, groupDenylist []string) *Service {
	deny := make(map[string]bool, len(groupDenylist))
	for _, g := range groupDenylist {
		if g = strings.TrimSpace(g); g != "" {
			deny[g] = true
		}
	}
	return &Service{store: s, prov: p, audit: a, denylist: deny}
}

// SubmitInput is the public form payload (already CAPTCHA-verified upstream).
type SubmitInput struct {
	Email           string
	Username        string
	ReviewText      string
	InviteCode      string   // optional plaintext; never logged
	Services        []string // requested service/profile ids (advisory; validated)
	CaptchaProvider string
	SubmittedIP     string
}

// SubmitResult is intentionally minimal so the HTTP layer can return a uniform
// response. AutoApproved is for internal logging/metrics, not for the user.
type SubmitResult struct {
	ApplicationID string
	AutoApproved  bool
}

// Submit records a registration application. If a valid invite code with a
// bound permission profile and spare capacity is supplied, that profile becomes
// the approved service, the code's use is reserved, and provisioning is
// attempted immediately (auto-approval). Otherwise the application waits for
// admin review.
//
// A duplicate live application (same email/username) is reported via
// store's unique indexes; callers should still return the uniform response.
func (s *Service) Submit(ctx context.Context, in SubmitInput) (*SubmitResult, error) {
	emailNorm := strings.ToLower(strings.TrimSpace(in.Email))
	usernameNorm := strings.ToLower(strings.TrimSpace(in.Username))

	app := &store.Application{
		ID:                uuid.NewString(),
		Email:             strings.TrimSpace(in.Email),
		Username:          strings.TrimSpace(in.Username),
		ReviewText:        strings.TrimSpace(in.ReviewText),
		RequestedServices: s.validServices(ctx, in.Services),
		Status:            store.StatusPending,
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
			app.ApprovedProfileID = t.ProfileID
			app.RequestedServices = []string{*t.ProfileID}
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

// tryReserve validates an invite code with an auto-approval profile and
// atomically reserves one use. Returns the token on success, nil on any failure
// (caller falls back to review).
func (s *Service) tryReserve(ctx context.Context, code, emailNorm string) *store.RegistrationToken {
	t, err := s.store.GetTokenByValue(ctx, code)
	if err != nil {
		return nil
	}
	if t.ProfileID == nil || *t.ProfileID == "" {
		return nil
	}
	if _, err := s.store.GetProfile(ctx, *t.ProfileID); err != nil {
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
// profile. Groups are resolved from the profile server-side. Token-backed retry
// approvals keep the original token-bound profile; an admin cannot turn a failed
// invite provisioning into a different permission grant.
func (s *Service) Approve(ctx context.Context, id, profileID, note string, actor store.AdminUser) error {
	appBeforeClaim, err := s.store.GetApplication(ctx, id)
	if err != nil {
		return err
	}
	if appBeforeClaim.Status == store.StatusProvisioningFailed &&
		appBeforeClaim.TokenID != nil &&
		appBeforeClaim.ApprovedProfileID != nil {
		profileID = *appBeforeClaim.ApprovedProfileID
	}
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

// Decide rejects an application. A held invite reservation (if any) is released
// so the use returns to the pool.
func (s *Service) Decide(ctx context.Context, id, status, note string, actor store.AdminUser) (bool, error) {
	if status != store.StatusRejected {
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
	if err != nil && errors.Is(err, provisioner.ErrUsernameNotSet) && userID != "" {
		// Partial success: the account exists (and the IdP may already have sent
		// the activation email), so failing here would strand it — a retry hits
		// "email already exists" and forces a manual IdP cleanup. Approve with a
		// warning; the admin can set the username in the IdP later.
		_ = s.audit.Record(ctx, actor, "application.provision.warn", "application", app.ID,
			map[string]any{"providerUserId": userID, "message": err.Error()})
		err = nil
	}
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

// validServices filters a requested service selection down to ids that are
// actually public-selectable. Tampered/unknown ids are silently dropped — the
// field is advisory (the real groups are resolved from a profile at approval),
// so this just keeps the stored intent clean. Best-effort: on a DB error the
// selection is dropped entirely rather than trusted.
func (s *Service) validServices(ctx context.Context, requested []string) []string {
	if len(requested) == 0 {
		return nil
	}
	allowed, err := s.store.PublicServiceIDs(ctx)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(requested))
	for _, id := range requested {
		id = strings.TrimSpace(id)
		if allowed[id] && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// ----- permission-profile administration (ADR-0012) -----

// PublicServices returns the anonymous-safe option list for the public form.
func (s *Service) PublicServices(ctx context.Context) ([]store.PublicService, error) {
	return s.store.ListPublicServices(ctx)
}

// ListProfiles returns all profiles (admin view, includes groups).
func (s *Service) ListProfiles(ctx context.Context) ([]store.PermissionProfile, error) {
	return s.store.ListProfiles(ctx)
}

// AvailableGroups returns the target-IdP groups a profile may draw from: the
// live catalog minus the denylist, sorted by name for a stable UI.
func (s *Service) AvailableGroups(ctx context.Context) ([]provisioner.Group, error) {
	all, err := s.prov.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]provisioner.Group, 0, len(all))
	for _, g := range all {
		if !s.denylist[g.Name] {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ProfileInput is the admin-supplied shape for creating/updating a profile.
type ProfileInput struct {
	ID               string // create only; ignored on update
	Label            string
	Description      string
	Groups           []string
	PublicSelectable bool
	PublicLabel      string
	SortOrder        int64
}

// validateGroups rejects a profile whose groups are not all in the allowed
// catalog (exist in the target IdP and are not denied). This is the server-side
// guard that a denied/admin group can never be smuggled into a profile.
func (s *Service) validateGroups(ctx context.Context, groups []string) error {
	allowed, err := s.AvailableGroups(ctx)
	if err != nil {
		return errors.New("could not load the group catalog from the target IdP: " + err.Error())
	}
	ok := make(map[string]bool, len(allowed))
	for _, g := range allowed {
		ok[g.Name] = true
	}
	for _, g := range groups {
		if !ok[g] {
			return errors.New("group not assignable (unknown or restricted): " + g)
		}
	}
	return nil
}

// CreateProfile validates and stores a new permission profile.
func (s *Service) CreateProfile(ctx context.Context, in ProfileInput, actor store.AdminUser) (*store.PermissionProfile, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" || strings.ContainsAny(id, " \t\n") {
		return nil, errors.New("a non-empty id without whitespace is required")
	}
	if strings.TrimSpace(in.Label) == "" {
		return nil, errors.New("a label is required")
	}
	groups := normalizeGroups(in.Groups)
	if err := s.validateGroups(ctx, groups); err != nil {
		return nil, err
	}
	p := &store.PermissionProfile{
		ID:               id,
		Label:            strings.TrimSpace(in.Label),
		Description:      strings.TrimSpace(in.Description),
		Groups:           groups,
		PublicSelectable: in.PublicSelectable,
		PublicLabel:      strings.TrimSpace(in.PublicLabel),
		SortOrder:        in.SortOrder,
	}
	if err := s.store.CreateProfile(ctx, p); err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, actor, "profile.create", "profile", p.ID, map[string]any{"groups": p.Groups})
	return p, nil
}

// UpdateProfile validates and replaces the mutable fields of a profile.
func (s *Service) UpdateProfile(ctx context.Context, id string, in ProfileInput, actor store.AdminUser) (*store.PermissionProfile, error) {
	if strings.TrimSpace(in.Label) == "" {
		return nil, errors.New("a label is required")
	}
	groups := normalizeGroups(in.Groups)
	if err := s.validateGroups(ctx, groups); err != nil {
		return nil, err
	}
	p := &store.PermissionProfile{
		ID:               id,
		Label:            strings.TrimSpace(in.Label),
		Description:      strings.TrimSpace(in.Description),
		Groups:           groups,
		PublicSelectable: in.PublicSelectable,
		PublicLabel:      strings.TrimSpace(in.PublicLabel),
		SortOrder:        in.SortOrder,
	}
	if err := s.store.UpdateProfile(ctx, p); err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, actor, "profile.update", "profile", p.ID, map[string]any{"groups": p.Groups})
	return p, nil
}

// DeleteProfile removes a profile (store returns ErrConflict if referenced).
func (s *Service) DeleteProfile(ctx context.Context, id string, actor store.AdminUser) error {
	if err := s.store.DeleteProfile(ctx, id); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, actor, "profile.delete", "profile", id, nil)
	return nil
}

// normalizeGroups trims, drops empties, and de-duplicates a group list.
func normalizeGroups(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, g := range in {
		if g = strings.TrimSpace(g); g != "" && !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	return out
}

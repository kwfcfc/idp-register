// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/audit"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/config"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/provisioner"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/store"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/token"

	"github.com/google/uuid"
)

type fakeProvisioner struct {
	groups       []provisioner.Group
	usersByEmail map[string]*provisioner.User
	created      []provisioner.NewUser
	failCreate   bool
	failUsername bool // create succeeds but the username step fails (partial success)
	failInit     bool
}

func newFakeProvisioner() *fakeProvisioner {
	return &fakeProvisioner{
		groups: []provisioner.Group{
			{ID: "g-matrix", Name: "svc:matrix:user"},
			{ID: "g-forgejo", Name: "svc:forgejo:user"},
			{ID: "g-admin", Name: "infra:admin"},
		},
		usersByEmail: map[string]*provisioner.User{},
	}
}

func (f *fakeProvisioner) Name() string { return "fake" }

func (f *fakeProvisioner) ListGroups(context.Context) ([]provisioner.Group, error) {
	return append([]provisioner.Group(nil), f.groups...), nil
}

func (f *fakeProvisioner) FindUserByEmail(_ context.Context, email string) (*provisioner.User, bool, error) {
	u, ok := f.usersByEmail[strings.ToLower(email)]
	return u, ok, nil
}

func (f *fakeProvisioner) CreateUser(_ context.Context, in provisioner.NewUser) (string, error) {
	if f.failCreate {
		return "", errors.New("forced create failure")
	}
	userID := fmt.Sprintf("fake-user-%d", len(f.created)+1)
	f.created = append(f.created, in)
	f.usersByEmail[strings.ToLower(in.Email)] = &provisioner.User{
		ID:     userID,
		Email:  in.Email,
		Groups: append([]string(nil), in.Groups...),
	}
	if f.failUsername {
		return userID, fmt.Errorf("%w: forced username failure", provisioner.ErrUsernameNotSet)
	}
	return userID, nil
}

func (f *fakeProvisioner) InitCredentials(context.Context, string, string) (*string, error) {
	if f.failInit {
		return nil, errors.New("forced init failure")
	}
	return nil, nil
}

type appHarness struct {
	ctx    context.Context
	store  *store.Store
	prov   *fakeProvisioner
	apps   *Service
	tokens *token.Service
	actor  store.AdminUser
}

func newAppHarness(t *testing.T) *appHarness {
	t.Helper()
	ctx := context.Background()
	cfg := &config.Config{
		DBDriver: config.DriverSQLite,
		DBDSN:    fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString()),
	}
	st, err := store.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	prov := newFakeProvisioner()
	log := audit.New(st)
	return &appHarness{
		ctx:    ctx,
		store:  st,
		prov:   prov,
		apps:   New(st, prov, log, []string{"infra:admin"}),
		tokens: token.New(st, log),
		actor:  store.AdminUser{Sub: "admin", Email: "admin@example.test", DisplayName: "Admin"},
	}
}

func (h *appHarness) createProfile(t *testing.T, id string, groups []string, public bool) {
	t.Helper()
	_, err := h.apps.CreateProfile(h.ctx, ProfileInput{
		ID:               id,
		Label:            strings.ToUpper(id[:1]) + id[1:],
		Description:      "test profile",
		Groups:           groups,
		PublicSelectable: public,
		PublicLabel:      id + " service",
		SortOrder:        10,
	}, h.actor)
	if err != nil {
		t.Fatalf("create profile %s: %v", id, err)
	}
}

func (h *appHarness) mintToken(t *testing.T, code, profileID string, uses *int64, email *string, expiry *int64) *store.RegistrationToken {
	t.Helper()
	tok, err := h.tokens.Mint(h.ctx, token.MintParams{
		Token:           code,
		UsesAllowed:     uses,
		ExpiryTime:      expiry,
		EmailConstraint: email,
		ProfileID:       &profileID,
	}, h.actor)
	if err != nil {
		t.Fatalf("mint token %s: %v", code, err)
	}
	return tok
}

func TestM3ManualReviewPathProvisionsApprovedProfileGroups(t *testing.T) {
	h := newAppHarness(t)
	h.createProfile(t, "matrix", []string{"svc:matrix:user"}, true)

	services, err := h.apps.PublicServices(h.ctx)
	if err != nil {
		t.Fatalf("public services: %v", err)
	}
	if len(services) != 1 || services[0].ID != "matrix" || services[0].Label != "matrix service" {
		t.Fatalf("unexpected public services: %+v", services)
	}

	result, err := h.apps.Submit(h.ctx, SubmitInput{
		Email:      "manual@example.test",
		Username:   "manual",
		ReviewText: "please approve",
		Services:   []string{"matrix", "missing", "matrix"},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if result.AutoApproved {
		t.Fatalf("manual-review submission must not auto-approve")
	}

	app, err := h.apps.Get(h.ctx, result.ApplicationID)
	if err != nil {
		t.Fatalf("get application: %v", err)
	}
	if app.Status != store.StatusPending {
		t.Fatalf("want pending, got %s", app.Status)
	}
	if !reflect.DeepEqual(app.RequestedServices, []string{"matrix"}) {
		t.Fatalf("requested services not filtered: %+v", app.RequestedServices)
	}
	if app.ApprovedProfileID != nil {
		t.Fatalf("manual-review application should not have approved profile yet")
	}

	if err := h.apps.Approve(h.ctx, app.ID, "matrix", "ok", h.actor); err != nil {
		t.Fatalf("approve: %v", err)
	}
	app, err = h.apps.Get(h.ctx, app.ID)
	if err != nil {
		t.Fatalf("get approved application: %v", err)
	}
	if app.Status != store.StatusApproved || app.ProviderUserID == nil {
		t.Fatalf("application was not approved with provider id: %+v", app)
	}
	if len(h.prov.created) != 1 {
		t.Fatalf("want one provisioned user, got %d", len(h.prov.created))
	}
	if !reflect.DeepEqual(h.prov.created[0].Groups, []string{"svc:matrix:user"}) {
		t.Fatalf("provisioned wrong groups: %+v", h.prov.created[0].Groups)
	}
}

func TestM3InviteAutoApprovalOverridesRequestedServicesAndCompletesToken(t *testing.T) {
	h := newAppHarness(t)
	h.createProfile(t, "matrix", []string{"svc:matrix:user"}, true)
	h.createProfile(t, "forgejo", []string{"svc:forgejo:user"}, true)
	uses := int64(1)
	tok := h.mintToken(t, "auto-code", "matrix", &uses, nil, nil)

	result, err := h.apps.Submit(h.ctx, SubmitInput{
		Email:      "auto@example.test",
		Username:   "auto",
		ReviewText: "invite path",
		InviteCode: tok.Token,
		Services:   []string{"forgejo"},
	})
	if err != nil {
		t.Fatalf("submit with invite: %v", err)
	}
	if !result.AutoApproved {
		t.Fatalf("valid invite should auto-approve")
	}

	app, err := h.apps.Get(h.ctx, result.ApplicationID)
	if err != nil {
		t.Fatalf("get application: %v", err)
	}
	if app.Status != store.StatusApproved {
		t.Fatalf("want approved, got %s", app.Status)
	}
	if app.ApprovedProfileID == nil || *app.ApprovedProfileID != "matrix" {
		t.Fatalf("approved profile not token-bound: %+v", app.ApprovedProfileID)
	}
	if !reflect.DeepEqual(app.RequestedServices, []string{"matrix"}) {
		t.Fatalf("requested services should be token-bound profile only: %+v", app.RequestedServices)
	}
	if len(h.prov.created) != 1 || !reflect.DeepEqual(h.prov.created[0].Groups, []string{"svc:matrix:user"}) {
		t.Fatalf("invite provisioned wrong groups: %+v", h.prov.created)
	}

	gotToken, err := h.store.GetTokenByID(h.ctx, tok.ID)
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if gotToken.Pending != 0 || gotToken.Completed != 1 {
		t.Fatalf("want pending=0 completed=1, got pending=%d completed=%d", gotToken.Pending, gotToken.Completed)
	}
}

func TestM3InvalidInvitesFallBackToPendingWithoutConsumingUses(t *testing.T) {
	h := newAppHarness(t)
	h.createProfile(t, "matrix", []string{"svc:matrix:user"}, true)
	uses := int64(1)
	expiredAt := int64(1)
	allowedEmail := "allowed@example.test"

	disabled := h.mintToken(t, "disabled-code", "matrix", &uses, nil, nil)
	if ok, err := h.tokens.SetActive(h.ctx, disabled.ID, false, h.actor); err != nil || !ok {
		t.Fatalf("disable token: ok=%v err=%v", ok, err)
	}
	exhausted := h.mintToken(t, "exhausted-code", "matrix", &uses, nil, nil)
	if ok, err := h.store.ReserveToken(h.ctx, exhausted.Token, 0); err != nil || !ok {
		t.Fatalf("pre-reserve exhausted token: ok=%v err=%v", ok, err)
	}
	emailBound := h.mintToken(t, "email-code", "matrix", &uses, &allowedEmail, nil)
	expired := h.mintToken(t, "expired-code", "matrix", &uses, nil, &expiredAt)

	cases := []struct {
		name      string
		code      string
		tokenID   string
		wantPend  int64
		wantDone  int64
		wantEmail string
	}{
		{name: "unknown", code: "unknown-code", wantEmail: "unknown@example.test"},
		{name: "disabled", code: disabled.Token, tokenID: disabled.ID, wantEmail: "disabled@example.test"},
		{name: "exhausted", code: exhausted.Token, tokenID: exhausted.ID, wantPend: 1, wantEmail: "exhausted@example.test"},
		{name: "wrong-email", code: emailBound.Token, tokenID: emailBound.ID, wantEmail: "wrong@example.test"},
		{name: "expired", code: expired.Token, tokenID: expired.ID, wantEmail: "expired@example.test"},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := h.apps.Submit(h.ctx, SubmitInput{
				Email:      tc.wantEmail,
				Username:   fmt.Sprintf("invalid%d", i),
				InviteCode: tc.code,
				Services:   []string{"matrix"},
			})
			if err != nil {
				t.Fatalf("submit: %v", err)
			}
			if result.AutoApproved {
				t.Fatalf("invalid invite must fall back to manual review")
			}
			app, err := h.apps.Get(h.ctx, result.ApplicationID)
			if err != nil {
				t.Fatalf("get application: %v", err)
			}
			if app.Status != store.StatusPending || app.TokenID != nil {
				t.Fatalf("invalid invite should produce pending application without token id: %+v", app)
			}
			if tc.tokenID != "" {
				got, err := h.store.GetTokenByID(h.ctx, tc.tokenID)
				if err != nil {
					t.Fatalf("get token: %v", err)
				}
				if got.Pending != tc.wantPend || got.Completed != tc.wantDone {
					t.Fatalf("token counters changed: got pending=%d completed=%d", got.Pending, got.Completed)
				}
			}
		})
	}
	if len(h.prov.created) != 0 {
		t.Fatalf("invalid invites should not provision users: %+v", h.prov.created)
	}
}

func TestM3ProvisioningFailureCanRetryOrRejectHeldInviteReservation(t *testing.T) {
	h := newAppHarness(t)
	h.createProfile(t, "matrix", []string{"svc:matrix:user"}, true)
	h.createProfile(t, "forgejo", []string{"svc:forgejo:user"}, true)
	uses := int64(2)
	retryToken := h.mintToken(t, "retry-code", "matrix", &uses, nil, nil)

	h.prov.failCreate = true
	retryResult, err := h.apps.Submit(h.ctx, SubmitInput{
		Email:      "retry@example.test",
		Username:   "retry",
		InviteCode: retryToken.Token,
	})
	if err != nil {
		t.Fatalf("submit retry case: %v", err)
	}
	retryApp, err := h.apps.Get(h.ctx, retryResult.ApplicationID)
	if err != nil {
		t.Fatalf("get failed app: %v", err)
	}
	if retryApp.Status != store.StatusProvisioningFailed || retryApp.ProvisioningError == nil {
		t.Fatalf("want provisioning_failed with error, got %+v", retryApp)
	}
	assertTokenCounters(t, h, retryToken.ID, 1, 0)

	h.prov.failCreate = false
	if err := h.apps.Approve(h.ctx, retryApp.ID, "forgejo", "retry", h.actor); err != nil {
		t.Fatalf("retry approve: %v", err)
	}
	retryApp, err = h.apps.Get(h.ctx, retryApp.ID)
	if err != nil {
		t.Fatalf("get retried app: %v", err)
	}
	if retryApp.Status != store.StatusApproved {
		t.Fatalf("retry did not approve application: %+v", retryApp)
	}
	if retryApp.ApprovedProfileID == nil || *retryApp.ApprovedProfileID != "matrix" {
		t.Fatalf("invite retry changed approved profile: %+v", retryApp.ApprovedProfileID)
	}
	if want := []string{"svc:matrix:user"}; !reflect.DeepEqual(want, h.prov.created[0].Groups) {
		t.Fatalf("invite retry should keep token-bound profile groups: got %+v want %+v", h.prov.created[0].Groups, want)
	}
	assertTokenCounters(t, h, retryToken.ID, 0, 1)

	rejectToken := h.mintToken(t, "reject-code", "matrix", &uses, nil, nil)
	h.prov.failCreate = true
	rejectResult, err := h.apps.Submit(h.ctx, SubmitInput{
		Email:      "reject@example.test",
		Username:   "reject",
		InviteCode: rejectToken.Token,
	})
	if err != nil {
		t.Fatalf("submit reject case: %v", err)
	}
	assertTokenCounters(t, h, rejectToken.ID, 1, 0)

	ok, err := h.apps.Decide(h.ctx, rejectResult.ApplicationID, store.StatusRejected, "no longer needed", h.actor)
	if err != nil || !ok {
		t.Fatalf("reject failed: ok=%v err=%v", ok, err)
	}
	rejectApp, err := h.apps.Get(h.ctx, rejectResult.ApplicationID)
	if err != nil {
		t.Fatalf("get rejected app: %v", err)
	}
	if rejectApp.Status != store.StatusRejected {
		t.Fatalf("want rejected, got %+v", rejectApp)
	}
	assertTokenCounters(t, h, rejectToken.ID, 0, 0)
}

// TestUsernameFailureIsPartialSuccess: when the target IdP creates the account
// but cannot apply the preferred username, the provision must still be approved
// (the account exists; a retry would hit "email already exists") with the
// warning recorded in the audit log.
func TestUsernameFailureIsPartialSuccess(t *testing.T) {
	h := newAppHarness(t)
	h.createProfile(t, "matrix", []string{"svc:matrix:user"}, true)
	h.prov.failUsername = true

	result, err := h.apps.Submit(h.ctx, SubmitInput{
		Email:    "partial@example.test",
		Username: "partial",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := h.apps.Approve(h.ctx, result.ApplicationID, "matrix", "", h.actor); err != nil {
		t.Fatalf("approve should tolerate a username-only failure, got: %v", err)
	}

	app, err := h.apps.Get(h.ctx, result.ApplicationID)
	if err != nil {
		t.Fatalf("get application: %v", err)
	}
	if app.Status != store.StatusApproved || app.ProviderUserID == nil {
		t.Fatalf("want approved with provider id, got %+v", app)
	}

	entries, err := h.store.ListAudit(h.ctx, 50)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	var warned bool
	for _, e := range entries {
		if e.Action == "application.provision.warn" && e.TargetID == app.ID {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("partial success must leave an application.provision.warn audit entry, got %+v", entries)
	}
}

func assertTokenCounters(t *testing.T, h *appHarness, tokenID string, pending, completed int64) {
	t.Helper()
	got, err := h.store.GetTokenByID(h.ctx, tokenID)
	if err != nil {
		t.Fatalf("get token %s: %v", tokenID, err)
	}
	if got.Pending != pending || got.Completed != completed {
		t.Fatalf("want token pending=%d completed=%d, got pending=%d completed=%d", pending, completed, got.Pending, got.Completed)
	}
}

// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"context"
	"testing"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/config"

	"github.com/google/uuid"
)

// openTestStore opens a fresh in-memory SQLite store with schema + seeds applied.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	cfg := &config.Config{DBDriver: config.DriverSQLite, DBDSN: "file::memory:?cache=shared"}
	s, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSchemaAndSeeds(t *testing.T) {
	s := openTestStore(t)
	profiles, err := s.ListProfiles(context.Background())
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(profiles) != 3 {
		t.Fatalf("want 3 seeded profiles, got %d", len(profiles))
	}
	dev, err := s.GetProfile(context.Background(), "developer")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if len(dev.Groups) != 3 {
		t.Fatalf("developer should have 3 groups, got %v", dev.Groups)
	}
}

// TestProfileCRUD covers create/update/delete and the public-form views.
func TestProfileCRUD(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	p := &PermissionProfile{
		ID: "matrix-only", Label: "Matrix", Description: "chat",
		Groups: []string{"svc:matrix:user"}, PublicSelectable: true,
		PublicLabel: "Matrix 聊天", SortOrder: 5,
	}
	if err := s.CreateProfile(ctx, p); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.CreateProfile(ctx, p); err != ErrConflict {
		t.Fatalf("duplicate create should be ErrConflict, got %v", err)
	}

	got, err := s.GetProfile(ctx, "matrix-only")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.PublicSelectable || got.PublicLabel != "Matrix 聊天" || got.SortOrder != 5 {
		t.Fatalf("public fields not persisted: %+v", got)
	}

	// Public views expose the option but never the groups.
	svcs, err := s.ListPublicServices(ctx)
	if err != nil {
		t.Fatalf("list public: %v", err)
	}
	if len(svcs) != 1 || svcs[0].ID != "matrix-only" || svcs[0].Label != "Matrix 聊天" {
		t.Fatalf("unexpected public services: %+v", svcs)
	}
	ids, err := s.PublicServiceIDs(ctx)
	if err != nil || !ids["matrix-only"] {
		t.Fatalf("public ids missing matrix-only: %v %v", ids, err)
	}

	// Update toggles it off.
	got.PublicSelectable = false
	got.Label = "Matrix v2"
	if err := s.UpdateProfile(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	if svcs, _ := s.ListPublicServices(ctx); len(svcs) != 0 {
		t.Fatalf("profile should no longer be public, got %+v", svcs)
	}

	// Updating a missing profile is ErrNotFound.
	if err := s.UpdateProfile(ctx, &PermissionProfile{ID: "ghost", Label: "x"}); err != ErrNotFound {
		t.Fatalf("update missing should be ErrNotFound, got %v", err)
	}

	// Delete blocked while a token references the profile.
	tok := &RegistrationToken{
		ID: uuid.NewString(), Token: "ref", Active: true,
		ProfileID: strptr("matrix-only"), CreatedBySub: "a", CreatedByEmail: "a@x",
	}
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := s.DeleteProfile(ctx, "matrix-only"); err != ErrConflict {
		t.Fatalf("delete of referenced profile should be ErrConflict, got %v", err)
	}

	// Remove the reference, then delete succeeds; deleting again is ErrNotFound.
	if _, err := s.DeleteToken(ctx, tok.ID); err != nil {
		t.Fatalf("delete token: %v", err)
	}
	if err := s.DeleteProfile(ctx, "matrix-only"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.DeleteProfile(ctx, "matrix-only"); err != ErrNotFound {
		t.Fatalf("delete missing should be ErrNotFound, got %v", err)
	}
}

func strptr(s string) *string { return &s }

// TestReserveTokenAtomic verifies the use-count guard: a 2-use token can be
// reserved exactly twice, never a third time.
func TestReserveTokenAtomic(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	uses := int64(2)
	tok := &RegistrationToken{
		ID: uuid.NewString(), Token: "test-code-123", UsesAllowed: &uses,
		Active: true, ProfileID: strptr("developer"), CreatedBySub: "admin", CreatedByEmail: "a@example.com",
	}
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatalf("create token: %v", err)
	}

	ok1, _ := s.ReserveToken(ctx, "test-code-123", 0)
	ok2, _ := s.ReserveToken(ctx, "test-code-123", 0)
	ok3, _ := s.ReserveToken(ctx, "test-code-123", 0)
	if !ok1 || !ok2 {
		t.Fatalf("first two reservations should succeed (%v,%v)", ok1, ok2)
	}
	if ok3 {
		t.Fatalf("third reservation must fail once uses are exhausted")
	}

	// Completing a use should not reopen capacity (pending-- but completed++).
	if err := s.CompleteTokenUse(ctx, tok.ID); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if ok, _ := s.ReserveToken(ctx, "test-code-123", 0); ok {
		t.Fatalf("reservation must still fail after a completion")
	}

	got, err := s.GetTokenByID(ctx, tok.ID)
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if got.Pending != 1 || got.Completed != 1 {
		t.Fatalf("want pending=1 completed=1, got pending=%d completed=%d", got.Pending, got.Completed)
	}
}

// TestExpiredTokenRejected verifies the expiry guard in reservation.
func TestExpiredTokenRejected(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	exp := int64(1000)
	tok := &RegistrationToken{
		ID: uuid.NewString(), Token: "expired", ExpiryTime: &exp,
		Active: true, ProfileID: strptr("developer"), CreatedBySub: "admin", CreatedByEmail: "a@example.com",
	}
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatalf("create: %v", err)
	}
	if ok, _ := s.ReserveToken(ctx, "expired", 2000); ok {
		t.Fatalf("expired token must not reserve")
	}
}

func TestCreateApplicationPersistsApprovedProfile(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	profileID := "developer"
	app := &Application{
		ID:                uuid.NewString(),
		Email:             "auto@example.test",
		Username:          "auto",
		Status:            StatusProvisioning,
		ApprovedProfileID: &profileID,
		RequestedServices: []string{profileID},
	}
	if err := s.CreateApplication(ctx, app, "auto@example.test", "auto"); err != nil {
		t.Fatalf("create application: %v", err)
	}

	got, err := s.GetApplication(ctx, app.ID)
	if err != nil {
		t.Fatalf("get application: %v", err)
	}
	if got.ApprovedProfileID == nil || *got.ApprovedProfileID != profileID {
		t.Fatalf("approved profile was not persisted: %+v", got.ApprovedProfileID)
	}
}

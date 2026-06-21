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

// TestReserveTokenAtomic verifies the use-count guard: a 2-use token can be
// reserved exactly twice, never a third time.
func TestReserveTokenAtomic(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	uses := int64(2)
	tok := &RegistrationToken{
		ID: uuid.NewString(), Token: "test-code-123", UsesAllowed: &uses,
		Active: true, CreatedBySub: "admin", CreatedByEmail: "a@example.com",
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
		Active: true, CreatedBySub: "admin", CreatedByEmail: "a@example.com",
	}
	if err := s.CreateToken(ctx, tok); err != nil {
		t.Fatalf("create: %v", err)
	}
	if ok, _ := s.ReserveToken(ctx, "expired", 2000); ok {
		t.Fatalf("expired token must not reserve")
	}
}

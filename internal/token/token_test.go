// SPDX-License-Identifier: GPL-3.0-or-later

package token

import (
	"context"
	"errors"
	"testing"

	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/audit"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/config"
	"forgejo.goba.ip-dynamic.org/gobro/idp-register/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	cfg := &config.Config{DBDriver: config.DriverSQLite, DBDSN: "file::memory:?cache=shared"}
	s, err := store.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestMintRequiresProfile(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	svc := New(s, audit.New(s))
	actor := store.AdminUser{Sub: "admin", Email: "admin@example.test"}

	if _, err := svc.Mint(ctx, MintParams{}, actor); !errors.Is(err, ErrProfileRequired) {
		t.Fatalf("mint without profile should be ErrProfileRequired, got %v", err)
	}

	missing := "missing"
	if _, err := svc.Mint(ctx, MintParams{ProfileID: &missing}, actor); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("mint with missing profile should be ErrNotFound, got %v", err)
	}

	profileID := "developer"
	tok, err := svc.Mint(ctx, MintParams{ProfileID: &profileID}, actor)
	if err != nil {
		t.Fatalf("mint with profile: %v", err)
	}
	if tok.ProfileID == nil || *tok.ProfileID != profileID {
		t.Fatalf("profile was not stored on token: %+v", tok.ProfileID)
	}
}

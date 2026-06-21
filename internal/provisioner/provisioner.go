// SPDX-License-Identifier: GPL-3.0-or-later

// Package provisioner abstracts the *target* identity provider where end users
// are created (distinct from the admin-auth OIDC RP). Rauthy is the only
// implementation today; Kanidm is planned. All provider-specific API calls live
// inside an implementation — nothing Rauthy-specific leaks past this interface
// (AGENTS.md invariant #2).
package provisioner

import (
	"context"
	"errors"
)

// ErrUserExists indicates the target IdP already has a user with that email.
var ErrUserExists = errors.New("user already exists in target idp")

// NewUser is the provider-neutral input for creating a user.
type NewUser struct {
	Email    string
	Username string   // preferred username; may be empty
	Groups   []string // resolved server-side from a permission profile
	Language string
	Timezone string
}

// User is the provider-neutral view of an IdP user.
type User struct {
	ID     string
	Email  string
	Groups []string
}

// Group is the provider-neutral view of an assignable IdP group. Permission
// profiles are composed from these (see docs/DECISIONS.md ADR-0012); the app
// never invents group names, it only references what the target IdP defines.
type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Provisioner creates users in a target IdP and kicks off credential setup.
type Provisioner interface {
	// Name identifies the implementation (e.g. "rauthy"), for logs/audit.
	Name() string

	// ListGroups returns the groups defined in the target IdP (the catalog a
	// permission profile may draw from). Read-only.
	ListGroups(ctx context.Context) ([]Group, error)

	// FindUserByEmail returns the user and true if one exists, false if not.
	FindUserByEmail(ctx context.Context, email string) (*User, bool, error)

	// CreateUser creates the user and returns its IdP id. Some providers may
	// send the initial credential setup email as part of creation.
	CreateUser(ctx context.Context, in NewUser) (userID string, err error)

	// InitCredentials triggers credential setup (password/passkey) when the
	// provider did not already do it during CreateUser. Some IdPs send their own
	// activation email and return nil; others return a reset link this service
	// must deliver. Hence the optional *string return.
	InitCredentials(ctx context.Context, userID, email string) (resetLink *string, err error)
}

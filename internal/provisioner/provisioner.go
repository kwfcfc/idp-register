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

// Provisioner creates users in a target IdP and kicks off credential setup.
type Provisioner interface {
	// Name identifies the implementation (e.g. "rauthy"), for logs/audit.
	Name() string

	// FindUserByEmail returns the user and true if one exists, false if not.
	FindUserByEmail(ctx context.Context, email string) (*User, bool, error)

	// CreateUser creates the user silently (no email) and returns its IdP id.
	CreateUser(ctx context.Context, in NewUser) (userID string, err error)

	// InitCredentials triggers credential setup (password/passkey) for the
	// user. Some IdPs send their own activation email and return nil (Rauthy);
	// others return a reset link this service must deliver (Kanidm). Hence the
	// optional *string return.
	InitCredentials(ctx context.Context, userID, email string) (resetLink *string, err error)
}

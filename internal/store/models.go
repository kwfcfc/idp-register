// SPDX-License-Identifier: GPL-3.0-or-later

package store

import "errors"

// ErrNotFound is returned by repository lookups when no row matches.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on a constraint clash: a duplicate primary key, or a
// delete blocked because another row still references the target.
var ErrConflict = errors.New("conflict")

// Application lifecycle states (mirror the CHECK list in the schema design).
const (
	StatusPending            = "pending"
	StatusProvisioning       = "provisioning"
	StatusProvisioningFailed = "provisioning_failed"
	StatusApproved           = "approved"
	StatusRejected           = "rejected"
)

// AdminUser is the authenticated admin identity carried in a session.
type AdminUser struct {
	Sub         string   `json:"sub"`
	Email       string   `json:"email"`
	DisplayName string   `json:"displayName"`
	Groups      []string `json:"groups"`
}

// PermissionProfile is a named, server-side bundle of target-IdP groups. The
// public form may only reference a profile by id; it can never set groups.
type PermissionProfile struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Groups      []string `json:"groups"`
	// PublicSelectable offers this profile as a choice in the public form;
	// PublicLabel is the user-facing text (falls back to Label); SortOrder is
	// the ascending display order. See docs/DECISIONS.md ADR-0012.
	PublicSelectable bool   `json:"publicSelectable"`
	PublicLabel      string `json:"publicLabel"`
	SortOrder        int64  `json:"sortOrder"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
}

// RegistrationToken is a plaintext, Synapse-aligned invite code.
type RegistrationToken struct {
	ID              string  `json:"id"`
	Token           string  `json:"token"` // plaintext; omit from logs, never from list-to-admin
	UsesAllowed     *int64  `json:"usesAllowed"`
	Pending         int64   `json:"pending"`
	Completed       int64   `json:"completed"`
	ExpiryTime      *int64  `json:"expiryTime"` // epoch ms
	Active          bool    `json:"active"`
	EmailConstraint *string `json:"emailConstraint"`
	ProfileID       *string `json:"profileId"`
	Note            string  `json:"note"`
	CreatedBySub    string  `json:"createdBySub"`
	CreatedByEmail  string  `json:"createdByEmail"`
	CreatedAt       int64   `json:"createdAt"`
}

// Valid reports whether the token may still be used at time nowMS.
func (t *RegistrationToken) Valid(now int64) bool {
	if !t.Active {
		return false
	}
	if t.ExpiryTime != nil && *t.ExpiryTime <= now {
		return false
	}
	if t.UsesAllowed != nil && t.Pending+t.Completed >= *t.UsesAllowed {
		return false
	}
	return true
}

// Application is a submitted registration request moving through review.
type Application struct {
	ID                string   `json:"id"`
	TokenID           *string  `json:"tokenId"`
	Email             string   `json:"email"`
	Username          string   `json:"username"`
	ReviewText        string   `json:"reviewText"`
	RequestedServices []string `json:"requestedServices"`
	Status            string   `json:"status"`
	CaptchaProvider   *string  `json:"captchaProvider"`
	SubmittedIP       *string  `json:"submittedIp"`
	ApprovedProfileID *string  `json:"approvedProfileId"`
	ProviderUserID    *string  `json:"providerUserId"`
	ProvisioningError *string  `json:"provisioningError"`
	DecisionNote      *string  `json:"decisionNote"`
	ReviewedAt        *int64   `json:"reviewedAt"`
	ReviewedBySub     *string  `json:"reviewedBySub"`
	ReviewedByEmail   *string  `json:"reviewedByEmail"`
	CreatedAt         int64    `json:"createdAt"`
	UpdatedAt         int64    `json:"updatedAt"`
}

// AuditEntry is one admin action recorded in the audit log.
type AuditEntry struct {
	ID         string `json:"id"`
	ActorSub   string `json:"actorSub"`
	ActorEmail string `json:"actorEmail"`
	Action     string `json:"action"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Details    string `json:"details"` // raw JSON
	CreatedAt  int64  `json:"createdAt"`
}

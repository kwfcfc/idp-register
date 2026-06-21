// SPDX-License-Identifier: GPL-3.0-or-later
// These shapes mirror the JSON emitted by the Go backend (internal/store/models.go).
// All timestamps are epoch milliseconds (integers); nullable fields are `| null`.

export type AdminUser = {
  sub: string;
  email: string;
  displayName: string;
  groups: string[];
};

export type ApplicationStatus =
  | 'pending'
  | 'provisioning'
  | 'provisioning_failed'
  | 'approved'
  | 'rejected'
  | 'needs_changes';

export type PermissionProfile = {
  id: string;
  label: string;
  description: string;
  groups: string[];
  // Public service-selector fields (ADR-0012). `publicSelectable` offers the
  // profile in the public form; `publicLabel` is the user-facing text (falls
  // back to `label`); `sortOrder` is the ascending display order.
  publicSelectable: boolean;
  publicLabel: string;
  sortOrder: number;
  createdAt: number;
  updatedAt: number;
};

// An assignable group from the target IdP catalog (GET /api/admin/groups).
// Profiles are composed only from these (ADR-0012); the app never invents names.
export type Group = {
  id: string;
  name: string;
};

// The anonymous-safe public form configuration (GET /api/form). Service options
// carry no IdP group names — only an id the form echoes back as `services`.
export type PublicService = {
  id: string;
  label: string;
  description: string;
};

export type FormConfig = {
  services: PublicService[];
  selectionMode: 'single'; // multi-select deferred (ADR-0012)
};

export type Application = {
  id: string;
  tokenId: string | null;
  email: string;
  username: string;
  reviewText: string;
  requestedServices: string[] | null;
  status: ApplicationStatus;
  captchaProvider: string | null;
  submittedIp: string | null;
  approvedProfileId: string | null;
  providerUserId: string | null;
  provisioningError: string | null;
  decisionNote: string | null;
  reviewedAt: number | null;
  reviewedBySub: string | null;
  reviewedByEmail: string | null;
  createdAt: number;
  updatedAt: number;
};

// Plaintext, Synapse-aligned invite code (ADR-0004). `usesAllowed`/`expiryTime`
// are null when unlimited / never-expires.
export type RegistrationToken = {
  id: string;
  token: string;
  usesAllowed: number | null;
  pending: number;
  completed: number;
  expiryTime: number | null;
  active: boolean;
  emailConstraint: string | null;
  profileId: string | null;
  note: string;
  createdBySub: string;
  createdByEmail: string;
  createdAt: number;
};

export type AuditEntry = {
  id: string;
  actorSub: string;
  actorEmail: string;
  action: string;
  targetType: string;
  targetId: string;
  details: string;
  createdAt: number;
};

// Derived, client-side status for an invite code (the backend returns raw
// counters; validity mirrors RegistrationToken.Valid in Go).
export type TokenStatus = 'active' | 'revoked' | 'exhausted' | 'expired';

export function tokenStatus(t: RegistrationToken, now: number): TokenStatus {
  if (!t.active) return 'revoked';
  if (t.expiryTime !== null && t.expiryTime <= now) return 'expired';
  if (t.usesAllowed !== null && t.pending + t.completed >= t.usesAllowed) return 'exhausted';
  return 'active';
}

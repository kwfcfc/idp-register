<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Logical Schema

This document describes the database shape independent of a specific SQL dialect. The
authoritative runtime schemas are:

- `internal/store/schema_sqlite.sql`
- `internal/store/schema_pg.sql`

The operator-facing initial deployment SQL is:

- `migrations/sqlite/001_init.sql`
- `migrations/postgres/001_init.sql`

## Portability Rules

The schema deliberately avoids provider-specific rich types so the same Go store layer can
work against SQLite and PostgreSQL.

| Logical type | SQLite | PostgreSQL | Rule |
|---|---|---|---|
| ID / UUID | `TEXT` | `TEXT` | Generated in Go, never by database defaults. |
| Timestamp | `INTEGER` | `BIGINT` | Epoch milliseconds generated in Go. |
| Boolean | `INTEGER` | `INTEGER` | `0` = false, `1` = true. |
| String list | `TEXT` | `TEXT` | JSON array encoded/decoded in Go. |
| JSON object | `TEXT` | `TEXT` | JSON object encoded/decoded in Go. |
| IP address | `TEXT` | `TEXT` | Do not use PostgreSQL `INET`. |

Partial indexes are allowed because both target engines support them.

## Tables

### `permission_profiles`

Server-side bundles of target-IdP groups. The public form may reference a profile by id,
but it never submits raw IdP groups.

| Column | Meaning |
|---|---|
| `id` | Stable profile id. |
| `label` | Admin-facing name. |
| `description` | Admin-facing description and public-service fallback text. |
| `groups` | JSON array of target-IdP group names. |
| `public_selectable` | `1` when offered on the public form. |
| `public_label` | Public-facing service label; blank falls back to `label`. |
| `sort_order` | Ascending public display order. |
| `created_at`, `updated_at` | Epoch milliseconds. |

### `registration_tokens`

Plaintext Synapse-style invite codes. Codes must not be logged.

| Column | Meaning |
|---|---|
| `id` | Go-generated id. |
| `token` | Plaintext invite code, unique. |
| `uses_allowed` | Maximum successful uses; `NULL` means unlimited. |
| `pending` | Reserved but not completed uses. |
| `completed` | Successfully provisioned uses. |
| `expiry_time` | Expiry epoch milliseconds; `NULL` means never. |
| `active` | Manual enable/disable switch. |
| `email_constraint` | Optional lowercased email binding. |
| `profile_id` | Required permission profile for auto-approval. |
| `note` | Admin note. |
| `created_by_sub`, `created_by_email` | Admin identity. |
| `created_at` | Epoch milliseconds. |

Valid invite use reserves capacity with `pending++`. Successful provisioning completes it
with `pending--, completed++`. Rejection of a failed auto-approval releases the reservation
with `pending--`.

### `applications`

Submitted registration applications.

| Column | Meaning |
|---|---|
| `id` | Go-generated id. |
| `token_id` | Invite token used, if any. |
| `email`, `email_normalized` | Submitted email and normalized lookup key. |
| `username`, `username_normalized` | Optional submitted username and normalized lookup key. |
| `review_text` | Applicant-provided note for administrators. |
| `requested_services` | JSON array of public profile ids requested by the applicant. |
| `status` | `pending`, `provisioning`, `provisioning_failed`, `approved`, or `rejected`. |
| `captcha_provider`, `captcha_verified_at` | Anti-abuse metadata. |
| `submitted_ip` | Submitted request IP as text. |
| `approved_profile_id` | Admin-approved or invite-bound profile. |
| `provider_user_id` | Target-IdP user id after provisioning. |
| `provisioning_error` | Last provisioning failure message. |
| `decision_note` | Admin decision note. |
| `reviewed_at`, `reviewed_by_sub`, `reviewed_by_email` | Admin review metadata. |
| `created_at`, `updated_at` | Epoch milliseconds. |

Unique partial indexes enforce one live application per email and per non-empty username
while status is `pending`, `provisioning`, `provisioning_failed`, or `approved`.

### `admin_sessions`

Opaque admin login sessions.

| Column | Meaning |
|---|---|
| `id` | Go-generated session id. |
| `token_digest` | Digest of the opaque cookie value. |
| `subject`, `email`, `display_name` | Admin identity from OIDC. |
| `groups` | JSON array of OIDC groups. |
| `expires_at`, `created_at`, `last_seen_at` | Epoch milliseconds. |

### `audit_log`

Append-only admin/system action log.

| Column | Meaning |
|---|---|
| `id` | Go-generated id. |
| `actor_sub`, `actor_email` | Actor identity. |
| `action` | Action string such as `token.create` or `application.approve`. |
| `target_type`, `target_id` | Audited target. |
| `details` | JSON object encoded as text. |
| `created_at` | Epoch milliseconds. |

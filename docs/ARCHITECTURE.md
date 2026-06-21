<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Architecture

## Goals & non-goals

**Goals.** A small self-hosted service that (1) collects registration applications via a
public form with anti-abuse, (2) supports **invite codes** that auto-approve (limited by
use-count and expiry, à la Synapse registration tokens), (3) gives admins a review panel
for everything else, and (4) on approval provisions the user into an OIDC IdP and lets the
IdP deliver a single activation email. Self-hostable as one OCI image behind Nginx/Cloudflare.

**Non-goals.** Not an IdP. Not an email-verification system of its own (the IdP's activation
link does that). Not a general form builder.

## Topology

```
            ┌────────────┐   admin login (OIDC Authorization Code + PKCE)
 admin ────▶│ OIDC IdP   │◀───────────────────────────────┐
            │ (Rauthy /  │                                 │
            │ Authentik) │                                 │
            └────────────┘                                 │
                                                           │
 user/admin ─▶ Cloudflare (CDN/WAF/Turnstile, TLS) ─▶ Nginx ─▶ ┌─────────────────────┐
                                                               │ idp-register (Go)   │
                                                               │  • embeds Svelte SPA │
                                                               │  • /api/*            │
                                                               │  • sessions, tokens, │
                                                               │    review, audit     │
                                                               └──────┬───────┬───────┘
                                                                      │       │ Provisioner API
                                                          SQLite/Postgres   ┌──▼─────────────┐
                                                                            │ Target IdP     │
                                                                            │ (Rauthy→Kanidm)│
                                                                            └────────────────┘
```

## Two decoupled IdP roles

This is the single most important design point.

| Role | Who uses it | Coupling |
|---|---|---|
| **Admin-auth IdP** | admins logging into the review panel | **Generic OIDC RP only.** Discovery + ID-token verification (`coreos/go-oidc`). No provider-specific APIs. |
| **Provisioning target IdP** | end users being created | Behind the **`Provisioner` interface**. Provider-specific code is isolated per implementation. |

They are often the *same* Rauthy instance, but the code must not assume so. Admin authz is
configurable: a **groups/roles claim allow-list**, or a **sub/email allow-list** (needed
for IdPs without a groups claim).

## Components

- **Frontend** — SvelteKit built with **`adapter-static` (SPA, `fallback` page)** → pure
  static assets, **embedded in the Go binary via `go:embed`** and served at `/`. Public
  register page + legal pages may be prerendered; admin panel is CSR calling `/api`.
- **Go HTTP layer** (`internal/web`) — serves embedded assets + JSON `/api`; CSRF on
  mutations; sets an opaque **HttpOnly** session cookie after OIDC login.
- **Data layer** (`internal/store`) — `database/sql` + portable SQL + repository
  interfaces; one implementation, two dialect schema files.
- **Token service** (`internal/token`) — mint/list/validate invite codes; atomic reserve.
- **Application state machine** (`internal/application`) — review + provisioning lifecycle.
- **Provisioner** (`internal/provisioner`) — `rauthy` impl now; `kanidm` later.
- **Admin OIDC RP + sessions** (`internal/oidcauth`).
- **Audit log** (`internal/audit`).
- **Reconcile job** — periodically confirms IdP-side activation to advance token counters.

## Registration flow (state machine)

```
submit (email, optional username, ToS, Turnstile, optional invite code)
  │  uniform response: "application received"   ← never reveal if email/code exists
  ├─ code present & valid & has capacity ──▶ AUTO-APPROVE
  │        reserve: pending++  ─▶ provision ─▶ status=provisioning→approved
  ├─ code present but invalid ─▶ soft error (re-enter or leave blank)
  └─ no code ─▶ status=pending ─▶ admin review
                 ├─ approve ─▶ provision
                 └─ reject / needs_changes

provision = claim (conditional UPDATE pending→provisioning)
          → Provisioner.CreateUser → set username/groups
          → Provisioner.InitCredentials (IdP sends activation email, or returns a link we send)
          → status=approved ; failure → provisioning_failed (safe retry)

user clicks activation link, sets password/passkey (= email verified + account active)
  → reconcile/landing confirms → if via invite code: pending--, completed++
  → activation link expires unused → pending-- (release the slot)
```

Happy path = **one** user-facing email (the IdP activation mail).

## Invite-code model (Synapse-aligned)

Plaintext code, addressed by the code string. Semantics mirror Synapse registration tokens:

| Field | Meaning |
|---|---|
| `token` | plaintext, ≤64 chars, charset `[A-Za-z0-9._~-]`; random if not supplied |
| `uses_allowed` | max successful registrations; `NULL` = unlimited |
| `pending` | reserved-but-not-completed; `+1` on use, `-1` on completion/expiry |
| `completed` | successful registrations |
| `expiry_time` | epoch **ms**; `NULL` = never |
| `active` | manual disable switch |

**Validity**: `active AND (expiry_time IS NULL OR expiry_time > now) AND (uses_allowed IS NULL OR pending+completed < uses_allowed)`.

**Atomic reservation** (portable across both engines):
```sql
UPDATE registration_tokens SET pending = pending + 1
 WHERE token = ?
   AND active = 1
   AND (expiry_time IS NULL OR expiry_time > ?)
   AND (uses_allowed IS NULL OR pending + completed < uses_allowed);
-- success iff RowsAffected == 1
```

Optional extensions carried over from the draft: `email_constraint` (bind a code to one
email) and a default `permission_profile`.

## Data model (portable)

Tables: `registration_tokens`, `applications`, `permission_profiles`, `admin_sessions`,
`audit_log`. Portability rules (see AGENTS.md invariant #1): app-generated UUID `TEXT` PKs,
epoch-ms `INTEGER` timestamps, JSON-in-`TEXT` for lists (`groups`, `requested_services`),
`INET`→`TEXT`. The application lifecycle states: `pending`, `provisioning`,
`provisioning_failed`, `approved`, `rejected`, `needs_changes`.

## Provisioner interface

```go
type Provisioner interface {
    FindUserByEmail(ctx context.Context, email string) (*User, bool, error)
    CreateUser(ctx context.Context, in NewUser) (userID string, err error)
    // Trigger credential setup. Some IdPs send their own email (Rauthy); others
    // return a reset link this service must deliver (Kanidm). Hence the optional link.
    InitCredentials(ctx context.Context, userID string) (resetLink *string, err error)
}
```

- **Rauthy**: `POST /users` (silent create) then trigger the set-password/reset email;
  `PUT /users/{id}/self/preferred_username`; `GET /users/email/{email}`. Auth header
  `API-Key <name>$<secret>`. (Confirmed: `POST /users` does **not** send mail by itself.)
- **Kanidm** (planned): create person, then issue a credential-reset token/URL that this
  service emails. `InitCredentials` returns the link.

## Anti-abuse & security

- **Cloudflare Turnstile** at the form; server-side verification in Go. Rate-limit by IP.
- **Account enumeration**: identical form responses regardless of email/code existence;
  differentiate only via email content.
- Public form cannot set IdP groups (only `permission_profiles`).
- Opaque HttpOnly session cookies; CSRF on mutations; secrets never reach the browser.

## Deployment

Single Go image (embedded frontend) behind the maintainer's existing Nginx + Cloudflare.
SQLite needs only a mounted volume; PostgreSQL is an optional external dependency. TLS
terminates at Cloudflare (Full strict) / Nginx; real client IP restored from
`CF-Connecting-IP`.

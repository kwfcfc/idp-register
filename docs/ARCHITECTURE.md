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

## Registration flow (state machine)

```
submit (email, optional username, ToS, Turnstile, optional invite code)
  │  uniform response: "application received"   ← never reveal if email/code exists
  ├─ code present & valid & has capacity & bound profile ──▶ AUTO-APPROVE
  │        requested service = token.profile_id
  │        reserve: pending++  ─▶ provision ─▶ pending--, completed++ ─▶ approved
  ├─ code present but invalid ─▶ ordinary pending application (uniform response)
  └─ no code ─▶ status=pending ─▶ admin review
                 ├─ approve ─▶ provision
                 └─ reject

provision = claim (conditional UPDATE pending→provisioning)
          → Provisioner.CreateUser → set username/groups
          → Provisioner.InitCredentials if needed by the provider
          → status=approved ; failure → provisioning_failed (safe retry)

user clicks activation link, sets password/passkey (= email verified + account active)
```

Happy path = **one** user-facing email (the IdP activation mail).

### Provisioning failure semantics

`provisioning_failed` means this service did not successfully create the target
IdP user. It is a recovery state for operator-visible IdP/API problems, not a
second applicant-facing review loop.

- Backend/API errors between this service and the target IdP (bad API key,
  missing create/group rights, network outage, target IdP unavailable, invalid
  target group/profile mapping) are retryable. Manual applications keep their
  application record and may be retried with a corrected profile. Invite-backed
  applications keep the original invite-bound profile; retry only re-attempts
  the same grant.
- Target-IdP registration conflicts are not retried as a different grant. If the
  email already exists in Rauthy, or an existing user is asking for a different
  profile / elevation, the application should be rejected here and handled in
  the target IdP's own user-management UI.
- Rauthy email delivery is target-IdP responsibility. For Rauthy, `POST /users`
  creates the user and enqueues/sends the first-password email; if mail delivery
  later fails, this service still treats provisioning as successful because the
  target IdP user exists. Rauthy administrators handle SMTP failures, bounces,
  and re-sending password/setup links.
- Provisioning runs synchronously inside the request; an application still in
  `provisioning` after a restart was interrupted mid-flight. On startup the
  service sweeps such rows into `provisioning_failed` so they re-enter the
  normal retry/reject path (a held invite reservation stays held, as for any
  other failure).
- Unreachable applicant mailboxes and unused activation links are not reconciled
  back into invite counters. Rauthy's first-password magic link has its own
  lifetime (`magic_link_pwd_first`, default 4320 minutes in Rauthy 0.35.x), and
  Rauthy's magic-link cleanup deletes users that never set a password/passkey
  after the unused first-password link expires. In this service, the invite use
  remains `completed` once target-IdP creation succeeds.

## Invite-code model (Synapse-aligned)

Plaintext code, addressed by the code string. Semantics mirror Synapse registration tokens:

| Field | Meaning |
|---|---|
| `token` | plaintext, ≤64 chars, charset `[A-Za-z0-9._~-]`; random if not supplied |
| `uses_allowed` | max successful registrations; `NULL` = unlimited |
| `pending` | reserved-but-not-completed; `+1` on use, `-1` on provisioning success or explicit release |
| `completed` | successful registrations |
| `expiry_time` | epoch **ms**; `NULL` = never |
| `active` | manual disable switch |
| `profile_id` | required permission profile; valid invite codes only auto-approve into this profile |

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
`provisioning_failed`, `approved`, `rejected`.

## Provisioner interface

```go
type Provisioner interface {
    FindUserByEmail(ctx context.Context, email string) (*User, bool, error)
    CreateUser(ctx context.Context, in NewUser) (userID string, err error)
    // Trigger credential setup if CreateUser did not already do it. Some IdPs
    // send their own email; others return a reset link this service must deliver.
    InitCredentials(ctx context.Context, userID, email string) (resetLink *string, err error)
}
```

- **Rauthy 0.35.2**: `POST /users` creates the user and, when SMTP is configured,
  sends the initial "New Password" activation email with a `type=new_user` reset link.
  `POST /users/request_reset` is the public password-reset path and requires a `pow`
  payload even with an API key, so the Rauthy `InitCredentials` implementation is a
  no-op for newly created users. `PUT /users/{id}/self/preferred_username`;
  `GET /users/email/{email}`. Auth header `API-Key <name>$<secret>`.
- **Kanidm** (planned): create person, then issue a credential-reset token/URL that this
  service emails. `InitCredentials` returns the link.

## Anti-abuse & security

- Public-form abuse is countered by a **server-verified human-verification challenge**,
  not by application-level IP rate limiting (ADR-0016). **Cloudflare Turnstile** is the
  implemented provider; self-hostable proof-of-work verifiers (e.g. `sebadob/spow` as an
  embeddable widget, or `TecharoHQ/anubis` as an edge interstitial in front of the app)
  are the planned/possible alternatives. The Turnstile site key is runtime config served
  to the browser via `GET /api/form` (no build-time key; one published image fits every
  deployment), and site key + secret must be set together. Production deployments should
  always configure a challenge — with it unset, verification is skipped. Edge rate limiting (Cloudflare,
  Nginx) remains available as defense in depth but is a deployment concern, not app code.
- **Registration rules & ToS consent**: deployers provide their own rules text
  (`FORM_RULES_TEXT` / `FORM_RULES_FILE`) and a terms-of-service link (`FORM_TERMS_URL`),
  all runtime config served via `GET /api/form` like the Turnstile site key. If any is
  configured, the form renders a required consent checkbox and the server rejects
  submissions without `termsAccepted: true` (consent requiredness is public config, so the
  rejection leaks nothing). Rules render as plain text, never HTML. Consent is not
  persisted per application: the server-side check means every stored application implies
  consent to the rules in force at submission time.
- **Account enumeration**: identical form responses regardless of email/code existence;
  differentiate only via email content.
- Public form cannot set IdP groups (only `permission_profiles`).
- Opaque HttpOnly session cookies; CSRF on mutations; secrets never reach the browser.

## Deployment

Single Go image (embedded frontend) behind the maintainer's existing Nginx + Cloudflare.
SQLite needs only a mounted volume; PostgreSQL is an optional external dependency. TLS
terminates at Cloudflare (Full strict) / Nginx; real client IP restored from
`CF-Connecting-IP`.

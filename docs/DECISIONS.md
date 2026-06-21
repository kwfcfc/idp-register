<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Decision log (ADRs)

Lightweight architecture decision records. Newest decisions append to the end. Each entry:
**Context → Decision → Consequences**. Status is `accepted` unless noted.

---

## ADR-0001 — Go backend instead of full-stack SvelteKit
**Status:** accepted (supersedes the initial SvelteKit-Node draft)

**Context.** The first AI draft was a full-stack SvelteKit (Node) app. We want a long-lived
infra service: tiny self-contained image, strong concurrency for the reconcile job, simple
ops.

**Decision.** Backend in **Go**; frontend stays Svelte but is decoupled (ADR-0002). All
`src/lib/server/**` and `*.server.ts` logic is rewritten in Go.

**Consequences.** Single static binary, ~15 MB image, no Node runtime in production. Cost:
the draft's server layer is rewritten; two languages in the repo (Go + TS for UI only).

---

## ADR-0002 — SvelteKit `adapter-static`, embedded in the Go binary
**Status:** accepted

**Context.** We want Svelte's DX but no second runtime to deploy.

**Decision.** Build the frontend with **`adapter-static` in SPA mode** (`fallback`
page) → pure static assets, **embedded via `go:embed`** and served by Go. Frontend talks
to Go over `/api`. Public/legal pages may be prerendered; admin is CSR.

**Consequences.** Truly single-artifact deploy; clean front/back separation. Cost: admin
data loading moves from `+page.server.ts` to client `+page.ts` → `/api`; need CSRF +
cookie-based session handling across the API boundary.

---

## ADR-0003 — `database/sql` + portable SQL, dual SQLite/PostgreSQL
**Status:** accepted

**Context.** Deployers should be able to run on SQLite (single-file, simplest) or
PostgreSQL. Migrations are out of scope for now (startup applies schema).

**Decision.** Use **`database/sql`** with hand-written **portable SQL** behind repository
interfaces — no ORM. Drivers: **pgx (stdlib)** and **modernc.org/sqlite** (pure Go, no
CGO). Portability rules: app-side UUID + timestamps, epoch-ms INTEGER time, JSON-in-TEXT
lists, `INET`→TEXT. Two dialect schema files; partial indexes allowed.

**Consequences.** Full control over the atomic token-reservation query; predictable SQL.
Cost: must keep SQL within the portable subset and **test the store layer against both
engines** in CI. Rejected: GORM (reflection/magic, awkward array/JSON mapping), sqlc
(duplicate per-dialect query files).

---

## ADR-0004 — Plaintext invite codes, Synapse-aligned semantics
**Status:** accepted (supersedes the draft's HMAC-digest invites)

**Context.** Codes are shareable invites (distributed in `?token=` links), not passwords.
The maintainer already uses Synapse registration tokens and wants the same mental model.

**Decision.** Store codes **plaintext**, addressed by the code string. Fields & semantics
mirror Synapse: `uses_allowed` / `pending` / `completed` / `expiry_time` (+ `active`).
Reserve with `pending++`, confirm with `completed++`. Admin API shape mirrors Synapse
(`.../registration_tokens`, `/new`, `PUT`, `DELETE`, `?valid=`).

**Consequences.** Familiar model; scripts/intuition transfer from Synapse. Codes must
**never be logged**; mitigate guessing with rate-limit + expiry + use caps. Drops the
draft's `code_digest`/`code_prefix` and `INVITE_HMAC_KEY`.

---

## ADR-0005 — Provisioner abstraction (Rauthy first, Kanidm planned)
**Status:** accepted

**Context.** Rauthy is today's target IdP, but Kanidm (and others) expose similar
create-user + credential-reset APIs.

**Decision.** All user creation goes through a **`Provisioner` interface**. Provider code
is isolated per implementation. `InitCredentials` returns an **optional reset link** so the
interface fits both "IdP sends its own email" (Rauthy) and "IdP returns a link we email"
(Kanidm).

**Consequences.** Swappable backends; no Rauthy assumptions leak out. Cost: a small
abstraction overhead and per-provider conformance tests.

---

## ADR-0006 — Admin auth is a generic OIDC RP; no GitHub OAuth2 adapter
**Status:** accepted

**Context.** Admins might log in via Rauthy, Authentik, or another OIDC IdP. GitHub is
OAuth2 (no standard OIDC discovery for user login).

**Decision.** Implement admin login as a **standards-only OIDC Relying Party**
(`coreos/go-oidc` + `golang.org/x/oauth2`). **No bespoke GitHub OAuth2 adapter** — if
GitHub is wanted, broker it through an OIDC IdP. Admin authz is configurable: groups/roles
claim allow-list, or sub/email allow-list.

**Consequences.** Simple, provider-neutral auth code. Cost: GitHub-only setups need an
OIDC broker in front.

---

## ADR-0007 — One user-facing email; verification deferred into IdP activation
**Status:** accepted

**Context.** A naive flow sends several emails (verify, then onboarding, then credential).
A human/code review gate already filters applicants.

**Decision.** Send **no** separate verification email. On approval, provision the user and
let the **IdP's activation link** do email-verification + credential setup + onboarding in
one action. Rejection mail is optional (needs the service's own SMTP).

**Consequences.** Happy path = 1 email; better sender reputation; less code. Cost: a typo'd
email is only discovered when the activation mail bounces (acceptable under human review).

---

## ADR-0008 — License: GPLv3-or-later
**Status:** accepted

**Decision.** The project is licensed **GPL-3.0-or-later**. `LICENSE` holds the full text;
every source file carries `SPDX-License-Identifier: GPL-3.0-or-later`.

**Consequences.** Strong copyleft; derivatives stay open.

---

## ADR-0009 — Permission profiles; the public form never sets IdP groups
**Status:** accepted (carried over from the draft)

**Context.** Group/role assignment is security-sensitive and must not be attacker-controlled.

**Decision.** Keep server-side **`permission_profiles`** (named bundles of target-IdP
groups). The public form may only reference a profile (or none); the actual groups are
resolved server-side at approval.

**Consequences.** Public input can't escalate privileges. Cost: profiles must be maintained
by admins.

---

## ADR-0010 — CI/CD with Crow CI (Jsonnet), deferred
**Status:** accepted; implementation deferred

**Context.** The repo is hosted on a self-managed Forgejo, but the chosen CI engine is
**Crow CI** (Woodpecker-derived), not Forgejo Actions.

**Decision.** Author pipelines in **Jsonnet** under a **`.crow/`** directory. Planned
stages: lint → test (SQLite + PostgreSQL) → multi-arch image build. **Do not build the CI
pipelines or deployment automation yet** — focus on the application first. Syntax reference:
<https://crowci.dev/v5-13/usage/jsonnet/>.

**Consequences.** Jsonnet gives reusable functions/imports over plain YAML. Cost: a less
common engine; contributors must learn Crow/Woodpecker conventions. No CI gate until wired.

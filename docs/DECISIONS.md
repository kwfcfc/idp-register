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
In this application, every invite code must also bind to one `permission_profiles.id`;
valid invite use is always automatic approval into that profile/service. There is no
profile-less or semi-automatic invite path.

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
interface fits both "IdP sends its own email" and "IdP returns a link we email" (Kanidm).
For Rauthy 0.35.2 specifically, the initial credential email is sent by `POST /users`
when SMTP is configured, and the public `request_reset` endpoint requires PoW.

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

---

## ADR-0011 — Support front/back-separated deployment; three build targets
**Status:** accepted; only Mode 1 implemented (Modes 2 & 3 deferred)

**Context.** Today the frontend is embedded in the Go binary and served by it (one
origin, ADR-0002). We also want the option to deploy the SPA **separately** (nginx, a CDN,
Cloudflare Pages) with the Go service acting as an API-only backend, so the UI can be
hosted/cached independently. This must not break the same-origin cookie + `Origin`-CSRF
model unless explicitly opted into.

**Decision.** Keep one SPA source and one Go codebase, packaged three ways
(see [`DEPLOYMENT.md`](DEPLOYMENT.md) for the how):
1. **All-in-one** — Go embeds + serves the SPA (current default; implemented).
2. **API-only** — Go serves no SPA, gated by a `SERVE_FRONTEND` flag (deferred).
3. **Standalone static frontend** — built bundle on nginx/CDN (deferred).

For separation, **prefer same-origin via an edge reverse-proxy** (the static host proxies
`/api` + `/auth` to the API): the SPA keeps using **relative** API paths, so the existing
`SameSite=Lax` cookie + CSRF model is unchanged and there is **no CORS**. True cross-origin
(separate API domain) is the fallback and requires CORS + `SameSite=None;Secure` + an
allowed-frontend-origin config + a configurable post-login redirect + a `VITE_API_BASE` in
the SPA. These three packagings become the **three Crow CI build targets** (ADR-0010).

**Consequences.** Flexible hosting (e.g. SPA on Cloudflare, API on a small box) without
forking the code. Cost: an unembedded build needs a parameterized adapter-static output
dir and the `SERVE_FRONTEND` flag; the cross-origin path additionally needs real CORS and
cookie/redirect changes, so keep the relative-path + edge-proxy route as the default.

---

## ADR-0012 — Profiles draw from the live IdP group catalog; admin CRUD; a public service selector
**Status:** accepted; backend implemented (frontend deferred)

**Context.** ADR-0009 keeps group assignment server-side in `permission_profiles`, but
those profiles were seeded with **hard-coded** group strings (`svc:matrix:user`, …) and had
only a read endpoint. Admins had no way to create/edit a profile, which blocked the whole
approval chain (approve requires an existing profile). We also want the public form to offer
a curated, admin-controlled set of "services to register for" rather than a free-text field.

**Decision.**
1. **Groups come from the target IdP, not the app.** Add `Provisioner.ListGroups` (Rauthy:
   `GET /groups`, needs `Groups:read` on the API key). Profiles are composed only from group
   names the IdP actually defines; submitted groups are validated against this live catalog.
2. **A denylist hides infra/admin groups** from the profile editor and rejects them on write
   (`PROFILE_GROUP_DENYLIST`, default `admin,rauthy_admin`, always merged with the app's own
   `OIDC_ADMIN_GROUP`). Service-admin groups like `svc:gotosocial:admin` stay assignable —
   only true infrastructure-admin groups are withheld.
3. **Profiles are admin-CRUD.** `POST/PUT/DELETE /api/admin/profiles[/{id}]`, plus
   `GET /api/admin/groups` for the editor. Delete is refused (409) while a token or
   application still references the profile (FK), preserving audit history.
4. **A public service selector.** Profiles carry `public_selectable` + `public_label` +
   `sort_order`. `GET /api/form` returns the anonymous-safe option list (**no group names**)
   and a `selectionMode`. The public form's selection is stored as advisory
   `requested_services` (validated to public ids); the **real** profile/groups are still
   resolved by an admin at approval — ADR-0009 is unchanged.
5. **Single-select first.** `selectionMode` is fixed to `"single"` for now. Multi-select
   needs union-of-groups provisioning and storing multiple approved profiles per application;
   that is a separate roadmap milestone, not part of this slice.

**Consequences.** Admins manage profiles from the panel with a real group picker; the public
form is curated without code changes; privilege escalation via crafted group names is
impossible (server validates against the catalog minus denylist). Cost: profile writes now
depend on a live IdP call. The current database policy is documented in
`docs/DATABASE.md`: fresh deployments may choose SQLite or PostgreSQL, while automatic
old-database upgrades are deferred until a future breaking schema change requires an
explicit upgrade path.

---

## ADR-0013 — Admin decisions are approve or reject only
**Status:** accepted

**Context.** The service does not send applicant-facing email. The happy-path email is sent
by the provisioning target IdP after approval, and invalid/missing invite codes already fall
back to ordinary manual review through the uniform public response.

**Decision.** Keep the admin review state machine to two human decisions: **approve** or
**reject**. There is no `needs_changes` state and no "request more information" workflow.
If provisioning fails after an invite reservation, an admin either retries approval after
fixing the cause, or rejects the application to release the held invite use.

**Consequences.** The review UI stays simpler and matches the no-service-email constraint.
Applicants who made a mistake can submit again; the service does not need to manage an
outbound correspondence loop.

---

## ADR-0014 — Target-IdP activation remains outside the broker state machine
**Status:** accepted

**Context.** Rauthy has two distinct user-creation paths. Its public registration
endpoint hides duplicate-email details for anti-enumeration, while the admin/API
`POST /users` path used by this broker returns normal API errors for create-time
problems. Rauthy creates the user first, then sends or queues the first-password
mail. SMTP delivery failures, bounces, and unused activation links are target-IdP
operational concerns.

**Decision.** Treat provisioning success as "the target IdP user was created",
not "the applicant completed activation". If Rauthy returns a create-time error
such as duplicate email, the application remains `provisioning_failed` for admin
resolution and should normally be rejected if the account already exists. If
Rauthy creates the user but its mail delivery fails later, this broker keeps the
application `approved` and the invite use `completed`; Rauthy administrators
repair SMTP or resend/reset credentials from Rauthy.

Invite-backed retries keep the original token-bound profile. Manual-review
retries may change the profile to correct an operator mistake, but existing-user
profile changes or elevation requests are rejected here and handled in the target
IdP's user-management workflow.

**Consequences.** The broker does not need to poll Rauthy for activation state or
reconcile expired first-password links into invite counters. This keeps the first
stable baseline simple and preserves the "one user-facing email" invariant. A
future reconciliation feature would require an explicit new ADR because it would
change invite accounting semantics after successful target-IdP creation.

---

## ADR-0015 — Publish project documentation as an mdBook site
**Status:** accepted; implementation deferred

**Context.** The repository now has several operator- and developer-facing documents:
README, architecture notes, ADRs, database policy, deployment modes, production Compose
templates, and local Rauthy harness instructions. Plain Markdown files are fine during early
development, but installation and operations need a browsable, versioned documentation site
once users deploy the project from published images.

**Decision.** Add a medium-term documentation site built with Rust's **mdBook**. The book
will cover project introduction, concepts, installation, production Compose deployment,
Rauthy/OIDC configuration, database choices, operations, troubleshooting, architecture, and
ADR references. It is documentation only: it does not replace the Svelte application
frontend and it does not change the all-in-one production deployment.

The docs build becomes part of the future Crow CI workflow after the application lint/test
steps. CI should publish the generated static book artifact to the maintainer's preferred
static host: Forgejo Pages if available for the canonical Forgejo instance, otherwise
Cloudflare Pages. The pipeline should eventually distinguish stable published docs from
main-branch preview docs.

**Consequences.** Operators get a single browsable install/config/deploy guide, and docs can
be published from the same CI system as the images. Cost: the repository must avoid stale
duplicate documentation by either organizing the existing Markdown into the book or making
clear which files are source-of-truth and which are rendered/curated book chapters.

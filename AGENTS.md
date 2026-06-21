<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# AGENTS.md — Context for AI agents & human developers

> Read this first. It captures the **why** and the **hard constraints** that are not
> obvious from the code. Detailed design lives in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md);
> the rationale for each major choice lives in [`docs/DECISIONS.md`](docs/DECISIONS.md).

## What this is

`idp-register` is a small, self-hosted **user-registration broker** that sits in front of
an OIDC identity provider (IdP). It is the OIDC-era equivalent of Synapse's built-in
**registration tokens**: people apply through a public form, an **invite code** (limited
by use-count and expiry) can **auto-approve** them, otherwise an admin reviews the
application. On approval the service **provisions the user into the target IdP** and lets
the IdP send a single "set your password / activate" email.

Origin: the maintainer runs a Matrix homeserver (Synapse) plus other services
(GoToSocial, Forgejo) behind **Rauthy** as the OIDC IdP, and wants the Synapse
registration-token UX, but for Rauthy.

## Status (read before assuming)

- An **initial AI-generated draft** exists as a **full-stack SvelteKit (TypeScript)**
  app. It is **being re-based** to **Go backend + decoupled SvelteKit static frontend**.
- The draft implements the **admin site only**; the **public application form is not built yet**.
- The draft is **PostgreSQL-only** and stores invite codes as **HMAC digests**. Both
  decisions are **superseded** (dual-DB; plaintext codes). See `docs/DECISIONS.md`.

What survives the re-base (port the *design*, not the TS): the `.svelte` UI components &
routes, the SQL schema *design* (application state machine, audit log, permission
profiles, admin sessions), the generic OIDC login concept, and the Rauthy API client
logic. Everything under `src/lib/server/**` and `*.server.ts` gets **rewritten in Go**.

## Hard invariants — do not violate without a new ADR

1. **Dual-DB portability (SQLite + PostgreSQL).** No PG-only column types in shared schema.
   Generate **UUIDs and timestamps in Go** (not DB defaults); store time as **epoch
   milliseconds (INTEGER)**; store small lists/objects as **JSON in TEXT** (handled in Go);
   `INET`→`TEXT`. Partial indexes are fine (both engines support them). Drivers:
   **pgx (stdlib)** for PG, **modernc.org/sqlite** (pure-Go, no CGO) for SQLite.
2. **Two distinct IdP roles — never conflate them.**
   - **Admin-auth IdP**: where *admins* log in. A plain, standards-only **OIDC Relying
     Party**. **Never** assume Rauthy-specific endpoints here — it may be Authentik, Dex,
     etc. (No bespoke GitHub OAuth2 adapter; broker GitHub via an OIDC IdP if needed.)
   - **Provisioning target IdP**: where *end users* get created. Hidden behind the
     **`Provisioner` interface** (Rauthy today, Kanidm planned). Rauthy API calls live
     **only** inside the Rauthy implementation.
3. **Invite codes are plaintext** "limited-use, time-limited" codes (Synapse-aligned:
   `uses_allowed` / `pending` / `completed` / `expiry_time`). **Never log a code.**
   Use-count is reserved with `pending++` and confirmed with `completed++` (see ARCHITECTURE).
4. **The public form must never choose IdP groups directly.** It may only reference
   server-side `permission_profiles`; group assignment happens server-side on approval.
5. **One user-facing email on the happy path.** Defer email verification into the IdP's
   activation link; don't add a separate "verify your email" round-trip. Keep form
   responses **uniform** to avoid account enumeration.
6. **Single self-contained binary.** Frontend is built to static assets and embedded via
   `go:embed`; build with `CGO_ENABLED=0`.
7. **License: GPLv3-or-later.** Every source file starts with
   `// SPDX-License-Identifier: GPL-3.0-or-later` (or the comment form for that language).

## Target repository layout

```
cmd/server/            main.go (wires config, db, provisioner, oidc, http)
internal/
  config/              env → typed config
  store/               database/sql repos + portable schema (schema_pg.sql, schema_sqlite.sql)
  provisioner/         Provisioner interface + rauthy/ (+ kanidm/ later)
  oidcauth/            admin OIDC RP (coreos/go-oidc) + opaque sessions
  token/               invite-code logic (mint, validate, reserve/complete)
  application/         application state machine + review
  audit/               audit log
  web/                 HTTP handlers, embeds frontend build
web/                   SvelteKit frontend (adapter-static → build/ embedded by Go)
docs/                  ARCHITECTURE.md, DECISIONS.md
migrations/            (later; startup applies schema_*.sql for now)
```

## Build / test / CI (target)

- **Build**: `CGO_ENABLED=0 go build`; multi-stage Dockerfile (Node builds frontend → Go
  embeds it → distroless/scratch final, ~15 MB).
- **Test**: run the `store` layer against **both** SQLite (in-memory) and PostgreSQL
  (testcontainers) to catch dialect drift — this is the main reason dual-DB needs CI cover.
- **CI/CD: Crow CI** (NOT Forgejo Actions), repo on `forgejo.goba.ip-dynamic.org`.
  - Pipelines are authored in **Jsonnet**. Config lives in a **`.crow/`** directory
    (`.jsonnet`/`.libsonnet`/`.yaml`); alternatively a single `.crow.jsonnet`. Crow is
    Woodpecker-derived, so `.woodpecker*` is a fallback Crow also recognizes.
  - A Jsonnet file evaluates to JSON/YAML **before** any other processing, then matrix
    expansion → `${CI_*}` substitution → lint → compile. Structure: a top-level object with
    `steps: [{ name, image, commands, when?, depends_on?, services? }]`; returning a JSON
    **array** yields multiple workflows. CI metadata via `std.extVar("CI_PIPELINE_EVENT")`
    etc. Share helpers via `.libsonnet` imports. Syntax ref:
    <https://crowci.dev/v5-13/usage/jsonnet/>.
  - Planned pipeline: lint (golangci-lint) → test (both DBs) → buildx multi-arch image.
- **Deferred for now**: the **CI/CD pipelines and deployment/testing automation are NOT
  built yet** (per maintainer). Focus is the application itself; wire Crow CI later.

## Glossary

- **Invite code / token** — public, plaintext, limited-use string that auto-approves an application.
- **Application** — a submitted registration request moving through a review state machine.
- **Permission profile** — a named server-side bundle of target-IdP groups (e.g. `developer`).
- **Provision** — create the user in the target IdP and trigger its activation email.

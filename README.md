<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# idp-register

A small, self-hosted **user-registration broker** for an OIDC identity provider — the
OIDC-era equivalent of Synapse's built-in **registration tokens**.

People apply through a public form. An **invite code** (limited by use-count and expiry)
can **auto-approve** them; otherwise an **admin reviews** the application. On approval the
service **provisions the user into the target IdP** (Rauthy today; Kanidm planned) and lets
the IdP send a single "set your password / activate" email — which also verifies the address.

Built for a self-hosted stack (Matrix/Synapse + GoToSocial + Forgejo behind **Rauthy**),
deployable as one OCI image behind Nginx/Cloudflare.

## Two IdP roles (don't conflate them)

- **Admin-auth IdP** — where admins log in. A standards-only **OIDC Relying Party**; never
  assumes provider-specific APIs.
- **Provisioning target IdP** — where end users are created. Hidden behind a `Provisioner`
  interface; provider-specific code is isolated.

## Status

> ⚠️ **In progress.** The Go backend (`cmd/`, `internal/`) is implemented and the SvelteKit
> frontend has been ported to `web/` (`adapter-static`, embedded via `go:embed`). CI/CD and
> some hardening remain. Treat the docs as the source of truth.

**Stack:** Go backend · SvelteKit frontend (`adapter-static`, built into
`internal/web/assets` and embedded via `go:embed`) · `database/sql` over SQLite **or**
PostgreSQL · generic OIDC admin login · plaintext Synapse-style invite codes · GPLv3-or-later.

## Documentation

- [`AGENTS.md`](AGENTS.md) — context & hard invariants for AI agents and developers (read first).
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system design, flows, data model.
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — architecture decision records (the *why*).
- [`docs/DATABASE.md`](docs/DATABASE.md) — supported DB baselines, fresh-deploy policy,
  and manual backend-switching notes.
- [`docs/SCHEMA.md`](docs/SCHEMA.md) — logical schema and SQLite/PostgreSQL type mapping.
- [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) — deployment modes (all-in-one / API-only /
  standalone static frontend) and the three future CI build targets.

## Features (target)

- Public application form: optional fields, ToS consent, Cloudflare Turnstile, optional invite code.
- Invite codes: use-count + expiry + manual disable; auto-approve on valid code (Synapse
  `uses_allowed` / `pending` / `completed` / `expiry_time` semantics).
- Admin review panel (OIDC-protected): mint/list/disable codes, review/approve/reject applications.
- Provisioning into the target IdP with safe retry on failure; server-side permission profiles
  map to IdP groups (public input never sets groups directly).
- One user-facing email on the happy path (the IdP activation link).
- Admin action audit log.

## Development

The frontend (`web/`, a SvelteKit SPA) builds straight into `internal/web/assets`, which the
Go binary embeds via `//go:embed all:assets`. Copy `.env.example` to `.env` and fill it in.

```sh
# Frontend dev (hot reload on :5173, proxies /api + /auth to the Go server on :8080)
pnpm install
pnpm --filter idp-register-web dev      # or: pnpm dev

# Run the Go backend (serves the embedded SPA + JSON API on :8080)
go run ./cmd/server

# Production build: build the SPA, then the binary embeds it
pnpm --filter idp-register-web build    # writes internal/web/assets
CGO_ENABLED=0 go build -o idp-register ./cmd/server

# Or build the single OCI image (Node → Go → distroless, multi-stage)
docker build -t idp-register .

# Tests (store layer; SQLite in-memory by default)
go test ./...
```

> The embedded assets are generated and git-ignored; `internal/web/assets/.gitkeep` keeps the
> Go package compiling before the first frontend build. See `AGENTS.md` for the full layout.

## License

[GPL-3.0-or-later](LICENSE). Every source file carries an SPDX header.

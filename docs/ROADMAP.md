<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Roadmap

Single source of truth for what's done and what's next. Architecture and the *why*
live in [`ARCHITECTURE.md`](ARCHITECTURE.md) and [`DECISIONS.md`](DECISIONS.md); this
file is just the sequenced worklist. Keep it in lock-step with reality.

## Status legend

`✅ done` · `🔄 in progress` · `⬜ todo` · `🧊 deferred`

---

## Milestones

### M0 — Re-base to Go + Svelte ✅
Go backend (dual SQLite/PostgreSQL), SvelteKit `adapter-static` SPA embedded via `go:embed`,
multi-stage Dockerfile, local Rauthy harness (`deploy/dev/`) verified end-to-end. See ADRs
0001–0011.

### M1 — Permission-profile management (backend) ✅
The functional unblock for the approval chain. See **ADR-0012**.
- ✅ `Provisioner.ListGroups` + Rauthy `GET /groups`; dev API key granted `Groups:read`.
- ✅ Group denylist (`PROFILE_GROUP_DENYLIST`, merges `OIDC_ADMIN_GROUP`).
- ✅ Profile CRUD: `POST/PUT/DELETE /api/admin/profiles`, `GET /api/admin/groups`;
  server-side group validation against the live catalog; delete guarded by FK (409).
- ✅ Public service selector backend: `public_selectable`/`public_label`/`sort_order`
  columns, `GET /api/form` (anonymous-safe, no group names), `services` accepted on
  `POST /api/register` and stored as advisory `requested_services`.
- ✅ `ErrNotFound → 404`, `ErrConflict → 409` mapping (closes the old "404 returns 500" bug).
- ✅ Store unit tests for profile CRUD + public views.

### M2 — Permission-profile management (frontend) ✅
Make the admin panel and public form actually drive M1.
- ✅ Admin **Profiles** page: list / create / edit / delete, with a group multi-select
  populated from `GET /api/admin/groups` (degrades gracefully if the catalog call fails),
  and the `publicSelectable` / `publicLabel` / `sortOrder` controls.
- ✅ Public registration form: renders the single-select service picker from `GET /api/form`
  (loaded in `+page.ts`, tolerant of failure), submits `services`.
- ✅ Approval UI picks from existing profiles (`admin/applications/[id]` + invite-code mint),
  and the application detail view shows the applicant's `requestedServices`.

### M3 — First end-to-end smoke test of the whole flow ✅
The "第一版测试": prove the complete chain against the local Rauthy harness.

Harness plumbing is now **verified working** (2026-06-21): admin OIDC login completes
(needed adding **S256 PKCE** to the RP — Rauthy's client requires `code_challenge`) and
`GET /api/admin/groups` returns the live catalog (needed the bootstrap API key's
`Groups:read`, which only takes effect after a `docker compose down -v` volume reset — the
key is create-if-absent). Both gotchas are captured in the dev-harness notes.

Verified flow:
- ✅ Admin creates a profile from real Rauthy groups → marks it public.
- ✅ Public user submits the form selecting that service.
- ✅ Admin approves with a profile → user is provisioned in Rauthy with the right groups.
  Mailcrab-backed testing showed Rauthy 0.35.2 sends the `type=new_user` set-password
  email directly from `POST /users` when SMTP is configured.
- ✅ `POST /users/request_reset` tested separately: it requires `pow` and sends no mail
  when only `email` is provided, even with the provisioning API key. The Rauthy
  provisioner therefore treats `InitCredentials` as a no-op for newly created users.
- ✅ Invite-code auto-approval path is profile-bound only: each token must name a
  permission profile/service, and a valid token always provisions directly into that
  profile. The run also found and fixed the app-side bug where `approved_profile_id`
  was not persisted during auto-approval application creation.
- ✅ Captured as repeatable application-service integration tests with a fake
  `Provisioner` in `internal/application/application_test.go`, covering public-service
  filtering, manual approval provisioning, profile-bound invite auto-approval, invalid
  invite fallback, token counters, and provisioning-failure retry/reject cleanup.
- ✅ Manual checklist against the real `deploy/dev/` Rauthy + Mailcrab harness is
  documented in `deploy/dev/README.md`.

### M4 — Multi-select services 🧊
Deferred from ADR-0012. Needs union-of-groups provisioning and storing multiple approved
profiles per application; `selectionMode` becomes admin-configurable (single/multi).

### M5 — Provisioning recovery / cleanup ✅
Invite counters now advance on provisioning success (`pending--`, `completed++`) instead
of waiting for a separate activation confirmation. Admins can retry a failed provisioning
by approving again, or reject the application to release any held invite reservation.
The recovery/cleanup path is covered in `internal/application/application_test.go`: retry
approval keeps the invite reservation until success, while rejection releases it. The admin
UI now labels failed provisioning as retryable and explains when rejecting releases an
invite-code pending reservation.

### M6 — Schema discipline & database policy ✅
Document the fresh-deployment schema and database support policy without promising automatic
old-database upgrades or SQLite↔PostgreSQL conversion.
- ✅ `docs/DATABASE.md`: supported baselines, fresh deployment, upgrade policy, manual
  backend-switching notes.
- ✅ `docs/SCHEMA.md`: logical schema and SQLite/PostgreSQL type mapping.
- ✅ Initial deployment SQL split by dialect:
  `migrations/sqlite/001_init.sql`, `migrations/postgres/001_init.sql`.
- 🧊 Automatic old-database upgrades are deferred until a future release introduces a
  breaking schema change that actually needs one.

### M7 — PostgreSQL test coverage ⬜
Run the store suite against PostgreSQL (testcontainers) alongside SQLite, to lock the
dual-DB invariant.

### M8 — Front/back-separated build (ADR-0011 Modes 2 & 3) ⬜
**The "把 Go 后端和 Svelte 前端分离" task.** Same SPA source + same Go codebase, packaged
three ways. Mode 1 (embedded) is done; this milestone adds:
- ⬜ `SERVE_FRONTEND` flag (default true) → Mode 2 (API-only Go).
- ⬜ Parameterized `adapter-static` output → Mode 3 (standalone static frontend on
  nginx/CDN), with the edge-reverse-proxy (relative paths, no CORS) as the default wiring.
- ⬜ Optional true cross-origin path: CORS + `SameSite=None;Secure` + allowed-origin config
  + configurable post-login redirect + `VITE_API_BASE`.
- ⬜ **Sub-path deployment (`BASE_PATH`)** — serve the whole app under a path prefix on an
  existing domain (e.g. `id.example.com/register` behind an Nginx `proxy_pass`). Needs:
  SvelteKit `kit.paths.base` injected at build time; frontend links/API calls rewritten via
  `$app/paths` `base` (they are hard-coded absolute `/api`/`/admin`/`/auth` today); Go-side
  `BASE_PATH` config with `http.StripPrefix`, prefixed login/callback redirects, and cookie
  `Path`. Constraints to document: the base is **baked into the frontend at build time**
  (sub-path deployments rebuild the SPA, the stock all-in-one image stays root-path), and
  pure Nginx prefix-stripping cannot work without this feature — the SPA's absolute URLs
  escape the prefix, and on the IdP's own domain `/auth/*` collides with Rauthy's paths.
  See `DEPLOYMENT.md` § Sub-path deployment.

### M9 — CI/CD with Crow CI (ADR-0010) 🧊
Author `.crow/` pipelines in Jsonnet: lint → test (SQLite + PostgreSQL) → multi-arch image.
The three packagings from M8 become the three build targets. Deferred until the app stabilizes.

### M10 — mdBook documentation site ⬜
Medium-term documentation target: turn the project docs into a versioned static documentation
site built with Rust's `mdBook`, published from CI.
- ⬜ Add an mdBook tree (for example `book/` or `docs/book/`) with `book.toml`, `SUMMARY.md`,
  and chapters covering introduction, concepts, installation, production Compose deployment,
  Rauthy/OIDC configuration, database choices, operations, troubleshooting, architecture,
  and ADR index.
- ⬜ Keep source-of-truth content in sync with existing repository docs (`README.md`,
  `docs/ARCHITECTURE.md`, `docs/DECISIONS.md`, `docs/DATABASE.md`, `docs/DEPLOYMENT.md`,
  `deploy/prod/README.md`, and `deploy/dev/README.md`) without duplicating stale copies.
- ⬜ Add local developer commands for `mdbook build` and `mdbook serve`.
- ⬜ Add a Crow CI docs job that builds the static book artifact after lint/test succeeds.
- ⬜ Publish the built book to the maintainer's preferred static host: Forgejo Pages if
  available in the deployment, otherwise Cloudflare Pages.
- ⬜ Decide the public URL and versioning policy (`stable` docs vs `main`/preview docs).

### M11 — Pluggable human-verification providers 🧊
Per **ADR-0016**, anti-abuse is a challenge, not IP rate limiting. Turnstile is implemented;
this milestone adds a small provider abstraction (form widget + server verify) so a
self-hostable proof-of-work verifier (`sebadob/spow` embeddable, or `TecharoHQ/anubis` as an
edge interstitial documented in deployment) can replace Turnstile. Deferred until someone
actually needs a Cloudflare-free deployment.

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

### M2 — Permission-profile management (frontend) ⬜
Make the admin panel and public form actually drive M1.
- ⬜ Admin **Profiles** page: list / create / edit / delete, with a group multi-select
  populated from `GET /api/admin/groups`, and the `publicSelectable` / `publicLabel` /
  `sortOrder` controls.
- ⬜ Public registration form: render the single-select service picker from `GET /api/form`,
  submit `services`.
- ⬜ Wire approval UI to pick from existing profiles (already required by the chain).

### M3 — First end-to-end smoke test of the whole flow 🔄
The "第一版测试": prove the complete chain against the local Rauthy harness.
- ⬜ Admin creates a profile from real Rauthy groups → marks it public.
- ⬜ Public user submits the form selecting that service.
- ⬜ Admin approves with a profile → user is provisioned in Rauthy with the right groups →
  activation email path exercised.
- ⬜ Invite-code auto-approval path (token bound to a profile) exercised.
- ⬜ Capture as a repeatable script/integration test (httptest + a Rauthy stub, plus a
  manual checklist against `deploy/dev/`).

### M4 — Multi-select services 🧊
Deferred from ADR-0012. Needs union-of-groups provisioning and storing multiple approved
profiles per application; `selectionMode` becomes admin-configurable (single/multi).

### M5 — Reconcile job ⬜
Periodic job that confirms IdP-side activation and advances invite-code counters
(pending → completed / released on expiry). See ARCHITECTURE.md.

### M6 — Schema migrations ⬜
There is no migration framework yet (schema is `CREATE TABLE IF NOT EXISTS`; new columns
require recreating dev DBs). Introduce real, ordered migrations before any persistent
deployment. Prerequisite for trusting M7's PostgreSQL coverage across versions.

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

### M9 — CI/CD with Crow CI (ADR-0010) 🧊
Author `.crow/` pipelines in Jsonnet: lint → test (SQLite + PostgreSQL) → multi-arch image.
The three packagings from M8 become the three build targets. Deferred until the app stabilizes.

<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Deployment modes & build targets

The SvelteKit SPA (`web/`) and the Go backend (`cmd/`, `internal/`) are identical in
all three modes below — only **how the frontend is packaged and reached** differs. This
maps directly onto the future Crow CI build targets (CI itself is deferred, ADR-0010).

| Mode | Frontend served by | Origins | Status |
|---|---|---|---|
| 1. All-in-one | Go (`go:embed`) | one | **implemented** |
| 2. Go API only | nothing (API only) | — | needs `SERVE_FRONTEND` flag |
| 3. Standalone static | nginx / Cloudflare / CDN | one (proxy) or two (CORS) | build works; wiring TODO |

## Mode 1 — All-in-one (current default)

Go embeds the built SPA (`//go:embed all:assets`, `internal/web/web.go`) and serves the
SPA at `/` plus the API at `/api`, `/auth` on a single origin.

```sh
pnpm --filter idp-register-web build      # → internal/web/assets/spa
CGO_ENABLED=0 go build ./cmd/server       # binary embeds it
```

The multi-stage `Dockerfile` already does exactly this (Node → Go → distroless).
**Pros:** one artifact; same-origin so the existing model works as-is — `SameSite=Lax`
session cookie + `Origin`-based CSRF check (`internal/web` `checkCSRF`), **zero CORS**.

### Production Compose Shape

The intended near-term production deployment is:

1. Build and publish the all-in-one OCI image to the operator's registry.
2. Copy `deploy/prod/.env.example` to `deploy/prod/.env` on the host and fill in the
   production Rauthy, OIDC, claim, cookie, and database values.
3. Run `deploy/prod/compose.yml`, which pulls the image and defaults to SQLite. Operators
   who want PostgreSQL uncomment `DATABASE_URL` / PostgreSQL credentials and start with
   the `postgres` profile.

This keeps frontend/backend separation out of the first production deployment. The browser
sees one origin, the Go service owns `/`, `/api`, and `/auth`, and the reverse proxy only
needs to forward that origin to the app container.

## Mode 2 — Go API only

The same binary, but it does **not** serve the SPA — the `/` catch-all is disabled and
the frontend is deployed separately (Mode 3).

- **TODO:** a config flag (e.g. `SERVE_FRONTEND`, default `true`) that, when false, skips
  registering `spaHandler()` on `/`. The `go:embed` still compiles (the embed dir only
  needs `.gitkeep`); the binary just never serves the SPA.
- **Image size is NOT the point.** The embedded SPA is a small fraction (~hundreds of KB)
  of the ~15–25 MB binary, so a no-embed API image is only marginally smaller. The real
  reason for Mode 2 is **separation of responsibility**: cache the UI on a CDN and
  scale/redeploy the API independently. A simpler variant: keep **one** image and toggle
  `SERVE_FRONTEND` at runtime (then Modes 1 and 2 are the same image, two run modes).

## Mode 3 — Standalone static frontend

`pnpm build` produces a static bundle hosted on nginx, Cloudflare Pages, or any CDN. The
SPA calls **relative** paths (`/api`, `/auth/login`). Two ways to make those reach the API:

### 3a. Edge reverse-proxy (recommended — no backend changes beyond Mode 2)

The static host proxies `/api` and `/auth` to the Go API. The **browser still sees one
origin**, so the current cookie + CSRF model keeps working and there is **no CORS**.

- **nginx:** serve the bundle as the site root and add
  `location /api/ { proxy_pass http://idp-register-api:8080; }` (same for `/auth/`).
- **Cloudflare:** Pages for the bundle + a `/api/*` (and `/auth/*`) route to a Worker /
  origin that forwards to the API; or Pages `_redirects`/`_routes.json` proxying.

### 3b. Separate API domain (true cross-origin)

Frontend on `app.example.com`, API on `api.example.com`. More moving parts; only choose
this if an edge proxy isn't an option.

**Cross-origin ≠ cross-site — and it matters a lot here.** CORS keys on **origin**
(scheme+host+port), cookies key on **site** (registrable domain). Two subdomains of the
same registrable domain (`app.example.com` / `api.example.com`) are **cross-origin but
same-site**:

- **Same registrable domain (subdomains):** the session cookie stays **first-party**, so
  `SameSite=Lax` still delivers it and it is **not** hit by third-party-cookie blocking
  (Safari ITP / Chrome). You keep `Lax` — no `SameSite=None` needed. But CORS + the config
  points below **still apply** (it's still cross-origin). ← strongly prefer this over ↓.
- **Different registrable domains (`app.com` / `api.io`):** the cookie is **third-party** →
  blocked by modern browsers regardless of `SameSite=None;Secure`. Cookie auth effectively
  breaks; you'd need token-based auth instead. Avoid.

Required changes (all **TODO**):

- **Go CORS**: handle `OPTIONS` preflight; send `Access-Control-Allow-Origin: <frontend>`,
  `Access-Control-Allow-Credentials: true`, allowed methods/headers.
- **Cookie**: same-site subdomains keep `SameSite=Lax`; only true cross-**site** needs
  `SameSite=None; Secure` (and even then is likely blocked — see above).
- **CSRF / origin config**: `checkCSRF` must accept the *frontend* origin; add a config
  value for the allowed frontend origin(s) separate from the API origin.
- **Post-login redirect**: after `/auth/callback`, Go must redirect to the frontend's
  `/admin` (its origin), not the API host — make the post-login target configurable.
- **SPA API base**: `web/src/lib/api.ts` must prefix requests with a configurable base
  (e.g. `VITE_API_BASE`) instead of relative paths.

## Sub-path deployment (`BASE_PATH`) — planned, M8

Deploying under a path prefix on an existing domain (`id.example.com/register` behind an
Nginx `proxy_pass`) is **not supported yet** and cannot be achieved with proxy config
alone:

- The SPA build hard-codes absolute URLs (`/_app/...` assets, `/api`, `/auth`, `/admin`
  links), so a prefix-stripping `proxy_pass` leaks every browser-side request back to the
  domain root. SvelteKit's `paths.base` fixes that, but it is **baked in at build time** —
  it cannot be changed at container runtime.
- Sharing the target IdP's own domain is an extra trap: on a Rauthy host, `/auth/*` is
  already Rauthy's — this app's `/auth/login`/`/auth/callback` cannot live at the root of
  that domain at all, so prefix-stripping is structurally impossible there.

The planned feature (see ROADMAP M8): `kit.paths.base` from a build-time env, frontend
links/fetches via `$app/paths` `base`, and a Go `BASE_PATH` config (`http.StripPrefix`,
prefixed redirects, cookie `Path`). `OIDC_REDIRECT_URI` already accommodates the prefixed
callback. Consequence for packaging: sub-path deployments rebuild the frontend with their
prefix; published all-in-one images remain root-path. Until then, use a dedicated
(sub)domain such as `register.example.com` — zero changes required.

## Build output location

adapter-static currently writes into `internal/web/assets/spa` (for embedding, Mode 1).
For a standalone bundle (Mode 3), the output dir should be **parameterized** (e.g. an env
read in `web/svelte.config.js`) so the same source builds either the embedded artifact or
a free-standing `build/` to upload. The build content is otherwise identical when using
relative API paths (3a) — only cross-origin (3b) needs a different build (`VITE_API_BASE`).

## Crow CI build targets (when wired — ADR-0010, deferred)

1. **all-in-one image** — Mode 1 multi-stage `Dockerfile` → single OCI image.
2. **api image** — Mode 2 (`SERVE_FRONTEND=false`) → smaller API-only OCI image.
3. **static frontend** — Mode 3 `pnpm build` → tarball / upload artifact; deploy step is
   host-specific (nginx: copy to served dir; Cloudflare: `wrangler pages deploy`).

<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Production Compose Deployment

This is the intended near-term production shape: one Docker Compose deployment
pulls an already-built all-in-one image, reads `.env`, and runs the Go service
that serves both the embedded Svelte frontend and the `/api` + `/auth` backend.
Frontend/backend separation remains a later deployment target.

## Files

- `compose.yml` — app container, default SQLite volume, optional PostgreSQL profile.
- `.env.example` — all required runtime configuration placeholders.

## Rauthy Setup

Create these in the production Rauthy instance before starting idp-register:

1. **OIDC client for admin login**
   - Redirect URI: `https://register.example.com/auth/callback`
   - Client ID / secret: put into `OIDC_CLIENT_ID` and `OIDC_CLIENT_SECRET`
   - Issuer: usually `https://auth.example.com/auth/v1/` with the trailing slash
   - Scopes: `openid email profile groups` unless your IdP uses a different claim setup

2. **Provisioning API key**
   - Name: put into `RAUTHY_API_KEY_NAME`
   - Secret: put into `RAUTHY_API_KEY_SECRET`
   - Required rights: users read/create/update, plus groups read

3. **Admin authorization**
   - Prefer a group emitted in the configured `OIDC_GROUPS_CLAIM`
   - Put that group into `OIDC_ADMIN_GROUP`
   - Use `OIDC_ADMIN_EMAILS` or `OIDC_ADMIN_SUBS` only if the IdP cannot emit groups

4. **Mail**
   - Rauthy SMTP must work. idp-register does not send the happy-path activation email;
     Rauthy sends the first-password email when the user is created.

## SQLite Deployment

SQLite is the default when `DATABASE_URL` is unset.

```sh
cp .env.example .env
$EDITOR .env
docker compose up -d
```

The database file is stored in the `app-data` volume at `/data/idp-register.db`.

## PostgreSQL Deployment

Uncomment and fill the PostgreSQL variables in `.env`, especially
`DATABASE_URL` and `POSTGRES_PASSWORD`, then start with the profile enabled:

```sh
docker compose --profile postgres up -d
```

Use PostgreSQL when you want external database backups, monitoring, or higher
write concurrency. Do not switch between SQLite and PostgreSQL without a manual
data migration; see `docs/DATABASE.md`.

## Reverse Proxy

Point Nginx, Caddy, Cloudflare Tunnel, or another reverse proxy at the app port.
The browser-facing URL must match `ORIGIN`, and the Rauthy OIDC client redirect
URI must match `OIDC_REDIRECT_URI`.

The compose file binds to `127.0.0.1:8080` by default. Change
`IDP_REGISTER_BIND` only when the service must be reachable from another host.

## First Production Check

After startup:

1. Open `/healthz`.
2. Open `/login` and complete admin OIDC login.
3. In the admin UI, confirm Rauthy groups load.
4. Create a public permission profile from real Rauthy groups.
5. Submit a public application without an invite and approve it.
6. Confirm Rauthy created the user with the expected groups and sent exactly one
   activation / first-password email.
7. Mint an invite code bound to a profile and confirm the auto-approval path.

## Turnstile

Set `TURNSTILE_SITE_KEY` and `TURNSTILE_SECRET` together (from the same
Cloudflare Turnstile site) to enable the challenge. The site key is a public
value served to the browser at runtime via `GET /api/form`, so the stock
published image works — no frontend rebuild is needed. The app refuses to start
with only one of the two set. Leaving both unset disables verification, which is
not recommended for an internet-facing form (ADR-0016).

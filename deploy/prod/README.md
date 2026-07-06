<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Production Compose Deployment

This is the intended near-term production shape: one Docker Compose deployment
pulls an already-built all-in-one image, reads `.env`, and runs the Go service
that serves both the embedded Svelte frontend and the `/api` + `/auth` backend.
Frontend/backend separation remains a later deployment target.

## Files

- `compose.yml` — app container, default SQLite volume, optional PostgreSQL profile.
- `.env.example` — all required runtime configuration placeholders.

## Image & Pulling

Set `IDP_REGISTER_IMAGE` to the published multi-arch (amd64 + arm64) image, e.g.
`forgejo.goba.ip-dynamic.org/gobro/idp-register:0.2.0`. Tags: `X.Y.Z` and `X.Y`
per release, `latest` follows `main`. Pin a version tag or digest in production.

**Known issue — anonymous pull fails with `401 Unauthorized` on containerd-based
clients.** The package is public and anonymous pulls work with the classic Docker
client, but clients that pull through containerd's resolver — Docker Engine 29+
with the containerd image store (its default), Kubernetes/k3s kubelets, nerdctl —
get `401` on the manifest request. Cause: Forgejo's `WWW-Authenticate` challenge
advertises `scope="*"`; containerd's resolver then obtains an anonymous token
without the repository pull scope, which Forgejo rejects. Workarounds:

- `docker login forgejo.goba.ip-dynamic.org` with a Forgejo access token that has
  `package:read` (on Kubernetes, the same credentials as an `imagePullSecret`); or
- switch Docker back to the classic image store
  (`/etc/docker/daemon.json`: `{"features": {"containerd-snapshotter": false}}`
  and restart the daemon).

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

## Registration rules & terms of service

Deployments can show their own registration rules and require consent, all at
runtime — the stock image needs no rebuild:

- `FORM_RULES_TEXT` — rules shown above the form (plain text, line breaks
  preserved, never rendered as HTML). For multi-line rules prefer
  `FORM_RULES_FILE` (a path inside the container; bind-mount the file), since
  `.env` values are single-line. Text and file are mutually exclusive.
- `FORM_TERMS_URL` — link to your terms-of-service page, shown in the consent
  checkbox.

If any of these is set, the form renders an "I have read and agree" checkbox
and the server rejects submissions that do not carry the consent flag. Leaving
all unset removes both the rules panel and the checkbox.

## Secrets via sops (optional)

Instead of keeping secrets in plaintext `.env`, they can live in a
[sops](https://github.com/getsops/sops)-encrypted file and be injected only for
the lifetime of the `docker compose` command:

```sh
# secrets.enc.yaml — flat KEY: value pairs, encrypted with sops (age/SSH keys):
#   OIDC_CLIENT_SECRET: ...
#   RAUTHY_API_KEY_SECRET: ...
#   TURNSTILE_SITE_KEY: 0x4AAA...      # public, but convenient to keep with its secret
#   TURNSTILE_SECRET: 0x4AAA...

sops exec-env secrets.enc.yaml 'docker compose up -d'
```

`sops exec-env` decrypts into the environment of the child process only —
nothing plaintext touches the disk. `compose.yml` forwards these variables into
the container via `${VAR:-}` interpolation, which reads the process environment
first and falls back to `.env`, so a plain-`.env` deployment keeps working
unchanged. Non-secret settings (image, origin, OIDC issuer, …) stay in `.env`.

Notes:

- Only variables listed under the app service's `environment:` block are
  forwarded this way (`OIDC_CLIENT_SECRET`, `RAUTHY_API_KEY_SECRET`,
  `TURNSTILE_SITE_KEY`, `TURNSTILE_SECRET`). To move another secret (e.g.
  `DATABASE_URL` with an embedded password) into sops, add the same
  `VAR: ${VAR:-}` mapping to `compose.yml`.
- The deploy host must hold a private key matching a recipient in `.sops.yaml`
  (age key or SSH ed25519 key). Add the host's key as a recipient and re-run
  `sops updatekeys` before deploying from that host.
- An encrypted `secrets.enc.yaml` is safe to commit if you want it versioned;
  keep it out of the image build context regardless (it is not needed there).
- Any `docker compose` invocation that (re)creates the app container needs the
  wrapper (`up`, `run`); plain `start`/`restart`/`stop` of an existing
  container does not re-read the environment.

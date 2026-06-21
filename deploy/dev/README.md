<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Local IdP test setup (Rauthy)

The simplest way to exercise idp-register end-to-end against a real OIDC IdP.
Compose runs **Rauthy** (HTTP, `:8080`) plus **Mailcrab** (`:1080`) for captured
test email. Rauthy is fully bootstrapped on first boot; idp-register runs on the
**host** at `:8081`.

## Why idp-register runs on the host, not in compose

The OIDC issuer must look identical to the browser *and* to idp-register's
server-side discovery/token calls. With Rauthy published at `localhost:8080` and
idp-register on the host, both see `http://localhost:8080` — no DNS/issuer
mismatch. (Putting idp-register in the same compose network would make it resolve
`localhost` to itself; that needs `/etc/hosts` aliasing, which we avoid here.)

## What gets bootstrapped

| Thing | Value |
|---|---|
| Admin login | `admin@localhost` / `TestAdmin1234!` |
| OIDC client | `idp-register` (redirect `http://localhost:8081/auth/callback`) |
| Provisioning API key | `idp-register` (Users: read/create/update) |
| Captured email | Mailcrab at `http://localhost:1080/` |

All secrets are throwaway test values.

## Steps

```sh
# 1. Start Rauthy + Mailcrab (first boot bootstraps the admin, API key, and OIDC client)
docker compose -f deploy/dev/docker-compose.yml up -d
docker compose -f deploy/dev/docker-compose.yml logs -f rauthy   # wait for "listening"

# 2. Build the SPA and run idp-register on the host at :8081
pnpm --filter idp-register-web build
set -a; source deploy/dev/idp-register.env; set +a
go run ./cmd/server
```

Then:

- **Public form** — http://localhost:8081/
- **Admin** — http://localhost:8081/login → "登录" → Rauthy login
  (`admin@localhost` / `TestAdmin1234!`). idp-register authorizes this account via
  `OIDC_ADMIN_EMAILS`.
- **Captured mail** — http://localhost:1080/
- **Provisioning test** — mint an invite code in the admin UI, submit the public
  form with it; idp-register calls the Rauthy API to create the user. Verify in
  Rauthy's own admin UI at http://localhost:8080/auth/v1/admin.

## M3 smoke-test checklist

Use this checklist after code changes that touch registration, profiles, invite codes,
provisioning, OIDC login, or Rauthy/Mailcrab harness config.

### Harness

- Start Rauthy + Mailcrab and wait for Rauthy to listen.
- Build the SPA and start idp-register on `http://localhost:8081`.
- Open `http://localhost:8081/login`, complete Rauthy admin login, and confirm the admin
  dashboard loads.
- Open Mailcrab at `http://localhost:1080/` and clear old messages before provisioning
  assertions.

### Profiles

- In the admin UI, open **Profiles**.
- Confirm the group picker loads real Rauthy groups from `GET /api/admin/groups`.
- Create or update a profile that:
  - uses only non-denylisted groups;
  - is marked public-selectable;
  - has a public label and sort order.
- Open the public form and confirm the service selector shows the public profile label, not
  raw IdP group names.

### Manual-review path

- Submit the public form without an invite code and with the public service selected.
- Confirm the user gets the same uniform "received" response as every other submission.
- In the admin UI, confirm the application is `pending` and `requestedServices` contains
  the selected profile id.
- Approve it with a permission profile.
- Confirm the application becomes `approved`, Rauthy has created the user with the expected
  groups, and Mailcrab has exactly one new `Rauthy IAM - New Password` message.

### Invite auto-approval path

- Create an invite code in the admin UI. A `profileId` is required; creation without one
  must return `400`.
- Submit the public form with that valid invite code. Any submitted `services` value is
  advisory and must be overridden by the token-bound profile.
- Confirm the public response is still the uniform `202`.
- Confirm the application becomes `approved`, `approvedProfileId` equals the token-bound
  profile, and `requestedServices` is exactly `[token.profile_id]`.
- Confirm the token counters end at `pending=0, completed=1` after provisioning succeeds.
- Confirm Mailcrab has exactly one new `Rauthy IAM - New Password` message.

### Invalid or missing invite behavior

- Submit with no invite code: the application must remain `pending` for admin review.
- Submit with a malformed, expired, disabled, over-capacity, wrong-email, or unknown invite
  code: the application must also remain `pending`.
- Confirm these cases return the same public response as a successful invite submission;
  do not expose whether the code or email exists.
- Confirm no invite counter changes for invalid or non-reserved codes.

### Provisioning-failure cleanup

- Force or simulate a provisioning failure after a valid invite has reserved a slot.
- Confirm the application becomes `provisioning_failed` and the invite reservation remains
  held (`pending` stays incremented).
- From the application detail page, retry approval after fixing the cause.
- If the application should not proceed, reject it and confirm the held invite reservation
  is released (`pending` decrements).

## Reset

```sh
docker compose -f deploy/dev/docker-compose.yml down -v   # wipes the Rauthy DB
rm -f idp-register-dev.db                                 # wipes idp-register's DB
```

## Verified (Rauthy 0.35.2, arm64 macOS / Docker linux/arm64)

Booted clean: image is multi-arch (no arch issue on Apple Silicon), `clients.json`
bootstrapped ("Migrated 1 clients"), API key returns 200 on `/auth/v1/users`,
Rauthy connects to Mailcrab SMTP, and idp-register completes OIDC discovery +
`/auth/login` 302-redirects to Rauthy.

Mail behavior verified on 2026-06-21:

- `POST /auth/v1/users` with `roles: []` creates the user and sends one Mailcrab
  message with subject `Rauthy IAM - New Password` and a `type=new_user` set-password
  link.
- `POST /auth/v1/users/request_reset` with only `email` returns 400
  (`missing field pow`) and sends no mail. This is true both with and without the
  provisioning API key.

## Gotchas found on first boot (read before migrating to another machine)

1. **`config.toml` is mandatory.** Rauthy 0.35.x **panics** (`Cannot read config file
   from ./config.toml`) if the file is absent — *even when every value is set via env
   vars*. We mount an effectively-empty `config.toml`; env vars supply the real config.
2. **Issuer has a TRAILING SLASH.** Rauthy's `iss` is `http://localhost:8080/auth/v1/`.
   `coreos/go-oidc` requires an exact match, so `OIDC_ISSUER` **must** end with `/`.
   (`RAUTHY_API_BASE` is fine without it — the Go config trims trailing slashes.)
3. **Quote env values with spaces.** `OIDC_SCOPES="openid email profile groups"` must be
   quoted or `set -a; source ...` runs `email` as a command.
4. **First-login password change.** Rauthy logs *"a bootstrap password has been given …
   Please change it immediately"*; it may prompt a password change on first Rauthy login.
   That's expected — change it once in Rauthy's UI, then idp-register login works.
5. **Architecture was NOT the problem here** — but if a future Rauthy tag is single-arch,
   add `platform: linux/amd64` under the `rauthy` service to run it via emulation.

## Two-process dev (frontend HMR + separate backend)

The steps above are **single-origin** (build the SPA, Go serves it + API on `:8081`).
For a split with frontend hot-reload, run two processes. The browser now lives on the
Vite port (`:5173`), so that becomes the public origin — `:5173/auth/callback` is already
registered in `bootstrap/clients.json`.

```sh
# Terminal 1 — Go backend on :8081, but public origin = the Vite port :5173
set -a; source deploy/dev/idp-register.env; set +a
export ORIGIN=http://localhost:5173
export OIDC_REDIRECT_URI=http://localhost:5173/auth/callback
go run ./cmd/server

# Terminal 2 — Vite dev server on :5173, proxying /api + /auth to the backend
BACKEND_ORIGIN=http://localhost:8081 pnpm --filter idp-register-web dev
```

Open **http://localhost:5173/**. Vite proxies `/api` and `/auth` to `:8081`; cookies are
host-scoped so they work across ports.

Caveats:
- **`vite preview` will NOT work as the frontend here** — its server doesn't proxy
  `/api`/`/auth` (SvelteKit's preview middleware intercepts them). To exercise the
  *production* build, use the single-origin embedded mode above (that IS the prod artifact).
- **`go run` leaves a zombie.** It spawns the compiled server as a child; killing `go run`
  (or closing the terminal) can leave the server bound to `:8081`, so the next start
  silently fails to bind and you hit the stale one. Stop it with Ctrl-C in the foreground,
  or `lsof -ti tcp:8081 | xargs kill`. (`go build -o /tmp/srv ./cmd/server && /tmp/srv`
  avoids the indirection.)

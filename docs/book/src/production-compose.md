<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Production Compose

The production Compose example runs the all-in-one image behind a reverse proxy.
SQLite is enabled by default through a persistent Docker volume. PostgreSQL is
available through a Compose profile.

## Minimal Compose File

This is the same shape used by `deploy/prod/compose.yml`:

```yaml
services:
  app:
    image: ${IDP_REGISTER_IMAGE:?set IDP_REGISTER_IMAGE in .env}
    env_file:
      - path: .env
        required: false
    environment:
      ADDR: ":8080"
      SQLITE_PATH: ${SQLITE_PATH:-/data/idp-register.db}
      OIDC_CLIENT_SECRET: ${OIDC_CLIENT_SECRET:-}
      RAUTHY_API_KEY_SECRET: ${RAUTHY_API_KEY_SECRET:-}
      TURNSTILE_SITE_KEY: ${TURNSTILE_SITE_KEY:-}
      TURNSTILE_SECRET: ${TURNSTILE_SECRET:-}
    ports:
      - "${IDP_REGISTER_BIND:-127.0.0.1:8080}:8080"
    volumes:
      - app-data:/data
    restart: unless-stopped

volumes:
  app-data:
```

The default bind address, `127.0.0.1:8080`, is appropriate when Caddy, Nginx, or
Cloudflare Tunnel runs on the same host.

## Example .env

Start from `deploy/prod/.env.example`. A compact SQLite configuration looks like
this:

```dotenv
IDP_REGISTER_IMAGE=forgejo.goba.ip-dynamic.org/gobro/idp-register:0.2.0
IDP_REGISTER_BIND=127.0.0.1:8080

ORIGIN=https://register.example.com
TRUST_CF_CONNECTING_IP=true
SQLITE_PATH=/data/idp-register.db

OIDC_ISSUER=https://auth.example.com/auth/v1/
OIDC_CLIENT_ID=idp-register
OIDC_CLIENT_SECRET=replace-with-oidc-client-secret
OIDC_REDIRECT_URI=https://register.example.com/auth/callback
OIDC_SCOPES=openid email profile groups
OIDC_GROUPS_CLAIM=groups
OIDC_ADMIN_GROUP=svc:idp-register:admin

PROFILE_GROUP_DENYLIST=admin,rauthy_admin

PROVISIONER=rauthy
RAUTHY_API_BASE=https://auth.example.com/auth/v1
RAUTHY_API_KEY_NAME=idp-register
RAUTHY_API_KEY_SECRET=replace-with-rauthy-api-key-secret
RAUTHY_DEFAULT_LANGUAGE=en
RAUTHY_DEFAULT_TIMEZONE=UTC

SESSION_TTL_HOURS=12
SECURE_COOKIES=true
APP_ENV=production

# Optional but recommended for an internet-facing public form.
TURNSTILE_SITE_KEY=
TURNSTILE_SECRET=

# Optional public-form rules and terms.
FORM_RULES_FILE=/data/rules.txt
FORM_TERMS_URL=https://example.com/terms
```

Do not put quotes around values unless your shell or Compose workflow requires
them. For multi-line registration rules, mount a file into the container and set
`FORM_RULES_FILE`; `.env` values are better kept single-line.

## Start With SQLite

SQLite is used when `DATABASE_URL` is unset:

```sh
cd deploy/prod
cp .env.example .env
$EDITOR .env
docker compose up -d
```

The database file is stored at `SQLITE_PATH` inside the app container. With the
sample Compose file, that means `/data/idp-register.db` in the `app-data`
volume.

## Start With PostgreSQL

Uncomment and fill these values in `.env`:

```dotenv
DATABASE_URL=postgres://idp_register:replace-with-db-password@postgres:5432/idp_register?sslmode=disable
POSTGRES_DB=idp_register
POSTGRES_USER=idp_register
POSTGRES_PASSWORD=replace-with-db-password
```

Then start the stack with the profile:

```sh
docker compose --profile postgres up -d
```

Use PostgreSQL when you want external database backup tooling, monitoring, or
higher write concurrency. Do not switch an existing deployment between SQLite
and PostgreSQL just by changing `DATABASE_URL`; that requires a data migration.

## First Production Check

After startup, check the deployment in this order:

1. `GET /healthz` responds with HTTP 200.
2. `/login` redirects to the admin-auth IdP and returns to `/auth/callback`.
3. The logged-in admin reaches `/admin`.
4. The permission profile editor loads groups from Rauthy.
5. A public application can be submitted.
6. Approval creates the user in Rauthy with the expected groups.
7. Rauthy sends exactly one activation or first-password email.

If login works but the profile editor cannot load groups, the OIDC client is
probably fine and the Rauthy provisioning API key is the part to inspect.

<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Configuration Reference

`idp-register` is configured entirely through environment variables. In Compose,
put non-secret values in `.env`. Secrets can also come from the process
environment, a secrets manager, or `sops exec-env`.

## HTTP

| Variable | Required | Default | Description |
|---|---:|---|---|
| `ADDR` | no | `:8080` | Listen address inside the container or process. |
| `ORIGIN` | yes | | Public browser-facing origin, for example `https://register.example.com`. No trailing slash is needed. |
| `TRUST_CF_CONNECTING_IP` | no | `true` | Use `CF-Connecting-IP` as the client IP when behind Cloudflare. |

`ORIGIN` must match the URL users type in the browser. It is used for CSRF
checks on admin mutations, so a mismatch can break POST, PUT, and DELETE
requests after login.

## Database

| Variable | Required | Default | Description |
|---|---:|---|---|
| `DATABASE_URL` | no | | PostgreSQL DSN. When set, PostgreSQL is used and `SQLITE_PATH` is ignored. |
| `SQLITE_PATH` | no | `idp-register.db` | SQLite path or DSN. In the container, use a mounted path such as `/data/idp-register.db`. |

Choose one database engine for a deployment. Switching engines later is a manual
migration, not a configuration toggle.

## Admin OIDC Login

| Variable | Required | Default | Description |
|---|---:|---|---|
| `OIDC_ISSUER` | yes | | Issuer URL from the admin-auth IdP discovery metadata. |
| `OIDC_CLIENT_ID` | yes | | Client ID for the admin UI OIDC client. |
| `OIDC_CLIENT_SECRET` | yes | | Client secret for the admin UI OIDC client. |
| `OIDC_REDIRECT_URI` | yes | | Callback URL, normally `https://register.example.com/auth/callback`. |
| `OIDC_SCOPES` | no | `openid email profile groups` | Space- or comma-separated scopes. |
| `OIDC_GROUPS_CLAIM` | no | `groups` | Claim containing admin group or role names. |
| `OIDC_ADMIN_GROUP` | no | `svc:idp-register:admin` | Group value that grants admin access. |
| `OIDC_ADMIN_SUBS` | no | | Space- or comma-separated subject allow-list. |
| `OIDC_ADMIN_EMAILS` | no | | Space- or comma-separated email allow-list. |

Prefer group-based authorization. Use subject or email allow-lists only when the
IdP cannot emit a reliable group or role claim.

## Provisioning Target IdP

| Variable | Required | Default | Description |
|---|---:|---|---|
| `PROVISIONER` | no | `rauthy` | Provisioner implementation. Only `rauthy` is currently implemented. |
| `RAUTHY_API_BASE` | yes, for Rauthy | | Rauthy API base URL, usually ending in `/auth/v1` without a trailing slash. |
| `RAUTHY_API_KEY_NAME` | yes, for Rauthy | | Name of the Rauthy API key. |
| `RAUTHY_API_KEY_SECRET` | yes, for Rauthy | | Secret value of the Rauthy API key. |
| `RAUTHY_DEFAULT_LANGUAGE` | no | `en` | Language assigned to created Rauthy users. |
| `RAUTHY_DEFAULT_TIMEZONE` | no | `UTC` | Timezone assigned to created Rauthy users. |
| `PROFILE_GROUP_DENYLIST` | no | `admin,rauthy_admin` | Groups never offered for registrant permission profiles. `OIDC_ADMIN_GROUP` is always added automatically. |

The admin-auth IdP and the provisioning target IdP are separate roles. They may
point at the same Rauthy instance, but they still use different credentials and
different protocols.

## Sessions and Cookies

| Variable | Required | Default | Description |
|---|---:|---|---|
| `SESSION_TTL_HOURS` | no | `12` | Admin session lifetime. Invalid or non-positive values fall back to 12 hours. |
| `SECURE_COOKIES` | no | `true` when `APP_ENV=production`, otherwise `false` | Adds the `Secure` cookie attribute. Must be true behind HTTPS. |
| `APP_ENV` | no | | Set to `production` to make secure cookies the default. |

For public HTTPS deployments, use `APP_ENV=production` and
`SECURE_COOKIES=true`.

## Public Form Protection

| Variable | Required | Default | Description |
|---|---:|---|---|
| `TURNSTILE_SITE_KEY` | no | | Cloudflare Turnstile site key. Public value served to the browser. |
| `TURNSTILE_SECRET` | no | | Cloudflare Turnstile secret used by the backend. |

Set both Turnstile variables to enable the challenge. Leave both unset to
disable it. Setting only one is a startup error.

## Registration Rules and Terms

| Variable | Required | Default | Description |
|---|---:|---|---|
| `FORM_RULES_TEXT` | no | | Plain text rules shown above the public form. |
| `FORM_RULES_FILE` | no | | Path to a file containing plain text rules. Mutually exclusive with `FORM_RULES_TEXT`. |
| `FORM_TERMS_URL` | no | | Terms-of-service URL linked from the consent checkbox. |

If any of these values is set, the public form requires consent and the server
rejects submissions that do not include it.

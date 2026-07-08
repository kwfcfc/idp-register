<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# OIDC and Rauthy Setup

Rauthy currently appears in two places:

- as a possible admin-auth OIDC provider;
- as the implemented provisioning target IdP.

Those roles are configured separately even when the same Rauthy instance provides
both. Do not reuse the OIDC client secret as the Rauthy provisioning API key.

## Admin OIDC Client

Create an OIDC client in the IdP used by administrators. This IdP can be
Rauthy, Authentik, Dex, or another standards-compliant OIDC provider.

Typical values:

```dotenv
OIDC_ISSUER=https://auth.example.com/auth/v1/
OIDC_CLIENT_ID=idp-register
OIDC_CLIENT_SECRET=replace-with-oidc-client-secret
OIDC_REDIRECT_URI=https://register.example.com/auth/callback
OIDC_SCOPES=openid email profile groups
OIDC_GROUPS_CLAIM=groups
OIDC_ADMIN_GROUP=svc:idp-register:admin
```

In the IdP client configuration, set:

- redirect URI: `https://register.example.com/auth/callback`;
- scopes: usually `openid email profile groups`;
- client type: confidential web application, if your IdP asks;
- PKCE: allowed or required.

The issuer string must match the IdP discovery metadata exactly. For Rauthy,
that commonly means `https://auth.example.com/auth/v1/` with the trailing slash.

## Admin Authorization

After OIDC login, `idp-register` only admits users that match one of these:

- their configured groups claim contains `OIDC_ADMIN_GROUP`;
- their `sub` claim appears in `OIDC_ADMIN_SUBS`;
- their email appears in `OIDC_ADMIN_EMAILS`.

Group-based authorization is the best default because it stays managed in the
IdP. Use `OIDC_ADMIN_SUBS` or `OIDC_ADMIN_EMAILS` only for IdPs that cannot emit
a group or role claim.

## Provisioning API Key

Create a Rauthy API key for the provisioner:

```dotenv
PROVISIONER=rauthy
RAUTHY_API_BASE=https://auth.example.com/auth/v1
RAUTHY_API_KEY_NAME=idp-register
RAUTHY_API_KEY_SECRET=replace-with-rauthy-api-key-secret
RAUTHY_DEFAULT_LANGUAGE=en
RAUTHY_DEFAULT_TIMEZONE=UTC
```

The API key needs rights to:

- read users;
- create users;
- update users;
- read groups.

Group read permission is required because permission profiles are validated
against the live target-IdP group catalog.

## Permission Profiles

After the first admin login, open the permission profile page and create one or
more profiles from Rauthy groups. If a profile should be selectable on the
public form, mark it public and give it a user-facing label.

Use `PROFILE_GROUP_DENYLIST` to hide infrastructure groups from the profile
editor. The value of `OIDC_ADMIN_GROUP` is always denied automatically.

## Mail

Rauthy SMTP must work in production. `idp-register` relies on Rauthy to send the
activation or first-password email after a user is created.

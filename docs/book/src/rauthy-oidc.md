<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Rauthy and OIDC Setup

Rauthy currently appears in two places:

- as a possible admin-auth OIDC provider;
- as the implemented provisioning target IdP.

Those roles are configured separately even when the same Rauthy instance provides
both.

## Admin OIDC Client

Create an OIDC client for the admin UI:

- redirect URI: `https://register.example.com/auth/callback`;
- scopes: usually `openid email profile groups`;
- issuer: Rauthy normally uses a path ending in `/auth/v1/`.

The issuer string must match Rauthy's discovery metadata exactly, including the
trailing slash when Rauthy emits one.

## Provisioning API Key

Create a Rauthy API key for the provisioner with rights to read, create, and
update users, plus read groups. The group read permission is required because
permission profiles are validated against the live target-IdP group catalog.

## Mail

Rauthy SMTP must work in production. `idp-register` relies on Rauthy to send the
activation or first-password email after a user is created.

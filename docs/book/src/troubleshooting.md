<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Troubleshooting

## OIDC Login Fails

Check that `OIDC_ISSUER` exactly matches the IdP discovery issuer. Rauthy commonly
requires the trailing slash in `/auth/v1/`.

Check that `OIDC_REDIRECT_URI` exactly matches the client configuration in the
admin-auth IdP.

## Groups Do Not Load

For Rauthy, the provisioning API key must include group read permission. The
profile editor validates selected groups against the live target-IdP catalog.

## Public Form Accepts Submissions Without Challenge

If both Turnstile variables are unset, challenge verification is disabled. Set
both `TURNSTILE_SITE_KEY` and `TURNSTILE_SECRET` for an internet-facing form.

## Duplicate Existing User

If the target IdP reports that the user already exists, the application should
normally be rejected or handled manually in the IdP. The broker treats successful
target-IdP user creation as provisioning success; later activation state belongs
to the IdP.

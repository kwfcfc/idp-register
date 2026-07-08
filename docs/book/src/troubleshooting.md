<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Troubleshooting

## Service Does Not Start

Check the container logs:

```sh
docker compose logs --tail=200 app
```

Common startup configuration errors:

- missing `ORIGIN`;
- missing OIDC variables;
- missing Rauthy API variables when `PROVISIONER=rauthy`;
- only one of `TURNSTILE_SITE_KEY` and `TURNSTILE_SECRET` is set;
- both `FORM_RULES_TEXT` and `FORM_RULES_FILE` are set;
- `FORM_RULES_FILE` points to a file that is not mounted into the container.

## OIDC Login Fails

Check that `OIDC_ISSUER` exactly matches the IdP discovery issuer. Rauthy commonly
requires the trailing slash in `/auth/v1/`.

Check that `OIDC_REDIRECT_URI` exactly matches the client configuration in the
admin-auth IdP.

Check that the browser-facing URL, `ORIGIN`, and `OIDC_REDIRECT_URI` use the
same scheme and host. For example:

```dotenv
ORIGIN=https://register.example.com
OIDC_REDIRECT_URI=https://register.example.com/auth/callback
```

## Login Succeeds But Admin Access Is Denied

The OIDC login worked, but the user did not match admin authorization. Check:

- the groups or roles claim name in `OIDC_GROUPS_CLAIM`;
- the configured admin group in `OIDC_ADMIN_GROUP`;
- whether the IdP actually includes the group claim in the ID token;
- `OIDC_ADMIN_EMAILS` or `OIDC_ADMIN_SUBS` if using allow-lists.

## Groups Do Not Load

For Rauthy, the provisioning API key must include group read permission. The
profile editor validates selected groups against the live target-IdP catalog.

Also check that `RAUTHY_API_BASE` is the API base, usually
`https://auth.example.com/auth/v1`, not just the root website URL.

## Public Form Accepts Submissions Without Challenge

If both Turnstile variables are unset, challenge verification is disabled. Set
both `TURNSTILE_SITE_KEY` and `TURNSTILE_SECRET` for an internet-facing form.

If the widget appears but every submission fails, confirm that the site key and
secret belong to the same Turnstile site.

## Duplicate Existing User

If the target IdP reports that the user already exists, the application should
normally be rejected or handled manually in the IdP. The broker treats successful
target-IdP user creation as provisioning success; later activation state belongs
to the IdP.

## Form Requires Consent Unexpectedly

The consent checkbox appears when any of these are set:

- `FORM_RULES_TEXT`
- `FORM_RULES_FILE`
- `FORM_TERMS_URL`

Unset all three to remove the consent requirement.

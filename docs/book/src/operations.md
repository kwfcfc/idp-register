<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Operations

## Public Form Protection

Production deployments should enable Turnstile by setting both
`TURNSTILE_SITE_KEY` and `TURNSTILE_SECRET`. Setting only one of them is a
startup configuration error.

## Registration Rules and Terms

Use `FORM_RULES_TEXT` or `FORM_RULES_FILE` to show deployment-specific
registration rules. Use `FORM_TERMS_URL` to link terms of service. When any of
these are set, the public form requires consent and the server rejects
submissions without it.

## Invite Recovery

If provisioning fails after an invite code reserves a slot, the reservation stays
held. An admin can retry approval after fixing the cause, or reject the
application to release the pending invite reservation.

## Secrets

Plain `.env` files are supported. Production deployments can also use `sops
exec-env` to inject selected secrets into `docker compose` without writing
plaintext secrets to disk.

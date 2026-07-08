<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Operations

## Health Checks

Use `/healthz` for a basic process and database check:

```sh
curl -fsS https://register.example.com/healthz
```

If this fails, inspect the app container logs first:

```sh
docker compose logs --tail=200 app
```

## Backups

Back up the database before upgrades.

For SQLite, back up the `/data/idp-register.db` file from the `app-data` volume.
Stop the container or use a SQLite-aware backup method to avoid copying a file
mid-write.

For PostgreSQL, use your normal PostgreSQL backup tooling, for example
`pg_dump`, scheduled snapshots, or managed database backups.

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

Do not search logs for invite codes. Codes are plaintext by design, but they
must not be logged or pasted into support tickets.

## Secrets

Plain `.env` files are supported. Production deployments can also use `sops
exec-env` to inject selected secrets into `docker compose` without writing
plaintext secrets to disk.

Example:

```sh
sops exec-env secrets.enc.yaml 'docker compose up -d'
```

The included production Compose file forwards these secret variables from the
process environment into the container:

- `OIDC_CLIENT_SECRET`
- `RAUTHY_API_KEY_SECRET`
- `TURNSTILE_SITE_KEY`
- `TURNSTILE_SECRET`

If you move another secret, such as a PostgreSQL DSN with an embedded password,
into `sops`, add it to the Compose `environment` block as well.

## Upgrades

For a pinned image tag:

```sh
docker compose pull
docker compose up -d
```

Check `/healthz`, admin login, group loading, and one test application after the
upgrade.

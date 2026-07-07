<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Production Compose

The production Compose deployment pulls a published all-in-one image and runs
the Go service behind a reverse proxy.

Basic shape:

1. Copy `deploy/prod/.env.example` to `deploy/prod/.env`.
2. Set `IDP_REGISTER_IMAGE` to a published version tag or digest.
3. Fill in `ORIGIN`, OIDC, Rauthy provisioning, cookie, and database settings.
4. Start the stack with `docker compose up -d`.

SQLite is the default when `DATABASE_URL` is unset. PostgreSQL is available via
the Compose profile in `deploy/prod/compose.yml`.

After startup, check:

- `/healthz` responds;
- `/login` completes admin OIDC login;
- the admin profile editor can load Rauthy groups;
- a public application can be submitted and approved;
- Rauthy creates the user and sends exactly one activation or first-password
  email.

For the complete deployment checklist, see `deploy/prod/README.md` in the source
repository.

<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# idp-register

A small, self-hosted **user-registration broker** for an OIDC identity provider — the
OIDC-era equivalent of Synapse's built-in **registration tokens**.

People apply through a public form. An **invite code** (limited by use-count and expiry)
can **auto-approve** them; otherwise an **admin reviews** the application. On approval the
service **provisions the user into the target IdP** (Rauthy today; Kanidm planned) and lets
the IdP send a single "set your password / activate" email — which also verifies the address.

Built for a self-hosted stack (Matrix/Synapse + GoToSocial + Forgejo behind **Rauthy**),
deployable as one OCI image behind Nginx/Cloudflare.

## Two IdP roles (don't conflate them)

- **Admin-auth IdP** — where admins log in. A standards-only **OIDC Relying Party**; never
  assumes provider-specific APIs.
- **Provisioning target IdP** — where end users are created. Hidden behind a `Provisioner`
  interface; provider-specific code is isolated.

## Status

> ⚠️ **In progress / being re-based.** An initial AI-generated **full-stack SvelteKit**
> draft exists (admin site only, PostgreSQL-only, HMAC-digest codes). It is being re-based
> to the **target architecture below**. Treat the docs as the source of truth, not the
> current `src/` tree.

**Target stack:** Go backend · SvelteKit frontend (`adapter-static`, embedded via
`go:embed`) · `database/sql` over SQLite **or** PostgreSQL · generic OIDC admin login ·
plaintext Synapse-style invite codes · GPLv3-or-later.

## Documentation

- [`AGENTS.md`](AGENTS.md) — context & hard invariants for AI agents and developers (read first).
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system design, flows, data model.
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — architecture decision records (the *why*).

## Features (target)

- Public application form: optional fields, ToS consent, Cloudflare Turnstile, optional invite code.
- Invite codes: use-count + expiry + manual disable; auto-approve on valid code (Synapse
  `uses_allowed` / `pending` / `completed` / `expiry_time` semantics).
- Admin review panel (OIDC-protected): mint/list/disable codes, review/approve/reject applications.
- Provisioning into the target IdP with safe retry on failure; server-side permission profiles
  map to IdP groups (public input never sets groups directly).
- One user-facing email on the happy path (the IdP activation link).
- Admin action audit log.

## Development

> Re-base to Go is underway; commands below will change. See `AGENTS.md` for the target layout.

## License

[GPL-3.0-or-later](LICENSE). Every source file carries an SPDX header.

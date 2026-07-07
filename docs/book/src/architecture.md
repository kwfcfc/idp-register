<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Architecture

The backend is Go. The frontend is a SvelteKit static app built with
`adapter-static` and embedded into the Go binary through `go:embed`.

The main packages are:

- `internal/config`: environment to typed configuration;
- `internal/store`: portable SQLite/PostgreSQL repositories;
- `internal/oidcauth`: generic OIDC relying party for admin login;
- `internal/provisioner`: target-IdP provisioning interface and implementations;
- `internal/token`: invite-code minting and reservation logic;
- `internal/application`: application state machine and review operations;
- `internal/web`: HTTP handlers and embedded frontend serving.

The full architecture document remains `docs/ARCHITECTURE.md` in the source
repository. The book is the operator-facing rendered documentation and should
stay aligned with that source.

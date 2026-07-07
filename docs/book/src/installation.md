<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Installation

The recommended first deployment is the all-in-one image: Go serves both the API
and the embedded SvelteKit frontend from one origin.

For local development:

```sh
pnpm install
pnpm --filter idp-register-web build
go run ./cmd/server
```

For a production-style binary build:

```sh
pnpm --filter idp-register-web build
CGO_ENABLED=0 go build -o idp-register ./cmd/server
```

For a production image:

```sh
docker build -t idp-register .
```

The full production Compose path is described in the next chapter. Keep the
browser-facing URL, `ORIGIN`, and OIDC redirect URI in sync.

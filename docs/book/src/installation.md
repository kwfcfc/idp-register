<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Installation

The recommended deployment is the all-in-one container image. The Go server
serves both the API and the embedded frontend from one origin, which keeps
cookies, CSRF protection, and OIDC redirects simple.

## Requirements

- Docker Compose or another container runtime.
- A public HTTPS URL for the service, such as `https://register.example.com`.
- An OIDC client for administrator login.
- A provisioning target IdP. Rauthy is currently implemented.
- Working SMTP in the target IdP, because it sends the activation email.

For production, put `idp-register` behind a reverse proxy such as Caddy, Nginx,
Traefik, or Cloudflare Tunnel. Bind the container to localhost unless another
host must reach it directly.

## Container Image

Published images are multi-arch for amd64 and arm64:

```sh
docker pull forgejo.goba.ip-dynamic.org/gobro/idp-register:<version>
```

Pin a version tag or image digest in production. Avoid deploying an unpinned
moving tag unless you intentionally want automatic upgrades.

Release images (version tags, not the moving `latest`) are signed with cosign
and carry an SPDX software bill of materials (SBOM). The signing public key is
committed at the repository root as `cosign.pub`:

```sh
cosign verify --key cosign.pub \
  forgejo.goba.ip-dynamic.org/gobro/idp-register:<version>
```

### Inspect the SBOM

The SBOM is attached to the image index as an in-toto attestation, so you can
audit the exact package set of a release without pulling and unpacking it:

```sh
docker buildx imagetools inspect \
  forgejo.goba.ip-dynamic.org/gobro/idp-register:<version> \
  --format '{{ json .SBOM }}'
```

Feed the JSON to a scanner such as `grype sbom:-` or `trivy sbom` to check the
release against known-vulnerability databases before rolling it out.

## Quick Start With Compose

The repository includes a production Compose example under `deploy/prod/`. If
you have not cloned the repository, copy the Compose file and `.env` shown in
the [Production Compose](production-compose.md) chapter instead — pulling the
image is the only hard requirement.

```sh
cd deploy/prod
cp .env.example .env
$EDITOR .env
docker compose up -d
```

At minimum, fill in:

- `IDP_REGISTER_IMAGE`
- `ORIGIN`
- `OIDC_ISSUER`
- `OIDC_CLIENT_ID`
- `OIDC_CLIENT_SECRET`
- `OIDC_REDIRECT_URI`
- `RAUTHY_API_BASE`
- `RAUTHY_API_KEY_NAME`
- `RAUTHY_API_KEY_SECRET`

Open `/healthz` on the public URL, then visit `/login` and complete the admin
OIDC login. The first login and everything after it are covered in
[Administration](administration.md).

## Build from Source

Building your own image is optional. It is useful for air-gapped sites, a
private registry, or a supply-chain audit where you want to reproduce the
published artifact yourself. The frontend is built with pnpm, embedded into the
Go binary, and the whole thing ships as one multi-stage image.

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

For a production image, then push it to your own registry:

```sh
docker build -t registry.example.com/idp-register:local .
docker push registry.example.com/idp-register:local
```

Set `IDP_REGISTER_IMAGE` in `.env` to the tag you pushed. To match the release
supply chain, buildx can also emit an SBOM (`--sbom=true`) and cosign can sign
your own image with a key you control.

For development, copy `.env.example` to `.env` at the repository root and fill
in the same runtime values used by the container.

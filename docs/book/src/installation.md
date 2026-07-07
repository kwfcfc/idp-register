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

## Published Images

Multi-arch (amd64/arm64) images are published to the Forgejo registry:

```sh
docker pull forgejo.goba.ip-dynamic.org/gobro/idp-register:<version>
```

Release images (version tags only — not `latest`) are signed with cosign and
carry an SPDX SBOM attestation. The signing public key is committed at the
repository root as `cosign.pub`:

```text
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE0gudYF0EBgUcToTPxQ3z69+X+b2e
6hsdC080EpNEDuOHYazjUsQDWozkw9CrsO08g2z+awRUplZGJNz+SFqTeQ==
-----END PUBLIC KEY-----
```

Verify a release before deploying it:

```sh
cosign verify --key cosign.pub \
  forgejo.goba.ip-dynamic.org/gobro/idp-register:<version>
```

Inspect the attached SBOM:

```sh
docker buildx imagetools inspect \
  forgejo.goba.ip-dynamic.org/gobro/idp-register:<version> \
  --format '{{ json .SBOM }}'
```

The full production Compose path is described in the next chapter. Keep the
browser-facing URL, `ORIGIN`, and OIDC redirect URI in sync.

# syntax=docker/dockerfile:1
# SPDX-License-Identifier: GPL-3.0-or-later

# --- Stage 1: build the SvelteKit SPA (adapter-static) into internal/web/assets ---
FROM node:24-alpine AS web
WORKDIR /src
RUN corepack enable
COPY . .
# adapter-static writes the build straight into ../internal/web/assets so the Go
# stage can embed it (see web/svelte.config.js).
# --frozen-lockfile: a release image must build from the committed lockfile,
# never resolve dependencies on the fly.
RUN pnpm install --filter idp-register-web... --frozen-lockfile \
 && pnpm --filter idp-register-web build

# --- Stage 2: build the static Go binary, embedding the SPA ---
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY --from=web /src ./
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/idp-register ./cmd/server \
 && mkdir /out/data

# --- Stage 3: minimal runtime (~static, single binary) ---
FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION=dev
LABEL org.opencontainers.image.title="idp-register" \
      org.opencontainers.image.description="Self-hosted user-registration broker for an OIDC IdP (Synapse-style invite codes)" \
      org.opencontainers.image.source="https://forgejo.goba.ip-dynamic.org/gobro/idp-register" \
      org.opencontainers.image.licenses="GPL-3.0-or-later" \
      org.opencontainers.image.version="${VERSION}"

COPY --from=build /out/idp-register /idp-register
# Ship /data owned by the runtime user (distroless nonroot = 65532): a named
# volume mounted here inherits this ownership on first use. Without it the
# mountpoint is created root-owned and SQLite cannot create its database file.
COPY --from=build --chown=65532:65532 /out/data /data
EXPOSE 8080
ENTRYPOINT ["/idp-register"]

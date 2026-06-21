# syntax=docker/dockerfile:1
# SPDX-License-Identifier: GPL-3.0-or-later

# --- Stage 1: build the SvelteKit SPA (adapter-static) into internal/web/assets ---
FROM node:24-alpine AS web
WORKDIR /src
RUN corepack enable
COPY . .
# adapter-static writes the build straight into ../internal/web/assets so the Go
# stage can embed it (see web/svelte.config.js).
RUN pnpm install --filter idp-register-web... --no-frozen-lockfile \
 && pnpm --filter idp-register-web build

# --- Stage 2: build the static Go binary, embedding the SPA ---
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY --from=web /src ./
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/idp-register ./cmd/server

# --- Stage 3: minimal runtime (~static, single binary) ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/idp-register /idp-register
EXPOSE 8080
ENTRYPOINT ["/idp-register"]

// SPDX-License-Identifier: GPL-3.0-or-later
// Shared constants for the Crow CI workflows (ADR-0010).
{
  // Everything runs on the arm64 agent: it is the strong box (tier:medium),
  // and the Dockerfile cross-compiles (BUILDPLATFORM stages), so multi-arch
  // images need no QEMU and the weak amd64 agent stays out of the hot path.
  // This is a plain platform label — adding another arm64 agent (e.g. a
  // temporary laptop runner) picks up work without touching these files.
  labels: { platform: 'linux/arm64' },

  // Keep in lock-step with the Dockerfile stage images.
  goImage: 'golang:1.26-alpine',
  nodeImage: 'node:24-alpine',
  postgresImage: 'postgres:16-alpine',

  // e2e.jsonnet dependencies: the IdP under test plus its mail sink, and a
  // small curl image for the readiness gates. Rauthy/mailcrab pins mirror
  // deploy/dev and deploy/e2e.
  rauthyImage: 'ghcr.io/sebadob/rauthy:0.35.2',
  mailcrabImage: 'marlonb/mailcrab:latest',
  curlImage: 'curlimages/curl:8.11.1',

  registry: 'forgejo.goba.ip-dynamic.org',
  imageRepo: 'forgejo.goba.ip-dynamic.org/gobro/idp-register',

  // Official Crow buildx plugin; the server must list it in
  // CROW_PLUGINS_PRIVILEGED (it starts a Docker daemon).
  buildxPlugin: 'codefloe.com/crow-plugins/docker-buildx',
}

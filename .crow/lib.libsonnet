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
  docsImage: 'alpine:3.24',
  postgresImage: 'postgres:16-alpine',
  // Keep in lock-step with pkgs.mdbook in the flake devshell (nixpkgs 26.05),
  // so a book that builds locally builds identically in CI.
  mdbookVersion: '0.5.2',
  gitPushPlugin: 'appleboy/drone-git-push:1.2.5',

  // e2e.jsonnet dependencies: the IdP under test plus its mail sink, and a
  // small curl image for the readiness gates. Rauthy/mailcrab pins mirror
  // deploy/dev and deploy/e2e.
  rauthyImage: 'ghcr.io/sebadob/rauthy:0.35.2',
  mailcrabImage: 'marlonb/mailcrab:v1.10.0',
  curlImage: 'curlimages/curl:8.23.0',

  registry: 'forgejo.goba.ip-dynamic.org',
  imageRepo: 'forgejo.goba.ip-dynamic.org/gobro/idp-register',

  // Official Crow buildx plugin; the server must list it in
  // CROW_PLUGINS_PRIVILEGED (it starts a Docker daemon). Pinned because it
  // runs privileged with the registry and cosign secrets. A tagless allowlist
  // entry still matches: the default `semver` match type ignores the tag
  // when the entry has none.
  buildxPlugin: 'codefloe.com/crow-plugins/docker-buildx:2.7.0',

  // Build cache (Crow >= 6.11, https://docs.crowci.com/v7-0/usage/caching/).
  // Use as `cache: [lib.goCache, ...]` on the workflow plus the matching
  // `*CacheEnv` on the steps.
  //  - Only the workspace survives between steps and runs, so the tool caches
  //    are pointed into .cache/ there. `go ... ./...` skips dot-dirs; .cache/
  //    is also git- and docker-ignored.
  //  - Keys are immutable: a new cache is saved only when a checksummed file
  //    changes. Other runs restore the newest cache with the same prefix,
  //    looking in their own ref, then the PR target, then the default branch.
  //    Bump the vN segment to throw a cache away.
  //  - The server needs CROW_CACHE_BACKEND; without it the section is ignored
  //    (server-side lint warning only). `crow exec` always ignores it.
  goCache: {
    key: 'go-{{ arch }}-v1-{{ checksum "go.sum" }}',
    restore_keys: ['go-{{ arch }}-v1-'],
    paths: ['.cache/go-mod', '.cache/go-build'],
  },
  goCacheEnv: {
    GOMODCACHE: '${CI_WORKSPACE}/.cache/go-mod',
    GOCACHE: '${CI_WORKSPACE}/.cache/go-build',
  },
  // The root package.json is in the key because its packageManager pin
  // decides which pnpm corepack downloads into COREPACK_HOME. Deliberately no
  // restore_keys: a store restored from an older lockfile would carry its
  // stale packages into every later save, and `pnpm store prune` can't trim
  // it safely (it judges by hard-link count, so it empties a store whose
  // packages were cloned/reflinked instead of hard-linked). A lockfile change
  // costs one cold install instead.
  pnpmCache: {
    key: 'pnpm-{{ arch }}-v1-{{ checksum "pnpm-lock.yaml" "package.json" }}',
    paths: ['.cache/pnpm-store', '.cache/corepack'],
  },
  pnpmCacheEnv: {
    // pnpm 11 reads pnpm_config_* (not npm_config_*) settings from the env.
    pnpm_config_store_dir: '${CI_WORKSPACE}/.cache/pnpm-store',
    COREPACK_HOME: '${CI_WORKSPACE}/.cache/corepack',
  },
}

// SPDX-License-Identifier: GPL-3.0-or-later
// Multi-arch image build + push to the Forgejo registry. Runs after `test`,
// only for pushes to main and for tags.
//
// Operator prerequisites (one-time, on the Crow server / repo):
//   - CROW_PLUGINS_PRIVILEGED must include codefloe.com/crow-plugins/docker-buildx
//     (the plugin starts its own Docker daemon).
//   - Repo secrets `registry_username` / `registry_password`: a Forgejo
//     account/token with package:write on gobro/idp-register.
//   - Repo secrets `cosign_private_key` / `cosign_password`: the same key pair
//     used by gobro/simple-git-server, so one key verifies all our images
//     (cosign verify --key cosign.pub <image>).
local lib = import 'lib.libsonnet';

// Release-only extras: an SPDX SBOM attestation (merged into the attestation
// manifests buildx already emits for provenance, so no new registry entries)
// and the plugin's built-in cosign signing. Cosign signs every manifest in the
// index and pushes the signatures as sha256-* tags — those show up as extra
// package versions in the Forgejo UI, sorted above the release tag; known
// cosmetic quirk, accepted. Tag builds only: `latest` from main is a moving
// target and stays unsigned, matching simple-git-server.
local releaseSettings = {
  sbom: 'true',
  cosign: true,
  'cosign-key': { from_secret: 'cosign_private_key' },
  'cosign-password': { from_secret: 'cosign_password' },
};

// One publish step per trigger, differing only in the VERSION build arg
// (baked into the org.opencontainers.image.version label): tags use the tag
// name, main pushes use the commit SHA. auto_tag then derives the image tags
// (semver set for tags, `latest` for the default branch).
local publish(name, version, when) = {
  name: name,
  image: lib.buildxPlugin,
  settings: {
    repo: lib.imageRepo,
    registry: lib.registry,
    dockerfile: 'Dockerfile',
    platforms: 'linux/amd64,linux/arm64',
    auto_tag: true,
    build_args: { VERSION: version },
    username: { from_secret: 'registry_username' },
    password: { from_secret: 'registry_password' },
  },
  when: [when],
};

{
  depends_on: ['test'],
  labels: lib.labels,
  when: [
    { event: 'push', branch: 'main' },
    { event: 'tag' },
  ],

  steps: [
    publish('publish-main', '${CI_COMMIT_SHA}', { event: 'push', branch: 'main' }),
    publish('publish-tag', '${CI_COMMIT_TAG}', { event: 'tag' }) + {
      settings+: releaseSettings,
    },
  ],
}

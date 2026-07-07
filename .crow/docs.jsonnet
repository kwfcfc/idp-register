// SPDX-License-Identifier: GPL-3.0-or-later
// Build the mdBook documentation with Crow CI and publish the generated static
// files for GitHub Pages by preparing a gh-pages worktree and pushing it with
// the appleboy/drone-git-push plugin. The Forgejo push mirror then syncs stable
// + gh-pages to GitHub.
//
// Operator prerequisites:
//   - GitHub Pages source: Deploy from a branch, gh-pages / root.
//   - Forgejo push mirror includes stable and gh-pages.
//   - Crow repo secrets `registry_username` / `forgejo_token`: a Forgejo
//     account/token allowed to push the gh-pages branch.
//   - Versioned documentation publishes only for release-line tags in the form
//     `vMAJOR.MINOR` (for example `v1.0`). Extra image tags like `v1` and
//     `v1.0.0` do not publish duplicate docs.
local lib = import 'lib.libsonnet';

local installTools = [
  'apk add --no-cache ca-certificates curl git tar gzip >/dev/null',
  'case "$(uname -m)" in aarch64|arm64) mdbook_target=aarch64-unknown-linux-musl;; x86_64|amd64) mdbook_target=x86_64-unknown-linux-musl;; *) echo "unsupported mdBook CI architecture: $(uname -m)" >&2; exit 1;; esac; curl -fsSL "https://github.com/rust-lang/mdBook/releases/download/v' + lib.mdbookVersion + '/mdbook-v' + lib.mdbookVersion + '-${mdbook_target}.tar.gz" | tar -xz -C /usr/local/bin mdbook',
  'mdbook --version',
];

local publishEnv = {
  PAGES_BRANCH: 'gh-pages',
};

local pushSettings(message) = {
  path: 'docs-build/pages',
  remote_name: 'origin',
  branch: 'gh-pages',
  local_branch: 'gh-pages',
  commit: true,
  commit_message: message,
  author_name: 'idp-register docs bot',
  author_email: 'docs-bot@goba.ip-dynamic.org',
  username: { from_secret: 'registry_username' },
  password: { from_secret: 'forgejo_token' },
};

{
  depends_on: ['test'],
  labels: lib.labels,
  when: [
    { event: 'push', branch: 'stable' },
    { event: 'tag' },
  ],

  steps: [
    {
      name: 'prepare-stable-docs',
      image: lib.docsImage,
      environment: publishEnv {
        MDBOOK_OUTPUT__HTML__SITE_URL: '/idp-register/stable/',
      },
      commands: installTools + [
        'rm -rf docs-build/stable',
        'mdbook build docs/book -d ../../docs-build/stable',
        'sh tools/docs/prepare-github-pages.sh docs-build/stable stable docs-build/pages',
      ],
      depends_on: [],
      when: [{ event: 'push', branch: 'stable' }],
    },
    {
      name: 'push-stable-docs',
      image: lib.gitPushPlugin,
      settings: pushSettings('[skip ci] docs: publish stable documentation'),
      depends_on: ['prepare-stable-docs'],
      when: [{ event: 'push', branch: 'stable' }],
    },
    {
      name: 'prepare-tag-docs',
      image: lib.docsImage,
      environment: publishEnv {
        MDBOOK_OUTPUT__HTML__SITE_URL: '/idp-register/${CI_COMMIT_TAG}/',
      },
      commands: installTools + [
        'printf "%s\\n" "${CI_COMMIT_TAG}" | grep -Eq "^v[0-9]+\\.[0-9]+$" || { echo "refusing non-docs-release tag: ${CI_COMMIT_TAG}" >&2; exit 1; }',
        'rm -rf "docs-build/${CI_COMMIT_TAG}"',
        'mdbook build docs/book -d "../../docs-build/${CI_COMMIT_TAG}"',
        'sh tools/docs/prepare-github-pages.sh "docs-build/${CI_COMMIT_TAG}" "${CI_COMMIT_TAG}" docs-build/pages',
      ],
      depends_on: [],
      when: [{ event: 'tag', ref: 'refs/tags/v*' }],
    },
    {
      name: 'push-tag-docs',
      image: lib.gitPushPlugin,
      settings: pushSettings('[skip ci] docs: publish ${CI_COMMIT_TAG} documentation'),
      depends_on: ['prepare-tag-docs'],
      when: [{ event: 'tag', ref: 'refs/tags/v*' }],
    },
  ],
}

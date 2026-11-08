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
//   - Versioned documentation publishes for final release tags
//     `vMAJOR.MINOR.PATCH` (for example `v0.3.0`) into the release-line
//     directory `/vMAJOR.MINOR/`, so each patch release refreshes its line.
//     Pre-releases (`v0.3.0-rc.1`) and other tag shapes skip this workflow via
//     the evaluate filter below. (Release-line tags such as `v0.3` are not used
//     for docs: every tag also triggers image.jsonnet, so an extra docs-only tag
//     would rebuild, re-push and re-sign the image.)
local lib = import 'lib.libsonnet';

// [.] instead of \. — the evaluate string is parsed by expr-lang, which
// rejects \. inside double-quoted strings as an invalid char escape.
local docsReleaseTag = 'CI_COMMIT_TAG matches "^v[0-9]+[.][0-9]+[.][0-9]+$"';

local installTools = [
  'apk add --no-cache ca-certificates curl git tar gzip >/dev/null',
  // $$ escapes the shell variable so Crow's ${...} substitution pass leaves it
  // for the shell (a bare ${mdbook_target} would be replaced with empty at
  // config time, since no CI variable by that name exists). $(uname -m) is
  // command substitution, not ${...}, so Crow leaves it alone.
  'case "$(uname -m)" in aarch64|arm64) mdbook_target=aarch64-unknown-linux-musl;; x86_64|amd64) mdbook_target=x86_64-unknown-linux-musl;; *) echo "unsupported mdBook CI architecture: $(uname -m)" >&2; exit 1;; esac; curl -fsSL "https://github.com/rust-lang/mdBook/releases/download/v' + lib.mdbookVersion + '/mdbook-v' + lib.mdbookVersion + '-$${mdbook_target}.tar.gz" | tar -xz -C /usr/local/bin mdbook',
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
  // The push authenticates with gobro's own Forgejo token
  // (registry_username / forgejo_token), so attribute the commit to the same
  // identity instead of a separate bot persona.
  author_name: 'gobro',
  author_email: 'gobro@noreply.localhost',
  // netrc_machine is required: without it the plugin writes no ~/.netrc at all,
  // and git falls back to prompting for a username ("could not read Username
  // for 'https://...'") and fails non-interactively. It must be the git host,
  // which for this Forgejo instance is the same as the container registry host.
  netrc_machine: lib.registry,
  username: { from_secret: 'registry_username' },
  password: { from_secret: 'forgejo_token' },
};

{
  depends_on: ['test'],
  labels: lib.labels,
  when: [
    { event: 'push', branch: 'stable' },
    { event: 'tag', evaluate: docsReleaseTag },
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
        // mdBook resolves -d relative to the current working directory (the
        // repo checkout root), not the book root — so this is docs-build/...,
        // not ../../docs-build/.... The prepare script reads the same path.
        'mdbook build docs/book -d docs-build/stable',
        'sh tools/docs/prepare-github-pages.sh docs-build/stable stable docs-build/pages',
      ],
      depends_on: [],
      when: [{ event: 'push', branch: 'stable' }],
    },
    {
      name: 'push-stable-docs',
      image: lib.gitPushPlugin,
      settings: pushSettings('[skip ci] docs: publish stable documentation\n\nSource-Ref: stable\nSource-Commit: ${CI_COMMIT_SHA}'),
      depends_on: ['prepare-stable-docs'],
      when: [{ event: 'push', branch: 'stable' }],
    },
    {
      name: 'prepare-tag-docs',
      image: lib.docsImage,
      environment: publishEnv,
      commands: installTools + [
        // v0.3.0 -> v0.3 (the tag already passed the vX.Y.Z filter). One shell
        // session per step, so `line` and the export carry to later commands;
        // $$ keeps Crow's config-time substitution off the shell variables.
        'line=$(printf %s "${CI_COMMIT_TAG}" | cut -d. -f1-2)',
        'export MDBOOK_OUTPUT__HTML__SITE_URL="/idp-register/$${line}/"',
        'rm -rf "docs-build/$${line}"',
        'mdbook build docs/book -d "docs-build/$${line}"',
        'sh tools/docs/prepare-github-pages.sh "docs-build/$${line}" "$${line}" docs-build/pages',
      ],
      depends_on: [],
      when: [{ event: 'tag', evaluate: docsReleaseTag }],
    },
    {
      name: 'push-tag-docs',
      image: lib.gitPushPlugin,
      settings: pushSettings('[skip ci] docs: publish ${CI_COMMIT_TAG} documentation\n\nSource-Ref: ${CI_COMMIT_TAG}\nSource-Commit: ${CI_COMMIT_SHA}'),
      depends_on: ['prepare-tag-docs'],
      when: [{ event: 'tag', evaluate: docsReleaseTag }],
    },
  ],
}

<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Branching and GitHub Mirror Policy

This document defines the branching, release, and mirror policy for `idp-register`.
It applies to both human developers and AI agents working in this repository.

## 1. Core Goals

This project uses the following model:

```text
Forgejo main
  Active development trunk, updated frequently
        |
        | selective promotion, testing, backports
        v
Forgejo stable
  Stable code line for external users
        |
        | low-frequency or manually triggered mirror
        v
GitHub stable
  Stable downstream mirror and public issue entry point
```

Basic principles:

- Forgejo is the only authoritative development repository.
- `main` remains the day-to-day development and integration trunk.
- `stable` is the stable branch promoted selectively from `main`.
- GitHub is not a second primary development repository; it only mirrors `stable`.
- GitHub may intentionally lag behind Forgejo.
- GitHub Issues primarily track problems found by users of the GitHub stable mirror.
- Forgejo Releases, tags, and release notes remain the authoritative release information.

## 2. Repository Roles

| Location | Role | Direct development allowed |
|---|---|---:|
| Forgejo `main` | Authoritative development trunk | Yes |
| Forgejo `stable` | Stable maintenance line and backport target | Only for promotions, backports, or emergency fixes |
| GitHub `stable` | Read-only-style public stable mirror | No |
| GitHub Issues | Public issue entry point for users without Forgejo accounts | Applicable |
| GitHub Pull Requests | Patch or suggestion entry point | Do not merge directly on GitHub |

GitHub repository:

```text
https://github.com/kwfcfc/idp-register
```

Canonical Forgejo repository:

```text
https://forgejo.goba.ip-dynamic.org/gobro/idp-register
```

The Forgejo repository URL should be set in the GitHub repository's About / Website
field and clearly noted in the README.

## 3. Branch Meanings

### `main`

`main` is the active development trunk on Forgejo.

Content allowed into `main` includes:

- New features.
- Refactors.
- Compatibility changes that have not yet been promoted to stable users.
- Fixes that have not yet been backported.
- Documentation and configuration changes preparing future versions.

Constraints for `main`:

- All day-to-day development starts from `main` by default.
- `main` is not mirrored directly to GitHub.
- `main` may be ahead of `stable`.
- A change landing on `main` does not mean that it is available to GitHub stable users.
- Do not bypass testing and modify `stable` directly just to update GitHub faster.

### `stable`

`stable` is the stable code line after explicit selection, testing, and approval.

Changes allowed into `stable` include:

- Bug fixes already verified on `main`.
- Stable features that were explicitly approved for promotion.
- Security fixes.
- Maintenance updates with acceptable compatibility risk.
- Documentation fixes aimed at stable users.
- Commits selectively backported from `main`.

Constraints for `stable`:

- Do not do ordinary feature development on `stable`.
- Do not merge an unverified `main` wholesale into `stable`.
- Promote individual commits by default through cherry-pick or a dedicated backport PR.
- A full merge or fast-forward is allowed only when every difference from `main` is
  confirmed suitable for stable users.
- Do not rebase or force-push published `stable` history.
- Emergency fixes may land on `stable` first, but must then be forward-ported back to
  `main`.
- `stable` may temporarily be ahead of GitHub because GitHub uses a low-frequency mirror.
- Except during the short window for emergency fixes, `stable` should not contain a
  long-lived divergence that never exists on `main`.

## 4. Development Flow

### 4.1 Ordinary Development

Ordinary development should start from Forgejo `main`:

```bash
git switch main
git pull --ff-only forgejo main
git switch -c feature/example
```

After completion, merge through a Forgejo Pull Request into `main`.

Do not use GitHub `stable` as the development baseline unless the task explicitly
requires reproducing a stable-version issue or preparing a backport.

### 4.2 Selective Backport to `stable`

When a `main` commit is suitable for stable users:

```bash
git switch stable
git pull --ff-only forgejo stable
git cherry-pick -x -S <main-commit>
git push forgejo stable
```

Notes:

- `-x` records the original commit in the commit message so the backport can be traced.
- `-S` signs the newly generated backport commit with the current maintainer's signing key.
- Cherry-picking creates a new commit ID.
- The original commit's cryptographic signature is not automatically preserved on the new
  commit. The author is usually preserved; the committer and signer are the maintainer
  performing the backport.
- After the backport, run the tests relevant to the change.

Prefer using a Forgejo PR from a backport branch into `stable` instead of pushing
directly to a protected `stable` branch.

### 4.3 Full Promotion

A full promotion is allowed only when all changes from `main` relative to `stable` have
passed stable review:

```bash
git switch stable
git pull --ff-only forgejo stable
git merge --ff-only main
git push forgejo stable
```

If this cannot fast-forward, do not rewrite history for convenience. Inspect the reason
for the divergence and use an explicit merge PR or selective backports.

### 4.4 Emergency Fixes on `stable`

An emergency fix may land on `stable` first:

```bash
git switch stable
git pull --ff-only forgejo stable

# edit and test

git commit -S -m "fix: ..."
git push forgejo stable
```

The fix must then be brought back to `main`:

```bash
git switch main
git pull --ff-only forgejo main
git cherry-pick -x -S <stable-hotfix-commit>
git push forgejo main
```

Do not let an emergency fix live permanently only on `stable`.

## 5. GitHub Mirror Policy

GitHub receives only Forgejo's `stable` branch.

### Forgejo Push Mirror Configuration

```text
Remote URL:
https://github.com/kwfcfc/idp-register.git

Authentication:
HTTPS + GitHub Fine-grained Personal Access Token

Username:
kwfcfc

Password:
GitHub Fine-grained Personal Access Token

Branch filter:
stable

Sync when new commits are pushed:
Disabled

Mirror interval:
168h

Signature trust model:
Use the Forgejo instance default
```

`168h` means the automatic mirror runs at most once every seven days. After an important
stable version is ready, maintainers may manually click "Sync Now" in Forgejo.

These two steps are independent:

1. Promote changes to Forgejo `stable`.
2. Mirror Forgejo `stable` to GitHub.

Landing on `stable` does not mean GitHub has been updated. GitHub may intentionally lag
behind Forgejo `stable`.

### Minimum GitHub Token Permissions

The fine-grained token should be limited to:

```text
Resource owner:
kwfcfc

Repository access:
Only select repositories
  - idp-register

Repository permissions:
Contents: Read and write
```

Grant the following only if the repository must mirror `.github/workflows/*`:

```text
Workflows: Read and write
```

Do not write the token into:

- Git URLs.
- Repository files.
- `.env.example`.
- CI logs.
- Commit messages.
- Issues or Pull Requests.

The token should only be stored in Forgejo's mirror credential storage.

## 6. GitHub Default Branch and Existing `main`

GitHub's default branch should be:

```text
stable
```

After migration, the old GitHub `main` no longer represents the current development trunk.

The old GitHub `main` may be deleted after confirming that:

- `stable` has mirrored successfully.
- GitHub's default branch has been changed to `stable`.
- No open Pull Request still uses GitHub `main` as its base.
- README, issue templates, and automation do not hard-code GitHub `main`.
- No external documentation link requires keeping GitHub `main`.

Forgejo `main` is unaffected and remains the authoritative development trunk.

Do not recreate or maintain an independent GitHub `main`. That would make users think
GitHub also hosts the development trunk.

## 7. First Migration Steps

The examples below assume these local remote names:

```text
forgejo
github
```

If the repository actually uses names such as `origin`, replace them accordingly.

### 7.1 Create `stable` from the Current Forgejo `main`

Choose the current commit that is suitable as the stable baseline. If the current `main`
is suitable as a whole:

```bash
git switch main
git pull --ff-only forgejo main
git switch -c stable
git push -u forgejo stable
```

If the stable baseline should stay at an earlier commit:

```bash
git switch -c stable <approved-commit>
git push -u forgejo stable
```

### 7.2 Configure the Forgejo Push Mirror

In the Forgejo repository settings:

1. Add an HTTPS push mirror pointing to GitHub.
2. Use a fine-grained PAT.
3. Set the branch filter to `stable`.
4. Disable immediate sync on push.
5. Set the mirror interval to `168h`.
6. Run one manual "Sync Now".

### 7.3 Adjust GitHub

On GitHub:

1. Confirm that `stable` exists.
2. Change the default branch to `stable`.
3. Update About / Website to point to the canonical Forgejo repository.
4. Update the repository description.
5. Check issue and PR templates for branch names.
6. Delete the old GitHub `main` after confirming there are no dependencies on it.

## 8. GitHub Issue Semantics

GitHub Issues are for users of the GitHub stable mirror.

The default interpretation is:

> The reporter is using GitHub `stable`, or a stable release corresponding to it, not
> Forgejo `main`.

Issue templates should require:

```text
Version or commit:
Installation method:
Expected behavior:
Actual behavior:
Relevant logs:
```

Recommended status labels:

```text
needs-triage
confirmed
fixed-on-main
backport-pending
available-on-stable
mirrored-to-github
```

Recommended lifecycle:

```text
1. A user reports a problem on GitHub for stable.
2. A maintainer confirms it and fixes it on Forgejo main.
3. The GitHub Issue is labeled fixed-on-main.
4. The maintainer evaluates whether to backport it.
5. The fix lands on Forgejo stable.
6. The Issue is labeled available-on-stable.
7. stable is mirrored to GitHub.
8. The Issue is labeled mirrored-to-github and closed.
```

Do not tell GitHub stable users that a fix is "available" merely because it exists on
Forgejo `main`.

A fix should be considered available to GitHub users only after it has entered `stable`
and the GitHub mirror has completed.

If a fix will not be backported, state that clearly:

- The fix will only appear in a future stable promotion.
- The current stable line will not include the fix.
- The user must test Forgejo `main` themselves.

## 9. Handling GitHub Pull Requests

GitHub Pull Requests can be accepted as an external contribution entry point, but GitHub
is not the final merge location.

Maintainers should:

1. Review the GitHub PR.
2. Fetch the patch locally or on Forgejo.
3. Merge the change into Forgejo `main`.
4. Decide whether it should be backported to `stable`.
5. After GitHub `stable` updates, close the GitHub PR or explain the outcome.

Do not click Merge on GitHub and merge the change into GitHub `stable`. The next Forgejo
mirror may overwrite that change and it would violate the rule that Forgejo is the only
authoritative repository.

## 10. Commit Identity, Signatures, and Remote Authentication

Commit signatures and push authentication are separate mechanisms.

This project follows these rules:

- One commit uses one project development identity and one cryptographic signature.
- The same commit keeps the same commit ID on Forgejo and GitHub.
- Forgejo login credentials are used to push to Forgejo.
- The GitHub fine-grained PAT is used only for Forgejo-to-GitHub mirroring.
- The GitHub token does not sign commits.
- Do not re-sign the same logical commit differently for different remotes.
- Re-signing or changing author/committer identity creates a new commit ID.

The signing public key can be registered on both Forgejo and GitHub so both platforms can
verify the same commit signature.

Forgejo's "signature trust model" only affects the trust status shown in the web UI. It
does not control GitHub mirror authentication and does not re-sign commits.

## 11. Releases and Tags

Forgejo is the authoritative release location:

- Official tags are created on Forgejo.
- Release notes are published on Forgejo.
- Binary attachments are published on Forgejo.
- GitHub Releases are optional downstream copies, not the authoritative source.

The current mirror policy is centered on the `stable` branch. Do not assume that all tags,
release descriptions, and binary attachments will automatically sync because the branch is
mirrored.

When a tag or Release must also be published on GitHub, use a separate and explicit
release step and state that Forgejo is the canonical source.

## 12. GitHub Repository Description

Suggested GitHub About fields:

```text
Description:
Stable downstream mirror of the canonical Forgejo repository. GitHub Issues are accepted for the mirrored stable version.

Website:
https://forgejo.goba.ip-dynamic.org/gobro/idp-register
```

Suggested neutral README note, suitable for both Forgejo and GitHub:

```markdown
> **Repository and release policy**
>
> Active development takes place in the canonical Forgejo repository:
> [Forgejo repository](https://forgejo.goba.ip-dynamic.org/gobro/idp-register)
>
> The GitHub repository mirrors the `stable` branch and may intentionally
> lag behind active development. GitHub Issues are accepted for users of
> the mirrored stable version.
```

## 13. Rules for AI Agents

AI agents working in this repository must check the current branch and task goal before
modifying code.

### Default Behavior

When the task does not explicitly specify a branch:

```text
target branch = Forgejo main
```

AI agents should:

- Create work branches from `main`.
- Put ordinary fixes and features into PRs targeting `main`.
- Not treat "GitHub is behind" as a reason to modify `stable` directly.
- Not directly modify mirror settings, GitHub tokens, or repository visibility.
- Not trigger stable promotion on their own.
- Not close GitHub Issues on their own.

### `stable` Requires Explicit Instruction

The following wording counts as explicit authorization:

```text
backport to stable
move this fix back to stable
prepare a stable update
promote these commits to stable
make a stable hotfix
```

Even with explicit authorization, AI agents must:

1. List the commits proposed for backport.
2. Check dependency commits.
3. Check for incompatible changes.
4. Use `cherry-pick -x` or an explicit backport PR.
5. Run tests.
6. Report the new commit ID.
7. Never force-push `stable`.

### Forbidden Actions

AI agents must not:

- Treat GitHub as the authoritative development repository.
- Commit directly to GitHub `stable`.
- Merge Pull Requests on GitHub.
- Reverse-overwrite Forgejo `main` from GitHub `stable`.
- Merge all of `main` into `stable` unless the task explicitly asks for it and all
  differences have been checked.
- Force-push `main` or `stable`.
- Rewrite published stable history.
- Print a GitHub PAT in logs or replies.
- Claim that GitHub users have received a fix just because it exists on `main`.
- Automatically create a GitHub Release and claim that it is authoritative.
- Modify this policy file to bypass these constraints unless the task explicitly asks to
  update the branching policy.

## 14. Maintainer Checklists

### Before Merging to `main`

- [ ] The change is suitable for the development trunk.
- [ ] Tests pass.
- [ ] The change does not require GitHub to update immediately.
- [ ] Issue status was not incorrectly marked as "available on stable".

### Before Promoting to `stable`

- [ ] The commits to promote are explicit.
- [ ] Dependency commits were checked.
- [ ] Compatibility and migration requirements were checked.
- [ ] Tests ran on the stable baseline.
- [ ] The backport commit can be traced to the original commit.
- [ ] `stable` history was not rewritten.

### Before Mirroring to GitHub

- [ ] Forgejo `stable` is healthy.
- [ ] The commit to mirror was checked.
- [ ] No private configuration, token, or secret is included.
- [ ] GitHub Issues that are about to be closed are actually included in `stable`.
- [ ] Trigger Forgejo "Sync Now" manually if needed.

### After Mirroring to GitHub

- [ ] GitHub `stable` points to the expected commit.
- [ ] GitHub's default branch is still `stable`.
- [ ] Release or README wording does not imply GitHub is authoritative.
- [ ] Issue labels were updated consistently.

## 15. Summary

```text
Forgejo main:
  fast, authoritative development trunk

Forgejo stable:
  selectively promoted stable line

GitHub stable:
  downstream mirror of Forgejo stable

GitHub Issues:
  primarily track issues for stable users

Default development base:
  main

Stable changes:
  explicit backport or promotion only

GitHub sync:
  delayed, optional, non-authoritative

Canonical releases:
  Forgejo
```

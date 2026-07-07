<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Introduction

`idp-register` is a small, self-hosted registration broker for an OIDC identity
provider. It gives an OIDC-based stack the same basic registration-token workflow
that Synapse operators are used to: people apply through a public form, a valid
invite code can auto-approve the application, and otherwise an admin reviews it.

On approval, `idp-register` provisions the user into the target identity provider
and lets that IdP send the activation or first-password email. The broker does
not send a second verification email on the happy path.

The near-term production shape is one OCI image containing:

- a Go HTTP server;
- an embedded static SvelteKit frontend;
- either SQLite or PostgreSQL storage;
- generic OIDC admin login;
- a target-IdP provisioner, with Rauthy implemented first.

Forgejo is the canonical development repository. GitHub is used as a stable
mirror and as the public GitHub Pages host for this generated documentation.

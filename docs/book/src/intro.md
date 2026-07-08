<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Introduction

`idp-register` is a small, self-hosted registration broker for an OIDC identity
provider. It gives an OIDC-based stack the registration-token workflow that
Synapse operators are used to:

- people apply through a public form;
- a valid invite code can auto-approve the application;
- otherwise an administrator reviews the request;
- approved users are created in the target identity provider.

On approval, `idp-register` provisions the user into the target identity provider
and lets that IdP send the activation or first-password email. The broker does
not send a second verification email on the happy path.

The normal deployment is one OCI image containing:

- a Go HTTP server;
- an embedded static SvelteKit frontend;
- either SQLite or PostgreSQL storage;
- generic OIDC admin login;
- a target-IdP provisioner, with Rauthy implemented first.

## Who This Manual Is For

This book is for operators deploying and maintaining `idp-register`. It focuses
on Compose deployment, environment variables, OIDC setup, Rauthy provisioning,
database choices, and operational checks.

Development internals live in the source repository under `docs/`.

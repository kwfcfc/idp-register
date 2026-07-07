<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Decision Records

Architecture decisions are tracked in `docs/DECISIONS.md` in the source
repository.

Key accepted decisions include:

- Go backend instead of full-stack SvelteKit;
- SvelteKit static frontend embedded in the Go binary;
- `database/sql` with portable SQLite/PostgreSQL schema;
- plaintext Synapse-style invite codes;
- a `Provisioner` interface for target-IdP user creation;
- generic OIDC relying-party admin auth;
- one user-facing email on the happy path;
- permission profiles instead of public group selection;
- Crow CI instead of Forgejo Actions;
- mdBook documentation published from CI.

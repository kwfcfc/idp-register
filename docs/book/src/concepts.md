<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Concepts

## Application

An application is a public registration submission. It starts as pending unless
a valid invite code can reserve a use and auto-approve it.

## Invite Code

Invite codes are plaintext, limited-use, time-limited strings. They follow the
Synapse-style counters:

- `uses_allowed`: maximum total successful uses;
- `pending`: reserved uses whose provisioning has not completed yet;
- `completed`: successfully provisioned uses;
- `expiry_time`: optional expiry timestamp.

Codes must never be logged. A valid invite code is bound to one permission
profile and always provisions into that profile.

## Permission Profile

A permission profile is a server-side bundle of target-IdP groups. The public
form can request a public profile as a service choice, but it never submits raw
group names. Final group assignment is resolved server-side during approval.

## Two IdP Roles

`idp-register` deliberately separates two identity-provider roles:

- the admin-auth IdP, where administrators log in through standards-only OIDC;
- the provisioning target IdP, where approved end users are created through a
  provider-specific implementation behind the `Provisioner` interface.

Rauthy-specific API calls belong only in the Rauthy provisioner. Admin login must
remain generic OIDC.

<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Concepts

## Registration Flow

The public form collects an application. If the applicant supplies a valid invite
code, the service reserves one use of that code and can approve the application
automatically. Without an invite code, the application waits for an admin.

When an application is approved, `idp-register` creates the user in the
provisioning target IdP and assigns the groups from the selected permission
profile. The target IdP then sends the activation or first-password email.

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

Administrators create profiles from the live target-IdP group catalog. Groups in
`PROFILE_GROUP_DENYLIST`, plus the admin group used by this application, are not
offered for registrant profiles.

## Two IdP Roles

`idp-register` deliberately separates two identity-provider roles:

- the admin-auth IdP, where administrators log in through standards-only OIDC;
- the provisioning target IdP, where approved end users are created through a
  provider-specific implementation behind the `Provisioner` interface.

Rauthy-specific API calls belong only in the Rauthy provisioner. Admin login must
remain generic OIDC.

Even when both roles point to the same Rauthy instance, configure them as two
separate integrations: one OIDC client for admin login, and one Rauthy API key
for user creation.

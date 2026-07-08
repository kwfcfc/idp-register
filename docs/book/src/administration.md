<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Administration

This chapter covers day-to-day use of the admin console after the service is
deployed: signing in, creating permission profiles, issuing invite codes,
reviewing applications, and reading the audit log.

## First Sign-In

There is no separate bootstrap password. The first administrator is simply the
first person who logs in through the admin-auth IdP and matches the
authorization rules:

- their groups or roles claim (`OIDC_GROUPS_CLAIM`) contains `OIDC_ADMIN_GROUP`;
- or their `sub` is listed in `OIDC_ADMIN_SUBS`;
- or their email is listed in `OIDC_ADMIN_EMAILS`.

So before the first deployment, make sure your own account already satisfies one
of these in the admin-auth IdP — most commonly by adding yourself to the admin
group. Then open `/login`, complete the OIDC login, and you land on `/admin`.

If login succeeds but you are denied admin access, see
[Troubleshooting](troubleshooting.md#login-succeeds-but-admin-access-is-denied).

## The Admin Console

The console lives under `/admin` and has four sections:

- **Applications** — the review queue for public registration requests.
- **Permission Profiles** — named bundles of target-IdP groups.
- **Invites** — registration codes that can auto-approve applications.
- **Audit Log** — a read-only record of admin actions.

## Recommended First-Run Order

Create at least one permission profile before anything else. Both issuing an
invite code and approving an application require a profile, so an empty install
cannot process registrations until one exists.

1. Sign in at `/login`.
2. Create a **Permission Profile** from the target-IdP groups.
3. Optionally issue an **Invite** bound to that profile for self-service signup.
4. Review incoming **Applications**.

## Permission Profiles

A permission profile is a server-side bundle of target-IdP groups. The public
form never submits raw group names; it can only request a profile that you have
marked public, and the group assignment is resolved during approval.

On the **Permission Profiles** page you can:

- create a profile from the live target-IdP group catalog;
- mark a profile public and give it a user-facing label so it appears as a
  choice on the public form;
- edit or delete existing profiles.

Groups listed in `PROFILE_GROUP_DENYLIST`, plus the value of `OIDC_ADMIN_GROUP`,
are never offered here — infrastructure and admin groups cannot be assigned to
registrants.

## Invites

Invite codes are the Synapse-style registration tokens. A valid code lets an
applicant be approved automatically and provisioned into the profile the code is
bound to.

On the **Invites** page, create a code with:

- **Profile** — required; the profile the code provisions into.
- **Code** — optional. Leave blank to auto-generate a random code, or supply
  your own.
- **Uses allowed** — optional. Blank means unlimited.
- **Expiry** — optional. Blank means it never expires.
- **Email constraint** — optional. Restricts the code to one email address.
- **Note** — optional free text for your own bookkeeping.

Each code tracks two counters against **uses allowed**: `pending` (reservations
whose provisioning has not finished) and `completed` (successful signups). You
can disable a code at any time with its active toggle, or delete it outright.

> Invite codes are stored and displayed in cleartext by design, so they remain
> visible to admins on this page. Treat them as secrets anyway: never paste a
> code into a support ticket, chat, or log. See
> [Invite Recovery](operations.md#invite-recovery) for handling a code whose
> provisioning failed mid-way.

## Applications

The **Applications** page is the review queue. Each entry opens a detail view
where you can:

- **Approve** — choose the permission profile to grant and optionally add a
  note. Approval creates the user in the target IdP with the profile's groups,
  and the target IdP sends the activation or first-password email. The broker
  does not send a second email on the happy path.
- **Reject** — record a decision with a status and optional note without
  provisioning anyone.

Applications submitted with a valid invite code are auto-approved into the
code's profile and appear here already resolved; codeless applications wait
in the queue for a manual decision.

## Audit Log

The **Audit Log** page is a read-only history of admin actions — profile
changes, invite creation, approvals, and rejections — with the acting admin and
a timestamp. Use it to answer "who approved this account" or "when was this code
created". Invite code values are never written to the audit log, only their
non-secret metadata (profile, expiry, usage limits).

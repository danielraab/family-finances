# account-sharing Specification

## Purpose

Sharing an account with other registered users: the four permission tiers
(`view`, `append`, `entry_admin`, `owner`) and what each grants, the
`account_shares` model, the share-management endpoints
(`/api/accounts/{id}/shares`), the email-invite flow (including the
unregistered-email/app-invite nudge), and revocation/self-leave semantics.
See `accounts` for the account a share applies to and its real owner, and
`account-entries` for how permission tiers gate entry reads/writes.

## Requirements

### Requirement: Four permission tiers, each a strict superset of the one before it

An account MAY be shared with any number of other registered users, each at
exactly one of four permission tiers: `view`, `append`, `entry_admin`, or
`owner`. A caller's tier SHALL be the sole determinant of what they may do
on the account, and each tier SHALL grant every capability of the tiers
below it. `view` grants reading the account's entries and balance. `append`
additionally grants creating entries and editing/deleting only entries the
same user created. `entry_admin` additionally grants editing/deleting any
entry on the account (not only ones that user created). `owner` additionally
grants editing every one of the account's own metadata fields (including
`type`, which is now plain text on the account rather than a reference into
a per-real-owner lookup), disabling/enabling/soft-deleting the account, and
managing shares (inviting, changing a permission, revoking). A shared
`owner`-tier grant carries every one of these rights identically to the
account's real owner (`accounts.owner_id`), which this capability never
reassigns — the real owner is always a distinct, always-knowable identity
from any `owner`-tier share.

#### Scenario: view grants reading only

- **WHEN** a user with `view` permission on an account requests its entries
  or balance
- **THEN** the response succeeds; a request to create, edit, or delete an
  entry, or to edit the account, is rejected

#### Scenario: append grants creating and editing only what was created

- **WHEN** a user with `append` permission creates an entry, then attempts
  to edit an entry created by a different user on the same account
- **THEN** the create succeeds and the edit of the other user's entry is
  rejected

#### Scenario: entry_admin grants editing any entry

- **WHEN** a user with `entry_admin` permission edits or deletes an entry
  created by a different user on the same account
- **THEN** the request succeeds

#### Scenario: owner grants full account management

- **WHEN** a user with a shared `owner`-tier permission edits the account's
  title, edits the account's `type`, disables the account, or invites
  another user to it
- **THEN** each request succeeds, identically to the real owner performing
  it

### Requirement: A user with any permission on an account can see every other user who has one

`GET /api/accounts/{id}/shares` SHALL require the caller to hold at least
`view` permission on the account (real ownership or any share), and SHALL
return the account's real owner (identified distinctly from the shares
below it) plus every current `account_shares` row for that account, each
including the shared user's identity, permission, who granted it, and when.
It SHALL respond `404` for a caller with no permission at all on the
account, matching how a non-owned, unshared account already behaves.

#### Scenario: A view-only user sees the full share list

- **WHEN** a user with only `view` permission on an account calls
  `GET /api/accounts/{id}/shares`
- **THEN** the response is `200` and includes the real owner and every
  other user who has any permission on the account

#### Scenario: A user with no permission gets a not-found response

- **WHEN** a user with neither ownership nor a share on an account calls
  `GET /api/accounts/{id}/shares`
- **THEN** the response is `404`

### Requirement: Only an owner-tier user may invite, change, or revoke a share

`POST` (invite), `PATCH` (change permission), and `DELETE` (revoke) on
`/api/accounts/{id}/shares` SHALL require the caller to be the account's
real owner or hold a shared `owner`-tier permission on it; any other
authenticated caller, including one with `view`/`append`/`entry_admin`
permission, SHALL receive `403`. A `PATCH`/`DELETE` naming the account's
real owner as the target user SHALL be rejected (`400`) — the real owner
carries no `account_shares` row to change or remove.

#### Scenario: entry_admin cannot manage shares

- **WHEN** a user with `entry_admin` permission calls `POST`, `PATCH`, or
  `DELETE` on `/api/accounts/{id}/shares`
- **THEN** the response is `403`

#### Scenario: A shared owner manages shares like the real owner

- **WHEN** a user with a shared `owner`-tier permission invites a new user,
  changes another share's permission, or revokes a share
- **THEN** each request succeeds

#### Scenario: The real owner cannot be targeted by a share change

- **WHEN** an owner-tier caller calls `PATCH` or `DELETE` on
  `/api/accounts/{id}/shares/{ownerUserId}` naming the account's real owner
- **THEN** the request is rejected (`400`) and no share is changed

### Requirement: Sharing by email tells the caller synchronously whether it matched, and whether an invite is currently possible

`POST /api/accounts/{id}/shares` SHALL accept `email` and `permission`
(one of the four tiers) from an owner-tier caller. When `email` (normalized
the same way `authentication` normalizes it) matches an existing,
non-disabled, non-soft-deleted user who is neither the caller nor the
account's real owner, the response SHALL be `201` with the created (or, per
the next requirement, updated) share, and SHALL send that user a
notification email naming the account, the granter, and the permission,
with a link into the application. Unlike `authentication`'s magic-link
start endpoint, this response SHALL NOT hide whether the email matched — the
caller already holds an authorized, owner-tier grant on a real account, so
this is not an anonymous-enumeration surface. When no match is found, the
response SHALL be `200` with `matched: false` and `invite_allowed`
reflecting whether the instance currently permits sending a new invite (the
same `AUTH_SIGNUP_ENABLED`/`AUTH_INVITE_ENABLED` logic
`POST /api/auth/invites` already applies), and no share SHALL be created.
Sharing with the caller's own email or the account's real owner's email
SHALL be rejected (`400`).

#### Scenario: Sharing with a registered user's email

- **WHEN** an owner-tier caller shares an account with the email of an
  existing, active user
- **THEN** the response is `201`, a share is created at the requested
  permission, and that user receives a notification email with an
  application link

#### Scenario: Sharing with an unregistered email is reported synchronously

- **WHEN** an owner-tier caller shares an account with an email matching no
  existing user
- **THEN** the response is `200` with `matched: false`, `invite_allowed`
  reflecting the instance's current invite/signup configuration, and no
  share is created

#### Scenario: An inviter can send an application invite afterward

- **WHEN** an owner-tier caller receives `matched: false` with
  `invite_allowed: true` and then calls `POST /api/auth/invites` for the
  same email
- **THEN** an invite is sent per `authentication`'s existing behavior,
  unmodified by this capability; sharing that email remains a separate,
  later action once they've signed up

#### Scenario: Sharing with your own or the real owner's email is rejected

- **WHEN** an owner-tier caller calls `POST /api/accounts/{id}/shares` with
  their own email, or (as a shared owner) the account's real owner's email
- **THEN** the request is rejected (`400`) and no share is created

### Requirement: Sharing an already-shared email updates the existing share

`POST /api/accounts/{id}/shares` for an email that already has a share on
that account SHALL overwrite that share's `permission` (and its granted-by/
updated timestamp) rather than creating a second row or rejecting as a
conflict.

#### Scenario: Re-sharing at a different tier updates the existing share

- **WHEN** an owner-tier caller shares an account with a user who already
  has `view` permission on it, specifying `append`
- **THEN** the response is `201`, the existing share's permission becomes
  `append`, and no second share row is created

### Requirement: Revoking or leaving a share removes all access immediately, without affecting entries already created

`DELETE /api/accounts/{id}/shares/{userId}` (owner-tier caller, any target
user other than the real owner) and a self-leave variant (the share's own
user, any tier, targeting only themselves) SHALL both remove the
`account_shares` row unconditionally — no soft delete, no grace period.
From the target's next request onward, the account and every entry on it
SHALL behave as if they do not exist for that user (`404`), including for
entries that user created themselves while they held access. Entries
created by the removed user SHALL remain on the account, unchanged and
fully visible/editable per tier, for every user who still has a permission
on it.

#### Scenario: Revoking removes all access, including to the revoked user's own entries

- **WHEN** an owner-tier caller revokes a user's `append` share, and that
  user had created entries on the account
- **THEN** that user can no longer read, edit, or delete the account or any
  of its entries, including the ones they created, while every remaining
  permission holder still sees those entries unchanged

#### Scenario: A user leaves a shared account voluntarily

- **WHEN** a user with any non-owner-real permission on a shared account
  calls the self-leave action
- **THEN** their share is removed and the account subsequently behaves as
  not-found for them, with no owner-tier action required

#### Scenario: The real owner cannot leave their own account

- **WHEN** the real owner of an account attempts the self-leave action on it
- **THEN** the request is rejected (`400`) — there is no share row to remove

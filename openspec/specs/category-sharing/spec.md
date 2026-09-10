# category-sharing Specification

## Purpose

Sharing a category with other registered users: the two permission tiers
(`view`, `append`) and what each grants, the `category_shares` model, the
share-management endpoints (`/api/categories/{id}/shares`), the
email-invite flow, and revocation/self-leave semantics. Unlike
`account-sharing`, there is no shareable "owner" tier — only a category's
real owner may ever edit its metadata, lifecycle, or shares. See
`entry-categories` for how the `append` tier plugs into an entry's
`category_id` validation, and `accounts`/`account-sharing` for the parallel
model this one deliberately mirrors.

## Requirements

### Requirement: Two permission tiers, append a strict superset of view

A category MAY be shared with any number of other registered users, each
at exactly one of two permission tiers: `view` or `append`. A caller's
tier on a category they don't own SHALL be the sole determinant of what
they may do with it. `view` grants seeing the category resolve wherever
it's referenced (entry-list and report category filters, and the name
shown on an entry already categorized under it) but SHALL NOT make it
selectable when creating or editing an entry. `append` additionally grants
selecting the category on a new or edited entry. Neither tier grants
editing the category's own name, icon, colour, or parent, disabling or
enabling it, reordering it among siblings, or deleting it — those remain
exclusive to the category's real owner (`categories.owner_id`), which
this capability never reassigns and for which there is no shareable
equivalent tier.

#### Scenario: view grants filtering but not selecting

- **WHEN** a user with `view` permission on a category filters entries by
  it, or views an entry already categorized under it
- **THEN** the filter and the entry's category both resolve normally; that
  category does not appear among the choosable options on a new or edited
  entry

#### Scenario: append grants selecting the category on entries

- **WHEN** a user with `append` permission on a category creates or edits
  an entry and selects that category
- **THEN** the entry is saved with that `category_id`

#### Scenario: Neither tier grants editing the category itself

- **WHEN** a user with `view` or `append` permission on a category
  attempts to rename it, change its icon/colour/parent, disable or enable
  it, reorder it, or delete it
- **THEN** the request is rejected — only the category's real owner may
  perform any of these

### Requirement: A user with any permission on a category can see every other user who has one

`GET /api/categories/{id}/shares` SHALL require the caller to hold at
least `view` permission on the category (real ownership or any share),
and SHALL return the category's real owner (identified distinctly from
the shares below it) plus every current `category_shares` row for that
category, each including the shared user's identity, permission, who
granted it, and when. It SHALL respond `404` for a caller with no
permission at all on the category.

#### Scenario: A view-only user sees the full share list

- **WHEN** a user with only `view` permission on a category calls
  `GET /api/categories/{id}/shares`
- **THEN** the response is `200` and includes the real owner and every
  other user who has any permission on the category

#### Scenario: A user with no permission gets a not-found response

- **WHEN** a user with neither ownership nor a share on a category calls
  `GET /api/categories/{id}/shares`
- **THEN** the response is `404`

### Requirement: Only the real owner may invite, change, or revoke a share

`POST` (invite), `PATCH` (change permission), and `DELETE` (revoke) on
`/api/categories/{id}/shares` SHALL require the caller to be the
category's real owner; any other authenticated caller, including one with
`append` permission via a share, SHALL receive `403`. Unlike
`account-sharing`, there is no shareable tier that also admits these
actions — a category has exactly two shareable tiers, neither of which is
"owner."

#### Scenario: An append-tier recipient cannot manage shares

- **WHEN** a user with `append` permission (via a share) calls `POST`,
  `PATCH`, or `DELETE` on `/api/categories/{id}/shares`
- **THEN** the response is `403`

#### Scenario: The real owner manages shares

- **WHEN** the category's real owner invites a new user, changes another
  share's permission, or revokes a share
- **THEN** each request succeeds

### Requirement: Sharing by email tells the caller synchronously whether it matched, and whether an invite is currently possible

`POST /api/categories/{id}/shares` SHALL accept `email` and `permission`
(`view` or `append`) from the category's real owner. When `email`
(normalized the same way `authentication` normalizes it) matches an
existing, non-disabled, non-soft-deleted user other than the caller, the
response SHALL be `201` with the created (or, per the next requirement,
updated) share, and SHALL send that user a notification email naming the
category, the granter, and the permission, with a link into the
application. This deliberately does not hide whether the email matched —
the caller already holds an authenticated, real-owner grant on a real
category, mirroring `account-sharing`'s equivalent decision. When no
match is found, the response SHALL be `200` with `matched: false` and
`invite_allowed` reflecting whether the instance currently permits
sending a new invite, and no share SHALL be created. Sharing with the
caller's own email SHALL be rejected (`400`).

#### Scenario: Sharing with a registered user's email

- **WHEN** a category's real owner shares it with the email of an
  existing, active user
- **THEN** the response is `201`, a share is created at the requested
  permission, and that user receives a notification email with an
  application link

#### Scenario: Sharing with an unregistered email is reported synchronously

- **WHEN** a category's real owner shares it with an email matching no
  existing user
- **THEN** the response is `200` with `matched: false`, `invite_allowed`
  reflecting the instance's current invite/signup configuration, and no
  share is created

#### Scenario: Sharing with your own email is rejected

- **WHEN** a category's real owner calls `POST /api/categories/{id}/
  shares` with their own email
- **THEN** the request is rejected (`400`) and no share is created

### Requirement: Sharing an already-shared email updates the existing share

`POST /api/categories/{id}/shares` for an email that already has a share
on that category SHALL overwrite that share's `permission` (and its
granted-by/updated timestamp) rather than creating a second row or
rejecting as a conflict.

#### Scenario: Re-sharing at a different tier updates the existing share

- **WHEN** a category's real owner shares it with a user who already has
  `view` permission on it, specifying `append`
- **THEN** the response is `201`, the existing share's permission becomes
  `append`, and no second share row is created

### Requirement: Revoking or leaving a share removes all access immediately, without affecting entries already categorized under it

`DELETE /api/categories/{id}/shares/{userId}` (real-owner caller, any
target user) and a self-leave variant (the share's own user, any tier,
targeting only themselves) SHALL both remove the `category_shares` row
unconditionally — no soft delete, no grace period. From the target's next
request onward, the category SHALL no longer be selectable by them on a
new or edited entry, SHALL no longer appear in their own `GET /api/
categories` response, and SHALL no longer resolve for their own
entry-list/report category filters. Any entry already carrying that
`category_id` — created by the removed user or anyone else — SHALL be
unaffected: its category continues to resolve and display normally for
every user who can otherwise see that entry, mirroring how disabling a
category already leaves existing references untouched.

#### Scenario: Revoking stops new selection without touching existing entries

- **WHEN** a category's real owner revokes a user's `append` share, and
  that user had already categorized entries under it
- **THEN** that user can no longer select the category on a new or edited
  entry, while their previously categorized entries keep showing that
  category normally

#### Scenario: A user leaves a shared category voluntarily

- **WHEN** a user with any permission on a category shared with them
  calls the self-leave action
- **THEN** their share is removed, with no owner action required

#### Scenario: The real owner cannot leave their own category

- **WHEN** a category's real owner attempts the self-leave action on it
- **THEN** the request is rejected (`400`) — there is no share row to
  remove

### Requirement: A shared category never cascades to its children

Sharing a category SHALL grant access to that category alone — none of
its child categories, if any, become visible or usable to the recipient
as a result. A child category remains private to its own owner unless it
is separately, individually shared.

#### Scenario: A child category is not implicitly shared

- **WHEN** a category with one or more children is shared with a user
- **THEN** that user gains no permission on any of its children, only on
  the shared category itself

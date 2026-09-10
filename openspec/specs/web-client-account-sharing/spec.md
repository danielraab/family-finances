# web-client-account-sharing Specification

## Purpose

The `/accounts/{id}/sharing` page: its entry point, permission-gated
read-only vs. management modes, the invite-by-email form (including the
"not registered, send an invite?" affordance), and the per-row permission/
revoke/leave controls. See `account-sharing` for the backend capability
this page is a client for, and `web-client-accounts` for the Share button
that links here.

## Requirements

### Requirement: The sharing page requires at least view permission

`/accounts/{id}/sharing` SHALL be reachable only by a visitor who holds at
least `view` permission on the account (real ownership or any share). An
anonymous visitor SHALL be redirected to `/login`; an authenticated visitor
with no permission on the account SHALL see the page reflect the backend's
`404`, matching every other accounts route.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/accounts/{id}/sharing`
- **THEN** the client redirects them to `/login`

#### Scenario: A visitor with no permission is not found

- **WHEN** an authenticated visitor with no permission on an account
  navigates to `/accounts/{id}/sharing`
- **THEN** the page reflects the backend's `404`

### Requirement: The sharing page lists the real owner and every share, read-only below owner tier

`/accounts/{id}/sharing` SHALL fetch and display `GET /api/accounts/{id}/
shares`: the real owner as a distinct, first, unremovable row, followed by
every current share with that user's name, permission, and who granted it.
A visitor whose own permission on the account is below `owner`-tier
(`view`, `append`, or `entry_admin`, including the real owner's own view of
their own list) SHALL see this list with no invite form, no permission
editor, and no revoke action on any row but their own — only their own row,
if present, SHALL show a Leave action. A visitor with `owner`-tier
permission (real or shared) SHALL additionally see the invite form and,
on every non-owner row, a permission editor and a Revoke action.

#### Scenario: A view-only visitor sees the list without management controls

- **WHEN** a visitor with only `view` permission opens the sharing page
- **THEN** they see the real owner and every share, with no invite form and
  no permission-editing or revoke controls on any row

#### Scenario: An owner-tier visitor sees full management controls

- **WHEN** a visitor with `owner`-tier permission (real or shared) opens
  the sharing page
- **THEN** they see the invite form, plus a permission editor and Revoke
  action on every share row other than the real owner's

#### Scenario: The real owner's row is never removable

- **WHEN** any visitor views the sharing page
- **THEN** the real owner's row shows no permission editor, Revoke, or
  Leave action

### Requirement: Inviting by email surfaces an unregistered-email nudge

The invite form (owner-tier visitors only) SHALL offer an email field and a
permission selector, submitting `POST /api/accounts/{id}/shares`. On a
matched response, the new share SHALL appear in the list immediately. On an
unmatched response, the page SHALL show text explaining the email is not
registered; when the response's `invite_allowed` is true, it SHALL
additionally offer a small inline form or link — prefilled with the same
email — that calls `POST /api/auth/invites` to send an application invite;
when `invite_allowed` is false, it SHALL instead explain that inviting is
currently unavailable, with no such affordance.

#### Scenario: Sharing with a registered user updates the list

- **WHEN** an owner-tier visitor submits the invite form with a registered
  user's email
- **THEN** the new share appears in the list without a page reload

#### Scenario: An unmatched email offers to send an application invite

- **WHEN** an owner-tier visitor submits the invite form with an email that
  matches no user, and the instance currently allows inviting
- **THEN** the page explains the email isn't registered and offers a
  prefilled way to send an application invite to it

#### Scenario: An unmatched email with inviting disabled shows no invite affordance

- **WHEN** an owner-tier visitor submits the invite form with an email that
  matches no user, and the instance currently does not allow inviting
- **THEN** the page explains the email isn't registered, with no invite
  affordance offered

### Requirement: Changing a permission or revoking a share requires confirmation

An owner-tier visitor's permission editor on a share row SHALL apply
immediately on selection (`PATCH /api/accounts/{id}/shares/{userId}`),
mirroring the account-types settings tab's direct-apply pattern for a
reversible change. Revoking a share, and leaving a shared account
(self-leave), SHALL each require an explicit confirmation step — the same
`@headlessui/react` `Dialog` pattern used throughout `/settings` — before
the request is sent.

#### Scenario: Changing a permission applies immediately

- **WHEN** an owner-tier visitor selects a different permission for a share
  row
- **THEN** `PATCH /api/accounts/{id}/shares/{userId}` is called immediately,
  with no confirmation dialog

#### Scenario: Revoking requires confirmation

- **WHEN** an owner-tier visitor activates Revoke on a share row
- **THEN** a confirmation dialog appears and `DELETE /api/accounts/{id}/
  shares/{userId}` is not called until confirmed

#### Scenario: Leaving requires confirmation

- **WHEN** a visitor activates Leave on their own share row
- **THEN** a confirmation dialog appears and the request is not sent until
  confirmed

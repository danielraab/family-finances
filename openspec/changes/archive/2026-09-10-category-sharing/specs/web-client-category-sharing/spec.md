## ADDED Requirements

### Requirement: The sharing page requires at least view permission

`/categories/{id}/sharing` SHALL be reachable only by a visitor who holds
at least `view` permission on the category (real ownership or any
share). An anonymous visitor SHALL be redirected to `/login`; an
authenticated visitor with no permission on the category SHALL see the
page reflect the backend's `404`.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/categories/{id}/sharing`
- **THEN** the client redirects them to `/login`

#### Scenario: A visitor with no permission is not found

- **WHEN** an authenticated visitor with no permission on a category
  navigates to `/categories/{id}/sharing`
- **THEN** the page reflects the backend's `404`

### Requirement: The sharing page lists the real owner and every share, read-only for a non-owner

`/categories/{id}/sharing` SHALL fetch and display `GET /api/categories/
{id}/shares`: the real owner as a distinct, first, unremovable row,
followed by every current share with that user's name, permission, and
who granted it. A visitor who is not the category's real owner SHALL see
this list with no invite form, no permission editor, and no revoke action
on any row but their own — only their own row, if present, SHALL show a
Leave action. The real owner SHALL additionally see the invite form and,
on every non-owner row, a permission editor and a Revoke action.

#### Scenario: A non-owner visitor sees the list without management controls

- **WHEN** a visitor who is not the category's real owner opens the
  sharing page
- **THEN** they see the real owner and every share, with no invite form
  and no permission-editing or revoke controls on any row

#### Scenario: The real owner sees full management controls

- **WHEN** the category's real owner opens the sharing page
- **THEN** they see the invite form, plus a permission editor and Revoke
  action on every share row other than their own

#### Scenario: The real owner's row is never removable

- **WHEN** any visitor views the sharing page
- **THEN** the real owner's row shows no permission editor, Revoke, or
  Leave action

### Requirement: Inviting by email surfaces an unregistered-email nudge

The invite form (real-owner visitors only) SHALL offer an email field and
a two-option permission selector (`view`, `append`), submitting
`POST /api/categories/{id}/shares`. On a matched response, the new share
SHALL appear in the list immediately. On an unmatched response, the page
SHALL show text explaining the email is not registered; when the
response's `invite_allowed` is true, it SHALL additionally offer a small
inline form or link — prefilled with the same email — that calls
`POST /api/auth/invites` to send an application invite; when
`invite_allowed` is false, it SHALL instead explain that inviting is
currently unavailable, with no such affordance.

#### Scenario: Sharing with a registered user updates the list

- **WHEN** the real owner submits the invite form with a registered
  user's email
- **THEN** the new share appears in the list without a page reload

#### Scenario: An unmatched email offers to send an application invite

- **WHEN** the real owner submits the invite form with an email that
  matches no user, and the instance currently allows inviting
- **THEN** the page explains the email isn't registered and offers a
  prefilled way to send an application invite to it

### Requirement: Changing a permission or revoking a share requires confirmation

The real owner's permission editor on a share row SHALL apply immediately
on selection (`PATCH /api/categories/{id}/shares/{userId}`). Revoking a
share, and leaving a shared category (self-leave), SHALL each require an
explicit confirmation step before the request is sent, mirroring
`web-client-account-sharing`'s equivalent behavior.

#### Scenario: Changing a permission applies immediately

- **WHEN** the real owner selects a different permission for a share row
- **THEN** `PATCH /api/categories/{id}/shares/{userId}` is called
  immediately, with no confirmation dialog

#### Scenario: Revoking requires confirmation

- **WHEN** the real owner activates Revoke on a share row
- **THEN** a confirmation dialog appears and `DELETE /api/categories/{id}/
  shares/{userId}` is not called until confirmed

#### Scenario: Leaving requires confirmation

- **WHEN** a visitor activates Leave on their own share row
- **THEN** a confirmation dialog appears and the request is not sent
  until confirmed

### Requirement: The categories page shows a "Shared with me" section, flat and read-only

`/categories` SHALL render, below the visitor's own tree, a separate
section listing every category shared with them (any tier), flat — with
no nesting, regardless of that category's `parent_id` in its real owner's
tree. Each row SHALL show the category's name (via `CategoryLabel`, with
its icon/colour badge when set), a shared badge naming the real owner,
and its permission tier. These rows SHALL offer no edit, reorder,
disable/enable, or delete controls; a row SHALL offer only a Leave action
(confirmed) and, implicitly via `CategoryLabel`, a link into
`/categories/{id}/sharing` for viewing who else has access.

#### Scenario: A shared category renders flat regardless of its real position

- **WHEN** a category shared with the visitor is a child category in its
  real owner's tree
- **THEN** it still renders as a single, unnested row in the "Shared with
  me" section

#### Scenario: Shared rows offer no owner-only controls

- **WHEN** the visitor views a row in the "Shared with me" section
- **THEN** no edit, reorder, disable/enable, or delete action is shown for
  it

#### Scenario: The section is absent when nothing is shared

- **WHEN** the visitor has no categories shared with them
- **THEN** the "Shared with me" section is not shown

### Requirement: An owned category offers a Share action

Each category in the visitor's own tree on `/categories` SHALL offer a
Share action, alongside its existing Edit action, linking to
`/categories/{id}/sharing`.

#### Scenario: Opening the sharing page from the categories tree

- **WHEN** the visitor activates Share on one of their own categories
- **THEN** the client navigates to `/categories/{id}/sharing`

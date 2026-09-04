# web-client-settings Specification

## Purpose

The authenticated /settings page: the sidebar link to it, its auth gate,
its Common tab (language/timezone/default-currency preferences), its My
Invitations tab (every authenticated visitor's own sent invitations, with
revoke), and its admin-only Users tab
(list/invite/disable/enable/delete/revoke). See `user-settings` and
`user-administration` for the backend capabilities it calls.

## Requirements

### Requirement: Settings link in the sidebar user menu

The authenticated sidebar user menu (`SidebarUser`) SHALL contain a
"Settings" item, positioned above "Log out", that navigates to `/settings`.

#### Scenario: Opening settings from the user menu

- **WHEN** an authenticated visitor opens the sidebar user menu and activates
  "Settings"
- **THEN** the client navigates to `/settings`

### Requirement: Settings route requires authentication

`/settings` (and its tabs) SHALL be accessible only to an authenticated
visitor. An anonymous visitor navigating to `/settings` SHALL be redirected
to `/login`. While `useAuth` is `loading`, the route SHALL render nothing
that would flash before the redirect-or-render decision is made.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/settings`
- **THEN** the client redirects them to `/login`

#### Scenario: Authenticated visitor sees the page

- **WHEN** an authenticated visitor navigates to `/settings`
- **THEN** the settings page renders with its tabs

### Requirement: Common tab

The settings page SHALL default to a **Common** tab, visible to every
authenticated visitor, containing three controls: display language
(English/German), timezone (populated from the browser's supported IANA
zones), and default currency (a three-letter code, validated client-side to
that shape). Each control SHALL save on change, calling
`PUT /api/settings` with only that field, with no separate save action.
Changing the language control SHALL also switch the running app's language
immediately, without a reload.

#### Scenario: Changing language applies immediately

- **WHEN** an authenticated visitor on the Common tab selects German
- **THEN** `PUT /api/settings` is called with `{ "language": "de" }`
- **AND** the app's UI text switches to German without a page reload

#### Scenario: Changing timezone does not affect other fields

- **WHEN** an authenticated visitor changes only the timezone control
- **THEN** the request updates only `timezone`, leaving language and default
  currency as they were

### Requirement: Users tab is admin-only

The settings page SHALL offer a **Users** tab only when the authenticated
visitor's `is_admin` is `true`; for a non-admin, the tab SHALL NOT be
rendered in the tab list, and navigating directly to `/settings/users` SHALL
redirect to `/settings` rather than rendering the tab's content.

#### Scenario: Non-admin does not see the Users tab

- **WHEN** a non-admin authenticated visitor opens `/settings`
- **THEN** only the Common tab is shown in the tab list

#### Scenario: Non-admin is redirected away from a direct link

- **WHEN** a non-admin authenticated visitor navigates directly to
  `/settings/users`
- **THEN** the client redirects them to `/settings`

#### Scenario: Admin sees the Users tab

- **WHEN** an admin authenticated visitor opens `/settings`
- **THEN** both the Common and Users tabs are shown

### Requirement: Users tab lists users and invitations

The Users tab SHALL, on mount, fetch and display every user
(`GET /api/auth/users`) and every invitation (`GET /api/auth/invites`,
including who invited each address). It SHALL offer a form to invite a new
address (`POST /api/auth/invites`), and, per listed user, controls to
disable/enable (`POST /api/auth/users/{id}/disable` /
`POST /api/auth/users/{id}/enable`) and to (soft) delete
(`DELETE /api/auth/users/{id}`). Per listed invitation, it SHALL offer a
Revoke control (`POST /api/auth/invites/{id}/revoke`). When there are no
invitations to display, the tab SHALL show text stating that plainly instead
of an empty list.

#### Scenario: Inviting a new address

- **WHEN** an admin submits the invite form with a valid email address
- **THEN** `POST /api/auth/invites` is called and, on success, the new
  invitation appears in the invitations list

#### Scenario: Disabling a user from the tab

- **WHEN** an admin activates "Disable" for a listed user
- **THEN** `POST /api/auth/users/{id}/disable` is called and, on success, that
  user's row reflects the disabled state

#### Scenario: Revoking an invitation from the tab

- **WHEN** an admin activates "Revoke" for a listed invitation
- **THEN** `POST /api/auth/invites/{id}/revoke` is called and, on success,
  that invitation's row reflects the revoked status without disappearing
  from the list

#### Scenario: Empty invitations list shows explanatory text

- **WHEN** an admin opens the Users tab and there are no invitations to show
- **THEN** the invitations section displays text saying there are none,
  instead of rendering nothing

### Requirement: Destructive user actions require confirmation

Disabling, enabling, and (soft) deleting a user, and revoking an invitation,
from the Users tab SHALL each require an explicit confirmation step before
the request is sent. When the target of disable or delete is the acting
admin's own account, the confirmation copy SHALL say so explicitly (including
that they will be signed out immediately).

#### Scenario: Confirmation blocks an accidental click

- **WHEN** an admin activates "Delete" for a listed user
- **THEN** a confirmation step appears and `DELETE /api/auth/users/{id}` is
  not called until it is confirmed

#### Scenario: Self-targeting is called out

- **WHEN** an admin activates "Disable" or "Delete" on their own account
- **THEN** the confirmation step's copy states that they are acting on their
  own account and will be signed out immediately

#### Scenario: Revoking an invitation is confirmed before sending

- **WHEN** an admin activates "Revoke" for a listed invitation
- **THEN** a confirmation step appears and
  `POST /api/auth/invites/{id}/revoke` is not called until it is confirmed

### Requirement: Self-disable or self-delete signs the admin out locally

When an admin's disable or delete action targets their own account and
succeeds, the client SHALL immediately transition `useAuth` to `anonymous`
(the same local transition `logout()` performs) and navigate away from the
admin-only view, without waiting for a subsequent request to discover the
`401`.

#### Scenario: Self-disable signs out immediately

- **WHEN** an admin successfully disables their own account from the Users
  tab
- **THEN** the client's auth state becomes `anonymous` and the visitor is
  navigated away from `/settings/users` without needing to reload

### Requirement: My Invitations tab

The settings page SHALL offer a **My Invitations** tab to every authenticated
visitor (no `is_admin` requirement), positioned between Common and the
admin-only Users tab. On mount it SHALL fetch and display the visitor's own
invitations (`GET /api/auth/invites/mine`) — those they personally created —
with each row's status (pending, accepted, revoked) and, per row, a Revoke
control (`POST /api/auth/invites/{id}/revoke`) behind the same confirmation
pattern used on the Users tab. When there are no invitations to display, the
tab SHALL show text stating that plainly instead of an empty list.

#### Scenario: Every authenticated visitor sees the tab

- **WHEN** any authenticated visitor, admin or not, opens `/settings`
- **THEN** the tab list includes "My Invitations" alongside Common

#### Scenario: Tab lists only the visitor's own invitations

- **WHEN** an authenticated visitor opens the My Invitations tab
- **THEN** `GET /api/auth/invites/mine` is called and only invitations they
  personally created are shown

#### Scenario: Revoking from the My Invitations tab

- **WHEN** a visitor confirms "Revoke" for one of their own listed
  invitations
- **THEN** `POST /api/auth/invites/{id}/revoke` is called and, on success,
  that invitation's row reflects the revoked status without disappearing
  from the list

#### Scenario: Empty state

- **WHEN** a visitor opens the My Invitations tab and has created no
  invitations, or all of theirs have been soft-deleted
- **THEN** the tab displays text saying there are none, instead of rendering
  an empty list

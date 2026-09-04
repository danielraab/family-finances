## MODIFIED Requirements

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

## ADDED Requirements

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

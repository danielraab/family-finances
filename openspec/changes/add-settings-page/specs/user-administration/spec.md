## ADDED Requirements

### Requirement: Admin-only user and invitation listing

`GET /api/auth/users` SHALL require authentication and `is_admin`, and SHALL
return every non-soft-deleted user (id, email, display_name, is_admin,
disabled, created_at). `GET /api/auth/invites` SHALL require authentication
and `is_admin`, and SHALL return every invitation (email, who invited them,
created_at, expires_at, accepted_at), regardless of whether it has been
accepted or has expired. Both SHALL respond `403` for an authenticated
non-admin and `401` for no valid session.

#### Scenario: Admin lists users

- **WHEN** an admin calls `GET /api/auth/users`
- **THEN** the response is `200` with every non-deleted user, each including
  `disabled`

#### Scenario: Non-admin is forbidden

- **WHEN** an authenticated non-admin calls `GET /api/auth/users` or
  `GET /api/auth/invites`
- **THEN** the response is `403`

#### Scenario: Deleted users are excluded

- **WHEN** a user has been soft-deleted and an admin calls
  `GET /api/auth/users`
- **THEN** that user does not appear in the response

#### Scenario: Invite listing shows the inviter

- **WHEN** an admin calls `GET /api/auth/invites`
- **THEN** each invitation includes the identity of the user who created it

### Requirement: Disabling and enabling a user

`POST /api/auth/users/{id}/disable` SHALL require authentication and
`is_admin`, SHALL set the target user's `disabled` flag to `true`, and SHALL
immediately delete every session row belonging to that user (not merely rely
on expiry). `POST /api/auth/users/{id}/enable` SHALL require the same
authorization and SHALL clear the `disabled` flag; it SHALL NOT restore any
previously-revoked session. An admin MAY disable or enable their own account,
including when they are the only remaining admin; no check prevents this.

#### Scenario: Disabling revokes active sessions immediately

- **WHEN** an admin calls `POST /api/auth/users/{id}/disable` for a user with
  an active session
- **THEN** the target's `disabled` becomes `true`
- **AND** a request using that user's previously-valid session token is now
  `401`

#### Scenario: Enabling does not restore sessions

- **WHEN** an admin re-enables a previously disabled user
- **THEN** `disabled` becomes `false`
- **AND** the user must sign in again to obtain a new session

#### Scenario: Admin may disable themselves, even as the last admin

- **WHEN** the only remaining admin calls
  `POST /api/auth/users/{id}/disable` on their own id
- **THEN** the request succeeds and their own sessions are revoked
  immediately, with no server-side check blocking it

### Requirement: Soft-deleting a user

`DELETE /api/auth/users/{id}` SHALL require authentication and `is_admin`,
SHALL set the target user's `deleted_at` to the current time, and SHALL
immediately delete every session row belonging to that user. A soft-deleted
user SHALL be excluded from `GET /api/auth/users`. This change introduces no
mechanism to reverse a soft delete. An admin MAY soft-delete their own
account, including when they are the only remaining admin; no check prevents
this.

#### Scenario: Soft-deleting revokes access immediately

- **WHEN** an admin calls `DELETE /api/auth/users/{id}` for a user with an
  active session
- **THEN** `deleted_at` is set and that session's next request is `401`

#### Scenario: Soft-deleted user is hidden from the listing

- **WHEN** a user has been soft-deleted
- **THEN** they no longer appear in `GET /api/auth/users`

#### Scenario: No self-lockout protection

- **WHEN** the only remaining admin calls `DELETE /api/auth/users/{id}` on
  their own id
- **THEN** the request succeeds with no server-side check blocking it

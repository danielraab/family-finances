## RENAMED Requirements

- FROM: `### Requirement: Admin-only user and invitation listing`
- TO: `### Requirement: Admin-only user listing`

## MODIFIED Requirements

### Requirement: Admin-only user listing

`GET /api/auth/users` SHALL require authentication and `is_admin`, and SHALL
return every non-soft-deleted user (id, email, display_name, is_admin,
disabled, created_at). It SHALL respond `403` for an authenticated non-admin
and `401` for no valid session.

#### Scenario: Admin lists users

- **WHEN** an admin calls `GET /api/auth/users`
- **THEN** the response is `200` with every non-deleted user, each including
  `disabled`

#### Scenario: Non-admin is forbidden

- **WHEN** an authenticated non-admin calls `GET /api/auth/users`
- **THEN** the response is `403`

#### Scenario: Deleted users are excluded

- **WHEN** a user has been soft-deleted and an admin calls
  `GET /api/auth/users`
- **THEN** that user does not appear in the response

## ADDED Requirements

### Requirement: Invitation listing

`GET /api/auth/invites` SHALL require authentication and `is_admin`, and
SHALL return every non-soft-deleted invitation regardless of status (pending,
accepted, expired, or revoked), each including who invited them, `created_at`,
`expires_at`, `accepted_at`, and `revoked_at`. It SHALL respond `403` for an
authenticated non-admin and `401` for no valid session.

`GET /api/auth/invites/mine` SHALL require authentication only — no
`is_admin` requirement — and SHALL return every non-soft-deleted invitation
created by the calling user (`invited_by` matching the caller), in the same
shape and regardless of status.

#### Scenario: Admin lists every invitation

- **WHEN** an admin calls `GET /api/auth/invites`
- **THEN** the response is `200` with every non-deleted invitation regardless
  of status, each including `revoked_at`

#### Scenario: Non-admin is forbidden from the all-invitations listing

- **WHEN** an authenticated non-admin calls `GET /api/auth/invites`
- **THEN** the response is `403`

#### Scenario: Invite listing shows the inviter

- **WHEN** an admin calls `GET /api/auth/invites`
- **THEN** each invitation includes the identity of the user who created it

#### Scenario: A user sees only the invitations they created

- **WHEN** an authenticated user calls `GET /api/auth/invites/mine`
- **THEN** the response is `200` with only the invitations whose `invited_by`
  is that user, regardless of the caller's `is_admin` status

#### Scenario: Soft-deleted invitations are excluded from both listings

- **WHEN** an invitation has been soft-deleted
- **THEN** it does not appear in `GET /api/auth/invites` or
  `GET /api/auth/invites/mine`

#### Scenario: Revoked invitations remain listed

- **WHEN** an invitation has been revoked but not soft-deleted
- **THEN** it still appears in both `GET /api/auth/invites` and (for its
  inviter) `GET /api/auth/invites/mine`, showing its `revoked_at`

### Requirement: Revoking an invitation

`POST /api/auth/invites/{id}/revoke` SHALL require authentication. It SHALL
be permitted for the invitation's own inviter (`invited_by` matches the
caller) or for a caller with `is_admin`; any other authenticated caller SHALL
receive `403`. A request naming an id that does not exist, or that has been
soft-deleted, SHALL receive `404`.

The action SHALL set the invitation's `revoked_at` to the current time only
if it is not already set, and SHALL be idempotent: a repeat call on an
already-revoked invitation SHALL succeed (`200`) without changing the stored
`revoked_at`. Revoking SHALL be permitted regardless of the invitation's
current status (pending, accepted, or expired). A revoked invitation SHALL
remain visible in both invitation listings and SHALL NOT be removed or
hidden by revoking alone.

#### Scenario: Inviter revokes their own invitation

- **WHEN** a non-admin user calls `POST /api/auth/invites/{id}/revoke` for an
  invitation they created
- **THEN** the response is `200` and the invitation's `revoked_at` is set

#### Scenario: Admin revokes someone else's invitation

- **WHEN** an admin calls `POST /api/auth/invites/{id}/revoke` for an
  invitation created by a different user
- **THEN** the response is `200` and the invitation's `revoked_at` is set

#### Scenario: Non-admin, non-inviter is forbidden

- **WHEN** an authenticated user who neither created the invitation nor has
  `is_admin` calls `POST /api/auth/invites/{id}/revoke`
- **THEN** the response is `403` and `revoked_at` is unchanged

#### Scenario: Revoking twice is idempotent

- **WHEN** `POST /api/auth/invites/{id}/revoke` is called a second time for
  an already-revoked invitation
- **THEN** the response is `200` and `revoked_at` is unchanged from its
  original value

#### Scenario: Revoking an accepted or expired invitation still succeeds

- **WHEN** `POST /api/auth/invites/{id}/revoke` is called for an invitation
  that has already been accepted, or whose `expires_at` has passed
- **THEN** the response is `200` and `revoked_at` is set

#### Scenario: Unknown or soft-deleted invitation is not found

- **WHEN** `POST /api/auth/invites/{id}/revoke` is called for an id that does
  not exist, or that has been soft-deleted
- **THEN** the response is `404`

### Requirement: Soft-deleting a revoked invitation

`DELETE /api/auth/invites/{id}` SHALL require authentication and `is_admin`.
It SHALL be rejected with `409` when the target invitation's `revoked_at` is
not set. Otherwise it SHALL set the invitation's `deleted_at` to the current
time. A soft-deleted invitation SHALL be excluded from
`GET /api/auth/invites` and `GET /api/auth/invites/mine`, and a subsequent
revoke or delete call naming it SHALL respond `404`. This capability
introduces no mechanism to reverse a soft delete.

#### Scenario: Admin soft-deletes a revoked invitation

- **WHEN** an admin calls `DELETE /api/auth/invites/{id}` for an invitation
  whose `revoked_at` is set
- **THEN** the response is `204` and the invitation's `deleted_at` is set

#### Scenario: Deleting a non-revoked invitation is rejected

- **WHEN** an admin calls `DELETE /api/auth/invites/{id}` for an invitation
  whose `revoked_at` is not set
- **THEN** the response is `409` and no column is changed

#### Scenario: Soft-deleted invitation disappears from both listings

- **WHEN** an invitation has been soft-deleted
- **THEN** it no longer appears in `GET /api/auth/invites` or
  `GET /api/auth/invites/mine`

#### Scenario: Non-admin is forbidden

- **WHEN** a non-admin authenticated user calls `DELETE /api/auth/invites/{id}`
- **THEN** the response is `403`, including when the caller is the
  invitation's own inviter

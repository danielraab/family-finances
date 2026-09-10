## RENAMED Requirements

- FROM: `### Requirement: An account has exactly one owner and is visible only to them`
- TO: `### Requirement: An account has exactly one real owner; visibility and lifecycle extend to every permission holder`

## MODIFIED Requirements

### Requirement: An account has exactly one real owner; visibility and lifecycle extend to every permission holder

Every account SHALL carry a required `owner_id`, set to the authenticated
caller at creation and never changed thereafter — the account's real owner.
Reading an account and its balance SHALL be permitted for any caller who
holds at least `view` permission on it (real ownership, or a share per
`account-sharing`); creating, listing, editing, disabling/enabling, and soft-
deleting an account, and managing its shares, SHALL be permitted only for
the real owner or a caller holding a shared `owner`-tier permission on it,
identically. An account a caller holds no permission on at all — real
ownership or any share — SHALL behave as if it does not exist (`404`),
including for an admin, unchanged from before this capability existed.
`GET /api/accounts` SHALL return every account the caller owns or has any
share on, each carrying the caller's effective permission and, for an
account the caller does not really own, the real owner's identity.

#### Scenario: Owner sees their own account

- **WHEN** an authenticated user calls `GET /api/accounts/{id}` for an
  account they own
- **THEN** the response is `200` with that account

#### Scenario: A user with any share tier can read a shared account

- **WHEN** a user with `view` permission (via a share) calls
  `GET /api/accounts/{id}`
- **THEN** the response is `200` with the account, including the real
  owner's identity

#### Scenario: A user with no permission cannot see the account

- **WHEN** an authenticated user with neither ownership nor a share on an
  account calls `GET /api/accounts/{id}`
- **THEN** the response is `404`

#### Scenario: Admin without a share still cannot see the account

- **WHEN** a user with `is_admin = true` and no share on an account calls
  `GET /api/accounts/{id}` for an account they do not own
- **THEN** the response is `404` — `is_admin` grants no visibility, exactly
  as before this capability existed

#### Scenario: Listing includes owned and shared accounts

- **WHEN** an authenticated user who owns one account and has a share on
  another calls `GET /api/accounts`
- **THEN** both accounts appear, each carrying the caller's effective
  permission, and the shared one additionally carries its real owner's
  identity

#### Scenario: A view or append tier cannot edit account metadata

- **WHEN** a user with `view`, `append`, or `entry_admin` permission calls
  `PATCH /api/accounts/{id}`, `POST /api/accounts/{id}/disable`, or
  `DELETE /api/accounts/{id}`
- **THEN** the response is `403`

#### Scenario: A shared owner manages the account like the real owner

- **WHEN** a user with a shared `owner`-tier permission edits, disables,
  enables, or soft-deletes the account
- **THEN** each request succeeds identically to the real owner performing it

### Requirement: `type_id` remains reassignable only by the real owner

An account's `type_id` SHALL be changeable only by the account's real owner
(`owner_id`), even when a caller holds a shared `owner`-tier permission on
the account — `account_types` remains a per-real-owner lookup, unextended
by sharing. `PATCH /api/accounts/{id}` from a shared `owner`-tier caller
that includes `type_id` SHALL be rejected (`403`); the same request with
every other field, and no `type_id`, SHALL succeed normally.

#### Scenario: A shared owner cannot reassign the account's type

- **WHEN** a user with a shared `owner`-tier permission calls
  `PATCH /api/accounts/{id}` including a `type_id`
- **THEN** the response is `403` and the account's `type_id` is unchanged

#### Scenario: A shared owner can edit every other field

- **WHEN** a user with a shared `owner`-tier permission calls
  `PATCH /api/accounts/{id}` changing `title`, `description`, `icon`,
  `color`, or other non-`type_id` fields
- **THEN** the update succeeds

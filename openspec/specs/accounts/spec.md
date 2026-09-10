# accounts Specification

## Purpose

Accounts a user owns for tracking their finances — title,
description, type, currency, financial institute, opening/closing
dates — their ownership/visibility rules, the reversible `disabled`
flag, and soft delete. `type` is a free-text label stored on the
account; `GET /api/account-types` lists the caller's distinct in-use
values for autocomplete. See `account-entries` for the entries recorded
against an account and its live balance, and `web-client-accounts`
for the client surface.

## Requirements

### Requirement: Account fields and creation

An account SHALL carry: `title` (required, non-empty), `description`
(optional), `type` (required, a free-text label the backend only trims of
leading/trailing whitespace before storing — non-empty after trimming, not
case-folded, not checked against any canonical list or lookup table),
`currency` (required, ISO-4217 shape — three uppercase letters, not checked
against a canonical list, validated the same way as `user-settings`'
`default_currency`), `financial_institute` (optional), `opening_date`
(required), and `closing_date` (optional). When `closing_date` is present it
SHALL NOT be before `opening_date`. `type` SHALL be accepted on
`POST /api/accounts` and `PATCH /api/accounts/{id}` and SHALL appear on
every response that carries an `Account`.

#### Scenario: Creating an account with valid fields

- **WHEN** an authenticated user calls `POST /api/accounts` with a title, a
  non-empty `type`, a three-letter currency, and an opening date
- **THEN** the response is `201` with the created account, owned by the
  caller, its `type` stored with surrounding whitespace trimmed and its
  casing preserved verbatim

#### Scenario: A blank type is rejected

- **WHEN** `POST /api/accounts` or `PATCH /api/accounts/{id}` sends a
  `type` that is empty or only whitespace
- **THEN** the request is rejected (`400` — the same `ErrInvalidValue`
  mapping as a blank `title`) and nothing is created or changed

#### Scenario: Closing date before opening date rejected

- **WHEN** `POST /api/accounts` or an update sets `closing_date` earlier
  than `opening_date`
- **THEN** the request is rejected (`422`) and nothing is changed

#### Scenario: Invalid currency shape rejected

- **WHEN** an account is created or updated with a `currency` that is not
  three uppercase letters
- **THEN** the request is rejected (`422`)

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

### Requirement: In-use account types are listed for autocomplete

`GET /api/account-types` SHALL return a JSON array of strings: the
distinct, non-empty `type` values present on the authenticated caller's own
non-deleted accounts, compared verbatim (case-sensitively, after the same
trimming applied on write), sorted case-insensitively ascending. It SHALL
never include values from another user's accounts. The endpoint is
read-only — there is no endpoint to create, rename, disable, enable, or
delete an account type, and `type` is only ever set by writing an account.

#### Scenario: The caller's distinct in-use types are returned

- **WHEN** an authenticated user with accounts of type `Checking`,
  `Savings`, and a second `Checking` calls `GET /api/account-types`
- **THEN** the response is `200` with `["Checking", "Savings"]`

#### Scenario: Another user's types are not included

- **WHEN** an authenticated user whose own accounts are all `Checking`
  calls `GET /api/account-types`, while a different user has an account of
  type `Brokerage`
- **THEN** the response contains `Checking` and does not contain
  `Brokerage`

#### Scenario: A user with no accounts gets an empty list

- **WHEN** an authenticated user with no non-deleted accounts calls
  `GET /api/account-types`
- **THEN** the response is `200` with `[]`

### Requirement: An account can be disabled and re-enabled, blocking new entries without hiding it

Every account SHALL carry a `disabled` flag, `false` by default, settable
by its owner via `POST /api/accounts/{id}/disable` and reversed via
`POST /api/accounts/{id}/enable`. Disabling SHALL NOT remove the account
from listings, reads, or updates, and SHALL NOT affect its existing
entries in any way — its only effect is that creating a *new* entry
against a disabled account is rejected (see `account-entries`). `disabled`
is independent of `closing_date` (informational only) and of `deleted_at`
(soft delete) — any combination of the three MAY hold at once.

#### Scenario: Disabling blocks new entries but not visibility

- **WHEN** an owner disables an account
- **THEN** `GET /api/accounts/{id}` and `GET /api/accounts` still show it,
  its existing entries are unaffected, and a subsequent
  `POST /api/entries` against it is rejected

#### Scenario: Enabling reverses it

- **WHEN** an owner enables a previously disabled account
- **THEN** creating a new entry against it succeeds again

#### Scenario: Disabling is independent of closing date

- **WHEN** an account has a `closing_date` in the past but `disabled` is
  `false`
- **THEN** creating a new entry against it still succeeds — `closing_date`
  alone does not block anything

### Requirement: Soft delete

Deleting an account SHALL set `deleted_at` rather than removing the row —
one-way, with no undelete endpoint, matching the existing
`users`/`invites` soft-delete convention. A soft-deleted account SHALL be
excluded from listings and SHALL behave as not found for reads, updates, and
further deletes.

#### Scenario: Soft-deleted account excluded from listing

- **WHEN** an account has been deleted and its owner calls
  `GET /api/accounts`
- **THEN** that account does not appear in the response

#### Scenario: Soft-deleted account not found on direct access

- **WHEN** its owner calls `GET /api/accounts/{id}` for a soft-deleted
  account
- **THEN** the response is `404`

### Requirement: An account carries an optional icon and colour

An account SHALL carry two optional fields, `icon` and `color`, each a short
opaque string the backend stores and returns but does not interpret. Each
field, when present, SHALL match `^[a-z0-9-]{1,40}$`; a value failing that
shape SHALL be rejected (`400`) and nothing changed. The two fields are
independent: an account MAY have neither, only `icon`, only `color`, or
both. Both SHALL be accepted on `POST /api/accounts` and
`PATCH /api/accounts/{id}`, and SHALL appear on every response that carries
an `Account`. On update, supplying a field as an empty string SHALL clear
it; omitting the field SHALL leave it unchanged. The backend SHALL NOT
validate `icon` against any icon set nor `color` against any palette — their
meaning is entirely a client concern.

#### Scenario: Creating an account with an icon and colour

- **WHEN** an authenticated user calls `POST /api/accounts` with a valid
  `icon` and `color` alongside the required fields
- **THEN** the response is `201` and the created account carries the given
  `icon` and `color` verbatim

#### Scenario: Icon and colour are optional

- **WHEN** an authenticated user calls `POST /api/accounts` with no `icon`
  and no `color`
- **THEN** the response is `201` and the account is created with neither
  field set

#### Scenario: A malformed icon or colour value is rejected

- **WHEN** `POST /api/accounts` or `PATCH /api/accounts/{id}` sends an
  `icon` or `color` containing characters outside `[a-z0-9-]` or longer than
  40 characters
- **THEN** the request is rejected (`400`) and the account is not created or
  changed

#### Scenario: Clearing the icon or colour on update

- **WHEN** the owner calls `PATCH /api/accounts/{id}` with `icon` set to
  `""` (empty string)
- **THEN** the update succeeds and the account's `icon` is thereafter unset,
  while its `color` is unchanged

#### Scenario: An unrelated update leaves icon and colour intact

- **WHEN** the owner calls `PATCH /api/accounts/{id}` changing only `title`,
  with no `icon` or `color` in the body
- **THEN** the account keeps whatever `icon` and `color` it already had

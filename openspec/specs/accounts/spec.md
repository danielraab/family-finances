# accounts Specification

## Purpose

Accounts a user owns for tracking their finances — title,
description, type, currency, financial institute, opening/closing
dates — their ownership/visibility rules, the reversible `disabled`
flag, soft delete, and the per-user, self-managed `account_types` lookup
they are classified by. See `account-entries` for the entries recorded
against an account and its live balance, and `web-client-accounts`
for the client surface.

## Requirements

### Requirement: Account fields and creation

An account SHALL carry: `title` (required, non-empty), `description`
(optional), `type_id` (required, references `account_types`), `currency`
(required, ISO-4217 shape — three uppercase letters, not checked against a
canonical list, validated the same way as `user-settings`'
`default_currency`), `financial_institute` (optional), `opening_date`
(required), and `closing_date` (optional). When `closing_date` is present it
SHALL NOT be before `opening_date`.

#### Scenario: Creating an account with valid fields

- **WHEN** an authenticated user calls `POST /api/accounts` with a title, a
  valid `type_id`, a three-letter currency, and an opening date
- **THEN** the response is `201` with the created account, owned by the
  caller

#### Scenario: Closing date before opening date rejected

- **WHEN** `POST /api/accounts` or an update sets `closing_date` earlier
  than `opening_date`
- **THEN** the request is rejected (`422`) and nothing is changed

#### Scenario: Invalid currency shape rejected

- **WHEN** an account is created or updated with a `currency` that is not
  three uppercase letters
- **THEN** the request is rejected (`422`)

### Requirement: An account has exactly one owner and is visible only to them

Every account SHALL carry a required `owner_id`, set to the authenticated
caller at creation and never changed by this change. Listing, reading,
updating, and deleting an account SHALL be scoped to
`owner_id = <authenticated user>`; an account belonging to a different user
SHALL behave as if it does not exist (`404`), including for an admin.
Sharing an account with another user or any permissions model beyond single
ownership is out of scope for this change.

#### Scenario: Owner sees their own account

- **WHEN** an authenticated user calls `GET /api/accounts/{id}` for an
  account they own
- **THEN** the response is `200` with that account

#### Scenario: Non-owner cannot see the account

- **WHEN** an authenticated user calls `GET /api/accounts/{id}` for an
  account owned by a different user
- **THEN** the response is `404`

#### Scenario: Admin cannot see another user's account either

- **WHEN** a user with `is_admin = true` calls `GET /api/accounts/{id}` for
  an account they do not own
- **THEN** the response is `404` — `is_admin` grants no visibility into
  another user's accounts

### Requirement: Account types are a per-user, self-managed lookup

`account_types` SHALL be private to the user who owns them, via a required
`owner_id` set to the authenticated caller at creation — not a single flat
table shared by the whole instance. `GET /api/account-types` SHALL return
only the authenticated caller's own types. Creating, updating, disabling,
enabling, and deleting an account type SHALL require only authentication —
there is no admin-only gate on any account-type operation. A type belonging
to a different owner SHALL behave as if it does not exist (`404` on direct
access; rejected as an invalid `type_id` when referenced as another user's
account's `type_id`), never `403`. Deleting an account type referenced by
at least one non-deleted account SHALL still be rejected (`409`) rather
than performed or cascaded — unchanged from before, now scoped to the
owner's own accounts.

#### Scenario: An authenticated user lists their own account types

- **WHEN** an authenticated user calls `GET /api/account-types`
- **THEN** the response is `200` with every non-deleted account type they
  own, and none belonging to any other user

#### Scenario: Any authenticated user manages their own account types

- **WHEN** an authenticated user calls `POST`, `PATCH`, `DELETE`, or the
  `/disable`/`/enable` actions on an account type they own
- **THEN** the request is not rejected for lack of admin privileges

#### Scenario: A user cannot see or use another user's account type

- **WHEN** an authenticated user calls an operation on an account type
  owned by a different user, or attempts to set it as `type_id` on their
  own account
- **THEN** the response is `404` for direct access, or the request is
  rejected as an invalid `type_id` for the account operation

#### Scenario: Deleting an in-use account type is rejected

- **WHEN** a user attempts to delete an account type referenced by a
  non-deleted account they own
- **THEN** the response is `409` and the account type is not deleted

### Requirement: A new user starts with a seeded set of default account types

Every new user account SHALL be seeded, at creation, with a fixed starter
set of account types owned by them (Checking, Savings, Cash, Credit Card,
Loan, Investment) — the same set for every user, in English, regardless of
any language preference. A user MAY freely rename, disable, or delete any
seeded type exactly as if they had created it themselves; seeding SHALL NOT
recur or top up a user's set after account creation.

#### Scenario: A freshly created user already has account types to choose from

- **WHEN** a new user account is created (via either sign-in method)
- **THEN** `GET /api/account-types` for that user immediately returns the
  full starter set, before they have created any account type themselves

#### Scenario: Seeded types are ordinary, fully editable types

- **WHEN** a user renames, disables, or deletes a seeded account type
- **THEN** the operation succeeds exactly as it would for a type they
  created themselves

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

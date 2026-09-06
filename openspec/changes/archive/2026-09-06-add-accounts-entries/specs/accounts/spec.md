## ADDED Requirements

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

### Requirement: Account types are an admin-managed, instance-global lookup

`account_types` SHALL be a single flat table shared by the whole instance
(not per-user). `GET /api/account-types` SHALL be available to any
authenticated user. Creating, updating, or deleting an account type SHALL
require `is_admin` (`403` otherwise, `401` unauthenticated) — the same gate
`user-administration` established for admin-only endpoints. Deleting an
account type referenced by at least one non-soft-deleted account SHALL be
rejected (`409`) rather than performed or cascaded.

#### Scenario: Any authenticated user can list account types

- **WHEN** an authenticated non-admin calls `GET /api/account-types`
- **THEN** the response is `200` with every account type

#### Scenario: Non-admin cannot manage account types

- **WHEN** an authenticated non-admin calls `POST`, `PUT`, or `DELETE` on
  `/api/account-types`
- **THEN** the response is `403`

#### Scenario: Deleting an in-use account type is rejected

- **WHEN** an admin attempts to delete an account type referenced by a
  non-deleted account
- **THEN** the response is `409` and the account type is not deleted

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

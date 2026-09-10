## MODIFIED Requirements

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

## ADDED Requirements

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

## REMOVED Requirements

### Requirement: `type_id` remains reassignable only by the real owner

**Reason**: `type` is now a plain text field on the account, not a
reference into a per-real-owner lookup, so there is nothing owner-scoped
about changing it. It becomes an ordinary `owner`-tier-editable field,
changeable by a shared `owner`-tier caller exactly like `title` (see the
`account-sharing` delta).

**Migration**: A `PATCH /api/accounts/{id}` from a shared `owner`-tier
caller that includes `type` now succeeds instead of returning `403`.
Clients that hid or locked the type field for shared owners SHALL stop
doing so.

### Requirement: Account types are a per-user, self-managed lookup

**Reason**: The `account_types` table and its CRUD API
(`GET/POST /api/account-types`, `PATCH/DELETE /api/account-types/{id}`,
`POST /api/account-types/{id}/disable`, `.../enable`) are removed. An
account's type is now a free-text `type` column on `accounts`; there is no
separate entity to own, disable, or protect from deletion.

**Migration**: Migration `0021` adds `accounts.type text NOT NULL`,
backfills it from `account_types.title` via the existing `type_id` foreign
key, drops `accounts.type_id`, and drops the `account_types` table. The
`AccountType` and `AccountTypeWrite` API schemas are removed;
`GET /api/account-types` is repurposed to return a `string[]` of the
caller's in-use type values (see "In-use account types are listed for
autocomplete"). The write endpoints are removed with no replacement — a
type is created implicitly by using it on an account, and a `409`
delete-in-use conflict no longer exists.

### Requirement: A new user starts with a seeded set of default account types

**Reason**: With no `account_types` table there is nothing to seed. The
starter labels (Checking, Savings, Cash, Credit Card, Loan, Investment)
move to a client-side constant that seeds the account form's type
autocomplete suggestions.

**Migration**: The `account.Service.SeedDefaults` new-user hook is removed
from `auth`'s new-user hook wiring. Existing users are unaffected; a new
user simply starts with no accounts and picks or types a `type` when
creating their first account, with the default labels offered as
suggestions by the client.

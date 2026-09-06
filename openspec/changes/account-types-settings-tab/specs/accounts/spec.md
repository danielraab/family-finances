## ADDED Requirements

### Requirement: Account types carry a title, an optional description, and a reversible disabled flag

Each account type SHALL carry `title` (required, non-empty, renamed from the
prior `name` field), an optional `description`, and `disabled` (boolean,
`false` by default). `disabled` SHALL be settable by an admin via
`POST /api/account-types/{id}/disable` and reversed via
`POST /api/account-types/{id}/enable` (`403` for a non-admin, `401`
unauthenticated). Disabling a type SHALL NOT change, hide, or otherwise
affect any account already assigned to it — its only effect is on new type
assignments (see the next requirement).

#### Scenario: Creating a type with a title and description

- **WHEN** an admin calls `POST /api/account-types` with a `title` and a
  `description`
- **THEN** the response is `201` with the created type, `disabled: false`

#### Scenario: Disabling a type does not affect existing accounts

- **WHEN** an admin disables a type that one or more accounts currently
  reference
- **THEN** those accounts are unchanged — still visible, editable, and
  usable for new entries — and the type itself still appears in
  `GET /api/account-types` with `disabled: true`

#### Scenario: Enabling reverses it

- **WHEN** an admin enables a previously disabled type
- **THEN** it becomes assignable again to new or edited accounts

### Requirement: A disabled account type cannot be newly assigned, including by leaving it in place on an edit

`POST /api/accounts` SHALL reject (`422`) a `type_id` that resolves to a
disabled type. `PATCH /api/accounts/{id}` SHALL likewise reject (`422`) any
update whose *effective* `type_id` — the value in the request body if
present, otherwise the account's current `type_id` — resolves to a disabled
type. This means an account whose current type has since been disabled
SHALL reject every `PATCH` (regardless of which other fields it changes)
until that same request also supplies a `type_id` for a different,
non-disabled type. Once such a request succeeds, later edits are governed
by the same rule against the account's new (live) type.

#### Scenario: Creating an account with a disabled type is rejected

- **WHEN** `POST /api/accounts` is called with a `type_id` that resolves to
  a disabled type
- **THEN** the response is `422` and no account is created

#### Scenario: Editing an unrelated field on an account with a disabled type is rejected

- **WHEN** an account's current type is disabled and its owner calls
  `PATCH /api/accounts/{id}` changing only `financial_institute` (no
  `type_id` in the body)
- **THEN** the response is `422` and nothing is changed

#### Scenario: Supplying a live type in the same edit succeeds

- **WHEN** an account's current type is disabled and its owner calls
  `PATCH /api/accounts/{id}` with both `financial_institute` and a `type_id`
  for a different, non-disabled type
- **THEN** the response is `200`, the account now has the new type, and
  `financial_institute` is also updated

#### Scenario: An account keeps working normally until its next edit

- **WHEN** an account's current type is disabled but its owner has not yet
  edited the account
- **THEN** the account is still readable, still shown in listings, and new
  entries can still be created against it

## MODIFIED Requirements

### Requirement: Account types are an admin-managed, instance-global lookup

`account_types` SHALL be a single flat table shared by the whole instance
(not per-user). `GET /api/account-types` SHALL be available to any
authenticated user and SHALL return every type regardless of `disabled`.
Creating, updating, disabling, enabling, or deleting an account type SHALL
require `is_admin` (`403` otherwise, `401` unauthenticated) — the same gate
`user-administration` established for admin-only endpoints. Deleting an
account type referenced by at least one non-deleted account SHALL be
rejected (`409`) rather than performed or cascaded; this is unchanged by
this account type gaining a `disabled` flag — delete stays the one-way,
only-when-unreferenced action, and disable is the reversible alternative
for retiring a type that is still in use.

#### Scenario: Any authenticated user can list account types, including disabled ones

- **WHEN** an authenticated non-admin calls `GET /api/account-types`
- **THEN** the response is `200` with every account type, including any
  with `disabled: true`

#### Scenario: Non-admin cannot manage account types

- **WHEN** an authenticated non-admin calls `POST`, `PATCH`, `DELETE`, or
  the `/disable`/`/enable` actions on `/api/account-types`
- **THEN** the response is `403`

#### Scenario: Deleting an in-use account type is rejected regardless of disabled state

- **WHEN** an admin attempts to delete an account type — disabled or not —
  that is referenced by a non-deleted account
- **THEN** the response is `409` and the account type is not deleted

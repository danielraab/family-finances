## MODIFIED Requirements

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

## ADDED Requirements

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

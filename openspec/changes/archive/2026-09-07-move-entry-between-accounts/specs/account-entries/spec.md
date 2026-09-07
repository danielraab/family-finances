## MODIFIED Requirements

### Requirement: An entry is a transaction or a balance adjustment

Every entry SHALL carry a required, immutable `kind`, one of `transaction`
(a relative amount applied to the account's running balance) or
`balance_adjustment` (an absolute amount the account's balance is set to at
that point in time). `kind` SHALL NOT be changeable after creation.
`account_id` MAY be changed after creation — see "An entry can be moved to a
different account of the same owner".

#### Scenario: Kind is immutable

- **WHEN** an update to an existing entry attempts to change `kind`
- **THEN** the request is rejected (`422`) and the entry is unchanged

## ADDED Requirements

### Requirement: An entry can be moved to a different account of the same owner

`PATCH /api/entries/{id}` SHALL accept `account_id`, changing which account
the entry belongs to. The new `account_id` SHALL be subject to the same
checks `POST /api/entries` applies when creating an entry against an
account: it SHALL reference an account owned by the caller, and SHALL NOT
reference a disabled account. This applies uniformly regardless of the
entry's `kind` — a `balance_adjustment` entry is movable exactly like a
`transaction` entry. No currency conversion or validation is performed: an
entry MAY be moved between accounts with different `currency` values, and
its stored `amount` is left unchanged. Moving an entry does not otherwise
change any of its other fields, and an update that changes `account_id`
alongside other fields is validated the same as any other
`PATCH /api/entries/{id}` call.

#### Scenario: Moving an entry to another of the caller's own accounts

- **WHEN** an authenticated user calls `PATCH /api/entries/{id}` with an
  `account_id` of a different, non-disabled account they own
- **THEN** the response is `200` and the entry's `account_id` is updated;
  the entry no longer counts toward the original account's balance and now
  counts toward the new account's

#### Scenario: Moving an entry to another user's account is rejected

- **WHEN** an authenticated user calls `PATCH /api/entries/{id}` with an
  `account_id` of an account owned by a different user
- **THEN** the request is rejected (`400`) and the entry's `account_id` is
  unchanged

#### Scenario: Moving an entry to a disabled account is rejected

- **WHEN** an authenticated user calls `PATCH /api/entries/{id}` with an
  `account_id` of an account they own that is disabled
- **THEN** the request is rejected (`422`) and the entry's `account_id` is
  unchanged — moving into a disabled account is rejected the same way
  creating a new entry against one is

#### Scenario: A balance adjustment can be moved like a transaction

- **WHEN** an authenticated user calls `PATCH /api/entries/{id}` on a
  `kind: balance_adjustment` entry with a new `account_id` of another
  account they own
- **THEN** the response is `200` and the entry's `account_id` is updated,
  the same as for a `transaction` entry

#### Scenario: Moving between accounts of different currencies is permitted

- **WHEN** an authenticated user calls `PATCH /api/entries/{id}` with an
  `account_id` of an account whose `currency` differs from the entry's
  current account
- **THEN** the response is `200`, the entry's `account_id` is updated, and
  its `amount` is left unchanged — no conversion is applied

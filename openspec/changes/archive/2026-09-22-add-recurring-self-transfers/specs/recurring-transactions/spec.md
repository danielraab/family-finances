## ADDED Requirements

### Requirement: A self-transfer recurring transaction moves money between two accounts

Every recurring transaction SHALL carry a required `kind`, one of
`transaction` (the default applied to every recurring transaction that
existed before this capability) or `self_transfer`. A `self_transfer`
recurring transaction SHALL additionally carry a required `to_account_id`,
the receiving account; every other kind's `to_account_id` SHALL be null.
`to_account_id` SHALL differ from `account_id`, and both accounts SHALL
share the same `currency` — a mismatch is rejected (`400`), the same way
`account-entries` rejects it for a `self_transfer` entry, since
cross-currency transfers are not supported at all. Both `kind` and
`to_account_id` SHALL be immutable after creation: neither is accepted on
`PATCH /api/recurring-transactions/{id}`. `amount` SHALL stay a signed
delta from `account_id`'s perspective, exactly as it is for a
`self_transfer` entry; the receiving side's effective delta is `-amount`.
Every response carrying a `self_transfer` recurring transaction SHALL
include `to_account_name` and `to_account_currency`, resolved server-side
regardless of the caller's own permission on that account — the same
unconditional resolution `account_currency` already follows.

#### Scenario: Creating a self-transfer recurring transaction

- **WHEN** an authenticated user with at least `append` permission on two
  same-currency, non-disabled accounts calls
  `POST /api/recurring-transactions` with `kind: self_transfer`, both
  account ids, an `amount` and a recurrence rule
- **THEN** the response is `201` with `to_account_id` set, and
  `to_account_name`/`to_account_currency` resolved

#### Scenario: to_account_id is required exactly for a self-transfer

- **WHEN** `POST /api/recurring-transactions` is called with
  `kind: self_transfer` and no `to_account_id`, or with
  `kind: transaction` and a `to_account_id`
- **THEN** the request is rejected (`400`) and no recurring transaction is
  created

#### Scenario: The two accounts must differ

- **WHEN** `POST /api/recurring-transactions` is called with
  `kind: self_transfer` and a `to_account_id` equal to `account_id`
- **THEN** the request is rejected (`400`) and no recurring transaction is
  created

#### Scenario: Mismatched currencies are rejected

- **WHEN** `POST /api/recurring-transactions` is called with
  `kind: self_transfer` naming two accounts of different currencies
- **THEN** the request is rejected (`400`) and no recurring transaction is
  created

#### Scenario: kind and to_account_id cannot be changed

- **WHEN** `PATCH /api/recurring-transactions/{id}` supplies a `kind` or a
  `to_account_id`
- **THEN** the request is rejected and the recurring transaction is
  unchanged

#### Scenario: An existing recurring transaction is a transaction

- **WHEN** a recurring transaction created before this capability is read
- **THEN** its `kind` is `transaction` and its `to_account_id` is null

### Requirement: Creating or editing a self-transfer recurring transaction requires permission on both accounts

Creating a `self_transfer` recurring transaction SHALL require at least
`append` permission on both `account_id` and `to_account_id`, neither
disabled — the same check `account-entries` applies to creating a
`self_transfer` entry, applied at template time so a template can never
describe a movement its author could not book. Editing one SHALL require
that the caller still holds `append`+ on both accounts at edit time, in
addition to the tier rules every recurring transaction already has;
a `self_transfer` recurring transaction whose far account has become
inaccessible SHALL be read-only (`403` on any edit, whatever field is being
changed). Deleting SHALL keep the existing rule, evaluated against
`account_id` alone.

#### Scenario: Append permission on only one side is rejected

- **WHEN** a user with `append` permission on `account_id` but only `view`
  permission on `to_account_id` calls `POST /api/recurring-transactions`
  with `kind: self_transfer`
- **THEN** the request is rejected (`422`) and no recurring transaction is
  created

#### Scenario: A disabled receiving account is rejected

- **WHEN** `POST /api/recurring-transactions` is called with
  `kind: self_transfer` and a `to_account_id` naming a disabled account
- **THEN** the request is rejected (`422`) and no recurring transaction is
  created

#### Scenario: Losing permission on the far account makes the template read-only

- **WHEN** the caller's share on a `self_transfer` recurring transaction's
  `to_account_id` is revoked and they then call
  `PATCH /api/recurring-transactions/{id}` changing only its title
- **THEN** the request is rejected (`403`) and the recurring transaction is
  unchanged

#### Scenario: Deleting is unaffected by the far account

- **WHEN** a caller who may delete a recurring transaction on `account_id`
  deletes a `self_transfer` one whose `to_account_id` they have no
  permission on, and no entry is linked to it
- **THEN** the response is `204` and the recurring transaction is soft
  deleted

### Requirement: A self-transfer recurring transaction is previewed from both sides, regardless of the listing flags

`GET /api/recurring-transactions/preview` SHALL include `self_transfer`
recurring transactions unconditionally, and SHALL project each one's
occurrences once per account of its two that is within the caller's
resolved account scope — twice when both are, with the receiving side's
`account_id`/`to_account_id` swapped and `amount` negated. It SHALL NOT
accept the `include_self_transfer` or `self_transfer_both_legs` parameters:
those govern the recurring-transaction listing only. Each preview item
SHALL carry `kind` and, for a `self_transfer`, `to_account_id` and
`to_account_name`.

#### Scenario: A transfer between two visible accounts is previewed twice

- **WHEN** a `self_transfer` recurring transaction's two accounts are both
  within the caller's resolved account scope and
  `GET /api/recurring-transactions/preview` is called
- **THEN** each projected occurrence appears twice, once per account, with
  opposite-signed amounts

#### Scenario: Only the receiving account is in scope

- **WHEN** `GET /api/recurring-transactions/preview` is filtered to the
  receiving account alone
- **THEN** each projected occurrence appears once, with `amount` negated
  and the two account ids swapped

#### Scenario: Preview ignores the listing flags

- **WHEN** `GET /api/recurring-transactions/preview` is called with no
  `include_self_transfer` parameter
- **THEN** `self_transfer` recurring transactions are still previewed

## MODIFIED Requirements

### Requirement: A recurring transaction belongs to exactly one account and is gated by the same permission tiers as entries

Every recurring transaction SHALL carry a required `account_id` and a
required, immutable `created_by` set to the authenticated caller at
creation. A `self_transfer` recurring transaction SHALL additionally carry
a required `to_account_id`, a second parent account (see "A self-transfer
recurring transaction moves money between two accounts"); every other
kind's `to_account_id` is null. Reading a recurring transaction SHALL be
gated by the caller's permission tier on the parent account, and creating,
editing, and deleting by that tier using the exact same rules
`account-entries` applies to entries: `view` may only read; `append` may
create and may edit/delete only recurring transactions it created itself;
`entry_admin` and `owner` may edit/delete any recurring transaction on the
account. For a `self_transfer`, `view`+ on either parent account SHALL be
sufficient to read it, and editing carries the additional both-accounts
requirement stated in "Creating or editing a self-transfer recurring
transaction requires permission on both accounts". A recurring transaction
none of whose parent accounts the caller has any permission on, or whose
only parent account(s) are soft-deleted, SHALL behave as if it does not
exist (`404`).

#### Scenario: Creating a recurring transaction against an account the caller has append+ permission on

- **WHEN** an authenticated user with at least `append` permission on an
  account calls `POST /api/recurring-transactions` with that `account_id`
- **THEN** the response is `201` with the created recurring transaction,
  `created_by` set to the caller

#### Scenario: View-tier caller cannot create

- **WHEN** a user with only `view` permission on an account calls
  `POST /api/recurring-transactions` with that `account_id`
- **THEN** the request is rejected (`422`, `account_id` not usable by the
  caller) and no recurring transaction is created

#### Scenario: Append-tier caller cannot edit another user's recurring transaction

- **WHEN** a user with `append` permission calls
  `PATCH /api/recurring-transactions/{id}` or
  `DELETE /api/recurring-transactions/{id}` on a recurring transaction
  created by a different user on the same account
- **THEN** the request is rejected (`403`) and the recurring transaction is
  unchanged

#### Scenario: Creation rejected against a disabled account

- **WHEN** an authenticated user calls `POST /api/recurring-transactions`
  with the `account_id` of an account they can append to that is disabled
- **THEN** the request is rejected (`422`) and no recurring transaction is
  created

#### Scenario: The receiving account alone is enough to read a transfer

- **WHEN** a user with `view` permission on only the `to_account_id` of a
  `self_transfer` recurring transaction calls
  `GET /api/recurring-transactions/{id}`
- **THEN** the response is `200` with that recurring transaction

### Requirement: A recurring transaction has the same content fields as a transaction entry

Every recurring transaction SHALL carry a required non-empty `title`, an
optional `description`, zero or more `tag_ids` (validated the same way
`account-entries` validates a transaction entry's tags), and a required
signed `amount` at the same fixed 4-decimal-place integer scale
(`account-entries`' `AmountScale`) as an entry's `amount`. Its remaining
content fields follow its `kind`, matching exactly what the entry it
materializes will accept: `category_id` is required for a `transaction`
and optional for a `self_transfer`, validated the same way
`account-entries` validates an entry's category in both cases (must be
usable: owned or `append`-shared, and not disabled); `counterparty` and
`location` are accepted for a `transaction` and rejected for a
`self_transfer`. There is no recurring equivalent of a balance
adjustment — `balance_adjustment` is not a valid `kind` here.

#### Scenario: Creating a recurring transaction with the minimum required fields

- **WHEN** `POST /api/recurring-transactions` is called with `account_id`,
  `title`, `category_id`, `amount`, and a recurrence rule (see below)
- **THEN** the response is `201` and its `kind` is `transaction`

#### Scenario: Missing category is rejected for a transaction

- **WHEN** `POST /api/recurring-transactions` is called with
  `kind: transaction` (or no `kind`) and no `category_id`
- **THEN** the request is rejected (`422`) and no recurring transaction is
  created

#### Scenario: A self-transfer needs no category

- **WHEN** `POST /api/recurring-transactions` is called with
  `kind: self_transfer` and no `category_id`
- **THEN** the response is `201`

#### Scenario: Counterparty and location are rejected on a self-transfer

- **WHEN** `POST /api/recurring-transactions` or
  `PATCH /api/recurring-transactions/{id}` supplies a `counterparty` or
  `location` on a `self_transfer` recurring transaction
- **THEN** the request is rejected (`400`) and no recurring transaction is
  created or changed

#### Scenario: A disabled or unusable category is rejected

- **WHEN** `POST /api/recurring-transactions` or
  `PATCH /api/recurring-transactions/{id}` supplies a `category_id` that is
  disabled, or owned by a different user with no `append` share
- **THEN** the request is rejected (`400`) and no recurring transaction is
  created or changed

#### Scenario: balance_adjustment is not a recurring kind

- **WHEN** `POST /api/recurring-transactions` is called with
  `kind: balance_adjustment`
- **THEN** the request is rejected (`400`) and no recurring transaction is
  created

### Requirement: Listing recurring transactions defaults to every account the caller has any permission on

`GET /api/recurring-transactions` SHALL accept an optional, repeatable
`account_id` filter, defaulting when omitted to every non-deleted account
the caller has any permission on (real ownership or a share of any tier) —
the same default `account-entries`' entry listing uses. An explicitly
supplied `account_id` SHALL be honored only when it names an account the
caller has at least `view` permission on. Each returned recurring
transaction SHALL include `account_currency`, resolved server-side, the
same way `account-entries` resolves it for an `Entry`.

It SHALL additionally accept two boolean parameters governing how
`self_transfer` recurring transactions appear, both defaulting to `false`:

- `include_self_transfer` `false`: `self_transfer` recurring transactions
  are excluded from the response entirely.
- `include_self_transfer` `true`, `self_transfer_both_legs` `false`: each
  `self_transfer` recurring transaction appears at most once, from its
  sending account's side — `account_id` and `amount` as stored — and only
  when that sending account is within the caller's resolved account scope.
- `include_self_transfer` `true`, `self_transfer_both_legs` `true`: each
  `self_transfer` recurring transaction appears once per account of its two
  that is within the caller's resolved account scope, and therefore twice
  when both are. The receiving side's occurrence SHALL have
  `account_id`/`to_account_id` swapped and `amount` negated, so it reads as
  money arriving; the two occurrences share the same `id`, which is
  therefore not unique within a response.

`self_transfer_both_legs` supplied while `include_self_transfer` is `false`
SHALL be ignored. Every other filter applies identically to both
occurrences of a `self_transfer`, since they represent the same underlying
recurring transaction.

#### Scenario: Omitting account_id includes every visible account

- **WHEN** a user who owns one account and has a share on another calls
  `GET /api/recurring-transactions` with no `account_id`
- **THEN** recurring transactions from both accounts are included

#### Scenario: Filtering by an account the caller has no permission on returns nothing

- **WHEN** a user calls `GET /api/recurring-transactions?account_id={id}`
  for an account they have no permission on
- **THEN** the response is `200` with an empty `items` list

#### Scenario: Self-transfers are hidden by default

- **WHEN** a caller with a `self_transfer` recurring transaction calls
  `GET /api/recurring-transactions` with no `include_self_transfer`
  parameter
- **THEN** it is not in the response

#### Scenario: One leg shown

- **WHEN** the caller calls `GET /api/recurring-transactions` with
  `include_self_transfer=true` and both of a transfer's accounts in scope
- **THEN** that recurring transaction appears exactly once, with
  `account_id` and `amount` as stored

#### Scenario: Both legs shown

- **WHEN** the caller calls `GET /api/recurring-transactions` with
  `include_self_transfer=true&self_transfer_both_legs=true` and both of a
  transfer's accounts in scope
- **THEN** that recurring transaction appears twice, with opposite-signed
  amounts and the two account ids swapped on the second

#### Scenario: Both legs requested but only the receiving account in scope

- **WHEN** the caller calls `GET /api/recurring-transactions` with
  `include_self_transfer=true&self_transfer_both_legs=true` and an
  `account_id` filter naming only the receiving account
- **THEN** that recurring transaction appears once, with `amount` negated
  and the two account ids swapped

#### Scenario: Only the receiving account in scope, one leg requested

- **WHEN** the caller calls `GET /api/recurring-transactions` with
  `include_self_transfer=true` (and no `self_transfer_both_legs`) and an
  `account_id` filter naming only the receiving account
- **THEN** that recurring transaction is not in the response

### Requirement: Recurring transaction totals can be summed per currency, excluding ended templates

`GET /api/recurring-transactions/summary` SHALL accept the same
`account_id` filter as `GET /api/recurring-transactions` (omitted meaning
every non-deleted account the caller has any permission on), and the same
`include_self_transfer` and `self_transfer_both_legs` parameters, resolved
to the same three modes — so the summary always sums exactly the set of
occurrences the listing under the same parameters would return. It SHALL
sum each matching, non-deleted recurring transaction occurrence's
`per_year_amount`, excluding any whose `ends_on` is before the current date
(in the caller's resolved timezone, per `user-settings`, the same
resolution `account-entries`' flow-summary already uses), grouped by the
account's currency the same way `GET /api/entries/summary` groups its sum.
The response SHALL be `{ sums, count }`, `sums` a list of
`{ currency, amount }`.

#### Scenario: Ended recurring transaction excluded from the total

- **WHEN** a recurring transaction's `ends_on` is in the past and
  `GET /api/recurring-transactions/summary` is called
- **THEN** that recurring transaction's `per_year_amount` is excluded from
  `sums` and it is not counted in `count`

#### Scenario: Multiple currencies summed separately

- **WHEN** matching recurring transactions belong to accounts of two
  different currencies
- **THEN** `sums` has one entry per currency, never combined

#### Scenario: Self-transfers excluded from the total by default

- **WHEN** `GET /api/recurring-transactions/summary` is called with no
  `include_self_transfer` parameter
- **THEN** no `self_transfer` recurring transaction contributes to `sums`
  or `count`

#### Scenario: Both legs of a transfer cancel

- **WHEN** `GET /api/recurring-transactions/summary` is called with
  `include_self_transfer=true&self_transfer_both_legs=true` and both of a
  transfer's accounts in scope
- **THEN** its two occurrences contribute equal and opposite amounts to
  that currency's sum, and `count` counts both

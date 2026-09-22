# recurring-transactions Specification

## Purpose

Recurring transaction templates recorded against an account: their fields,
recurrence rule, per-year amount and next-suggested-date calculations, and
their relationship to the entries created from or linked to them.

## Requirements

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

### Requirement: A recurring transaction's recurrence rule is an interval unit and count, not a named frequency

Every recurring transaction SHALL carry a required `interval_unit`, one of
`day`, `week`, `month`, or `year`, and a required `interval_count`, a
positive integer meaning "every `interval_count` `interval_unit`s". It
SHALL also carry a required `starts_on` date and an optional `ends_on` date;
when both are set, `ends_on` SHALL NOT be before `starts_on`. There is no
separate "yearly" or "quarterly" frequency value — those are expressed as
`interval_unit: year, interval_count: 1` and `interval_unit: month,
interval_count: 3` respectively.

#### Scenario: Quarterly expressed as every 3 months

- **WHEN** `POST /api/recurring-transactions` is called with
  `interval_unit: month, interval_count: 3`
- **THEN** the response is `201` and no separate "quarterly" value is
  needed or accepted

#### Scenario: interval_count must be positive

- **WHEN** `POST /api/recurring-transactions` or
  `PATCH /api/recurring-transactions/{id}` supplies `interval_count: 0` or a
  negative value
- **THEN** the request is rejected (`422`) and no recurring transaction is
  created or changed

#### Scenario: ends_on before starts_on is rejected

- **WHEN** `POST /api/recurring-transactions` or
  `PATCH /api/recurring-transactions/{id}` supplies an `ends_on` earlier
  than `starts_on`
- **THEN** the request is rejected (`422`) and no recurring transaction is
  created or changed

### Requirement: A recurring transaction's per-year amount is computed server-side from its recurrence rule

Every response carrying a recurring transaction SHALL include a computed
`per_year_amount`, `amount` multiplied by the number of occurrences per
year implied by its recurrence rule:

- `interval_unit: month` → `12 / interval_count` (exact, calendar-based)
- `interval_unit: year` → `1 / interval_count` (exact, calendar-based)
- `interval_unit: week` → `365.25 / (7 * interval_count)`
- `interval_unit: day` → `365.25 / interval_count`

`per_year_amount` SHALL be computed fresh on every read, never stored, and
SHALL be present regardless of whether the recurring transaction has since
ended (see "A recurring transaction can have an end date").

#### Scenario: Monthly per-year amount

- **WHEN** a recurring transaction has `amount: -80000` (i.e. -8.0000),
  `interval_unit: month`, `interval_count: 1`
- **THEN** its `per_year_amount` is `-960000` (12 × -80000)

#### Scenario: Quarterly per-year amount

- **WHEN** a recurring transaction has `amount: 300000`,
  `interval_unit: month`, `interval_count: 3`
- **THEN** its `per_year_amount` is `1200000` (4 × 300000)

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

### Requirement: A recurring transaction can have an end date

A recurring transaction with a non-null `ends_on` at or before the current
date (in the caller's resolved timezone) SHALL be reported as ended: every
response carrying it SHALL include a computed `ended` boolean. Being ended
SHALL NOT block reading, editing, deleting (subject to the linked-entries
rule below), creating a transaction from it, or linking an existing entry
to it — `ends_on` only affects `ended` and exclusion from the per-year sum.

#### Scenario: Past end date marks the template ended

- **WHEN** a recurring transaction's `ends_on` is yesterday (caller's local
  date)
- **THEN** its `ended` is `true`

#### Scenario: Ended template still accepts a manually created transaction

- **WHEN** an ended recurring transaction is used to create a new entry
  (see "Materializing a transaction from a recurring transaction")
- **THEN** the entry is created and linked normally

### Requirement: The next suggested booking date is computed from the last linked entry

Every response carrying a recurring transaction SHALL include a computed
`next_suggested_date`: if the recurring transaction has at least one
non-deleted linked entry (an entry whose `recurring_transaction_id`
references it), `next_suggested_date` is that entry's `booking_timestamp`
with the highest value among them, advanced by exactly one interval of the
recurring transaction's recurrence rule; if it has no linked entries yet,
`next_suggested_date` is `starts_on` itself. Advancing by one interval
SHALL use calendar arithmetic for `month` and `year` units (adding
`interval_count` calendar months/years, clamped to the target month's last
day when the original day doesn't exist there) and fixed day-counts for
`week` (`interval_count * 7` days) and `day` (`interval_count` days) units.
This value is never stored — it is recomputed on every read from the
current set of linked entries.

#### Scenario: No linked entries yet

- **WHEN** a recurring transaction with `starts_on: 2026-10-01` has no
  linked entries
- **THEN** its `next_suggested_date` is `2026-10-01`

#### Scenario: Monthly suggestion advances by one calendar month

- **WHEN** a monthly (`interval_count: 1`) recurring transaction's most
  recently booked linked entry has `booking_timestamp` of `2026-01-31`
- **THEN** its `next_suggested_date` is `2026-02-28` (clamped, since
  February has no 31st)

#### Scenario: Suggestion uses the latest linked entry, not creation order

- **WHEN** a recurring transaction has two linked entries booked
  `2026-01-01` and `2026-03-01`
- **THEN** its `next_suggested_date` is computed by advancing from
  `2026-03-01`, the later of the two, regardless of which entry was linked
  first

### Requirement: A recurring transaction with linked entries cannot be deleted

`DELETE /api/recurring-transactions/{id}` SHALL be rejected (`409`) while
one or more non-deleted entries reference it via `recurring_transaction_id`.
Deleting is a soft delete (`deleted_at`, one-way, no undelete endpoint),
permitted only once no non-deleted entry is linked to it.

#### Scenario: Delete blocked while entries are linked

- **WHEN** `DELETE /api/recurring-transactions/{id}` is called and at least
  one non-deleted entry has that id as its `recurring_transaction_id`
- **THEN** the request is rejected (`409`) and the recurring transaction is
  not deleted

#### Scenario: Delete succeeds once unlinked

- **WHEN** every entry previously linked to a recurring transaction has
  since been unlinked (its `recurring_transaction_id` cleared) or deleted,
  and `DELETE /api/recurring-transactions/{id}` is called
- **THEN** the response is `204` and the recurring transaction is soft
  deleted

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

A category or tag filter SHALL NOT widen the account scope above: unlike
`account-entries`' entry listing, a caller's permission on the filtered
category or tag alone never exposes a recurring transaction on an account
they cannot otherwise see.

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

#### Scenario: A category filter does not reach beyond the caller's accounts

- **WHEN** a user filters by a category they own, and a recurring
  transaction in that category exists on an account they have no
  permission on
- **THEN** that recurring transaction is not returned

### Requirement: Listing and summing recurring transactions can be filtered by category and tag

The recurring-transaction listing and summary endpoints SHALL each accept
an optional `category_id`, an optional `category_mode` (`subtree` |
`exact`, defaulting to `subtree`), and an optional `tag_id` —
`GET /api/recurring-transactions` and `GET /api/recurring-transactions/summary`
— resolved exactly as `GET /api/recurring-transactions/preview` already
resolves the same three parameters and as `account-entries`' entry
listing resolves its own.

`category_id` SHALL match a recurring transaction whose `category_id` is
that category or, unless `category_mode=exact`, any descendant of it
within the caller's own category tree. A recurring transaction with no
category SHALL never match a `category_id` filter. A `category_id` naming
a category the caller holds no permission on SHALL match nothing — never
every recurring transaction in scope. A `category_mode` that is neither
`subtree` nor `exact` SHALL be rejected with `400`.

`tag_id` SHALL match a recurring transaction carrying that tag among its
`tag_ids`.

Both parameters SHALL combine with `account_id`,
`include_self_transfer` and `self_transfer_both_legs` as a conjunction,
and SHALL apply identically to both operations — so the summary's total
remains the total of the rows the listing returns under the same
parameters.

#### Scenario: Filtering by a parent category includes its descendants

- **WHEN** a caller filters the listing by a category that has a child
  category, and recurring transactions exist in both
- **THEN** both are returned

#### Scenario: category_mode=exact excludes descendants

- **WHEN** that same caller adds `category_mode=exact`
- **THEN** only the recurring transaction in the named category itself is
  returned

#### Scenario: An uncategorized recurring transaction never matches a category filter

- **WHEN** a caller filters by any category and a recurring transaction in
  scope has no `category_id`
- **THEN** it is not returned

#### Scenario: A category the caller cannot see matches nothing

- **WHEN** a caller filters by a `category_id` they neither own nor hold a
  share on
- **THEN** the response is `200` with no recurring transactions, not the
  caller's unfiltered list

#### Scenario: An invalid category_mode is rejected

- **WHEN** a caller supplies `category_mode=ancestors`
- **THEN** the response is `400`

#### Scenario: Filtering by a tag

- **WHEN** a caller filters by a `tag_id` carried by one of their
  recurring transactions
- **THEN** only recurring transactions carrying that tag are returned

#### Scenario: Category and tag filters combine

- **WHEN** a caller supplies both `category_id` and `tag_id`
- **THEN** only recurring transactions matching both are returned

#### Scenario: The summary totals exactly the filtered rows

- **WHEN** a caller calls the listing and the summary with the same
  `account_id`, `category_id`, `category_mode`, `tag_id` and
  self-transfer parameters
- **THEN** the summary's per-currency totals are the per-year amounts of
  the non-ended recurring transactions the listing returned

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

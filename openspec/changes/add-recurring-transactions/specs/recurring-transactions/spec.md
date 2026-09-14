## Purpose

Recurring transaction templates recorded against an account: their fields,
recurrence rule, per-year amount and next-suggested-date calculations, and
their relationship to the entries created from or linked to them.

## ADDED Requirements

### Requirement: A recurring transaction belongs to exactly one account and is gated by the same permission tiers as entries

Every recurring transaction SHALL carry a required `account_id` and a
required, immutable `created_by` set to the authenticated caller at
creation. Reading, creating, editing, and deleting a recurring transaction
SHALL be gated by the caller's permission tier on the parent account, using
the exact same tiers and rules `account-entries` applies to entries: `view`
may only read; `append` may create and may edit/delete only recurring
transactions it created itself; `entry_admin` and `owner` may edit/delete
any recurring transaction on the account. A recurring transaction on an
account the caller has no permission on, or whose parent account is
soft-deleted, SHALL behave as if it does not exist (`404`).

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

### Requirement: A recurring transaction has the same content fields as a transaction entry

Every recurring transaction SHALL carry a required non-empty `title`, an
optional `description`, a required `category_id` (validated the same way
`account-entries` validates a transaction entry's category — must be
usable: owned or `append`-shared, and not disabled), optional `counterparty`
and `location`, zero or more `tag_ids` (validated the same way
`account-entries` validates a transaction entry's tags), and a required
signed `amount` at the same fixed 4-decimal-place integer scale
(`account-entries`' `AmountScale`) as an entry's `amount`. A recurring
transaction has no `kind` field — it always represents a `transaction`-kind
entry when materialized; there is no recurring equivalent of a balance
adjustment.

#### Scenario: Creating a recurring transaction with the minimum required fields

- **WHEN** `POST /api/recurring-transactions` is called with `account_id`,
  `title`, `category_id`, `amount`, and a recurrence rule (see below)
- **THEN** the response is `201`

#### Scenario: Missing category is rejected

- **WHEN** `POST /api/recurring-transactions` is called with no
  `category_id`
- **THEN** the request is rejected (`422`) and no recurring transaction is
  created

#### Scenario: A disabled or unusable category is rejected

- **WHEN** `POST /api/recurring-transactions` or
  `PATCH /api/recurring-transactions/{id}` supplies a `category_id` that is
  disabled, or owned by a different user with no `append` share
- **THEN** the request is rejected (`400`) and no recurring transaction is
  created or changed

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
every non-deleted account the caller has any permission on). It SHALL sum
each matching, non-deleted recurring transaction's `per_year_amount`,
excluding any whose `ends_on` is before the current date (in the caller's
resolved timezone, per `user-settings`, the same resolution
`account-entries`' flow-summary already uses), grouped by the account's
currency the same way `GET /api/entries/summary` groups its sum. The
response SHALL be `{ sums, count }`, `sums` a list of `{ currency, amount }`.

#### Scenario: Ended recurring transaction excluded from the total

- **WHEN** a recurring transaction's `ends_on` is in the past and
  `GET /api/recurring-transactions/summary` is called
- **THEN** that recurring transaction's `per_year_amount` is excluded from
  `sums` and it is not counted in `count`

#### Scenario: Multiple currencies summed separately

- **WHEN** matching recurring transactions belong to accounts of two
  different currencies
- **THEN** `sums` has one entry per currency, never combined

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

#### Scenario: Omitting account_id includes every visible account

- **WHEN** a user who owns one account and has a share on another calls
  `GET /api/recurring-transactions` with no `account_id`
- **THEN** recurring transactions from both accounts are included

#### Scenario: Filtering by an account the caller has no permission on returns nothing

- **WHEN** a user calls `GET /api/recurring-transactions?account_id={id}`
  for an account they have no permission on
- **THEN** the response is `200` with an empty `items` list

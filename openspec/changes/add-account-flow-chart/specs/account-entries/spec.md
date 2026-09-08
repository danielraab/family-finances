## MODIFIED Requirements

### Requirement: An entry is a transaction or a balance adjustment

Every entry SHALL carry a required, immutable `kind`, one of `transaction`
(a relative amount applied to the account's running balance) or
`balance_adjustment` (a point in time the account's balance is set to a
known reading). `kind` SHALL NOT be changeable after creation. `account_id`
MAY be changed after creation — see "An entry can be moved to a different
account of the same owner".

For a `transaction`, `amount` SHALL be the signed value supplied by the
caller, unchanged from today. For a `balance_adjustment`, the caller SHALL
supply the absolute reading as `balance`, not `amount`; the entry's `amount`
SHALL instead be computed automatically as the change from the account's
balance immediately before that entry (see "A balance adjustment's amount
is a computed delta from the balance immediately before it"). Supplying
`amount` for a `balance_adjustment`, or `balance` for a `transaction`, on
create or update SHALL be rejected.

#### Scenario: Kind is immutable

- **WHEN** an update to an existing entry attempts to change `kind`
- **THEN** the request is rejected (`422`) and the entry is unchanged

#### Scenario: Creating a balance adjustment supplies balance, not amount

- **WHEN** `POST /api/entries` creates a `kind: balance_adjustment` entry
  with `balance: 105000` and no `amount`
- **THEN** the response is `201`, the entry's `balance` is `105000`, and its
  `amount` is the computed delta from the balance immediately before it

#### Scenario: Supplying amount for a balance adjustment is rejected

- **WHEN** `POST /api/entries` or `PATCH /api/entries/{id}` supplies
  `amount` for an entry whose `kind` is `balance_adjustment`
- **THEN** the request is rejected (`400`) and no entry is created or
  changed

#### Scenario: Supplying balance for a transaction is rejected

- **WHEN** `POST /api/entries` or `PATCH /api/entries/{id}` supplies
  `balance` for an entry whose `kind` is `transaction`
- **THEN** the request is rejected (`400`) and no entry is created or
  changed

### Requirement: Account balance is always computed live

An account's balance at a given point in time SHALL be computed on every
request, never read from a cached or precomputed value, as the sum of every
non-deleted entry's `amount` at or before that point in time. This
reproduces the same result as always resetting to the latest
`balance_adjustment`'s reading and summing only the transactions after it,
because a `balance_adjustment`'s own `amount` is defined (see "A balance
adjustment's amount is a computed delta...") to make the running sum land
exactly on its `balance` reading at that point.

#### Scenario: No balance adjustment yet

- **WHEN** an account has only `transaction` entries and its balance is
  requested
- **THEN** the balance equals the sum of those transactions, computed as if
  starting from `0`

#### Scenario: Balance adjustment sets the baseline

- **WHEN** an account has a `balance_adjustment` with `balance: 10000`
  followed by a `transaction` of `-500`, and the balance is requested as of
  after both
- **THEN** the balance is `9500`

#### Scenario: Balance as of a past point in time ignores later entries

- **WHEN** an account has entries both before and after a given timestamp,
  and the balance is requested as of that timestamp
- **THEN** only entries at or before that timestamp are included

## ADDED Requirements

### Requirement: A balance adjustment's amount is a computed delta from the balance immediately before it

A `balance_adjustment` entry's `amount` SHALL always equal its `balance`
reading minus the account's balance computed strictly before that entry's
`(booking_timestamp, id)` position. It SHALL be recomputed, synchronously
and within the same operation, whenever any entry on the same account
(transaction or balance adjustment) is created, updated in a way that
changes its `amount`, `kind`-relevant timing, or deletion state, or deleted
— specifically, whenever such a change could alter the balance strictly
before some existing balance adjustment. Only the nearest affected balance
adjustment(s) SHALL be recomputed; recomputing one balance adjustment
correctly SHALL make every later entry's already-stored `amount` remain
correct without further changes, up to (but not including) the next
balance adjustment after it.

#### Scenario: A transaction inserted before an existing adjustment shifts its delta

- **WHEN** an account has a `balance_adjustment` with `balance: 20000`, and
  a new `transaction` is created with a `booking_timestamp` before that
  adjustment
- **THEN** the adjustment's `amount` is recomputed so the account's balance
  as of that adjustment is still exactly `20000`

#### Scenario: A later adjustment is unaffected by a change further upstream

- **WHEN** an account has, in order, `AdjustmentA`, some transactions, and
  `AdjustmentB`, and a transaction strictly before `AdjustmentA` is created,
  edited, or deleted
- **THEN** neither `AdjustmentA` nor `AdjustmentB`'s stored `amount` changes,
  because `AdjustmentA` was not between the changed entry and `AdjustmentB`

#### Scenario: Editing a transaction between two adjustments only affects the following one

- **WHEN** an account has, in order, `AdjustmentA`, a `transaction`, and
  `AdjustmentB`, and that transaction's amount is edited
- **THEN** `AdjustmentB`'s `amount` is recomputed to keep its `balance`
  reading exact, and `AdjustmentA`'s `amount` is unchanged

#### Scenario: Deleting an adjustment shifts the next adjustment's baseline

- **WHEN** an account has, in order, `AdjustmentA`, `AdjustmentB`, and
  `AdjustmentC`, and `AdjustmentB` is deleted
- **THEN** `AdjustmentC`'s `amount` is recomputed against `AdjustmentA` as
  its new immediately-preceding adjustment

#### Scenario: Moving an adjustment's booking timestamp recomputes both its old and new neighbors

- **WHEN** a `balance_adjustment`'s `booking_timestamp` is updated to a
  point after another existing adjustment that used to follow it
- **THEN** the adjustment whose position it vacated and the adjustment
  whose position it now precedes or follows both have their `amount`
  values recomputed as needed to keep every adjustment's `balance` reading
  exact

### Requirement: Entries can be summarized as income and outcome per month or per day

`GET /api/entries/flow-summary` SHALL accept `account_id` (repeatable,
omitted meaning every non-deleted account the caller owns, same as
`GET /api/entries`), `unit` (`month` or `day`), `year`, and `month`
(required when `unit` is `day`, rejected when `unit` is `month`). It SHALL
bucket the caller's own, non-deleted accounts' non-deleted entries by
`booking_timestamp` — one bucket per month of `year` when `unit` is
`month`, or one bucket per day of `year`/`month` when `unit` is `day` —
using the caller's resolved timezone setting (`user-settings`) to determine
bucket boundaries. Within each bucket, entries whose `amount` is positive
SHALL be summed into `income`, and the absolute value of entries whose
`amount` is negative SHALL be summed into `outcome`, each grouped per
currency (the currency of the entry's account) the same way
`GET /api/entries/summary` already groups its sum. This applies uniformly
to both `transaction` and `balance_adjustment` entries — unlike
`GET /api/entries/summary`, a `balance_adjustment`'s `amount` (now a
delta) is included. A bucket with no matching entries SHALL still appear,
with empty `income` and `outcome` lists.

#### Scenario: Monthly buckets for a year

- **WHEN** `GET /api/entries/flow-summary?account_id={id}&unit=month&year=2026`
  is called
- **THEN** the response has twelve buckets, one per calendar month of 2026
  in the caller's timezone

#### Scenario: Daily buckets for a month

- **WHEN**
  `GET /api/entries/flow-summary?account_id={id}&unit=day&year=2026&month=3`
  is called
- **THEN** the response has one bucket per day of March 2026 in the
  caller's timezone

#### Scenario: A transaction contributes to income or outcome by its sign

- **WHEN** a bucket's matching entries include a `transaction` with a
  positive `amount` and one with a negative `amount`
- **THEN** the positive one's `amount` is included in that bucket's
  `income`, and the absolute value of the negative one's `amount` is
  included in that bucket's `outcome`

#### Scenario: A balance adjustment's delta contributes like a transaction

- **WHEN** a bucket's matching entries include a `balance_adjustment` whose
  computed `amount` is negative
- **THEN** its absolute value is included in that bucket's `outcome`, the
  same as a negative transaction would be

#### Scenario: An empty bucket has no special case

- **WHEN** a requested month or day has no matching entries
- **THEN** its bucket is still present, with `income: []` and
  `outcome: []`

#### Scenario: unit=day requires month

- **WHEN** `GET /api/entries/flow-summary?unit=day&year=2026` is called
  with no `month`
- **THEN** the request is rejected (`400`)

#### Scenario: unit=month rejects month

- **WHEN** `GET /api/entries/flow-summary?unit=month&year=2026&month=3` is
  called
- **THEN** the request is rejected (`400`)

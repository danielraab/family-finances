# account-entries Specification

## Purpose

Entries (transactions and balance adjustments) recorded against an
account: their fields, immutability rules, category/tag
relationships, live balance computation, and the
filterable/searchable/sortable/cursor-paginated listing. See
`accounts` for the accounts entries belong to, `entry-categories`
for the category tree, `entry-tags` for per-user tags, and
`web-client-entries` for the client surface.

## Requirements

### Requirement: An entry belongs to exactly one account and one owner

Every entry SHALL carry a required `account_id` referencing an
account the caller owns, and a required `owner_id` set to that account's
owner at creation. Listing, reading, updating, and deleting an entry SHALL
be scoped to `owner_id = <authenticated user>` and to the entry's parent
account being non-deleted; an entry that does not satisfy both SHALL behave
as if it does not exist (`404`).

#### Scenario: Creating an entry against the caller's own account

- **WHEN** an authenticated user calls `POST /api/entries` with the
  `account_id` of an account they own
- **THEN** the response is `201` with the created entry, `owner_id` set to
  the caller

#### Scenario: Creating an entry against another user's account rejected

- **WHEN** an authenticated user calls `POST /api/entries` with the
  `account_id` of an account owned by a different user
- **THEN** the request is rejected (`422`, `account_id` not usable by the
  caller) and no entry is created

#### Scenario: Entry on a soft-deleted account is not accessible

- **WHEN** an account has been soft-deleted and its owner calls
  `GET /api/entries?account_id={id}`
- **THEN** the response is `200` with an empty `items` list — the
  account's entries are no longer reachable through it

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

### Requirement: A category is required for a transaction, optional for a balance adjustment

Every entry SHALL reference at most one category. A `transaction` entry
SHALL have a non-null `category_id`. A `balance_adjustment` entry MAY have a
null `category_id`.

#### Scenario: Transaction without a category rejected

- **WHEN** `POST /api/entries` creates a `kind: transaction` entry with no
  `category_id`
- **THEN** the request is rejected (`422`) and no entry is created

#### Scenario: Balance adjustment without a category accepted

- **WHEN** `POST /api/entries` creates a `kind: balance_adjustment` entry
  with no `category_id`
- **THEN** the response is `201` and the entry has a null `category_id`

#### Scenario: Balance adjustment with a category is also accepted

- **WHEN** `POST /api/entries` creates a `kind: balance_adjustment` entry
  that includes a `category_id`
- **THEN** the response is `201` and the entry has that `category_id`

### Requirement: An entry can only be categorized with its owner's own category

Setting `category_id` on an entry (at creation or update) SHALL require the
category to belong to the same owner as the entry — the same rule
`entry-tags` already applies to `tag_id`. Referencing a nonexistent
category id, or a category owned by a different user, SHALL be rejected.

#### Scenario: Categorizing with another user's category rejected

- **WHEN** a user attempts to create or update an entry with a
  `category_id` owned by a different user
- **THEN** the request is rejected (`400`) and the entry is not categorized
  with it

### Requirement: A disabled category cannot be newly set on an entry

Setting `category_id` on an entry — at creation, or on an update that
includes `category_id` in the request — SHALL be rejected when it resolves
to a disabled category. An update that does not include `category_id` SHALL
NOT be rejected on account of the entry's current category having since
become disabled — unlike a disabled account type, a disabled category does
not block unrelated edits to entries that already reference it.

#### Scenario: Creating an entry with a disabled category is rejected

- **WHEN** a user calls `POST /api/entries` with a `category_id` that
  resolves to a disabled category
- **THEN** the request is rejected (`400`) and no entry is created

#### Scenario: Editing an unrelated field does not require reselecting a since-disabled category

- **WHEN** an entry's current category has since been disabled and its
  owner calls `PATCH /api/entries/{id}` changing only an unrelated field
  (no `category_id` in the body)
- **THEN** the update succeeds and the entry keeps its (disabled) category

#### Scenario: Explicitly re-setting a disabled category on update is rejected

- **WHEN** a user calls `PATCH /api/entries/{id}` with a `category_id` that
  resolves to a disabled category
- **THEN** the request is rejected (`400`) and the entry's category is
  unchanged

### Requirement: A disabled tag cannot be newly attached to an entry

Setting `tag_ids` on an entry — at creation, or on an update that includes
`tag_ids` in the request — SHALL be rejected when any id in the request
that is not already among the entry's current tags resolves to a disabled
tag. A tag id already present on the entry before the update SHALL NOT
block that update on account of having since become disabled, even when
the same request resubmits it as part of the full `tag_ids` array —
`tag_ids` is always a full replacement list, so this exemption applies
specifically to ids the update does not actually change the presence of,
not to the request as a whole.

#### Scenario: Creating an entry with a disabled tag is rejected

- **WHEN** a user calls `POST /api/entries` with a `tag_ids` entry that
  resolves to a disabled tag
- **THEN** the request is rejected (`400`) and no entry is created

#### Scenario: Resubmitting an entry's existing, since-disabled tag succeeds

- **WHEN** an entry currently carries a tag that has since been disabled,
  and its owner calls `PATCH /api/entries/{id}` with a `tag_ids` array that
  still includes that same tag id (unchanged) alongside other, unrelated
  changes
- **THEN** the update succeeds and the entry keeps the disabled tag

#### Scenario: Adding a new disabled tag alongside an untouched existing one is rejected

- **WHEN** an entry currently carries tag A (since disabled), and its owner
  calls `PATCH /api/entries/{id}` with `tag_ids` containing both A and a
  different tag B that is also disabled
- **THEN** the request is rejected (`400`) and the entry's tags are
  unchanged

#### Scenario: Editing an unrelated field does not require dropping a since-disabled tag

- **WHEN** an entry's current tags include one that has since been
  disabled, and its owner calls `PATCH /api/entries/{id}` changing only an
  unrelated field (no `tag_ids` in the body)
- **THEN** the update succeeds and the entry keeps every one of its
  existing tags, including the disabled one

### Requirement: An entry has a booking timestamp, title, and optional description

Every entry SHALL carry a required `booking_timestamp` (millisecond
precision), a required non-empty `title`, and an optional `description`. It
MAY reference zero or more tags belonging to the same owner (see
`entry-tags`).

#### Scenario: Creating an entry with the minimum required fields

- **WHEN** `POST /api/entries` is called with `account_id`, `kind`,
  `amount`, `booking_timestamp`, `title`, and (for a transaction)
  `category_id`
- **THEN** the response is `201`

### Requirement: Entries ordered by booking timestamp break ties by insertion order

Wherever entries are ordered by time — listing and balance computation —
they SHALL be ordered by `booking_timestamp` first and, for entries sharing
the exact same millisecond timestamp, by insertion order (the order in
which they were created).

#### Scenario: Same-millisecond entries list in insertion order

- **WHEN** two entries on the same account share an identical
  `booking_timestamp` (to the millisecond) and are listed
- **THEN** they appear in the order they were created, not an unspecified
  order

### Requirement: Amounts are stored as integers at a fixed 4 decimal places

An entry's `amount` SHALL be an integer in the account's currency's minor
units, scaled by a fixed 4 decimal places, uniformly for every account and
currency in the instance. This scale is not configurable. (Per-user
*display* rounding is a separate concern — see `user-settings`'
`displayed_decimal_places`, which affects only how a client renders an
amount, never how it is stored or edited.)

#### Scenario: Amount stored and returned at 4 decimal places

- **WHEN** an entry is created with `amount: 105000`
- **THEN** it represents `10.5000` in the account's currency, and the API
  returns the same integer, `105000`, on every subsequent read

### Requirement: Entry creation is rejected against a disabled account

`POST /api/entries` SHALL reject creating an entry whose `account_id`
names an account with `disabled = true` (see `accounts`). This applies only
to creation — an entry that already existed before its account was
disabled remains fully readable, editable, and deletable.

#### Scenario: Creating an entry against a disabled account is rejected

- **WHEN** an authenticated user calls `POST /api/entries` with the
  `account_id` of an account they own that is disabled
- **THEN** the request is rejected (`422`) and no entry is created

#### Scenario: Existing entries on a newly disabled account are unaffected

- **WHEN** an account with existing entries is disabled
- **THEN** those entries remain listable, editable, and deletable, and
  still count toward the account's balance

### Requirement: Entry listing supports filtering, free-text search, sorting, and cursor-based pagination

`GET /api/entries` SHALL accept, all optional and combinable: `account_id`
(repeatable; omitted means every non-deleted account the caller owns),
`category_id` with an optional `category_mode` (`subtree`, the default —
matches that category and every descendant in the category tree — or
`exact`, matching only that category), `tag_id`, `kind`, `from`/`to` (an
inclusive `booking_timestamp` range), and `q` (a case-insensitive
substring match against `title` or `description`). It SHALL accept `sort`
(`booking_timestamp`, the default, or `amount`) and `dir` (`desc`, the
default, or `asc`). It SHALL accept `after`, an opaque cursor from a
previous response's `next_cursor`, and `limit` (a page size). The response
SHALL be `{ items, next_cursor }`, where `next_cursor` is `null` once no
further matching entries remain. Every filter applies before pagination;
results are always scoped to the caller's own, non-deleted accounts'
non-deleted entries. `category_mode` without `category_id` has no effect.

#### Scenario: Filtering by account

- **WHEN** `GET /api/entries?account_id={id}` is called
- **THEN** only entries on that account are returned

#### Scenario: Filtering by category includes descendants

- **WHEN** `GET /api/entries?category_id={parent}` is called (no
  `category_mode`) and some matching entries carry a child category of
  `{parent}` rather than `{parent}` itself
- **THEN** those entries are included in the results

#### Scenario: Filtering by category with an exact mode excludes descendants

- **WHEN** `GET /api/entries?category_id={parent}&category_mode=exact` is
  called and some matching entries carry a child category of `{parent}`
  rather than `{parent}` itself
- **THEN** those child-category entries are excluded from the results, and
  only entries carrying `{parent}` itself are returned

#### Scenario: Free-text search matches title or description

- **WHEN** `GET /api/entries?q=coffee` is called
- **THEN** only entries whose `title` or `description` contains "coffee"
  (case-insensitive) are returned

#### Scenario: Sorting by amount

- **WHEN** `GET /api/entries?sort=amount&dir=asc` is called
- **THEN** results are ordered from the smallest to the largest `amount`

#### Scenario: Paginating with a cursor

- **WHEN** a first page is fetched and its `next_cursor` is passed back as
  `after` on a second request with the same filters/sort
- **THEN** the second page continues immediately after the first with no
  gap or overlap

#### Scenario: Last page has a null cursor

- **WHEN** a page of results is fetched that reaches the end of the
  matching entries
- **THEN** `next_cursor` is `null`

### Requirement: Entry amounts can be summed per currency without paging through results

`GET /api/entries/summary` SHALL accept the same `account_id`,
`category_id`/`category_mode`, `tag_id`, `from`/`to`, and `q` filters as
`GET /api/entries` (no `sort`, `dir`, `after`, or `limit` — this is an
aggregate, not a page), scoped the same way to the caller's own,
non-deleted accounts' non-deleted entries. It SHALL always additionally
restrict to entries whose `kind` is `transaction`, regardless of whether
`kind` could otherwise be requested — a `balance_adjustment` is an
absolute reading, not a categorized delta, and including it in a sum would
misrepresent the total. The response SHALL be `{ sums, count }`, where
`sums` is a list of `{ currency, amount }` — one entry per distinct
currency (from the currency of each matching entry's account) present
among the matching entries, each `amount` being the sum of those entries'
`amount` values — and `count` is the total number of matching entries
across every currency. An empty result SHALL return `{ sums: [], count: 0
}`, not an error.

#### Scenario: Summing entries in a single currency

- **WHEN** `GET /api/entries/summary?category_id={id}` is called and every
  matching entry belongs to an account in the same currency
- **THEN** the response's `sums` has exactly one entry, for that currency,
  equal to the sum of the matching entries' `amount` values

#### Scenario: Summing entries across multiple currencies

- **WHEN** `GET /api/entries/summary?tag_id={id}` is called and matching
  entries belong to accounts of two different currencies
- **THEN** the response's `sums` has one entry per currency, each the sum
  of only that currency's matching entries — amounts are never added
  across currencies

#### Scenario: Balance adjustments are excluded from the sum

- **WHEN** `GET /api/entries/summary?category_id={id}` is called and some
  entries matching every other filter are `balance_adjustment` entries
- **THEN** those entries are excluded from both `sums` and `count`

#### Scenario: No matching entries

- **WHEN** `GET /api/entries/summary` is called with filters that match no
  entries
- **THEN** the response is `{ sums: [], count: 0 }`

#### Scenario: Exact category mode narrows the sum the same way it narrows the list

- **WHEN** `GET /api/entries/summary?category_id={parent}&category_mode=exact`
  is called
- **THEN** entries carrying a descendant category of `{parent}` are
  excluded from the sum, matching `GET /api/entries` with the same filters

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

### Requirement: A running-balance series can be sampled per day over a month

`GET /api/entries/balance-series` SHALL accept `account_id` (repeatable,
omitted meaning every non-deleted account the caller owns, same as
`GET /api/entries`), `unit` (only `day` is defined), `year`, and `month`
(1-12, required). It SHALL return the running balance of the matching
accounts sampled at each local midnight of `year`/`month` — one point at
`00:00` on each calendar day of that month, plus a closing point at the end
of the last day (equivalently, `00:00` on the first day of the following
month) — using the caller's resolved timezone setting (`user-settings`) to
determine those midnight boundaries.

Each point's value SHALL be the balance computed exactly as
`GET /api/accounts/{id}/balance` computes it as of that instant — the
running sum of `amount` over the account's non-deleted entries booked
strictly before the instant, with a `balance_adjustment` acting as an
anchor (everything before the latest adjustment at or before the instant is
ignored, and the result lands exactly on that adjustment's `balance`
reading). An entry booked exactly at a local midnight therefore belongs to
that day's activity, not the point that opens the day. When several
accounts match, each point SHALL carry one balance per currency (the
currency of the contributing accounts), the same per-currency grouping
shape as `GET /api/entries/flow-summary`; accounts sharing a currency are
summed, and every currency present in the selected accounts appears on
every point, including with an amount of `0`.

Unlike `GET /api/entries` and `GET /api/entries/summary`, this endpoint
SHALL NOT accept `category_id`, `category_mode`, `tag_id`, `from`, `to`, or
`q`. A balance is defined only by the account and the instant; a
`balance_adjustment` carries no category or tag, so honouring such a filter
would drop the anchors the running sum depends on and yield a number that
is not a balance.

#### Scenario: One point per day plus a closing point

- **WHEN** `GET /api/entries/balance-series?account_id={id}&unit=day&year=2026&month=3`
  is called
- **THEN** the response has 32 points — one at `00:00` on each of the 31
  days of March 2026 in the caller's timezone, and a final point at `00:00`
  on 1 April 2026

#### Scenario: The first point is the balance of all prior history

- **WHEN** an account has entries both before and during March 2026, and
  the balance series for March 2026 is requested
- **THEN** the first point's value equals the account's balance as of
  `00:00` on 1 March 2026 — reflecting every non-deleted entry booked
  before that instant and nothing booked on or after it

#### Scenario: A transaction moves the line the following day

- **WHEN** an account holds `100000` at `00:00` on 10 March 2026 and a
  single `transaction` of `-2500` is booked later on 10 March 2026
- **THEN** the point for 10 March is `100000` and the point for 11 March is
  `97500`

#### Scenario: A mid-month balance adjustment re-anchors the line

- **WHEN** a `balance_adjustment` with `balance: 50000` is booked on
  15 March 2026 for an account whose computed balance just before it was
  `48000`
- **THEN** every point from 16 March onward reflects `50000` plus any later
  entries, regardless of the balance on 14 March

#### Scenario: Timezone determines the day boundaries

- **WHEN** the caller's resolved timezone is `Europe/Vienna` and an entry
  is booked at `2026-03-09T23:30:00Z` (00:30 on 10 March local)
- **THEN** the entry's local day is 10 March, so it is reflected from the
  11 March point onward — not from the 10 March point, where a UTC reading
  of the same timestamp would place it

#### Scenario: Multiple accounts of different currencies

- **WHEN** the caller owns a EUR account and a USD account and requests the
  balance series with no `account_id`
- **THEN** each point carries a separate EUR balance and USD balance, and
  never a single combined figure

#### Scenario: A month with no entries is a flat line

- **WHEN** an account has no entries booked during or after the requested
  month
- **THEN** every point has the same value — the account's balance as of the
  month's first midnight

#### Scenario: Filtering parameters are rejected

- **WHEN** `GET /api/entries/balance-series?unit=day&year=2026&month=3&tag_id={tid}`
  is called
- **THEN** the request is rejected (`400`)

#### Scenario: month is required

- **WHEN** `GET /api/entries/balance-series?unit=day&year=2026` is called
  with no `month`
- **THEN** the request is rejected (`400`)

#### Scenario: A soft-deleted entry does not affect the series

- **WHEN** an entry that would otherwise move the balance during the
  requested month has been soft-deleted
- **THEN** the series is identical to one computed with that entry absent

### Requirement: Soft delete

Deleting an entry SHALL set `deleted_at` rather than removing the row —
one-way, matching the existing soft-delete convention, with no undelete
endpoint. A soft-deleted entry SHALL be excluded from listings and from
balance computation.

#### Scenario: Soft-deleted entry excluded from listing and balance

- **WHEN** an entry has been deleted
- **THEN** it does not appear in `GET /api/entries` and no longer
  contributes to `GET /api/accounts/{id}/balance`

## MODIFIED Requirements

### Requirement: An entry belongs to exactly one account and records who created it

Every entry SHALL carry a required `account_id`, and a required, immutable
`created_by` set to the authenticated caller at creation — the user who
logged it, which is not necessarily the account's real owner once
`account-sharing` is in effect. A `self_transfer` entry additionally carries
a required `to_account_id`, a second parent account (see "A self-transfer
entry moves money between two accounts"); every other kind's `to_account_id`
is null. Reading an entry (directly, in a listing, or in balance/summary
computation) SHALL be permitted for any caller who holds at least `view`
permission on the entry's parent account — for a `self_transfer`, either
parent account SHALL be sufficient. An entry none of whose parent accounts
the caller has any permission on, or whose only parent account(s) are
soft-deleted, SHALL behave as if it does not exist (`404`).

#### Scenario: Creating an entry against an account the caller has append+ permission on

- **WHEN** an authenticated user with at least `append` permission on an
  account (real ownership, or a share) calls `POST /api/entries` with that
  `account_id`
- **THEN** the response is `201` with the created entry, `created_by` set
  to the caller

#### Scenario: Creating an entry against an account with only view permission is rejected

- **WHEN** a user with only `view` permission on an account calls
  `POST /api/entries` with that `account_id`
- **THEN** the request is rejected (`422`, `account_id` not usable by the
  caller) and no entry is created

#### Scenario: A shared user's entry is created_by them, not the account's real owner

- **WHEN** a user with `append` permission (via a share, not real
  ownership) creates an entry on the account
- **THEN** the entry's `created_by` is that user, not the account's real
  owner

#### Scenario: A view-tier user reads entries they did not create

- **WHEN** a user with `view` permission on a shared account calls
  `GET /api/entries?account_id={id}` and some matching entries were
  created by a different user
- **THEN** those entries are included in the response

#### Scenario: Entry on an account with no permission is not accessible

- **WHEN** an authenticated user with no permission on an account calls
  `GET /api/entries?account_id={id}`
- **THEN** the response is `200` with an empty `items` list

#### Scenario: A self-transfer is visible via either of its two accounts

- **WHEN** a caller has permission on only one of a `self_transfer` entry's
  `account_id`/`to_account_id` pair and requests that account's entries
- **THEN** the entry is included in the response

### Requirement: An entry is a transaction, a balance adjustment, or a self-transfer

Every entry SHALL carry a required, immutable `kind`, one of `transaction`
(a relative amount applied to the account's running balance),
`balance_adjustment` (a point in time the account's balance is set to a
known reading), or `self_transfer` (a relative amount moved from one
account to another — see "A self-transfer entry moves money between two
accounts"). `kind` SHALL NOT be changeable after creation. `account_id` MAY
be changed after creation for a `transaction` or `balance_adjustment` — see
"An entry can be moved to a different account of the same owner" — but not
for a `self_transfer`, whose two accounts are set at creation and never
individually reassigned thereafter.

For a `transaction` or `self_transfer`, `amount` SHALL be the signed value
supplied by the caller, unchanged from today; for a `self_transfer`,
`amount` is signed from `account_id`'s perspective (negative leaves
`account_id` and arrives at `to_account_id` as the positive equivalent).
For a `balance_adjustment`, the caller SHALL supply the absolute reading as
`balance`, not `amount`; the entry's `amount` SHALL instead be computed
automatically as the change from the account's balance immediately before
that entry. Supplying `amount` for a `balance_adjustment`, `balance` for a
`transaction` or `self_transfer`, or `to_account_id` for a `transaction` or
`balance_adjustment`, on create or update SHALL be rejected.

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

#### Scenario: Supplying to_account_id for a transaction is rejected

- **WHEN** `POST /api/entries` supplies `to_account_id` for a `kind:
  transaction` or `kind: balance_adjustment` entry
- **THEN** the request is rejected (`400`) and no entry is created

#### Scenario: A self-transfer's amount is signed from the sending account's perspective

- **WHEN** a `kind: self_transfer` entry is created with `account_id: A`,
  `to_account_id: B`, and `amount: -10000`
- **THEN** the response is `201`, `A`'s balance reflects `-10000`, and `B`'s
  balance reflects `+10000`

### Requirement: A category is required for a transaction, optional for a balance adjustment or self-transfer

Every entry SHALL reference at most one category. A `transaction` entry
SHALL have a non-null `category_id`. A `balance_adjustment` or
`self_transfer` entry MAY have a null `category_id`.

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

#### Scenario: Self-transfer without a category accepted

- **WHEN** `POST /api/entries` creates a `kind: self_transfer` entry with no
  `category_id`
- **THEN** the response is `201` and the entry has a null `category_id`

### Requirement: Entry creation is rejected against a disabled account

`POST /api/entries` SHALL reject creating an entry whose `account_id`
names an account with `disabled = true` (see `accounts`); for a
`self_transfer`, this SHALL be checked for `to_account_id` as well. This
applies only to creation — an entry that already existed before one of its
accounts was disabled remains fully readable, and remains editable/
deletable per its kind's own permission rule.

#### Scenario: Creating an entry against a disabled account is rejected

- **WHEN** an authenticated user calls `POST /api/entries` with the
  `account_id` of an account they own that is disabled
- **THEN** the request is rejected (`422`) and no entry is created

#### Scenario: Existing entries on a newly disabled account are unaffected

- **WHEN** an account with existing entries is disabled
- **THEN** those entries remain listable, editable, and deletable, and
  still count toward the account's balance

#### Scenario: Creating a self-transfer to a disabled account is rejected

- **WHEN** an authenticated user calls `POST /api/entries` with `kind:
  self_transfer` and a `to_account_id` that is disabled
- **THEN** the request is rejected (`422`) and no entry is created

### Requirement: Entry listing supports filtering, free-text search, sorting, and cursor-based pagination

`GET /api/entries` SHALL accept, all optional and combinable: `account_id`
(repeatable; omitted means every non-deleted account the caller owns),
`category_id` with an optional `category_mode` (`subtree`, the default —
matches that category and every descendant in the category tree — or
`exact`, matching only that category), `tag_id`, `kind`, `from`/`to` (an
inclusive `booking_timestamp` range), and `q` (a case-insensitive
substring match against `title`, `description`, or `counterparty`). It
SHALL accept `sort` (`booking_timestamp`, the default, or `amount`) and
`dir` (`desc`, the default, or `asc`). It SHALL accept `after`, an opaque
cursor from a previous response's `next_cursor`, and `limit` (a page
size). The response SHALL be `{ items, next_cursor }`, where `next_cursor`
is `null` once no further matching entries remain. Every filter applies
before pagination; results are always scoped to the caller's own,
non-deleted accounts' non-deleted entries. `category_mode` without
`category_id` has no effect.

A `self_transfer` entry SHALL appear once per resolved account (from the
combination of any explicit `account_id` filter and the caller's own
visible-accounts scoping) it touches: once, amount as stored, when
`account_id` alone is in scope; once, amount sign flipped, when
`to_account_id` alone is in scope; and twice — both of the above — when
both accounts are in scope at once (for example, an unfiltered listing
covering every account the caller can see, or an explicit `account_id`
filter naming both). Every other filter (`category_id`, `tag_id`, `kind`,
`from`/`to`, `q`) applies identically to both occurrences, since they
represent the same underlying entry.

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
- **THEN** only entries whose `title`, `description`, or `counterparty`
  contains "coffee" (case-insensitive) are returned

#### Scenario: Free-text search matches counterparty alone

- **WHEN** `GET /api/entries?q=rewe` is called and a matching entry's
  `counterparty` is `"Rewe"` while its `title` and `description` contain
  neither "rewe" nor any substring of it
- **THEN** that entry is included in the results

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

#### Scenario: A self-transfer between two accounts in scope is listed twice

- **WHEN** `GET /api/entries` (no `account_id` filter) is called by a
  caller who can see both accounts of a `self_transfer` entry
- **THEN** the response's `items` includes that entry twice — once with
  its stored, negative-from-the-sender amount, once with the sign flipped
  for the receiving account

#### Scenario: A self-transfer with only one account in scope is listed once

- **WHEN** `GET /api/entries?account_id={id}` is called naming only one
  side of a `self_transfer` entry
- **THEN** the response's `items` includes that entry once, with the
  amount signed for the named account

### Requirement: Entry amounts can be summed per currency without paging through results

`GET /api/entries/summary` SHALL accept the same `account_id`,
`category_id`/`category_mode`, `tag_id`, `from`/`to`, and `q` filters as
`GET /api/entries` (no `sort`, `dir`, `after`, or `limit` — this is an
aggregate, not a page), scoped the same way to the caller's own,
non-deleted accounts' non-deleted entries. It SHALL always additionally
restrict to entries whose `kind` is `transaction` or `self_transfer` —
excluding `balance_adjustment`, since an absolute reading is not a
categorized delta and including it in a sum would misrepresent the total.
A `self_transfer` in scope of both its accounts contributes to the sum
twice, once per account, mirroring how `GET /api/entries` lists it twice
in the same circumstance. The response SHALL be `{ sums, count }`, where
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

#### Scenario: A self-transfer between two same-currency accounts nets to zero

- **WHEN** `GET /api/entries/summary` (no `account_id` filter) is called
  by a caller who can see both accounts of a `self_transfer` entry, and
  those accounts share a currency
- **THEN** that entry's two contributions to `sums` (one per account) sum
  to zero for that currency

### Requirement: Account balance is always computed live

An account's balance at a given point in time SHALL be computed on every
request, never read from a cached or precomputed value, as the sum of every
non-deleted entry's `amount` at or before that point in time — for an
entry whose `account_id` is this account, `amount` as stored; for a
`self_transfer` entry whose `to_account_id` is this account, `amount`
negated. This reproduces the same result as always resetting to the latest
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

#### Scenario: A self-transfer increases the receiving account's balance

- **WHEN** account `B` receives a `self_transfer` of `amount: -10000` from
  account `A` (i.e. `account_id: A`, `to_account_id: B`)
- **THEN** `B`'s balance, requested as of at or after the entry, includes
  `+10000` from it

### Requirement: Entries can be summarized as income and outcome per month or per day

`GET /api/entries/flow-summary` SHALL accept `account_id` (repeatable,
omitted meaning every non-deleted account the caller owns, same as
`GET /api/entries`), `category_id`, `category_mode` (`exact`, only
meaningful with `category_id`), `tag_id`, `unit` (`month` or `day`),
`year`, and `month` (required when `unit` is `day`, rejected when `unit`
is `month`) — the same `category_id`/`category_mode`/`tag_id` filters
`GET /api/entries/summary` already accepts, resolved the same way
(`category_id` alone includes the category's subtree; paired with
`category_mode=exact` it matches that category only). It SHALL bucket the
caller's own, non-deleted accounts' non-deleted entries by
`booking_timestamp` — one bucket per month of `year` when `unit` is
`month`, or one bucket per day of `year`/`month` when `unit` is `day` —
using the caller's resolved timezone setting (`user-settings`) to
determine bucket boundaries, after applying any given `category_id`/
`tag_id` filter. Within each bucket, entries whose `amount` is positive
SHALL be summed into `income`, and the absolute value of entries whose
`amount` is negative SHALL be summed into `outcome`, each grouped per
currency (the currency of the entry's account) the same way
`GET /api/entries/summary` already groups its sum. This applies uniformly
to `transaction`, `balance_adjustment`, and `self_transfer` entries —
unlike `GET /api/entries/summary`, a `balance_adjustment`'s `amount` (now a
delta) is included. A `self_transfer` in scope of both its accounts
contributes to each account's own bucket independently — as stored for
`account_id`, sign flipped for `to_account_id` — mirroring how
`GET /api/entries` and `GET /api/entries/summary` both treat it. A bucket
with no matching entries SHALL still appear, with empty `income` and
`outcome` lists.

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

#### Scenario: Filtering by category restricts the buckets to that category's entries

- **WHEN** `GET /api/entries/flow-summary?category_id={id}&unit=month&year=2026`
  is called
- **THEN** each bucket's `income`/`outcome` reflects only entries
  categorized under that category or one of its descendants

#### Scenario: category_mode=exact excludes subcategories

- **WHEN**
  `GET /api/entries/flow-summary?category_id={parent}&category_mode=exact&unit=month&year=2026`
  is called
- **THEN** entries categorized under a child of `{parent}` are excluded
  from every bucket

#### Scenario: Filtering by tag restricts the buckets to that tag's entries

- **WHEN** `GET /api/entries/flow-summary?tag_id={id}&unit=month&year=2026`
  is called
- **THEN** each bucket's `income`/`outcome` reflects only entries carrying
  that tag

#### Scenario: Category and tag filters combine with account filters

- **WHEN** `GET /api/entries/flow-summary` is called with `account_id`,
  `category_id`, and `tag_id` together
- **THEN** each bucket reflects only entries matching all three filters at
  once, the same intersection semantics `GET /api/entries/summary` already
  applies

#### Scenario: A self-transfer contributes to both accounts' buckets

- **WHEN** a `self_transfer` moves money from account `A` to account `B`,
  both within the caller's resolved account scope
- **THEN** `A`'s bucket includes the outgoing amount in `outcome` and `B`'s
  bucket includes the incoming amount in `income`, for the same period

## ADDED Requirements

### Requirement: A self-transfer entry moves money between two accounts the caller can write to

`POST /api/entries` with `kind: self_transfer` SHALL require `account_id`
(the sending account) and `to_account_id` (the receiving account), SHALL
require the caller to hold at least `append` permission on both, and SHALL
require both accounts to share the same `currency` — a mismatch SHALL be
rejected (`400`) the same way an unusable `account_id` is rejected for any
other kind. `account_id` and `to_account_id` SHALL differ; a self-transfer
naming the same account on both sides SHALL be rejected (`400`). Every response
carrying a `self_transfer` entry SHALL include `to_account_id`'s resolved
name and currency (`to_account_name`, `to_account_currency`), resolved
server-side regardless of the caller's own permission on `to_account_id` —
the same reasoning `account_id`'s own resolved `account_currency` already
follows — so a caller who can only see the sending side still sees where
the money went and in what currency.

#### Scenario: Creating a self-transfer with append+ on both accounts

- **WHEN** a caller with `append`+ permission on both `account_id: A` and
  `to_account_id: B`, sharing the same currency, calls `POST /api/entries`
  with `kind: self_transfer`
- **THEN** the response is `201` with the created entry

#### Scenario: Creating a self-transfer without permission on the receiving account is rejected

- **WHEN** a caller with `append`+ permission on `account_id` but no
  permission at all on `to_account_id` calls `POST /api/entries` with
  `kind: self_transfer`
- **THEN** the request is rejected (`400`) and no entry is created

#### Scenario: Creating a self-transfer with only view permission on either account is rejected

- **WHEN** a caller has only `view` permission on `account_id` or on
  `to_account_id` and calls `POST /api/entries` with `kind: self_transfer`
  naming it
- **THEN** the request is rejected (`400`) and no entry is created

#### Scenario: Cross-currency self-transfer is rejected

- **WHEN** a caller calls `POST /api/entries` with `kind: self_transfer`
  naming two accounts of different `currency`
- **THEN** the request is rejected (`400`) and no entry is created

#### Scenario: Self-transfer to the same account is rejected

- **WHEN** a caller calls `POST /api/entries` with `kind: self_transfer`,
  `account_id` and `to_account_id` naming the same account
- **THEN** the request is rejected (`400`) and no entry is created

#### Scenario: The receiving account's name and currency are always resolved

- **WHEN** a caller with permission only on a `self_transfer`'s
  `account_id` (none on `to_account_id`) fetches that entry
- **THEN** the response includes `to_account_name` and `to_account_currency`

### Requirement: Editing a self-transfer requires current permission on both accounts; deleting requires it on either

`PATCH /api/entries/{id}` on a `self_transfer` entry SHALL require the
caller to currently hold at least `append` permission on both `account_id`
and `to_account_id` — stricter than a `transaction`'s edit rule, since any
edit to a self-transfer's shared amount necessarily moves balance on both
accounts at once. A caller who has lost permission on either side (a
revoked share, or the account since disabled or soft-deleted) SHALL be
unable to edit the entry at all (`403`), even for fields unrelated to the
amount or accounts, and even if they still hold `entry_admin`/`owner` on
the side they do have access to. `DELETE /api/entries/{id}` on a
`self_transfer` SHALL instead use the same rule as any other entry —
`entry_admin`/`owner` may delete it, or `append` may delete it if
`created_by` matches them — evaluated against whichever of the two
accounts the caller currently holds permission on; it SHALL NOT require
permission on the other account.

#### Scenario: Editing a self-transfer with append+ on both accounts

- **WHEN** a caller with `append`+ permission on both of a `self_transfer`
  entry's accounts calls `PATCH /api/entries/{id}`
- **THEN** the update succeeds

#### Scenario: Editing a self-transfer after losing access to one side is rejected

- **WHEN** a caller's share on one of a `self_transfer` entry's two
  accounts is revoked, and they then call `PATCH /api/entries/{id}` on it
  (including a change unrelated to `amount` or either account id)
- **THEN** the request is rejected (`403`) and the entry is unchanged

#### Scenario: Deleting a self-transfer needs only the accessible side's permission

- **WHEN** a caller holds `entry_admin`/`owner` permission on one of a
  `self_transfer` entry's two accounts, and no permission at all on the
  other (revoked or never granted)
- **THEN** `DELETE /api/entries/{id}` succeeds

#### Scenario: Deleting a self-transfer still respects the append-created-by rule

- **WHEN** a caller with only `append` permission on their accessible side
  of a `self_transfer` did not create it
- **THEN** `DELETE /api/entries/{id}` is rejected (`403`)

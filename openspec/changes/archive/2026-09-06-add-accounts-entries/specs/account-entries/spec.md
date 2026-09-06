## ADDED Requirements

### Requirement: An entry belongs to exactly one account and one owner

Every entry SHALL carry a required, immutable `account_id` referencing an
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
`balance_adjustment` (an absolute amount the account's balance is set to at
that point in time). `account_id` and `kind` SHALL NOT be changeable after
creation.

#### Scenario: Kind is immutable

- **WHEN** an update to an existing entry attempts to change `kind` or
  `account_id`
- **THEN** the request is rejected (`422`) and the entry is unchanged

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
`category_id` (matches that category and every descendant in the category
tree), `tag_id`, `kind`, `from`/`to` (an inclusive `booking_timestamp`
range), and `q` (a case-insensitive substring match against `title` or
`description`). It SHALL accept `sort` (`booking_timestamp`, the default,
or `amount`) and `dir` (`desc`, the default, or `asc`). It SHALL accept
`after`, an opaque cursor from a previous response's `next_cursor`, and
`limit` (a page size). The response SHALL be `{ items, next_cursor }`,
where `next_cursor` is `null` once no further matching entries remain.
Every filter applies before pagination; results are always scoped to the
caller's own, non-deleted accounts' non-deleted entries.

#### Scenario: Filtering by account

- **WHEN** `GET /api/entries?account_id={id}` is called
- **THEN** only entries on that account are returned

#### Scenario: Filtering by category includes descendants

- **WHEN** `GET /api/entries?category_id={parent}` is called and some
  matching entries carry a child category of `{parent}` rather than
  `{parent}` itself
- **THEN** those entries are included in the results

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

### Requirement: Account balance is always computed live

An account's balance at a given point in time SHALL be computed on every
request, never read from a cached or precomputed value. It SHALL equal the
amount of the latest non-deleted `balance_adjustment` entry at or before
that point in time (ordered per the tie-break rule above), or `0` if no such
adjustment exists, plus the sum of every non-deleted `transaction` entry
after that adjustment (or from the beginning, if none exists) up to that
point in time.

#### Scenario: No balance adjustment yet

- **WHEN** an account has only `transaction` entries and its balance is
  requested
- **THEN** the balance equals the sum of those transactions, computed as if
  starting from `0`

#### Scenario: Balance adjustment sets the baseline

- **WHEN** an account has a `balance_adjustment` of `10000` followed by a
  `transaction` of `-500`, and the balance is requested as of after both
- **THEN** the balance is `9500`

#### Scenario: Balance as of a past point in time ignores later entries

- **WHEN** an account has entries both before and after a given timestamp,
  and the balance is requested as of that timestamp
- **THEN** only entries at or before that timestamp are included

### Requirement: Soft delete

Deleting an entry SHALL set `deleted_at` rather than removing the row —
one-way, matching the existing soft-delete convention, with no undelete
endpoint. A soft-deleted entry SHALL be excluded from listings and from
balance computation.

#### Scenario: Soft-deleted entry excluded from listing and balance

- **WHEN** an entry has been deleted
- **THEN** it does not appear in `GET /api/entries` and no longer
  contributes to `GET /api/accounts/{id}/balance`

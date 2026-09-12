## MODIFIED Requirements

### Requirement: An entry has a booking timestamp, title, and optional description

Every entry SHALL carry a required `booking_timestamp` (millisecond
precision), a required non-empty `title`, and an optional `description`. It
MAY reference zero or more tags belonging to the same owner (see
`entry-tags`). A transaction SHALL additionally accept two optional,
free-text fields: `counterparty` (the other party in the transaction — who
was paid, or who paid — regardless of the amount's sign) and `location`
(a free-text value the backend never interprets or validates beyond
allowing it to be empty — it may hold a typed address or a JSON-encoded
coordinate string; see `web-client-entries` for how a client renders
either). Both `counterparty` and `location` SHALL be rejected (`400`) on a
`balance_adjustment`, the same kind-gating `category_id` already has.

#### Scenario: Creating an entry with the minimum required fields

- **WHEN** `POST /api/entries` is called with `account_id`, `kind`,
  `amount`, `booking_timestamp`, `title`, and (for a transaction)
  `category_id`
- **THEN** the response is `201`

#### Scenario: Creating a transaction with a counterparty and location

- **WHEN** `POST /api/entries` is called with `kind: transaction` and both
  `counterparty` and `location` set
- **THEN** the response is `201`, carrying both values unchanged

#### Scenario: A balance adjustment rejects counterparty and location

- **WHEN** `POST /api/entries` is called with `kind: balance_adjustment`
  and either `counterparty` or `location` set
- **THEN** the response is `400` and no entry is created

#### Scenario: Counterparty and location can be cleared on update

- **WHEN** `PATCH /api/entries/{id}` is called on a transaction with
  `counterparty: ""` and/or `location: ""`
- **THEN** the response is `200` and the corresponding field(s) are empty
  on that entry going forward

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

## ADDED Requirements

### Requirement: The caller's distinct in-use counterparty values are listable for autocomplete

`GET /api/entries/counterparties` SHALL return a JSON array of strings:
the distinct, non-empty `counterparty` values present on the authenticated
caller's own non-deleted entries, compared verbatim (case-sensitively),
sorted case-insensitively ascending — structurally identical to
`GET /api/account-types`. It SHALL never include values from another
user's entries. The endpoint is read-only — there is no way to create,
rename, or delete a counterparty value independent of writing it onto an
entry.

#### Scenario: The caller's distinct in-use counterparties are returned

- **WHEN** an authenticated user with entries whose counterparties are
  `Rewe`, `Employer GmbH`, and a second `Rewe` calls
  `GET /api/entries/counterparties`
- **THEN** the response is `200` with `["Employer GmbH", "Rewe"]`

#### Scenario: Another user's counterparties are not included

- **WHEN** an authenticated user whose own entries all have counterparty
  `Rewe` calls `GET /api/entries/counterparties`, while a different user
  has an entry with counterparty `Spar`
- **THEN** the response contains `Rewe` and does not contain `Spar`

#### Scenario: A user with no counterparty values gets an empty list

- **WHEN** an authenticated user with no entries carrying a non-empty
  `counterparty` calls `GET /api/entries/counterparties`
- **THEN** the response is `200` with `[]`

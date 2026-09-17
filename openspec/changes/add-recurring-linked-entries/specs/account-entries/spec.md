## MODIFIED Requirements

### Requirement: Entry listing supports filtering, free-text search, sorting, and cursor-based pagination

`GET /api/entries` SHALL accept, all optional and combinable: `account_id`
(repeatable; omitted means every non-deleted account the caller owns),
`category_id` with an optional `category_mode` (`subtree`, the default —
matches that category and every descendant in the category tree — or
`exact`, matching only that category), `tag_id`, `kind`,
`recurring_transaction_id`, `from`/`to` (an inclusive `booking_timestamp`
range), and `q` (a case-insensitive substring match against `title`,
`description`, or `counterparty`). It SHALL accept `sort`
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

#### Scenario: Filtering by recurring transaction

- **WHEN** `GET /api/entries?recurring_transaction_id={id}` is called
- **THEN** only entries whose `recurring_transaction_id` equals `{id}` are
  returned, still scoped to the caller's own visible accounts

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

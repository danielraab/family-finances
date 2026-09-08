## MODIFIED Requirements

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

## ADDED Requirements

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

## MODIFIED Requirements

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

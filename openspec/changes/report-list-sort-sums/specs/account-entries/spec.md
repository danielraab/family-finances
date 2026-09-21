## MODIFIED Requirements

### Requirement: Entry amounts can be summed per currency without paging through results

`GET /api/entries/summary` SHALL accept the same `account_id`,
`category_id`/`category_mode`, `tag_id`, `from`/`to`, and `q` filters as
`GET /api/entries` (no `sort`, `dir`, `after`, or `limit` — this is an
aggregate, not a page), scoped the same way to the caller's own,
non-deleted accounts' non-deleted entries. It SHALL always additionally
restrict to entries whose `kind` is `transaction`, regardless of whether
`kind` could otherwise be requested — a `balance_adjustment` is an
absolute reading, not a categorized delta, and including it in a sum would
misrepresent the total. The response SHALL be `{ sums, income, outcome,
count }`, where `sums` is a list of `{ currency, amount }` — one entry per
distinct currency (from the currency of each matching entry's account)
present among the matching entries, each `amount` being the sum of those
entries' `amount` values — and `count` is the total number of matching
entries across every currency. `income` and `outcome` SHALL each be lists
of `{ currency, amount }` in the same one-entry-per-present-currency shape
as `sums`: within each currency, `income`'s `amount` SHALL be the sum of
the matching entries' positive `amount` values, and `outcome`'s `amount`
SHALL be the sum of the absolute value of the matching entries' negative
`amount` values — the same split `GET /api/entries/flow-summary` already
computes per bucket, applied here as one total per currency for the whole
filtered result. A currency present in `sums` SHALL always also appear in
`income` and/or `outcome` (whichever is non-zero for that currency); a
currency with no matching entries at all SHALL appear in none of the
three. An empty result SHALL return `{ sums: [], income: [], outcome: [],
count: 0 }`, not an error.

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
- **THEN** those entries are excluded from `sums`, `income`, `outcome`, and
  `count`

#### Scenario: No matching entries

- **WHEN** `GET /api/entries/summary` is called with filters that match no
  entries
- **THEN** the response is `{ sums: [], income: [], outcome: [], count: 0
  }`

#### Scenario: Exact category mode narrows the sum the same way it narrows the list

- **WHEN** `GET /api/entries/summary?category_id={parent}&category_mode=exact`
  is called
- **THEN** entries carrying a descendant category of `{parent}` are
  excluded from the sum, matching `GET /api/entries` with the same filters

#### Scenario: A currency's income and outcome split by sign

- **WHEN** matching entries in one currency include both a positive-amount
  transaction and a negative-amount transaction
- **THEN** that currency's `income` entry equals the sum of the positive
  amounts, and its `outcome` entry equals the sum of the absolute values
  of the negative amounts

#### Scenario: A self-transfer's two legs contribute to both income and outcome

- **WHEN** a self-transfer entry's sending and receiving accounts are both
  within the requested scope, so it is counted once per account (its
  outgoing, negative leg and its incoming, positive leg)
- **THEN** its outgoing leg is included in that currency's `outcome` and
  its incoming leg is included in that currency's `income`, even though
  the two legs cancel out in `sums`

#### Scenario: A currency with only income has no outcome entry

- **WHEN** every matching entry in a currency has a positive `amount`
- **THEN** that currency appears in `income` but not in `outcome`

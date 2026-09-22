## MODIFIED Requirements

### Requirement: Listing recurring transactions defaults to every account the caller has any permission on

`GET /api/recurring-transactions` SHALL accept an optional, repeatable
`account_id` filter, defaulting when omitted to every non-deleted account
the caller has any permission on (real ownership or a share of any tier) —
the same default `account-entries`' entry listing uses. An explicitly
supplied `account_id` SHALL be honored only when it names an account the
caller has at least `view` permission on. Each returned recurring
transaction SHALL include `account_currency`, resolved server-side, the
same way `account-entries` resolves it for an `Entry`.

A category or tag filter SHALL NOT widen that account scope: unlike
`account-entries`' entry listing, a caller's permission on the filtered
category or tag alone never exposes a recurring transaction on an account
they cannot otherwise see.

#### Scenario: Omitting account_id includes every visible account

- **WHEN** a user who owns one account and has a share on another calls
  `GET /api/recurring-transactions` with no `account_id`
- **THEN** recurring transactions from both accounts are included

#### Scenario: Filtering by an account the caller has no permission on returns nothing

- **WHEN** a user calls `GET /api/recurring-transactions?account_id={id}`
  for an account they have no permission on
- **THEN** the response is `200` with an empty `items` list

#### Scenario: A category filter does not reach beyond the caller's accounts

- **WHEN** a user filters by a category they own, and a recurring
  transaction in that category exists on an account they have no
  permission on
- **THEN** that recurring transaction is not returned

## ADDED Requirements

### Requirement: Listing and summing recurring transactions can be filtered by category and tag

`GET /api/recurring-transactions` and
`GET /api/recurring-transactions/summary` SHALL each accept an optional
`category_id`, an optional `category_mode` (`subtree` | `exact`,
defaulting to `subtree`), and an optional `tag_id`, resolved exactly as
`GET /api/recurring-transactions/preview` already resolves the same three
parameters and as `account-entries`' entry listing resolves its own.

`category_id` SHALL match a recurring transaction whose `category_id` is
that category or, unless `category_mode=exact`, any descendant of it
within the caller's own category tree. A recurring transaction with no
category SHALL never match a `category_id` filter. A `category_id` naming
a category the caller holds no permission on SHALL match nothing — never
every recurring transaction in scope. A `category_mode` that is neither
`subtree` nor `exact` SHALL be rejected with `400`.

`tag_id` SHALL match a recurring transaction carrying that tag among its
`tag_ids`.

Both parameters SHALL combine with `account_id`,
`include_self_transfer` and `self_transfer_both_legs` as a conjunction,
and SHALL apply identically to both operations — so the summary's total
remains the total of the rows the listing returns under the same
parameters.

#### Scenario: Filtering by a parent category includes its descendants

- **WHEN** a caller filters the listing by a category that has a child
  category, and recurring transactions exist in both
- **THEN** both are returned

#### Scenario: category_mode=exact excludes descendants

- **WHEN** that same caller adds `category_mode=exact`
- **THEN** only the recurring transaction in the named category itself is
  returned

#### Scenario: An uncategorized recurring transaction never matches a category filter

- **WHEN** a caller filters by any category and a recurring transaction in
  scope has no `category_id`
- **THEN** it is not returned

#### Scenario: A category the caller cannot see matches nothing

- **WHEN** a caller filters by a `category_id` they neither own nor hold a
  share on
- **THEN** the response is `200` with no recurring transactions, not the
  caller's unfiltered list

#### Scenario: An invalid category_mode is rejected

- **WHEN** a caller supplies `category_mode=ancestors`
- **THEN** the response is `400`

#### Scenario: Filtering by a tag

- **WHEN** a caller filters by a `tag_id` carried by one of their
  recurring transactions
- **THEN** only recurring transactions carrying that tag are returned

#### Scenario: Category and tag filters combine

- **WHEN** a caller supplies both `category_id` and `tag_id`
- **THEN** only recurring transactions matching both are returned

#### Scenario: The summary totals exactly the filtered rows

- **WHEN** a caller calls the listing and the summary with the same
  `account_id`, `category_id`, `category_mode`, `tag_id` and
  self-transfer parameters
- **THEN** the summary's per-currency totals are the per-year amounts of
  the non-ended recurring transactions the listing returned

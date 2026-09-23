# Spec Delta

## MODIFIED Requirements

### Requirement: Entry listing supports filtering, free-text search, sorting, and cursor-based pagination

`GET /api/entries` SHALL accept, all optional and combinable: `account_id` (repeatable; omitted means every non-deleted account the caller owns), `category_id` with an optional `category_mode` (`subtree`, the default — matches that category and every descendant in the category tree — or `exact`, matching only that category), `tag_id`, `kind`, `recurring_transaction_id`, `from`/`to` (an inclusive `booking_timestamp` range), `amount_from`/`amount_to` (inclusive signed `amount` bounds in the fixed stored scale), and `q` (a case-insensitive substring match against `title`, `description`, or `counterparty`). It SHALL reject a non-integer amount bound or a request whose `amount_from` exceeds its `amount_to` with `400`. It SHALL accept `sort` (`booking_timestamp`, the default, or `amount`) and `dir` (`desc`, the default, or `asc`). It SHALL accept `after`, an opaque cursor from a previous response's `next_cursor`, and `limit` (a page size). The response SHALL be `{ items, next_cursor }`, where `next_cursor` is `null` once no further matching entries remain. Every filter applies before pagination; results are always scoped to the caller's own, non-deleted accounts' non-deleted entries. `category_mode` without `category_id` has no effect.

A `self_transfer` entry SHALL appear once per resolved account (from the combination of any explicit `account_id` filter and the caller's own visible-accounts scoping) it touches: once, amount as stored, when `account_id` alone is in scope; once, amount sign flipped, when `to_account_id` alone is in scope; and twice — both of the above — when both accounts are in scope at once (for example, an unfiltered listing covering every account the caller can see, or an explicit `account_id` filter naming both). The amount bounds SHALL apply to that account-oriented signed amount, so the two occurrences may match differently. A balance adjustment's bounds SHALL likewise apply to its computed signed `amount` delta, not its absolute `balance` reading. Every other filter (`category_id`, `tag_id`, `kind`, `from`/`to`, `q`) applies identically to both self-transfer occurrences, since they represent the same underlying entry.

#### Scenario: Filtering by account

- **WHEN** `GET /api/entries?account_id={id}` is called
- **THEN** only entries on that account are returned

#### Scenario: Filtering by category includes descendants

- **WHEN** `GET /api/entries?category_id={parent}` is called (no `category_mode`) and some matching entries carry a child category of `{parent}` rather than `{parent}` itself
- **THEN** those entries are included in the results

#### Scenario: Filtering by category with an exact mode excludes descendants

- **WHEN** `GET /api/entries?category_id={parent}&category_mode=exact` is called and some matching entries carry a child category of `{parent}` rather than `{parent}` itself
- **THEN** those child-category entries are excluded from the results, and only entries carrying `{parent}` itself are returned

#### Scenario: Filtering by recurring transaction

- **WHEN** `GET /api/entries?recurring_transaction_id={id}` is called
- **THEN** only entries whose `recurring_transaction_id` equals `{id}` are returned, still scoped to the caller's own visible accounts

#### Scenario: Filtering by an open signed amount bound

- **WHEN** `GET /api/entries?amount_to=0` is called
- **THEN** entries whose signed amount is zero or negative are returned and positive entries are excluded

#### Scenario: Filtering by an inclusive amount range

- **WHEN** `GET /api/entries?amount_from=-1000000&amount_to=-500000` is called
- **THEN** entries whose signed amount is within those inclusive stored-scale bounds are returned

#### Scenario: Amount bounds filter self-transfer legs independently

- **WHEN** a self-transfer is listed for both accounts and its stored amount is negative, with an amount range that admits only negative amounts
- **THEN** only the sending account's negative occurrence is returned

#### Scenario: Amount bounds filter a balance adjustment by its delta

- **WHEN** a balance-adjustment entry has an absolute balance reading outside an amount range but a computed signed delta within it
- **THEN** the entry is returned

#### Scenario: An inverted amount range is rejected

- **WHEN** `GET /api/entries` is called with `amount_from` greater than `amount_to`
- **THEN** the response is `400`

#### Scenario: A malformed amount bound is rejected

- **WHEN** `GET /api/entries` is called with a non-integer `amount_from` or `amount_to`
- **THEN** the response is `400`

#### Scenario: Free-text search matches title or description

- **WHEN** `GET /api/entries?q=coffee` is called
- **THEN** only entries whose `title`, `description`, or `counterparty` contains "coffee" (case-insensitive) are returned

#### Scenario: Free-text search matches counterparty alone

- **WHEN** `GET /api/entries?q=rewe` is called and a matching entry's `counterparty` is `"Rewe"` while its `title` and `description` contain neither "rewe" nor any substring of it
- **THEN** that entry is included in the results

#### Scenario: Sorting by amount

- **WHEN** `GET /api/entries?sort=amount&dir=asc` is called
- **THEN** results are ordered from the smallest to the largest `amount`

#### Scenario: Paginating with a cursor

- **WHEN** a first page is fetched and its `next_cursor` is passed back as `after` on a second request with the same filters/sort
- **THEN** the second page continues immediately after the first with no gap or overlap

#### Scenario: Last page has a null cursor

- **WHEN** a page of results is fetched that reaches the end of the matching entries
- **THEN** `next_cursor` is `null`

#### Scenario: A self-transfer between two accounts in scope is listed twice

- **WHEN** `GET /api/entries` (no `account_id` filter) is called by a caller who can see both accounts of a `self_transfer` entry
- **THEN** the response's `items` includes that entry twice — once with its stored, negative-from-the-sender amount, once with the sign flipped for the receiving account

#### Scenario: A self-transfer with only one account in scope is listed once

- **WHEN** `GET /api/entries?account_id={id}` is called naming only one side of a `self_transfer` entry
- **THEN** the response's `items` includes that entry once, with the amount signed for the named account

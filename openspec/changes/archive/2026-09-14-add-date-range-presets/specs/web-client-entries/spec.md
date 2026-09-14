## MODIFIED Requirements

### Requirement: Entry list filters, search, and sort controls

`/entries` SHALL offer controls for every backend filter (`account_id`,
`category_id`, `tag_id`, `kind`, a date range via the shared date-range
filter — see `web-client-date-range-filter`), a free-text search input, and
a way to sort by booking timestamp or amount in either direction. When no
entries match the current filters, the page SHALL show text distinguishing
"no entries match these filters" from "no entries exist yet."

#### Scenario: Combining filters narrows the results

- **WHEN** an authenticated visitor sets an account filter, a category
  filter, and a date range together
- **THEN** only entries matching all three narrow the results

#### Scenario: No matches under the current filters

- **WHEN** the current filters match no entries but entries exist on the
  account
- **THEN** the page shows text indicating no entries match the current
  filters, not that none exist at all

## ADDED Requirements

### Requirement: The entry ledger defaults to the last 2 weeks

When `/entries` is opened with no date-range parameter (`range`, `from`, or
`to`) present in the URL, the entry ledger SHALL apply the "Last 2 weeks"
preset as its effective date filter, without writing that preset into the
URL. The date-range filter's dropdown SHALL show "Last 2 weeks" selected in
this state.

#### Scenario: Opening the ledger with no filters applies a two-week default

- **WHEN** an authenticated visitor opens `/entries` with no `range`,
  `from`, or `to` parameter
- **THEN** the list shows only entries within the last 2 weeks, the
  date-range filter shows "Last 2 weeks" selected, and the URL is not
  modified

#### Scenario: An explicit date filter overrides the default

- **WHEN** an authenticated visitor opens `/entries?range=this_month`
- **THEN** the list shows entries within the current month instead of the
  last-2-weeks default

# Spec Delta

## MODIFIED Requirements

### Requirement: The entry ledger's filter, search, and sort state lives in the URL

`/entries` SHALL represent its current account, category, tag, kind, date range, signed amount range, and free-text search filters, and its sort field and direction, as typed URL search parameters, readable and writable through TanStack Router's search-param APIs. The amount range SHALL use optional `amount_from` and `amount_to` stored-scale integer values. Reloading a URL with search parameters SHALL reproduce the same filtered/sorted view. Arriving at `/entries` with an `account_id` parameter already set (for example, via the link from an account's details page) SHALL apply that filter immediately on load. Every change to this state SHALL also be written to browser-local storage, keyed per visitor's browser (not synced to the account or the backend) — see "Returning to the entry ledger restores the last-applied filters" for when that stored state is read back.

#### Scenario: A filtered view survives a reload

- **WHEN** an authenticated visitor applies a category filter and a sort order, then reloads the page
- **THEN** the same category filter and sort order are applied after the reload

#### Scenario: An amount range survives a reload

- **WHEN** an authenticated visitor applies an amount start and/or end bound, then reloads the page
- **THEN** the same amount bounds are present in the URL and applied to the ledger

#### Scenario: Arriving with a preset account filter

- **WHEN** an authenticated visitor follows a link to `/entries?account_id={id}`
- **THEN** the entry list is immediately filtered to that account

#### Scenario: Changing a filter updates the URL

- **WHEN** an authenticated visitor changes the search text or a filter control
- **THEN** the corresponding URL search parameter changes to match, and the same state is written to browser-local storage

### Requirement: Entry list filters, search, and sort controls

`/entries` SHALL offer controls for every backend filter (`account_id`, `category_id`, `tag_id`, `kind`, a date range via the shared date-range filter — see `web-client-date-range-filter`, and optional signed amount start and end bounds), a free-text search input, and a way to sort by booking timestamp or amount in either direction. The amount controls SHALL accept full stored precision in major units and convert to/from the API's fixed stored scale; they SHALL permit either bound independently. When no entries match the current filters, the page SHALL show text distinguishing "no entries match these filters" from "no entries exist yet."

Those controls SHALL be laid out as a responsive grid that fills the available width — one column below the `sm` breakpoint, two from `sm`, and three from `lg` — with each control sized to its cell rather than to its content, and the free-text search input spanning the grid's full width.

#### Scenario: Combining filters narrows the results

- **WHEN** an authenticated visitor sets an account filter, a category filter, and a date range together
- **THEN** only entries matching all three narrow the results

#### Scenario: An open amount bound narrows the results

- **WHEN** an authenticated visitor sets only an amount end bound
- **THEN** the ledger requests and shows only entries at or below that signed amount

#### Scenario: No matches under the current filters

- **WHEN** the current filters match no entries but entries exist on the account
- **THEN** the page shows text indicating no entries match the current filters, not that none exist at all

#### Scenario: Controls stack in a single column on a phone

- **WHEN** an authenticated visitor views the ledger's filter controls on a viewport narrower than `sm`
- **THEN** each control occupies its own full-width row

### Requirement: The ledger offers a clear-all-filters action

`/entries` SHALL offer a single action clearing every filter at once, rendered only while at least one filter is active. Activating it SHALL remove the account, category, tag, kind, date-range, amount-range, free-text search, and upcoming-recurring parameters from the URL in one navigation, and SHALL leave the sort field and direction untouched. With no date-range parameter present the ledger's own default range applies again, per "The entry ledger defaults to the last 2 weeks".

A filter counts as active when its parameter is present in the URL; the date range counts as one active filter when any of `range`, `from`, or `to` is present, the amount range counts as one active filter when either `amount_from` or `amount_to` is present, and the implicitly applied default range does not count.

#### Scenario: Clearing every filter at once

- **WHEN** an authenticated visitor with an account filter, a tag filter, an amount bound, and search text applied activates the clear-all-filters action
- **THEN** all four parameters are removed from the URL and the ledger shows the default unfiltered view

#### Scenario: Clearing filters preserves the sort

- **WHEN** a visitor sorted by amount ascending activates the clear-all action
- **THEN** the results remain sorted by amount ascending

#### Scenario: No action is offered when nothing is filtered

- **WHEN** a visitor opens `/entries` with no filter parameters in the URL
- **THEN** no clear-all-filters action is rendered

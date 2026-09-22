## MODIFIED Requirements

### Requirement: The entry ledger's filter, search, and sort state lives in the URL

`/entries` SHALL represent its current account, category, tag, kind, date
range, and free-text search filters, and its sort field and direction, as
typed URL search parameters, readable and writable through TanStack
Router's search-param APIs. Reloading a URL with search parameters SHALL
reproduce the same filtered/sorted view. Arriving at `/entries` with an
`account_id` parameter already set (for example, via the link from an
account's details page) SHALL apply that filter immediately on load. Every
change to this state SHALL also be written to browser-local storage, keyed
per visitor's browser (not synced to the account or the backend) — see
"Returning to the entry ledger restores the last-applied filters" for when
that stored state is read back.

#### Scenario: A filtered view survives a reload

- **WHEN** an authenticated visitor applies a category filter and a sort
  order, then reloads the page
- **THEN** the same category filter and sort order are applied after the
  reload

#### Scenario: Arriving with a preset account filter

- **WHEN** an authenticated visitor follows a link to
  `/entries?account_id={id}`
- **THEN** the entry list is immediately filtered to that account

#### Scenario: Changing a filter updates the URL

- **WHEN** an authenticated visitor changes the search text or a filter
  control
- **THEN** the corresponding URL search parameter changes to match, and the
  same state is written to browser-local storage

### Requirement: The entry ledger defaults to the last 2 weeks

The entry ledger SHALL apply the "Last 2 weeks" preset as its effective
date filter when `/entries` is opened with no date-range parameter
(`range`, `from`, or `to`) present in the URL and no filter state is
restored from browser-local storage (see "Returning to the entry ledger
restores the last-applied filters"), without writing that preset into the
URL. The date-range filter's dropdown SHALL show "Last 2 weeks" selected in
this state.

#### Scenario: Opening the ledger with no filters and nothing persisted applies a two-week default

- **WHEN** an authenticated visitor with no previously persisted filter
  state opens `/entries` with no `range`, `from`, or `to` parameter
- **THEN** the list shows only entries within the last 2 weeks, the
  date-range filter shows "Last 2 weeks" selected, and the URL is not
  modified

#### Scenario: An explicit date filter overrides the default

- **WHEN** an authenticated visitor opens `/entries?range=this_month`
- **THEN** the list shows entries within the current month instead of the
  last-2-weeks default

## ADDED Requirements

### Requirement: Returning to the entry ledger restores the last-applied filters

`/entries` SHALL restore the visitor's most recently persisted
filter/search/sort state from browser-local storage when it is arrived at
with a completely bare URL — no `account_id`, `category_id`, `tag_id`,
`kind`, `range`, `from`, `to`, `q`, `sort`, `dir`, or `show_recurring`
parameter present at all — replacing the URL rather than leaving the bare
arrival in browser history, if any state has been persisted. This applies
to every ordinary navigation to bare `/entries` (a sidebar link, a saved
bookmark to `/entries` with no parameters, a link from elsewhere in the
app), not only a specific round trip. Arriving with any explicit parameter
already present SHALL NOT be overridden by persisted state — the explicit
parameters apply exactly as they would with nothing persisted. When
nothing has been persisted yet, a bare arrival SHALL fall back to the
default view (see "The entry ledger defaults to the last 2 weeks").

#### Scenario: A sidebar click restores the last-applied filters

- **WHEN** an authenticated visitor applies a tag filter and a custom date
  range on `/entries`, navigates elsewhere in the app, then clicks
  "Entries" in the sidebar
- **THEN** they land on `/entries` with the same tag filter and date range
  applied, not the default view

#### Scenario: Filters persist across a save-and-return round trip

- **WHEN** an authenticated visitor applies a tag filter and a custom date
  range on `/entries`, opens an entry, and saves it
- **THEN** they return to `/entries` with the same tag filter and date
  range applied, not just the account

#### Scenario: No persisted state falls back to the default view

- **WHEN** an authenticated visitor with nothing yet persisted opens a bare
  `/entries`
- **THEN** the default "Last 2 weeks" view applies, as described in "The
  entry ledger defaults to the last 2 weeks"

#### Scenario: An explicit link is not overridden by persisted state

- **WHEN** an authenticated visitor with a persisted category filter
  follows a link to `/entries?account_id={id}`
- **THEN** the entry list is filtered only by that account, not also by the
  previously persisted category

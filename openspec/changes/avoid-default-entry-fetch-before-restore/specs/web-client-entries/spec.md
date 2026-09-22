## MODIFIED Requirements

### Requirement: Returning to the entry ledger restores the last-applied filters

`/entries` SHALL restore the visitor's most recently persisted
filter/search/sort state from browser-local storage when it is arrived at with
a completely bare URL — no `account_id`, `category_id`, `tag_id`, `kind`,
`range`, `from`, `to`, `q`, `sort`, `dir`, or `show_recurring` parameter
present at all — replacing the URL rather than leaving the bare arrival in
browser history, if any state has been persisted. This applies to every
ordinary navigation to bare `/entries` (a sidebar link, a saved bookmark to
`/entries` with no parameters, a link from elsewhere in the app), not only a
specific round trip. Arriving with any explicit parameter already present SHALL
NOT be overridden by persisted state — the explicit parameters apply exactly as
they would with nothing persisted. When nothing has been persisted yet, a bare
arrival SHALL fall back to the default view (see "The entry ledger defaults to
the last 2 weeks").

When a bare arrival restores a persisted non-bare state, `/entries` SHALL NOT
issue `GET /api/entries` using the bare URL's default filter/search/sort state
before the restore navigation completes. The first entries query for that
arrival SHALL use the restored state.

#### Scenario: A sidebar click restores the last-applied filters

- **WHEN** an authenticated visitor applies a tag filter and a custom date
  range on `/entries`, navigates elsewhere in the app, then clicks "Entries" in
  the sidebar
- **THEN** they land on `/entries` with the same tag filter and date range
  applied, not the default view
- **AND** the entries ledger does not first query entries with the default
  last-2-weeks filter

#### Scenario: Filters persist across a save-and-return round trip

- **WHEN** an authenticated visitor applies a tag filter and a custom date
  range on `/entries`, opens an entry, and saves it
- **THEN** they return to `/entries` with the same tag filter and date range
  applied, not just the account
- **AND** the entries ledger does not first query entries with the default
  last-2-weeks filter

#### Scenario: No persisted state falls back to the default view

- **WHEN** an authenticated visitor with nothing yet persisted opens a bare
  `/entries`
- **THEN** the default view applies, as described in "The entry ledger defaults
  to the last 2 weeks"
- **AND** the entries ledger queries entries using that default view

#### Scenario: An explicit link is not overridden by persisted state

- **WHEN** an authenticated visitor with a persisted account filter follows a
  link to `/entries?category_id={id}`
- **THEN** the category filter from the URL applies
- **AND** the persisted account filter is not restored

## MODIFIED Requirements

### Requirement: Returning to the recurring list restores the last-applied filters

`/recurring` SHALL restore the visitor's most recently persisted filter state
from browser-local storage when it is arrived at with a completely bare URL —
no `account_id`, `category_id`, `tag_id`, `include_self_transfer`, or
`self_transfer_both_legs` parameter present at all — replacing the URL rather
than leaving the bare arrival in browser history, if any state has been
persisted. This applies to any ordinary navigation to bare `/recurring` (a
sidebar link, or any other ordinary navigation). Arriving with any explicit
parameter already present SHALL NOT be overridden by persisted state. When
nothing has been persisted yet, a bare arrival SHALL fall back to the
unfiltered recurring list.

When a bare arrival restores a persisted non-bare state, `/recurring` SHALL NOT
issue recurring list or summary requests using the bare URL's default filter
state before the restore navigation completes. The first recurring list and
summary queries for that arrival SHALL use the restored state.

#### Scenario: A sidebar click restores the last-applied filters

- **WHEN** an authenticated visitor applies an account filter on `/recurring`,
  navigates elsewhere in the app, then clicks "Recurring" in the sidebar
- **THEN** they land on `/recurring` with the same account filter applied, not
  the unfiltered list
- **AND** the recurring list does not first query using the unfiltered default
  state

#### Scenario: No persisted state falls back to the default view

- **WHEN** an authenticated visitor with nothing yet persisted opens a bare
  `/recurring`
- **THEN** the unfiltered recurring list applies
- **AND** the recurring list queries using that unfiltered state

#### Scenario: An explicit link is not overridden by persisted state

- **WHEN** an authenticated visitor with a persisted account filter follows a
  link to `/recurring?category_id={id}`
- **THEN** the category filter from the URL applies
- **AND** the persisted account filter is not restored

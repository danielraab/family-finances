## ADDED Requirements

### Requirement: Returning to the recurring list restores the last-applied filters

`/recurring` SHALL write its current filter state (`account_id`,
`category_id`, `tag_id`, `include_self_transfer`,
`self_transfer_both_legs`) to browser-local storage, keyed per visitor's
browser (not synced to the account or the backend), whenever it changes.
Arriving at `/recurring` with a completely bare URL — none of those
parameters present at all — SHALL restore the visitor's most recently
persisted filter state from that storage, replacing the URL rather than
leaving the bare arrival in browser history, if any state has been
persisted. This applies to any ordinary navigation to bare `/recurring` (a
sidebar link, a link from elsewhere in the app). Arriving with any explicit
parameter already present SHALL NOT be overridden by persisted state — the
explicit parameters apply exactly as they would with nothing persisted.
When nothing has been persisted yet, a bare arrival SHALL show the
unfiltered list, as today.

#### Scenario: A sidebar click restores the last-applied filters

- **WHEN** an authenticated visitor applies an account filter and enables
  "include self-transfers" on `/recurring`, navigates elsewhere in the
  app, then clicks "Recurring" in the sidebar
- **THEN** they land on `/recurring` with the same account filter and
  self-transfer toggle applied, not the unfiltered list

#### Scenario: No persisted state shows the unfiltered list

- **WHEN** an authenticated visitor with nothing yet persisted opens a bare
  `/recurring`
- **THEN** the unfiltered list is shown, as today

#### Scenario: An explicit link is not overridden by persisted state

- **WHEN** an authenticated visitor with a persisted category filter
  follows a link to `/recurring?account_id={id}`
- **THEN** the recurring list is filtered only by that account, not also by
  the previously persisted category

#### Scenario: Changing a filter persists it

- **WHEN** an authenticated visitor changes a filter control on
  `/recurring`
- **THEN** the corresponding URL search parameter changes to match, and the
  same state is written to browser-local storage

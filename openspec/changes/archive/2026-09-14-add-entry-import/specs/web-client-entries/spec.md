## ADDED Requirements

### Requirement: Entries link in the toolbar to import

`/entries`' toolbar SHALL offer an "Import" action alongside "New Entry",
navigating to `/entries/import`. When the ledger's current view has an
`account_id` filter applied, the Import action SHALL carry it as
`/entries/import?account_id={id}`, the same way "New Entry" already
carries the current filter.

#### Scenario: Navigating to import from the ledger

- **WHEN** an authenticated visitor activates "Import" on `/entries`
- **THEN** the client navigates to `/entries/import`

#### Scenario: The current account filter is carried over

- **WHEN** an authenticated visitor is viewing `/entries?account_id={id}`
  and activates "Import"
- **THEN** the client navigates to `/entries/import?account_id={id}`

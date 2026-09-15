## ADDED Requirements

### Requirement: An opt-in Upcoming block previews future recurring occurrences on the entry ledger

`/entries` SHALL offer a toggle, off by default and encoded as a
`show_recurring` URL search parameter, that reveals a dedicated "Upcoming"
block above the real, paginated entry list. While off, `/entries` behaves
exactly as it did before this capability existed — no additional request
is made. While on, the block SHALL fetch
`GET /api/recurring-transactions/preview` using the ledger's current
account/category/tag filters and a `to` resolved as
`min(the ledger's current effective date-range "to", if any; the caller's
recurring_preview_horizon setting resolved to a date)`, and SHALL render
one row per returned item, always sorted by `booking_timestamp` ascending,
independent of whatever sort the real list below is currently using.

#### Scenario: Toggling the preview on fetches and shows upcoming rows

- **WHEN** an authenticated visitor enables the "Upcoming" toggle on
  `/entries`
- **THEN** `GET /api/recurring-transactions/preview` is called with the
  ledger's current filters and the resolved cutoff, and matching rows
  render in a block above the real list, sorted by date

#### Scenario: The toggle is off by default

- **WHEN** an authenticated visitor opens `/entries` with no
  `show_recurring` parameter in the URL
- **THEN** no Upcoming block is shown and no preview request is made

#### Scenario: The block's own sort is independent of the ledger's sort

- **WHEN** the real entry list below is sorted by amount descending
- **THEN** the Upcoming block still renders its rows sorted by
  `booking_timestamp` ascending

#### Scenario: A filter's own end date bounds the preview tighter than the horizon

- **WHEN** the ledger's current date-range filter resolves a `to` earlier
  than the caller's recurring preview horizon
- **THEN** the preview request's cutoff uses that earlier `to`, not the
  horizon date

### Requirement: An overdue previewed occurrence is visually distinguished but stays in chronological order

A row in the Upcoming block whose `overdue` is `true` SHALL render with a
distinct, muted background tint from a non-overdue row, but SHALL NOT be
moved out of its normal position in the block's date-ascending order.

#### Scenario: An overdue row is tinted, not reordered

- **WHEN** the Upcoming block includes one overdue row and several
  non-overdue rows
- **THEN** the overdue row renders with the distinct background tint in
  its correct chronological position among the others, not pulled to the
  top or bottom

### Requirement: Each Upcoming row offers the existing Create transaction action, prefilled to that occurrence's date

Each row in the Upcoming block SHALL offer a "Create transaction" action,
mirroring the one `/recurring` already offers, navigating to
`/entries/new?recurring_transaction_id={id}&booking_timestamp={that row's
booking_timestamp}` (see `web-client-recurring-transactions`'s date-override
requirement).

#### Scenario: Activating Create transaction on an Upcoming row

- **WHEN** an authenticated visitor activates "Create transaction" on an
  Upcoming block row projected for 2026-12-01
- **THEN** the client navigates to `/entries/new` with that row's
  `recurring_transaction_id` and `booking_timestamp=2026-12-01`, and the
  form prefills accordingly

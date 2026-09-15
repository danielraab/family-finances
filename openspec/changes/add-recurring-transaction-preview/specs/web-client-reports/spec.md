## ADDED Requirements

### Requirement: An opt-in Upcoming block previews future recurring occurrences in a report

`/reports` SHALL offer a toggle, off by default and encoded as a
`show_recurring` URL search parameter, alongside its other filter
controls. Its state SHALL NOT trigger a fetch on its own, consistent with
every other `/reports` control — the preview is fetched only when
"Generate report" is activated (see `web-client-reports`'s "A report is
only generated on explicit action" requirement), alongside the existing
`GET /api/entries` and `GET /api/entries/summary` calls. When the toggle
is on at generation time, `GET /api/recurring-transactions/preview` SHALL
be called using the report's current category/tag/account filters and a
`to` resolved as `min(the report's current effective date-range "to", if
any; the caller's recurring_preview_horizon setting resolved to a date)`,
and its results SHALL render in a dedicated "Upcoming" block, sorted by
`booking_timestamp` ascending, alongside the generated results table. When
the toggle is off, no such request is made and no block renders.

#### Scenario: Generating with the toggle on fetches the preview

- **WHEN** an authenticated visitor enables the Upcoming toggle and
  activates "Generate report"
- **THEN** `GET /api/recurring-transactions/preview` is called alongside
  the existing entries/summary requests, and matching rows render in an
  Upcoming block

#### Scenario: The toggle does not fetch on its own

- **WHEN** an authenticated visitor enables the Upcoming toggle without
  activating "Generate report"
- **THEN** no preview request is made

#### Scenario: Generating with the toggle off shows no Upcoming block

- **WHEN** an authenticated visitor generates a report with the Upcoming
  toggle left off
- **THEN** no Upcoming block is rendered and no preview request is made

### Requirement: Previewed occurrences never affect the report's per-currency sum

The Upcoming block's rows SHALL be excluded from `/reports`' displayed
per-currency sum (`GET /api/entries/summary`'s response) under every
condition — the sum SHALL always reflect only real, matching entries,
regardless of whether the Upcoming toggle is on.

#### Scenario: The sum is unaffected by the Upcoming toggle

- **WHEN** a report is generated with the Upcoming toggle on and the
  Upcoming block shows one or more previewed rows
- **THEN** the displayed per-currency sum is identical to what it would be
  with the toggle off

### Requirement: An overdue previewed occurrence is visually distinguished but stays in chronological order

A row in the report's Upcoming block whose `overdue` is `true` SHALL
render with a distinct, muted background tint from a non-overdue row, but
SHALL NOT be moved out of its normal position in the block's
date-ascending order.

#### Scenario: An overdue row is tinted, not reordered

- **WHEN** the report's Upcoming block includes one overdue row among
  several non-overdue rows
- **THEN** the overdue row renders with the distinct background tint in
  its correct chronological position, not pulled to the top or bottom

### Requirement: Each Upcoming row offers the existing Create transaction action, prefilled to that occurrence's date

Each row in the report's Upcoming block SHALL offer a "Create transaction"
action, navigating to `/entries/new?recurring_transaction_id={id}
&booking_timestamp={that row's booking_timestamp}` (see
`web-client-recurring-transactions`'s date-override requirement).

#### Scenario: Activating Create transaction on a report's Upcoming row

- **WHEN** an authenticated visitor activates "Create transaction" on a
  report's Upcoming block row
- **THEN** the client navigates to `/entries/new` with that row's
  `recurring_transaction_id` and `booking_timestamp`, and the form
  prefills accordingly

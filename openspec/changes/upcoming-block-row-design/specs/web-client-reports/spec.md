## MODIFIED Requirements

### Requirement: An overdue previewed occurrence is visually distinguished but stays in chronological order

A row in the report's Upcoming block whose `overdue` is `true` SHALL
render with a distinct, muted background tint from a non-overdue row, and
SHALL carry the same legible, never-truncated overdue marker
`web-client-entries` defines for its own Upcoming block. It SHALL NOT be
moved out of its normal position in the block's date-ascending order.

#### Scenario: An overdue row is tinted, not reordered

- **WHEN** the report's Upcoming block includes one overdue row among
  several non-overdue rows
- **THEN** the overdue row renders with the distinct background tint and
  its overdue marker in full, in its correct chronological position, not
  pulled to the top or bottom

### Requirement: Each Upcoming row offers the existing Create transaction action, prefilled to that occurrence's date

Each row in the report's Upcoming block SHALL offer a "Create transaction"
action, navigating to `/entries/new?recurring_transaction_id={id}
&booking_timestamp={that row's booking_timestamp}` (see
`web-client-recurring-transactions`'s date-override requirement), and
rendering as a plus glyph alone below the `sm` breakpoint exactly as
`web-client-entries`' Upcoming block does.

#### Scenario: Activating Create transaction on a report's Upcoming row

- **WHEN** an authenticated visitor activates "Create transaction" on a
  report's Upcoming block row
- **THEN** the client navigates to `/entries/new` with that row's
  `recurring_transaction_id` and `booking_timestamp`, and the form
  prefills accordingly

## ADDED Requirements

### Requirement: A report's Upcoming row title opens the recurring summary

Each row in the report's Upcoming block SHALL render its title as an
activatable control opening that row's recurring transaction in the
read-only recurring summary modal — the same summary the recurring badge
in the report's results table already opens.

#### Scenario: Activating the title of a report's Upcoming row

- **WHEN** an authenticated visitor activates the title of a report's
  Upcoming block row
- **THEN** that recurring transaction's read-only summary modal opens,
  with no navigation away from the report

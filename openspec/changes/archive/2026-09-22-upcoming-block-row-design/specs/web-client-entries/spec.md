## MODIFIED Requirements

### Requirement: Each Upcoming row offers the existing Create transaction action, prefilled to that occurrence's date

Each row in the Upcoming block SHALL offer a "Create transaction" action,
mirroring the one `/recurring` already offers, navigating to
`/entries/new?recurring_transaction_id={id}&booking_timestamp={that row's
booking_timestamp}` (see `web-client-recurring-transactions`'s date-override
requirement).

Below the `sm` breakpoint the action SHALL render as a plus glyph alone,
and from `sm` up as that glyph beside its translated label. Its
accessible name SHALL be that same translated label at every width.

#### Scenario: Activating Create transaction on an Upcoming row

- **WHEN** an authenticated visitor activates "Create transaction" on an
  Upcoming block row projected for 2026-12-01
- **THEN** the client navigates to `/entries/new` with that row's
  `recurring_transaction_id` and `booking_timestamp=2026-12-01`, and the
  form prefills accordingly

#### Scenario: The action is icon-only on a phone

- **WHEN** an Upcoming block row is rendered below the `sm` breakpoint
- **THEN** its Create transaction action shows the plus glyph without its
  label, and its accessible name is still the translated label

### Requirement: An overdue previewed occurrence is visually distinguished but stays in chronological order

A row in the Upcoming block whose `overdue` is `true` SHALL render with a
distinct, muted background tint from a non-overdue row, and SHALL carry a
legible overdue marker beside its title that is never truncated or
ellipsised, however long the title is. It SHALL NOT be moved out of its
normal position in the block's date-ascending order.

#### Scenario: An overdue row is tinted, not reordered

- **WHEN** the Upcoming block includes one overdue row and several
  non-overdue rows
- **THEN** the overdue row renders with the distinct background tint in
  its correct chronological position among the others, not pulled to the
  top or bottom

#### Scenario: A long title never hides the overdue marker

- **WHEN** an overdue row's title is too long for the row and is
  truncated
- **THEN** the overdue marker beside it is still shown in full

## ADDED Requirements

### Requirement: An Upcoming row's title opens the recurring transaction's summary

Each row in the Upcoming block SHALL render its title as an activatable
control that opens that row's recurring transaction in the read-only
recurring summary modal (see `web-client-modals`), the same summary
`/recurring`'s rows and a linked entry's badge already open.

#### Scenario: Activating an Upcoming row's title

- **WHEN** an authenticated visitor activates the title of an Upcoming
  block row
- **THEN** that row's recurring transaction's read-only summary modal
  opens, with no navigation away from the ledger

### Requirement: An Upcoming row lays out so that no part of it overlaps another

An Upcoming row SHALL render its title and amount on one line and its
date, account and Create transaction action on a second, with each
shrinkable part truncating within its own bounds. At every width the
block is rendered at, no part of a row SHALL overlap another, and the
date and its separator SHALL stay on one line.

#### Scenario: A narrow row truncates rather than overlapping

- **WHEN** an Upcoming row with a long title and a long account name is
  rendered in the narrowest container the block is used in
- **THEN** the title and the account name truncate within their own
  bounds, and neither is drawn over the amount or the action

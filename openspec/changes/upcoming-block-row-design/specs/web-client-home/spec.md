## MODIFIED Requirements

### Requirement: An overdue previewed occurrence on an entry_list card is visually distinguished but stays in order

A row in an `entry_list` card's Upcoming block whose `overdue` is `true`
SHALL render with the same distinct, muted background tint and the same
never-truncated overdue marker `web-client-entries` defines for its own
Upcoming block, in its normal date-ascending position.

#### Scenario: An overdue row is tinted, not reordered

- **WHEN** an `entry_list` card's Upcoming block includes an overdue row
- **THEN** it renders tinted, with its overdue marker shown in full, in
  its correct chronological position

### Requirement: Each entry_list card's Upcoming row offers the existing Create transaction action

Each row in an `entry_list` card's Upcoming block SHALL offer the same
"Create transaction" action `web-client-entries`'s Upcoming block offers,
navigating to `/entries/new` with that row's `recurring_transaction_id`
and `booking_timestamp`, and rendering as a plus glyph alone below the
`sm` breakpoint exactly as that block does.

#### Scenario: Activating Create transaction on a card's Upcoming row

- **WHEN** a visitor activates "Create transaction" on an `entry_list`
  card's Upcoming block row
- **THEN** the client navigates to `/entries/new` with that row's
  `recurring_transaction_id` and `booking_timestamp`

#### Scenario: The action is icon-only on a phone

- **WHEN** an `entry_list` card's Upcoming block is rendered below the
  `sm` breakpoint
- **THEN** each row's Create transaction action shows the plus glyph
  without its label, and its accessible name is still the translated
  label

## ADDED Requirements

### Requirement: An entry_list card's Upcoming row title opens the recurring summary

Each row in an `entry_list` card's Upcoming block SHALL render its title
as an activatable control opening that row's recurring transaction in the
read-only recurring summary modal, the same summary the card's real entry
rows already reach through an entry's own summary.

#### Scenario: Activating the title of a card's Upcoming row

- **WHEN** a visitor activates the title of an `entry_list` card's
  Upcoming block row
- **THEN** that recurring transaction's read-only summary modal opens,
  with no navigation away from the dashboard

### Requirement: An entry_list card's Upcoming block is not drawn as a box inside the card

The Upcoming block rendered inside an `entry_list` card SHALL NOT draw
its own border, rounding or padding inside the card's. It SHALL keep its
heading and SHALL be separated from the card's real entry list by a rule.
The same block rendered on `/entries` and `/reports`, where it stands on
its own, SHALL keep its bordered box.

#### Scenario: The block renders without a nested box on the dashboard

- **WHEN** an `entry_list` card with the preview enabled renders its
  Upcoming block
- **THEN** the block renders with its heading and a rule separating it
  from the card's entry list, and with no border of its own inside the
  card's border

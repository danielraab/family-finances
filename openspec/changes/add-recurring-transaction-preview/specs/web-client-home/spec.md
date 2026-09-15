## ADDED Requirements

### Requirement: An entry_list card with the preview enabled shows an Upcoming block

An `entry_list` card whose `config.show_recurring_preview` is `true`
SHALL render a dedicated "Upcoming" block above its real (fixed-size)
entry list, populated from `GET /api/recurring-transactions/preview`
using the card's own inline account/category/tag filter and a `to`
resolved as `min(the card's own "range" resolved "to", if set; the
caller's recurring_preview_horizon setting resolved to a date)`. The
Upcoming block SHALL NOT count against the card's fixed page size — it is
a separate, additional section. A card whose `show_recurring_preview` is
`false` or absent SHALL render exactly as it did before this capability
existed.

#### Scenario: An entry_list card with the preview on shows upcoming rows

- **WHEN** an `entry_list` card's `config.show_recurring_preview` is `true`
- **THEN** the card renders an Upcoming block above its real entry list,
  populated from the preview endpoint using the card's own filter

#### Scenario: The preview is off by default

- **WHEN** an `entry_list` card's `config` has no `show_recurring_preview`
  set
- **THEN** no Upcoming block renders and no preview request is made for
  that card

#### Scenario: The Upcoming block does not reduce the real list's count

- **WHEN** an `entry_list` card has the preview enabled and both an
  Upcoming block and its real list have items
- **THEN** the real list still shows up to its normal fixed page size,
  unaffected by how many Upcoming rows are shown

### Requirement: An overdue previewed occurrence on an entry_list card is visually distinguished but stays in order

A row in an `entry_list` card's Upcoming block whose `overdue` is `true`
SHALL render with the same distinct, muted background tint
`web-client-entries` defines for its own Upcoming block, in its normal
date-ascending position.

#### Scenario: An overdue row is tinted, not reordered

- **WHEN** an `entry_list` card's Upcoming block includes an overdue row
- **THEN** it renders tinted, in its correct chronological position

### Requirement: Each entry_list card's Upcoming row offers the existing Create transaction action

Each row in an `entry_list` card's Upcoming block SHALL offer the same
"Create transaction" action `web-client-entries`'s Upcoming block offers,
navigating to `/entries/new` with that row's `recurring_transaction_id`
and `booking_timestamp`.

#### Scenario: Activating Create transaction on a card's Upcoming row

- **WHEN** a visitor activates "Create transaction" on an `entry_list`
  card's Upcoming block row
- **THEN** the client navigates to `/entries/new` with that row's
  `recurring_transaction_id` and `booking_timestamp`

### Requirement: A bar_chart card with the preview enabled stacks projected amounts on its bars

A `bar_chart` card whose `config.show_recurring_preview` is `true` SHALL,
for any bucket falling between today and its resolved cutoff (`min(the
horizon setting resolved to a date; the card has no date-range filter to
further bound it)`), fetch `GET /api/recurring-transactions/preview`
using the card's own inline account/category/tag filter, bucket the
returned items into the same month/day periods the card's real
`flow-summary` buckets already use, and render each affected bucket's
income/outcome bars with a second, distinctly-colored segment stacked on
top of the real segment representing the projected amount. A card whose
`show_recurring_preview` is `false` or absent, or a bucket with no
projected amount, SHALL render exactly as it did before this capability
existed — a plain, non-stacked bar.

#### Scenario: A future bucket shows a stacked projected segment

- **WHEN** a `bar_chart` card has the preview enabled and a displayed
  bucket beyond today has projected income
- **THEN** that bucket's income bar renders with a distinctly-colored
  segment, stacked above its real segment, sized to the projected amount

#### Scenario: A bucket spanning today mixes real and projected amounts in one bar

- **WHEN** the currently displayed period's bucket containing today has
  both real entries and projected occurrences later in the same bucket
- **THEN** that bucket's bar renders as one bar with both a real segment
  and a stacked projected segment

#### Scenario: The preview is off by default

- **WHEN** a `bar_chart` card's `config` has no `show_recurring_preview`
  set
- **THEN** every bucket renders as a plain, non-stacked bar, unchanged
  from before this capability existed

#### Scenario: A bucket beyond the resolved cutoff shows no projected segment

- **WHEN** a displayed bucket falls entirely after the card's resolved
  cutoff
- **THEN** that bucket's bars render with no stacked projected segment

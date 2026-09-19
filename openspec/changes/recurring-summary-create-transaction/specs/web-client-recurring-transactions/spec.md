## MODIFIED Requirements

### Requirement: A recurring transaction has a read-only summary modal

A recurring transaction SHALL have a read-only summary modal, opened from
its badge on a linked entry in the ledger and the `/reports` results
table, from a `/recurring` list row, and from an Upcoming block row's
title. The summary SHALL show the template's title, account, signed
amount, interval (count and unit), start date, end date when set, whether
it has ended, next suggested date, per-year amount, category, tags,
counterparty, and linked entry count, each omitted when not carried.
Nothing in the summary SHALL be editable.

The summary SHALL offer an Edit action navigating to
`/recurring/{id}/edit`.

The summary SHALL also offer a "Create transaction" action navigating to
`/entries/new?recurring_transaction_id={id}`, the same action
`/recurring`'s own rows offer. When the summary was opened for a specific
projected occurrence, that action SHALL additionally carry that
occurrence's `booking_timestamp`; when it was not, it SHALL send no
`booking_timestamp` and the entry form's own default date applies.

#### Scenario: The badge opens the summary

- **WHEN** a visitor activates the recurring badge on a linked entry
- **THEN** that recurring transaction's read-only summary modal opens

#### Scenario: The summary offers Edit

- **WHEN** the recurring summary modal is open
- **THEN** it offers an Edit action navigating to `/recurring/{id}/edit`

#### Scenario: The summary offers Create transaction

- **WHEN** the recurring summary modal is open
- **THEN** it offers a Create transaction action navigating to
  `/entries/new` with that recurring transaction's id

#### Scenario: A summary opened from an occurrence books that occurrence's date

- **WHEN** a visitor opens the summary from an Upcoming block row
  projected for 2026-12-01 and activates Create transaction
- **THEN** the client navigates to `/entries/new` with that recurring
  transaction's id and `booking_timestamp=2026-12-01`

#### Scenario: A summary opened from a badge sends no date

- **WHEN** a visitor opens the summary from a linked entry's badge or a
  `/recurring` row and activates Create transaction
- **THEN** the client navigates to `/entries/new` with the recurring
  transaction's id and no `booking_timestamp`, and the form prefills its
  own default date

#### Scenario: An ended template is shown as ended

- **WHEN** the summary opens for a recurring transaction whose `ended` is
  true
- **THEN** the summary indicates that it has ended

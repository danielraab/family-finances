## ADDED Requirements

### Requirement: A recurring transaction has a read-only summary modal

A recurring transaction SHALL have a read-only summary modal, opened from
its badge on a linked entry in the ledger and the `/reports` results
table. The summary SHALL show the template's title, account, signed
amount, interval (count and unit), start date, end date when set, whether
it has ended, next suggested date, per-year amount, category, tags,
counterparty, and linked entry count, each omitted when not carried.
Nothing in the summary SHALL be editable.

The summary SHALL offer an Edit action navigating to
`/recurring/{id}/edit`.

#### Scenario: The badge opens the summary

- **WHEN** a visitor activates the recurring badge on a linked entry
- **THEN** that recurring transaction's read-only summary modal opens

#### Scenario: The summary offers Edit

- **WHEN** the recurring summary modal is open
- **THEN** it offers an Edit action navigating to `/recurring/{id}/edit`

#### Scenario: An ended template is shown as ended

- **WHEN** the summary opens for a recurring transaction whose `ended` is
  true
- **THEN** the summary indicates that it has ended

### Requirement: The entry and recurring summaries cross-link in place

An entry summary for an entry with a `recurring_transaction_id` SHALL
offer a way to open that recurring transaction's summary, and the
recurring summary SHALL offer a way back to the entry summary it was
opened from. Following either SHALL replace the open modal's content
rather than opening a second modal on top of it.

#### Scenario: Opening the recurring summary from an entry summary

- **WHEN** the visitor follows the recurring-transaction link inside an
  entry's summary
- **THEN** the same modal's content is replaced by the recurring
  transaction's summary, with no second modal stacked above it

#### Scenario: Returning to the entry summary

- **WHEN** the visitor activates the back affordance on a recurring
  summary reached from an entry summary
- **THEN** the modal's content returns to that entry's summary

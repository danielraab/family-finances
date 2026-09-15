## MODIFIED Requirements

### Requirement: Creating a transaction from a recurring transaction is always a manual action

Each recurring transaction (on the list and on its edit page) SHALL offer a
"Create transaction" action. Activating it SHALL navigate to
`/entries/new` prefilled from the template — account, title, description,
category, counterparty, location, tags, and signed amount — with
`booking_timestamp` prefilled to the recurring transaction's
`next_suggested_date`, and with `recurring_transaction_id` carried through
so the entry is linked once submitted. Every prefilled field SHALL remain
editable before submission, and no entry SHALL be created without the
visitor explicitly submitting that form — there is no automatic or
scheduled creation.

`/entries/new` SHALL additionally accept an optional `booking_timestamp`
search parameter alongside `recurring_transaction_id`. When present, it
SHALL override the fetched recurring transaction's `next_suggested_date`
as the prefilled booking timestamp — used by a "Create transaction" action
on a previewed future occurrence (see `web-client-entries`'s and
`web-client-reports`'s Upcoming block) to prefill that specific occurrence's
date rather than always the template's own `next_suggested_date`. When
`booking_timestamp` is absent, behavior is unchanged from before this
capability existed.

#### Scenario: Create transaction prefills and links

- **WHEN** the visitor activates "Create transaction" on a recurring
  transaction and submits the prefilled form unchanged
- **THEN** a new entry is created matching the template's fields, booked on
  the template's `next_suggested_date`, with `recurring_transaction_id` set
  to that recurring transaction

#### Scenario: Prefilled fields remain editable

- **WHEN** the visitor activates "Create transaction" and changes the
  amount or booking date before submitting
- **THEN** the created entry reflects the edited values, still linked to
  the recurring transaction

#### Scenario: No background or scheduled creation

- **WHEN** a recurring transaction's `next_suggested_date` has passed with
  no visitor action taken
- **THEN** no entry is created automatically — the list still shows the
  template with its (past) `next_suggested_date`, unchanged until the
  visitor manually acts

#### Scenario: An explicit booking_timestamp overrides the fetched next_suggested_date

- **WHEN** the visitor arrives at
  `/entries/new?recurring_transaction_id={id}&booking_timestamp=2026-12-01`
  for a recurring transaction whose `next_suggested_date` is `2026-10-01`
- **THEN** the form prefills `booking_timestamp` to `2026-12-01`, not
  `2026-10-01`, with every other field still prefilled from the template

#### Scenario: Omitting booking_timestamp keeps today's behavior

- **WHEN** the visitor arrives at
  `/entries/new?recurring_transaction_id={id}` with no `booking_timestamp`
  parameter
- **THEN** the form prefills `booking_timestamp` to the template's
  `next_suggested_date`, exactly as before this capability existed

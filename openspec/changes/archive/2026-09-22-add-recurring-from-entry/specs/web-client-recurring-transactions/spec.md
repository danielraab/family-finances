## ADDED Requirements

### Requirement: Creating a recurring transaction from an existing entry prefills and auto-links it

`/recurring/new` SHALL accept an optional `from_entry_id` search
parameter. When present, it SHALL fetch that entry and prefill the create
form's account (locked, not changeable in this flow — mirroring
`/entries/new`'s `?account_id=` lock), title, description, category,
counterparty, location, tags, and signed amount from it, and SHALL default
`starts_on` to the entry's booking date (date portion only, no
time-of-day). Every prefilled field SHALL remain editable before
submission except the locked account.

On successful creation from this flow, the new recurring transaction SHALL
always be linked back to the originating entry — `PATCH
/api/entries/{id}` setting `recurring_transaction_id` to the newly created
template's id — with no visitor-facing opt-out. The visitor SHALL land on
`/recurring` after creation, the same destination `/recurring/new` already
navigates to on a normal (non-`from_entry_id`) successful creation.

#### Scenario: Prefilled from the entry

- **WHEN** the visitor arrives at `/recurring/new?from_entry_id=<id>`
- **THEN** the form is populated with that entry's title, description,
  category, counterparty, location, tags, and signed amount, the account
  field is locked to the entry's account, and `starts_on` defaults to the
  entry's booking date

#### Scenario: Prefilled fields remain editable

- **WHEN** the visitor arrives via `from_entry_id` and changes the title or
  amount before submitting
- **THEN** the created recurring transaction reflects the edited values

#### Scenario: Successful creation links back to the originating entry

- **WHEN** the visitor submits the form reached via
  `?from_entry_id=<id>` and creation succeeds
- **THEN** `PATCH /api/entries/{id}` is called with `recurring_transaction_id`
  set to the newly created recurring transaction's id, and the visitor
  lands on `/recurring`

#### Scenario: No prefill without from_entry_id

- **WHEN** the visitor opens `/recurring/new` with no `from_entry_id`
  parameter
- **THEN** the form starts empty as it does today, and no entry is
  fetched or linked

## ADDED Requirements

### Requirement: Linking or unlinking an existing entry to a recurring transaction happens only on the entry edit page

`/entries/{id}/edit` SHALL offer a field to set or clear the entry's
`recurring_transaction_id`, offering only recurring transactions on the
entry's current account as choices. `/entries/new` SHALL NOT expose this
field as a normal, visitor-facing control — the only way a newly created
entry gets linked is via the recurring transaction's own "Create
transaction" prefill flow (see `web-client-recurring-transactions`), which
carries the id through without a picker.

#### Scenario: Linking an existing entry from its edit page

- **WHEN** the visitor opens `/entries/{id}/edit` and selects a recurring
  transaction on the entry's account, then saves
- **THEN** `PATCH /api/entries/{id}` is called with that
  `recurring_transaction_id`, and the entry is now linked

#### Scenario: Unlinking

- **WHEN** the visitor clears the field on `/entries/{id}/edit` and saves
- **THEN** `PATCH /api/entries/{id}` is called with
  `recurring_transaction_id: null`

#### Scenario: New entry form has no link picker

- **WHEN** the visitor opens `/entries/new` directly (not via a recurring
  transaction's "Create transaction" action)
- **THEN** no recurring-transaction picker is shown on the form

### Requirement: A linked entry shows a badge to its recurring transaction in the ledger

`/entries` SHALL show a small icon/badge on any entry whose
`recurring_transaction_id` is set, distinct from the row's other content,
that navigates to that recurring transaction's edit page when activated.
An entry with no `recurring_transaction_id` SHALL show no such badge.

#### Scenario: Linked entry shows the badge

- **WHEN** the ledger lists an entry with a non-null
  `recurring_transaction_id`
- **THEN** its row shows the recurring-transaction badge

#### Scenario: Badge links to the recurring transaction

- **WHEN** the visitor activates a linked entry's badge
- **THEN** the browser navigates to that recurring transaction's edit page

#### Scenario: Unlinked entry shows no badge

- **WHEN** the ledger lists an entry with a null `recurring_transaction_id`
- **THEN** no recurring-transaction badge is shown on its row

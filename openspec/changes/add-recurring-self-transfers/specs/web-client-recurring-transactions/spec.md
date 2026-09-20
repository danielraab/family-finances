## ADDED Requirements

### Requirement: The recurring list has a self-transfer filter with a revealed second flag

`/recurring` SHALL offer a filter control above the list carrying two
checkboxes. The first, "Show self-transfers", SHALL be unchecked by
default and SHALL map to `include_self_transfer` on both the list and the
summary request. The second, "Show both sides (income and outcome)", SHALL
be rendered only while the first is checked, SHALL itself be unchecked by
default, and SHALL map to `self_transfer_both_legs` on the same two
requests. Unchecking the first SHALL hide the second and clear it, so the
two can never be left in the meaningless "both legs, transfers hidden"
combination. Both flags SHALL live in the URL's search parameters, the way
`/entries` keeps its own filter state, so a filtered list can be
bookmarked, shared, and restored by the browser's back button. The
per-currency total below the list SHALL always reflect the same two flags
as the list above it.

#### Scenario: Transfers hidden by default

- **WHEN** the visitor opens `/recurring` with a `self_transfer` recurring
  transaction among their templates
- **THEN** it is not listed, and the second checkbox is not rendered

#### Scenario: Checking the first flag reveals the second

- **WHEN** the visitor checks "Show self-transfers"
- **THEN** self-transfer templates appear in the list, each once, and the
  "Show both sides" checkbox becomes visible, unchecked

#### Scenario: Both sides shown

- **WHEN** the visitor additionally checks "Show both sides (income and
  outcome)" with both of a transfer's accounts visible to them
- **THEN** that template is listed twice, once as an outgoing row on the
  sending account and once as an incoming row on the receiving account

#### Scenario: Unchecking the first flag clears the second

- **WHEN** the visitor has both checkboxes checked and unchecks "Show
  self-transfers"
- **THEN** the second checkbox is hidden, its value is cleared from the
  URL, and no self-transfer rows are listed

#### Scenario: The total follows the flags

- **WHEN** the visitor toggles either flag
- **THEN** the per-currency per-year total below the list is refetched with
  the same flags, so it always totals the rows shown

### Requirement: A self-transfer row identifies the other account and its direction

A listed `self_transfer` recurring transaction SHALL show the other
account it moves money to or from, visually distinguished from a plain
transaction row the way a self-transfer entry already is in the ledger, and
SHALL make clear which side the row represents when both sides are shown.
Its category cell SHALL render as empty rather than broken when the
template has no category, and its counterparty and location cells SHALL be
empty, since a self-transfer carries neither.

#### Scenario: The other account is named on the row

- **WHEN** a `self_transfer` recurring transaction is listed
- **THEN** its row names the account on the other side of the transfer

#### Scenario: The two sides are distinguishable

- **WHEN** both sides of one transfer are listed together
- **THEN** each row shows its own account and an opposite-signed amount,
  and the two are not mistakable for two separate templates

#### Scenario: A category-less transfer renders cleanly

- **WHEN** a `self_transfer` recurring transaction with no category is
  listed
- **THEN** its category cell is empty and the row renders normally

## MODIFIED Requirements

### Requirement: Creating and editing a recurring transaction

`/recurring/new` and `/recurring/{id}/edit` SHALL present a kind selector
offering "Transaction" and "Self-transfer", plus the content fields the
entry create/edit form presents for that kind (account, title,
description, category, counterparty, location, tags, signed amount), plus
the recurrence rule: a preset picker (at least Weekly, Every 2 weeks,
Monthly, Every 2 months, Quarterly, Every 6 months, Yearly, and a Custom
option) that maps to the underlying `interval_unit`/`interval_count` pair,
a `starts_on` date, and an optional `ends_on` date. Selecting "Custom"
SHALL reveal direct `interval_unit`/`interval_count` inputs for a
combination not covered by a preset (e.g. every 10 days).

Selecting "Self-transfer" SHALL reveal a "To account" picker and hide
counterparty and location, and SHALL stop requiring a category — mirroring
what `/entries/new` already does for the same kind. The "To account"
picker SHALL offer only accounts the visitor holds at least `append`
permission on that are not disabled, share the selected source account's
currency, and are not the source account itself. On `/recurring/{id}/edit`
the kind selector and the "To account" picker SHALL be shown as
immutable — the backend rejects changing either — and the whole form SHALL
be read-only when the visitor no longer holds `append`+ on both accounts,
with an explanation rather than a save that fails.

#### Scenario: Preset maps to the underlying fields

- **WHEN** the visitor picks "Quarterly" on the create form
- **THEN** the submitted request has `interval_unit: month`,
  `interval_count: 3`

#### Scenario: Custom reveals raw inputs

- **WHEN** the visitor picks "Custom"
- **THEN** direct `interval_unit` and `interval_count` inputs are shown,
  editable to any valid combination

#### Scenario: Self-transfer reveals the to-account picker

- **WHEN** the visitor selects "Self-transfer" on `/recurring/new`
- **THEN** a "To account" picker appears, counterparty and location are
  hidden, and the form submits without a category

#### Scenario: The to-account picker excludes unusable accounts

- **WHEN** the visitor has selected a source account in EUR and opens the
  "To account" picker
- **THEN** the source account itself, any disabled account, any account
  they hold only `view` on, and any account in another currency are not
  offered

#### Scenario: Kind is not editable

- **WHEN** the visitor opens `/recurring/{id}/edit` for a `self_transfer`
  recurring transaction
- **THEN** the kind and "To account" are shown but cannot be changed

### Requirement: Creating a transaction from a recurring transaction is always a manual action

Each recurring transaction (on the list and on its edit page) SHALL offer a
"Create transaction" action. Activating it SHALL navigate to
`/entries/new` prefilled from the template — account, kind, title,
description, category, counterparty, location, tags, signed amount, and,
for a `self_transfer`, the receiving account — with `booking_timestamp`
prefilled to the recurring transaction's `next_suggested_date`, and with
`recurring_transaction_id` carried through so the entry is linked once
submitted. The prefill SHALL always resolve the template in its stored
orientation, so activating the action on an incoming (receiving-side) row
still creates an entry whose `account_id` is the template's sending
account — the orientation the link is validated against. Every prefilled
field SHALL remain editable before submission, and no entry SHALL be
created without the visitor explicitly submitting that form — there is no
automatic or scheduled creation.

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

#### Scenario: Creating from a self-transfer template

- **WHEN** the visitor activates "Create transaction" on a `self_transfer`
  recurring transaction
- **THEN** the entry form opens with kind "Self-transfer", both accounts
  prefilled, and submitting it creates a linked `self_transfer` entry

#### Scenario: Creating from the incoming side of a transfer

- **WHEN** the visitor activates "Create transaction" on the receiving-side
  row of a `self_transfer` recurring transaction shown with both sides
- **THEN** the created entry's `account_id` is the template's sending
  account, its `to_account_id` the receiving one, and the link is accepted

#### Scenario: No background or scheduled creation

- **WHEN** a recurring transaction's `next_suggested_date` has passed with
  no visitor action taken
- **THEN** no entry is created automatically — the list still shows the
  template with its (past) `next_suggested_date`, unchanged until the
  visitor manually acts

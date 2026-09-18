## ADDED Requirements

### Requirement: A transaction can be converted to a self-transfer from a dedicated route

`/entries/{id}/edit` SHALL offer a "Transfer to self-transfer" action
whenever the entry is a `kind: transaction` and the visitor otherwise
qualifies to edit it (the same permission rule the edit form's own fields
already require). Activating it SHALL navigate to a new, dedicated route
(distinct from the edit form itself — not a mode-switch within it) that
offers: a counterparty-account picker (offering only non-deleted,
non-disabled accounts the visitor holds at least `append` permission on,
sharing the entry's account's `currency`, excluding the entry's own
account), and an explicit choice of which role the entry's own account
plays in the resulting transfer — sender or receiver — with no default
inferred from the entry's current amount sign. Submitting calls `POST
/api/entries/{id}/self-transfer`. On success, the visitor SHALL be
navigated to `/entries?last=true` (see "Returning to the entry ledger
restores the last-applied filters"). The action SHALL NOT be offered for
an entry whose `kind` is already `balance_adjustment` or `self_transfer`.

#### Scenario: The conversion action appears on an editable transaction

- **WHEN** a visitor with sufficient permission opens `/entries/{id}/edit`
  for a `kind: transaction` entry
- **THEN** a "Transfer to self-transfer" action is shown

#### Scenario: The conversion action is absent for other kinds

- **WHEN** a visitor opens `/entries/{id}/edit` for a `kind:
  balance_adjustment` or `kind: self_transfer` entry
- **THEN** no "Transfer to self-transfer" action is shown

#### Scenario: The conversion action is absent on a read-only entry

- **WHEN** a visitor without sufficient permission to edit a `kind:
  transaction` entry opens `/entries/{id}/edit`
- **THEN** no "Transfer to self-transfer" action is shown

#### Scenario: The conversion route requires an explicit sender/receiver choice

- **WHEN** a visitor opens the conversion route for a `kind: transaction`
  entry
- **THEN** neither "sender" nor "receiver" is preselected, and submitting
  without choosing one shows a validation error rather than defaulting

#### Scenario: A successful conversion returns to the entries list with prior filters restored

- **WHEN** a visitor completes a conversion from `/entries` filtered to a
  specific category
- **THEN** they are navigated to `/entries?last=true`, and the category
  filter that was active before is applied again

### Requirement: Returning to the entry ledger restores the last-applied filters

`/entries` SHALL persist its current filter, search, and sort state (the
same fields carried in its URL search parameters — see "The entry ledger's
filter, search, and sort state lives in the URL") to browser-local storage
whenever that state changes. A navigation to `/entries?last=true`, with no
other search parameter present, SHALL resolve to that persisted state — or
to a bare, unfiltered `/entries` if nothing has been persisted yet —
replacing the URL rather than leaving the `?last=true` hop in browser
history. A navigation to `/entries` with `last=true` present alongside any
other filter parameter SHALL ignore `last` entirely; the explicit filter
parameters SHALL apply exactly as they would without `last` present.
Saving an edit on `/entries/{id}/edit` SHALL navigate to
`/entries?last=true` on success, replacing today's narrower redirect that
restores only `account_id`.

#### Scenario: Filters persist across a save-and-return round trip

- **WHEN** a visitor applies a tag filter and a custom date range on
  `/entries`, opens an entry, and saves it
- **THEN** they return to `/entries` with the same tag filter and date
  range applied, not just the account

#### Scenario: No persisted state falls back to the default view

- **WHEN** `/entries?last=true` is opened with nothing yet persisted (for
  example, a visitor's first visit in a fresh browser profile)
- **THEN** the entries list renders its ordinary unfiltered default view

#### Scenario: An explicit filter alongside last takes precedence

- **WHEN** `/entries?last=true&account_id={id}` is opened (a combination
  no part of this application produces on its own)
- **THEN** the list applies the `account_id` filter as usual and does not
  restore any other persisted filter

#### Scenario: The last=true hop does not linger in browser history

- **WHEN** `/entries?last=true` resolves to a persisted (or default)
  filter state
- **THEN** the resulting URL replaces the `?last=true` entry rather than
  adding a new history entry, so the browser's back button does not return
  to the bare `?last=true` URL

## MODIFIED Requirements

### Requirement: Creating and editing an entry

`/entries/new` SHALL offer a form for account (pre-selected and, when
arrived at via `?account_id=`, not changeable in that flow; otherwise a
choice among the visitor's own non-deleted accounts *and* every
non-deleted shared account the visitor holds at least `append` permission
on), kind, amount (entered and displayed at full stored precision,
independent of the visitor's display-rounding preference), booking
timestamp, title, description, a category (required unless kind is
`balance_adjustment` or `self_transfer`), and tags. When kind is
`transaction`, the form SHALL additionally offer a `counterparty` text
input (suggested values from `GET /api/entries/counterparties` via a
`<datalist>`) and a `location` field (a plain text input, editable
directly, alongside a "Use GPS" button and a "Pick on map" button — see the
location requirements below); both are hidden when kind is
`balance_adjustment` or `self_transfer`, mirroring how category is treated.
When kind is `self_transfer`, the form SHALL additionally offer a "To
account" `<select>`, offering only non-deleted, non-disabled accounts the
visitor holds at least `append` permission on, sharing the same `currency`
as the currently selected "from" account, and excluding whichever account
is currently selected as "from" — narrowed live as the "from" account
changes. Submitting calls `POST /api/entries`.

`/entries/{id}/edit` SHALL offer the same form pre-populated from the
existing entry, with kind rendered read-only (immutable per the backend)
and account rendered locked by default, changeable only through the
unlock interaction described below, submitting an update on save. The edit
page SHALL offer its full form, including the delete action behind a
confirmation step, only to a visitor with `entry_admin`+ permission on the
entry's account, or with `append` permission and `created_by` matching
them; a visitor with `view` permission, or `append` permission on an entry
they did not create, SHALL instead see the entry's details as read-only,
with no edit or delete action. For a `self_transfer` entry specifically,
the full editable form additionally requires the visitor to currently hold
at least `append` permission on both `account_id` and `to_account_id` — a
visitor who qualifies under the general rule above but has since lost
access to the other account SHALL still see the entry read-only, with no
edit action; the delete action, when otherwise available, is unaffected by
this narrower rule (see "A self-transfer's delete action does not require
the other account").

#### Scenario: Creating a transaction requires a category

- **WHEN** an authenticated visitor submits the create form with
  `kind: transaction` and no category selected
- **THEN** the form shows a validation error and does not submit

#### Scenario: Creating a balance adjustment allows an empty category

- **WHEN** an authenticated visitor submits the create form with
  `kind: balance_adjustment` and no category selected
- **THEN** the entry is created successfully

#### Scenario: Creating a self-transfer allows an empty category

- **WHEN** an authenticated visitor submits the create form with
  `kind: self_transfer`, a "to account" selected, and no category selected
- **THEN** the entry is created successfully

#### Scenario: Counterparty and location are hidden for a balance adjustment

- **WHEN** an authenticated visitor selects `kind: balance_adjustment` on
  either the create or edit form
- **THEN** the counterparty and location fields are not shown, and neither
  is included in the submitted request

#### Scenario: Counterparty and location are hidden for a self-transfer

- **WHEN** an authenticated visitor selects `kind: self_transfer` on
  either the create or edit form
- **THEN** the counterparty and location fields are not shown, and neither
  is included in the submitted request

#### Scenario: Counterparty input suggests the visitor's own history

- **WHEN** an authenticated visitor focuses the counterparty field on a
  `kind: transaction` form
- **THEN** the suggestion list is populated from
  `GET /api/entries/counterparties`

#### Scenario: Account and kind are not editable

- **WHEN** an authenticated visitor opens `/entries/{id}/edit`
- **THEN** the account and kind fields are shown but cannot be changed —
  the account field's unlock interaction (see below) is a separate,
  deliberate action, not a default-editable state

#### Scenario: Deleting an entry requires confirmation

- **WHEN** an authenticated visitor with sufficient permission activates
  "Delete" on the edit page
- **THEN** a confirmation step appears and the delete request is not sent
  until it is confirmed

#### Scenario: The new-entry account picker includes shared accounts

- **WHEN** a visitor with `append`+ permission on a shared account opens
  `/entries/new` with no preset `account_id`
- **THEN** that account appears among the choosable accounts

#### Scenario: A view-tier visitor sees an entry read-only

- **WHEN** a visitor with only `view` permission on a shared account opens
  one of its entries
- **THEN** the entry's details render read-only, with no edit or delete
  action

#### Scenario: An append-tier visitor cannot edit another user's entry

- **WHEN** a visitor with `append` permission opens an entry on the same
  account that a different user created
- **THEN** the entry's details render read-only, with no edit or delete
  action

#### Scenario: Selecting self-transfer reveals the "to account" picker

- **WHEN** an authenticated visitor selects `kind: self_transfer` on
  `/entries/new`
- **THEN** a "To account" `<select>` appears, offering only append+,
  non-disabled accounts sharing the "from" account's currency, excluding
  the "from" account itself

#### Scenario: The "to account" picker excludes different-currency accounts

- **WHEN** an authenticated visitor has `append`+ permission on an account
  of a different currency than the selected "from" account
- **THEN** that account does not appear among the "to account" choices

#### Scenario: A self-transfer missing permission on the other account is read-only

- **WHEN** a visitor who meets the general edit-permission rule for an
  entry's `account_id` opens a `kind: self_transfer` entry whose
  `to_account_id` they no longer have `append`+ permission on
- **THEN** the entry's details render read-only, with no edit action

## ADDED Requirements

### Requirement: A self-transfer entry shows a badge linking to its other account's leg

`/entries` and the `/reports` results table SHALL show a small icon/badge
on any `kind: self_transfer` entry row, distinct from the row's other
content, that navigates to `/entries/{id}/edit` for that same entry when
activated — the same entry, viewed from its other account. A `self_transfer`
rendered once (only one of its two accounts in the current view's scope)
still shows the badge; the badge does not depend on the entry appearing
twice.

#### Scenario: A self-transfer row shows the badge

- **WHEN** the ledger or reports table lists a `kind: self_transfer` entry
- **THEN** its row shows the self-transfer badge, naming the other account

#### Scenario: Activating the badge opens the entry's edit page

- **WHEN** the visitor activates a self-transfer row's badge
- **THEN** the browser navigates to that entry's own `/entries/{id}/edit`
  page

#### Scenario: A non-transfer entry shows no self-transfer badge

- **WHEN** the ledger or reports table lists a `transaction` or
  `balance_adjustment` entry
- **THEN** no self-transfer badge is shown on its row

### Requirement: A self-transfer's delete action does not require the other account

The edit page's delete action for a `kind: self_transfer` entry SHALL
follow the same permission rule as any other entry's delete action
(`entry_admin`/`owner` on the accessible account, or `append` with
`created_by` matching the visitor), evaluated only against whichever of
the entry's two accounts the visitor currently has permission on. Losing
access to the *other* account SHALL NOT hide or disable the delete action,
even though it does hide the edit action (see "Creating and editing an
entry").

#### Scenario: Delete remains available after losing access to the other account

- **WHEN** a visitor holds `entry_admin`/`owner` permission on one account
  of a `self_transfer` entry and has since lost all permission on the
  other
- **THEN** the edit page still shows the entry's fields read-only (no edit
  action) but keeps the Delete action available

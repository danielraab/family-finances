## MODIFIED Requirements

### Requirement: Creating and editing an entry

`/entries/new` SHALL offer a form for account (pre-selected and, when
arrived at via `?account_id=`, not changeable in that flow; otherwise a
choice among the visitor's own non-deleted accounts *and* every
non-deleted shared account the visitor holds at least `append` permission
on), kind, amount (entered and displayed at full stored precision,
independent of the visitor's display-rounding preference), booking
timestamp, title, description, a category (required unless kind is
`balance_adjustment`), and tags. Submitting calls `POST /api/entries`.
`/entries/{id}/edit` SHALL offer the same form pre-populated from the
existing entry, with kind rendered read-only (immutable per the backend)
and account rendered locked by default, changeable only through the
unlock interaction described below, submitting an update on save. The edit
page SHALL offer its full form, including the delete action behind a
confirmation step, only to a visitor with `entry_admin`+ permission on the
entry's account, or with `append` permission and `created_by` matching
them; a visitor with `view` permission, or `append` permission on an entry
they did not create, SHALL instead see the entry's details as read-only,
with no edit or delete action.

#### Scenario: Creating a transaction requires a category

- **WHEN** an authenticated visitor submits the create form with
  `kind: transaction` and no category selected
- **THEN** the form shows a validation error and does not submit

#### Scenario: Creating a balance adjustment allows an empty category

- **WHEN** an authenticated visitor submits the create form with
  `kind: balance_adjustment` and no category selected
- **THEN** the entry is created successfully

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

## ADDED Requirements

### Requirement: An entry not created by the current visitor shows who created it

Wherever an entry is rendered — the ledger (`/entries`), an account's
recent-entries list, and a single entry's details — a small annotation
naming the creator SHALL be shown whenever that entry's `created_by`
differs from the current visitor, using the `created_by_name` the backend
already resolves. An entry the visitor created themselves SHALL render
exactly as before this capability, with no such annotation.

#### Scenario: An entry created by someone else shows their name

- **WHEN** the entry ledger, an account's recent entries, or an entry's
  detail view renders an entry whose `created_by` differs from the current
  visitor
- **THEN** that row shows the creator's name

#### Scenario: A self-created entry shows no creator annotation

- **WHEN** any of those views renders an entry the current visitor created
  themselves
- **THEN** no creator annotation is shown, unchanged from before this
  capability existed

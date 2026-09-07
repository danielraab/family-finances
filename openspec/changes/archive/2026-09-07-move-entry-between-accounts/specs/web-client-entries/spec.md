## MODIFIED Requirements

### Requirement: Creating and editing an entry

`/entries/new` SHALL offer a form for account (pre-selected and, when
arrived at via `?account_id=`, not changeable in that flow; otherwise a
choice among the visitor's own non-deleted accounts), kind, amount (entered
and displayed at full stored precision, independent of the visitor's
display-rounding preference), booking timestamp, title, description, a
category (required unless kind is `balance_adjustment`), and tags.
Submitting calls `POST /api/entries`. `/entries/{id}/edit` SHALL offer the
same form pre-populated from the existing entry, with kind rendered
read-only (immutable per the backend) and account rendered locked by
default, changeable only through the unlock interaction described below,
submitting an update on save, and SHALL offer a delete action behind a
confirmation step.

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

- **WHEN** an authenticated visitor activates "Delete" on the edit page
- **THEN** a confirmation step appears and the delete request is not sent
  until it is confirmed

## ADDED Requirements

### Requirement: The edit form's account field is locked by default and unlocked via a button

On `/entries/{id}/edit`, the account field SHALL render, by default, as a
disabled display of the entry's current account with an adjacent icon
button (a pencil, labeled for assistive technology as changing the
account) beside it. Activating that button SHALL replace the disabled
display with a `<select>` listing the visitor's own non-deleted,
non-disabled accounts, alongside a second button that discards any pending
selection and returns the field to its locked, disabled display showing
the entry's original account. If the entry's current account has since
been disabled, it SHALL still render as the unlocked field's current,
selected value (mirroring the category picker's handling of a disabled
current category) but SHALL NOT appear as a choosable option once a
different account has been selected.

#### Scenario: Unlocking the account field

- **WHEN** an authenticated visitor activates the account field's unlock
  button on `/entries/{id}/edit`
- **THEN** the field becomes a `<select>` of the visitor's own non-deleted,
  non-disabled accounts, defaulted to the entry's current account

#### Scenario: Canceling a pending account change

- **WHEN** an authenticated visitor has unlocked the account field,
  selected a different account, and then activates the cancel button
  instead of saving
- **THEN** the field reverts to its locked, disabled display showing the
  entry's original account, and no request is sent

#### Scenario: A disabled current account still renders while unlocked

- **WHEN** an authenticated visitor unlocks the account field on an entry
  whose current account has since been disabled
- **THEN** that account renders as the field's current, selected value,
  but does not appear among the selectable options

### Requirement: Selecting a different-currency account shows a warning

On `/entries/{id}/edit`, once the account field is unlocked, selecting an
account whose `currency` differs from the entry's original account SHALL
show an inline warning next to the field, without preventing the
selection.

#### Scenario: Selecting a same-currency account shows no warning

- **WHEN** an authenticated visitor selects an account whose `currency`
  matches the entry's original account
- **THEN** no currency warning is shown

#### Scenario: Selecting a different-currency account shows a warning

- **WHEN** an authenticated visitor selects an account whose `currency`
  differs from the entry's original account
- **THEN** an inline warning is shown next to the account field, and the
  selection is not prevented

### Requirement: Saving a changed account requires confirmation

On `/entries/{id}/edit`, submitting the form with the account field's
selection unchanged from the entry's original `account_id` SHALL submit
the update immediately, the same as any other field change. Submitting
with a changed `account_id` SHALL first show a confirmation dialog naming
the destination account — repeating the currency-mismatch warning when
applicable — before any request is sent. Confirming SHALL submit the
update, including any other fields edited in the same visit; canceling
SHALL return to the form without sending a request.

#### Scenario: Saving without an account change skips confirmation

- **WHEN** an authenticated visitor saves the edit form without having
  changed the account field
- **THEN** the update is submitted immediately, with no confirmation
  dialog shown

#### Scenario: Saving a changed account shows a confirmation dialog

- **WHEN** an authenticated visitor saves the edit form after selecting a
  different account
- **THEN** a confirmation dialog appears naming the destination account
  before any request is sent

#### Scenario: Confirming submits the whole form

- **WHEN** an authenticated visitor confirms the account-change dialog
- **THEN** the update is submitted with the new `account_id` and any other
  fields edited in the same visit

#### Scenario: Canceling the confirmation submits nothing

- **WHEN** an authenticated visitor dismisses the account-change
  confirmation dialog without confirming
- **THEN** no request is sent and the form remains as it was, with the
  changed account selection still pending

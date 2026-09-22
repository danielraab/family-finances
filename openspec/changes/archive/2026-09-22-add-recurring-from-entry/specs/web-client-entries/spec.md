## ADDED Requirements

### Requirement: A transaction entry offers a shortcut to create a recurring transaction from it

`/entries/{id}/edit` SHALL offer a compact, icon-only action that starts a
new recurring transaction template prefilled from the current entry,
visible only when the entry's `kind` is `transaction` and the visitor has
the same edit permission the rest of the form requires (`entry_admin`+ on
the account, or `append` and `created_by` matching the visitor). It SHALL
NOT be shown for a `kind: balance_adjustment` entry, and SHALL NOT be shown
to a visitor who only sees the entry read-only.

The action's accessible name (and visible label, if any) SHALL depend on
whether the form has unsaved edits since it was loaded or last saved:

- With no unsaved edits, activating it SHALL navigate directly to
  `/recurring/new?from_entry_id=<id>` without submitting anything.
- With unsaved edits, its label SHALL indicate that saving happens first
  (e.g. "Save and create recurring transaction"). Activating it SHALL run
  the same validation and save request the entry form's normal Save action
  runs — including the account-change confirmation step when the account
  was changed — and, only on a successful save, SHALL navigate to
  `/recurring/new?from_entry_id=<id>` instead of the normal Save action's
  redirect to `/entries`.

#### Scenario: Action hidden for a balance adjustment

- **WHEN** the visitor opens `/entries/{id}/edit` for an entry with
  `kind: balance_adjustment`
- **THEN** the "create recurring transaction" action is not shown

#### Scenario: Action hidden for a read-only visitor

- **WHEN** a visitor with only `view` permission, or `append` permission on
  an entry they did not create, opens the entry
- **THEN** the "create recurring transaction" action is not shown

#### Scenario: Clean form navigates directly

- **WHEN** the visitor opens `/entries/{id}/edit` for a transaction entry,
  makes no changes, and activates the action
- **THEN** the browser navigates to `/recurring/new?from_entry_id=<id>`
  with no save request sent

#### Scenario: Dirty form saves first, then navigates

- **WHEN** the visitor changes a field on the entry form and activates the
  action
- **THEN** `PATCH /api/entries/{id}` is called with the changed values, and
  only after it succeeds does the browser navigate to
  `/recurring/new?from_entry_id=<id>`

#### Scenario: A failed save from this action does not navigate

- **WHEN** the visitor activates the action on a dirty form and the save
  request fails
- **THEN** the entry form shows its normal save error and the browser does
  not navigate to `/recurring/new`

#### Scenario: Account change confirmation still applies

- **WHEN** the visitor changes the entry's account and activates the
  action
- **THEN** the same account-change confirmation dialog the normal Save
  action shows is presented before the save request is sent

## REMOVED Requirements

### Requirement: Account Types tab is admin-only

**Reason**: the Account Types tab is no longer admin-gated — see the
modified "Account Types tab lists, creates, edits, disables/enables, and
deletes account types" requirement below, which now applies to every
authenticated visitor.

**Migration**: none — dropping the `is_admin` gate only widens who can see
and use the tab; no existing data or behavior for an admin changes.

## MODIFIED Requirements

### Requirement: Account Types tab lists, creates, edits, disables/enables, and deletes account types

The settings page SHALL offer an **Account Types** tab to every
authenticated visitor, positioned alongside Common and My Invitations —
not gated on `is_admin`. The tab SHALL list only the caller's own account
types (`GET /api/account-types`) showing each type's title, description,
and an Active/Disabled status, with actions to create a new type, edit an
existing type's title and description, disable or enable it, and delete
it. Each state-changing action SHALL be confirmed via the same
`@headlessui/react` `Dialog` pattern already used by the Users tab. A
delete rejected by the backend (`409`, still referenced by one of the
caller's own accounts) SHALL surface an inline error rather than silently
doing nothing.

#### Scenario: Any authenticated visitor sees the Account Types tab

- **WHEN** an authenticated visitor (admin or not) opens `/settings`
- **THEN** the Account Types tab is shown in the tab list

#### Scenario: A visitor creates a new account type

- **WHEN** an authenticated visitor submits the create form with a title
  and a description
- **THEN** `POST /api/account-types` is called and the new type appears in
  the list, Active

#### Scenario: A visitor disables a type in use

- **WHEN** an authenticated visitor disables a type that their own
  accounts reference
- **THEN** the type's status shows Disabled, and those accounts are
  unaffected

#### Scenario: Deleting an in-use type shows an error

- **WHEN** an authenticated visitor attempts to delete a type still
  referenced by one of their own accounts
- **THEN** the backend's `409` surfaces as an inline error and the type
  remains in the list

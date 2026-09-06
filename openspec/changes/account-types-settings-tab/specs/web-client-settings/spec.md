## ADDED Requirements

### Requirement: Account Types tab is admin-only

The settings page SHALL offer an **Account Types** tab only when the
authenticated visitor's `is_admin` is `true`; for a non-admin, the tab
SHALL NOT be rendered in the tab list, and navigating directly to
`/settings/account-types` SHALL redirect to `/settings` rather than
rendering the tab's content — the same defense-in-depth pattern the
existing Users tab uses.

#### Scenario: Non-admin does not see the Account Types tab

- **WHEN** a non-admin authenticated visitor opens `/settings`
- **THEN** the Account Types tab is not shown in the tab list

#### Scenario: Non-admin is redirected away from a direct link

- **WHEN** a non-admin authenticated visitor navigates directly to
  `/settings/account-types`
- **THEN** the client redirects them to `/settings`

### Requirement: Account Types tab lists, creates, edits, disables/enables, and deletes account types

The Account Types tab SHALL list every account type (`GET
/api/account-types`) showing its title, description, and an Active/Disabled
status, with actions to create a new type, edit an existing type's title and
description, disable or enable it, and delete it. Each state-changing
action SHALL be confirmed via the same `@headlessui/react` `Dialog` pattern
already used by the Users tab. A delete rejected by the backend (`409`,
still referenced by an account) SHALL surface an inline error rather than
silently doing nothing.

#### Scenario: Admin creates a new account type

- **WHEN** an admin submits the create form with a title and a description
- **THEN** `POST /api/account-types` is called and the new type appears in
  the list, Active

#### Scenario: Admin disables a type in use

- **WHEN** an admin disables a type that existing accounts reference
- **THEN** the type's status shows Disabled, and those accounts are
  unaffected

#### Scenario: Deleting an in-use type shows an error

- **WHEN** an admin confirms deleting a type that a non-deleted account
  still references
- **THEN** the request fails and the tab shows an inline error instead of
  removing the type from the list

### Requirement: The account form only offers live types for a new assignment

The account create/edit form's type selector SHALL exclude disabled types
when choosing a type for a new account or changing an existing account's
type. If the account being edited currently holds a disabled type, that
type SHALL still be shown (labeled distinctly, e.g. as disabled) so the
form does not appear to have lost the account's data, but it SHALL NOT be
resubmittable as the account's type — the form SHALL require a different,
non-disabled selection before the edit can be saved, mirroring how a blank
required field already blocks submission.

#### Scenario: Creating an account only offers live types

- **WHEN** an authenticated visitor opens the new-account form
- **THEN** the type dropdown lists only non-disabled account types

#### Scenario: Editing an account on a disabled type forces reselection

- **WHEN** an authenticated visitor edits an account whose current type is
  disabled
- **THEN** the form shows that type as the current, non-selectable value
  and blocks saving until a different, non-disabled type is chosen

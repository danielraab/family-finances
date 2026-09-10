## MODIFIED Requirements

### Requirement: Tags tab lists, creates, renames, disables/enables, and deletes tags

The settings page SHALL offer a **Tags** tab to every authenticated
visitor, positioned alongside Common and My Invitations — not gated on
`is_admin`. The tab SHALL list only the caller's own tags
(`GET /api/tags`) showing each tag's name, its `entry_count`, and an
Active/Disabled status, with actions to create a new tag, rename an
existing tag, disable or enable it, and delete it.

#### Scenario: Any authenticated visitor sees the Tags tab

- **WHEN** an authenticated visitor (admin or not) opens `/settings`
- **THEN** the Tags tab is shown in the tab list

#### Scenario: A visitor creates a new tag

- **WHEN** an authenticated visitor submits the create form with a name
- **THEN** `POST /api/tags` is called and the new tag appears in the list,
  Active, with an entry count of `0`

#### Scenario: A visitor renames a tag

- **WHEN** an authenticated visitor edits an existing tag's name and saves
- **THEN** `PATCH /api/tags/{id}` is called and the list reflects the new
  name

#### Scenario: The list shows how many entries use each tag

- **WHEN** a tag is currently attached to three of the visitor's entries
- **THEN** its row in the Tags tab shows `3`

### Requirement: Deleting a tag requires confirmation

Each tag row's delete action SHALL require an explicit confirmation step,
via the same `@headlessui/react` `Dialog` pattern used elsewhere in
`/settings`, before `DELETE /api/tags/{id}` is called. Unlike categories,
a tag delete is never blocked by use — the confirmation copy SHALL state
that the tag will be removed from every entry that currently carries it.

#### Scenario: Confirmation blocks an accidental delete

- **WHEN** an authenticated visitor activates "Delete" for a listed tag
- **THEN** a confirmation step appears and `DELETE /api/tags/{id}` is not
  called until it is confirmed

#### Scenario: Deleting a tag in use

- **WHEN** an authenticated visitor confirms deleting a tag currently
  attached to one or more of their entries
- **THEN** `DELETE /api/tags/{id}` is called, the tag is removed from the
  list, and it no longer appears on any entry

## REMOVED Requirements

### Requirement: Account Types tab lists, creates, edits, disables/enables, and deletes account types

**Reason**: The `account_types` entity and its CRUD API are removed; an
account's type is now a free-text field with no managed lifecycle, so
there is nothing for a settings tab to manage.

**Migration**: The `/settings/account-types` route and its entry in the
settings tab navigation are deleted, along with the `settings.accountTypes.*`
i18n keys in every locale. Users classify accounts by typing or picking a
`type` on the account create/edit form (see `web-client-accounts`), which
offers the former default titles plus their in-use values as suggestions.

### Requirement: The account form only offers live types for a new assignment

**Reason**: There is no longer a disabled/enabled distinction on account
types — `type` is plain text — so there is nothing to exclude from the
form or force reselection of.

**Migration**: The account create/edit form's type control becomes a
required free-text input with suggestions (see `web-client-accounts`'
"Creating and editing an account"). No account's stored `type` is affected;
any value that was a disabled type's title is simply shown as ordinary
text and remains resubmittable.

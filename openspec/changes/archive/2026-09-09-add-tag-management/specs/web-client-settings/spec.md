## ADDED Requirements

### Requirement: Tags tab lists, creates, renames, disables/enables, and deletes tags

The settings page SHALL offer a **Tags** tab to every authenticated
visitor, positioned alongside Common, My Invitations, and Account Types —
not gated on `is_admin`. The tab SHALL list only the caller's own tags
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

### Requirement: Disabling and enabling a tag apply immediately, without a confirmation dialog

Each tag row SHALL offer a toggle calling `POST /api/tags/{id}/disable` or
`/enable` as appropriate, applied immediately on activation — unlike
delete, disabling or enabling a tag SHALL NOT require a confirmation step,
since it is cheaply reversible and does not affect any entry already
carrying the tag.

#### Scenario: Disabling a tag in use

- **WHEN** an authenticated visitor disables a tag that existing entries
  currently carry
- **THEN** the tag's status shows Disabled immediately, with no
  confirmation step, and those entries remain visible and functional
  elsewhere in the app

### Requirement: Deleting a tag requires confirmation

Each tag row's delete action SHALL require an explicit confirmation step,
via the same `@headlessui/react` `Dialog` pattern used elsewhere in
`/settings`, before `DELETE /api/tags/{id}` is called. Unlike account types
or categories, delete is never blocked by use — the confirmation copy
SHALL state that the tag will be removed from every entry that currently
carries it.

#### Scenario: Confirmation blocks an accidental delete

- **WHEN** an authenticated visitor activates "Delete" for a listed tag
- **THEN** a confirmation step appears and `DELETE /api/tags/{id}` is not
  called until it is confirmed

#### Scenario: Deleting a tag in use

- **WHEN** an authenticated visitor confirms deleting a tag currently
  attached to one or more of their entries
- **THEN** `DELETE /api/tags/{id}` is called, the tag is removed from the
  list, and it no longer appears on any entry

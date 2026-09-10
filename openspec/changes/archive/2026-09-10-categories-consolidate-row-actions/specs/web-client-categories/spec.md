## MODIFIED Requirements

### Requirement: Reparenting via a "Move to…" picker, not drag-and-drop

The category **edit dialog** SHALL include a parent-category picker listing
the caller's own tree (including an option to make the category a root
category), rather than a standalone "Move to…" dialog or a drag gesture.
The picker SHALL exclude the category itself and its own descendants as a
UX nicety. Saving the edit dialog SHALL send a single
`PATCH /api/categories/{id}` carrying the category's `name`, `icon`,
`color`, and `parent_id` together; when the chosen parent is unchanged the
request MAY omit `parent_id` or send its current value. A `422` from a
self/descendant reparent that slips past the client filter SHALL surface as
an inline error in the dialog without corrupting the displayed tree. The
inline node row SHALL NOT carry a reparent control.

#### Scenario: Reparenting to a different category

- **WHEN** an authenticated visitor opens a category's edit dialog, picks a
  new parent, and saves
- **THEN** `PATCH /api/categories/{id}` is called with that `parent_id` and
  the tree re-renders with the category under its new parent

#### Scenario: Making a category a root category

- **WHEN** an authenticated visitor opens a category's edit dialog, picks
  "make root" from the parent picker, and saves
- **THEN** `PATCH /api/categories/{id}` is called with a null `parent_id`
  and the category renders as a top-level node

#### Scenario: The category's own subtree is not offered as a parent

- **WHEN** the parent picker is shown for a category that has descendants
- **THEN** neither the category itself nor any of its descendants appears as
  a selectable parent option

### Requirement: Disabling and enabling a category

The category **edit dialog** SHALL offer a Disable action (or Enable, when
the category is already disabled) that calls
`POST /api/categories/{id}/disable` or `/enable` as appropriate and applies
**immediately, with no confirmation step**, updating the category's
displayed status on success. The inline node row SHALL NOT carry a
disable/enable control.

#### Scenario: Disabling a category in use

- **WHEN** an authenticated visitor opens a category's edit dialog and
  activates Disable, where existing entries or child categories reference
  that category
- **THEN** the request is sent with no confirmation prompt, the category's
  status shows disabled, and those entries and child categories remain
  visible and functional elsewhere in the app

#### Scenario: Enabling a disabled category

- **WHEN** an authenticated visitor opens the edit dialog of a disabled
  category and activates Enable
- **THEN** `POST /api/categories/{id}/enable` is called with no
  confirmation prompt and the category's status shows enabled

### Requirement: Delete is only offered when the category has no children and no entry references

The category **edit dialog** SHALL offer a Delete action, disabled
(visibly, e.g. greyed out with an explanation) whenever the category
currently has any child category, re-evaluated from the same tree data the
page already has — without a separate request per node. Deleting an
eligible category calls `DELETE /api/categories/{id}` behind a confirmation
step; on success the edit dialog closes and the node is removed from the
tree. A `409` response (the category became in-use since the page loaded)
SHALL surface as an inline error rather than silently removing the node.
The inline node row SHALL NOT carry a delete control.

#### Scenario: Delete is unavailable on a category with children

- **WHEN** a visitor opens the edit dialog of a category that has at least
  one child category
- **THEN** the dialog's Delete action is shown disabled

#### Scenario: Deleting an eligible category

- **WHEN** an authenticated visitor opens the edit dialog of a category
  with no children and no entry references and confirms deletion
- **THEN** `DELETE /api/categories/{id}` is called, the dialog closes, and
  the category is removed from the tree

#### Scenario: Deleting a category still referenced by an entry

- **WHEN** the confirmed delete returns `409`
- **THEN** an inline error is shown and the node remains in the tree

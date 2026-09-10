# web-client-categories Specification

## Purpose

The authenticated `/categories` page: the sidebar link, the auth gate, the
tree view of the caller's own categories, and the only surface where a
category can be created or edited — including its mobile-friendly,
button-driven controls for renaming, disabling/enabling, reordering
siblings, reparenting, and deleting — plus the flat "Shared with me"
section and the Share entry point into `web-client-category-sharing`. See
`entry-categories`/`category-sharing` for the backend capabilities and
`web-client-entries` for how a category is picked on an entry.

## Requirements

### Requirement: Categories link in the sidebar

The `Sidebar` navigation SHALL contain a "Categories" item, visible to an
authenticated visitor, positioned alongside "Accounts" and "Entries", that
navigates to `/categories` and is shown as active for `/categories` and
every route nested under it.

#### Scenario: Navigating to categories from the sidebar

- **WHEN** an authenticated visitor activates "Categories" in the sidebar
- **THEN** the client navigates to `/categories`

### Requirement: Categories routes require authentication

`/categories` and every route nested under it SHALL be accessible only to
an authenticated visitor. An anonymous visitor navigating to `/categories`
SHALL be redirected to `/login`.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/categories`
- **THEN** the client redirects them to `/login`

### Requirement: The categories page renders the caller's tree, ordered by sibling order

`/categories` SHALL fetch the caller's categories (`GET /api/categories`,
now owned categories plus every category shared with the caller — see
`category-sharing`) and render the caller's own categories as a nested
tree, each node's children ordered by `sort_order`. Every category shared
with the caller SHALL render separately, in a distinct "Shared with me"
section, flat rather than nested (see the requirement below) — it SHALL
NOT be interleaved into the owned tree. The page SHALL indicate a
category's `disabled` status distinctly from an enabled one, SHALL show
each owned node's `entry_count` (the number of the caller's own entries
directly categorized under it) as a distinct, non-interactive indicator
on that node, and SHALL render at any viewport width down to a typical
mobile screen without requiring horizontal scrolling or drag gestures for
any control.

#### Scenario: The tree reflects parent/child structure

- **WHEN** the caller has a category with one or more children
- **THEN** those children are rendered nested under their parent, not as
  separate top-level entries

#### Scenario: Disabled categories are visually distinguished

- **WHEN** a category in the tree has `disabled: true`
- **THEN** it is shown with a visibly different status than an enabled
  category

#### Scenario: Each node shows how many entries use it

- **WHEN** a category is directly referenced by three of the caller's
  entries
- **THEN** that category's node shows `3`

#### Scenario: A category with no entries still shows its count

- **WHEN** a category has an `entry_count` of `0`
- **THEN** its node shows `0` rather than omitting the indicator

### Requirement: Categories can only be created and edited from this page

`/categories` SHALL be the only page offering a form to create a new
category or rename an existing one. Creating calls `POST /api/categories`;
renaming calls `PATCH /api/categories/{id}` with the new `name`. No other
page (including the entry form's category picker) SHALL offer category
creation.

#### Scenario: Creating a category

- **WHEN** an authenticated visitor submits the create form with a name and
  an optional parent
- **THEN** `POST /api/categories` is called and the new category appears in
  the tree, appended after its siblings

#### Scenario: Renaming a category

- **WHEN** an authenticated visitor edits an existing category's name and
  saves
- **THEN** `PATCH /api/categories/{id}` is called and the tree reflects the
  new name

### Requirement: Reordering siblings with move-up/move-down buttons

Each category node SHALL offer ▲/▼ buttons that call
`POST /api/categories/{id}/move-up` and `/move-down` respectively, moving
it one position among its current siblings. A button SHALL be disabled
(or otherwise inert) when the category is already first (▲) or last (▼)
among its siblings.

#### Scenario: Moving a category up

- **WHEN** an authenticated visitor activates the ▲ button on a category
  that is not first among its siblings
- **THEN** `POST /api/categories/{id}/move-up` is called and the category
  renders above its former previous sibling

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

### Requirement: The category create and edit forms offer an icon and colour picker

The inline category create form and the category edit form on
`/categories` SHALL each include the shared `IconColorPicker`, letting the
user set, change, or clear the category's `icon` and `color` independently.
On submit, the chosen values SHALL be sent on the `POST /api/categories` /
`PATCH /api/categories/{id}` body; an unset icon or colour SHALL be sent
such that the field is created without it, and clearing a previously set
field SHALL send the empty string so the backend clears it. Neither field
SHALL be required.

#### Scenario: Creating a category with an icon

- **WHEN** a visitor types a name in the create form, picks an icon, and
  submits
- **THEN** `POST /api/categories` is called with that `icon` and the new
  category carries it

#### Scenario: Changing a category's icon from the edit form

- **WHEN** a visitor opens the edit form for a category, picks a different
  icon, and submits
- **THEN** `PATCH /api/categories/{id}` is called with the new `icon` and
  the tree re-renders showing it

#### Scenario: Clearing a category's icon

- **WHEN** a visitor opens the edit form for a category that has an
  `icon`, clears it in the picker, and submits
- **THEN** `PATCH /api/categories/{id}` is called with `icon` as `""` and
  the category's icon becomes unset

### Requirement: The category tree renders each node's icon and colour before its name

The `/categories` tree SHALL render each category's `EntityIcon` badge
immediately before its name, via the shared `CategoryLabel` component, when
the category has an `icon` and/or `color` set. A category with neither set
SHALL render exactly as before. The badge SHALL appear consistently at every
depth of the tree.

#### Scenario: A nested category shows its badge

- **WHEN** the tree renders a child category that has an `icon` and a
  `color`
- **THEN** that node shows the icon/colour badge immediately before its
  name, aligned consistently with sibling and parent nodes

#### Scenario: A category with no icon or colour is unchanged

- **WHEN** the tree renders a category with neither `icon` nor `color`
- **THEN** that node shows just its name, with no badge

### Requirement: The categories page shows categories shared with the caller in a separate, flat section

`/categories` SHALL render a "Shared with me" section, positioned below
the caller's own tree, listing every category returned by `GET /api/
categories` with `shared: true` — one flat row per category, with no
nesting regardless of that category's real `parent_id`. Each row SHALL
show the category's name (with its icon/colour badge via
`CategoryLabel`, when set), a shared indicator naming the real owner (via
`owner_name`), and the caller's permission tier (`view` or `append`) on
it. This section SHALL be omitted entirely when the caller has no
categories shared with them. See `web-client-category-sharing` for the
per-row Leave action and the linked sharing page.

#### Scenario: A shared category appears in its own section, unnested

- **WHEN** the caller has a category shared with them that is a child
  category in its real owner's tree
- **THEN** it appears as a single row in the "Shared with me" section,
  not nested under any other category

#### Scenario: No section when nothing is shared

- **WHEN** the caller has no categories shared with them
- **THEN** the "Shared with me" section does not render at all

### Requirement: Each of the caller's own categories offers a Share action

Each category in the caller's own tree on `/categories` SHALL offer a
Share action (alongside its existing Edit action) that navigates to
`/categories/{id}/sharing`.

#### Scenario: Opening the sharing page

- **WHEN** the caller activates Share on one of their own categories
- **THEN** the client navigates to `/categories/{id}/sharing`

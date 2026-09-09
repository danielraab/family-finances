# web-client-categories Specification

## Purpose

The authenticated `/categories` page: the sidebar link, the auth gate, the
tree view of the caller's own categories, and the only surface where a
category can be created or edited — including its mobile-friendly,
button-driven controls for renaming, disabling/enabling, reordering
siblings, reparenting, and deleting. See `entry-categories` for the backend
capability and `web-client-entries` for how a category is picked on an
entry.

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

`/categories` SHALL fetch the caller's categories (`GET /api/categories`)
and render them as a nested tree, each node's children ordered by
`sort_order`. The page SHALL indicate a category's `disabled` status
distinctly from an enabled one, and SHALL render at any viewport width down
to a typical mobile screen without requiring horizontal scrolling or drag
gestures for any control.

#### Scenario: The tree reflects parent/child structure

- **WHEN** the caller has a category with one or more children
- **THEN** those children are rendered nested under their parent, not as
  separate top-level entries

#### Scenario: Disabled categories are visually distinguished

- **WHEN** a category in the tree has `disabled: true`
- **THEN** it is shown with a visibly different status than an enabled
  category

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

Each category node SHALL offer a "Move to…" action that opens a picker
listing the caller's own tree (including an option to make the category a
root category), rather than requiring a drag gesture. Selecting a
destination calls `PATCH /api/categories/{id}` with the chosen `parent_id`.
Attempting to move a category onto itself or one of its own descendants
SHALL be prevented or SHALL surface the resulting `422` as an inline error
without corrupting the displayed tree.

#### Scenario: Reparenting to a different category

- **WHEN** an authenticated visitor picks a new parent from the "Move to…"
  picker
- **THEN** `PATCH /api/categories/{id}` is called with that `parent_id` and
  the tree re-renders with the category under its new parent

#### Scenario: Making a category a root category

- **WHEN** an authenticated visitor picks "make root" from the "Move to…"
  picker
- **THEN** `PATCH /api/categories/{id}` is called with a null `parent_id`
  and the category renders as a top-level node

### Requirement: Disabling and enabling a category

Each category node SHALL offer a toggle calling
`POST /api/categories/{id}/disable` or `/enable` as appropriate, updating
the node's displayed status immediately on success.

#### Scenario: Disabling a category in use

- **WHEN** an authenticated visitor disables a category that existing
  entries or child categories currently reference
- **THEN** the category's status shows disabled, and those entries and
  child categories remain visible and functional elsewhere in the app

### Requirement: Delete is only offered when the category has no children and no entry references

Each category node's delete action SHALL be disabled (visibly, e.g. greyed
out with an explanation) whenever the category currently has any child
category or is referenced by any entry, re-evaluated from the same tree
data the page already has — without a separate request per node. Deleting
an eligible category calls `DELETE /api/categories/{id}` behind a
confirmation step. A `409` response (the category became in-use since the
page loaded) SHALL surface as an inline error rather than silently removing
the node.

#### Scenario: Delete is unavailable on a category with children

- **WHEN** a category has at least one child category
- **THEN** its delete action is shown disabled

#### Scenario: Deleting an eligible category

- **WHEN** an authenticated visitor confirms deleting a category with no
  children and no entry references
- **THEN** `DELETE /api/categories/{id}` is called and the category is
  removed from the tree

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

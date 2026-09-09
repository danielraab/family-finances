## ADDED Requirements

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

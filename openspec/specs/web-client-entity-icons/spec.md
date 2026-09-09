# web-client-entity-icons Specification

## Purpose

The client-side icon and colour vocabulary for accounts and categories: a
curated, statically-imported `lucide-react` icon set organised into groups,
a fixed theme-aware colour palette, the `EntityIcon` badge that renders
them, the `AccountLabel` / `CategoryLabel` wrappers that put the badge
before an entity's name everywhere except native `<select>` controls, and
the reusable `IconColorPicker`. The backend stores `icon` and `color` as
opaque tokens (see `accounts` and `entry-categories`); all meaning lives
here.

## Requirements

### Requirement: A curated, grouped icon set

The web client SHALL define a fixed, curated set of selectable icons, each
identified by a lowercase-kebab token and backed by a statically imported
`lucide-react` icon component. The set SHALL be organised into named groups
(for example Banking, Income, Home & bills, Food, Transport, Life,
Transfers), each group having an i18n-keyed heading and an ordered list of
icons. The icon components SHALL be imported by name (not via a wildcard
import or a runtime dynamic import) so the bundle contains only the icons in
the set. This curated map SHALL be the single source of which icon tokens
are offered anywhere in the client.

#### Scenario: Only curated icons are bundled

- **WHEN** the client bundle is built
- **THEN** it includes exactly the `lucide-react` icons named in the curated
  set and no others

#### Scenario: Every offered icon belongs to a group

- **WHEN** the icon picker renders its choices
- **THEN** each selectable icon appears under exactly one group heading, and
  every heading text comes from an i18n key

### Requirement: A fixed colour palette with light and dark values

The web client SHALL define a fixed palette of selectable colours, each
identified by a lowercase-kebab token (`blue`, `orange`, `aqua`, `yellow`,
`magenta`, `green`, `violet`, `red`) and mapped to a light-theme value and a
dark-theme value taken from the project's validated categorical palette. A
stored `color` value SHALL be a palette token, never a raw colour string.
Rendering a colour SHALL pick the light or dark value according to the
active theme.

#### Scenario: A palette colour adapts to the theme

- **WHEN** an entity with `color` `blue` is rendered in dark mode
- **THEN** the badge uses the palette's dark value for `blue`, and in light
  mode it uses the light value

### Requirement: EntityIcon renders an optional icon/colour badge

The client SHALL provide an `EntityIcon` component taking an optional `icon`
token and an optional `color` token. When `color` is set it SHALL render a
small rounded badge filled with the resolved palette colour; when `color` is
unset it SHALL use a neutral surface tint. When `icon` is set it SHALL
render the corresponding curated glyph inside the badge; when `icon` is
unset it SHALL render the badge (or neutral chip) with no glyph. When
**both** `icon` and `color` are unset the component SHALL render nothing.

#### Scenario: Icon and colour both set

- **WHEN** `EntityIcon` is given a curated `icon` token and a palette
  `color` token
- **THEN** it renders the coloured badge containing that glyph

#### Scenario: Colour only

- **WHEN** `EntityIcon` is given a palette `color` token and no `icon`
- **THEN** it renders the coloured chip with no glyph

#### Scenario: Neither set renders nothing

- **WHEN** `EntityIcon` is given neither `icon` nor `color`
- **THEN** it renders no element

### Requirement: Unknown icon or colour tokens fall back gracefully

When a stored `icon` token is not present in the curated icon set, the
client SHALL render no glyph for it (a colour chip if `color` resolves,
otherwise nothing) without raising an error or logging noise. When a stored
`color` token is not present in the palette, the client SHALL treat the
colour as unset. In every case the entity's name SHALL still render.

#### Scenario: An unrecognised icon token

- **WHEN** a category's `icon` is a token this build's curated set does not
  contain, and its `color` is a valid palette token
- **THEN** the label renders the colour chip and the category name, with no
  glyph and no console error

#### Scenario: An unrecognised colour token

- **WHEN** an account's `color` is a token this build's palette does not
  contain
- **THEN** the label renders as though `color` were unset, and the account
  title still shows

### Requirement: Account and category names render icon-before-label outside native selects

Wherever the web client displays an account or a category **by name** — the
accounts overview, the account detail page, the home account cards, the
category tree, the entry ledger rows, and the reports tables — the name
SHALL be preceded by that entity's `EntityIcon` badge when the entity has an
`icon` and/or `color` set, via a shared `AccountLabel` / `CategoryLabel`
component. Native `<select>` / `<option>` controls SHALL be the sole
exception and SHALL keep showing the bare name: this covers the entry form's
account and category pickers and the account filter dropdowns on the entry
ledger and the reports page.

#### Scenario: The accounts overview shows account badges

- **WHEN** the accounts overview lists an account that has an `icon` and a
  `color`
- **THEN** that account's row shows the icon/colour badge immediately before
  its title

#### Scenario: The category tree shows category badges

- **WHEN** the `/categories` tree renders a category that has an `icon`
- **THEN** the icon appears immediately before the category name in that
  tree node

#### Scenario: The entry ledger shows account and category badges

- **WHEN** the entry ledger lists an entry whose account and category each
  have a badge
- **THEN** each of those names is shown preceded by its badge

#### Scenario: Native select controls stay text-only

- **WHEN** the entry form's account picker or category picker, or an account
  filter dropdown on the ledger or reports page, is rendered
- **THEN** each option shows only the entity's name, with no icon

### Requirement: A reusable icon and colour picker

The client SHALL provide an `IconColorPicker` that opens from a trigger
control showing the current selection (the `EntityIcon` badge, or an
empty-state placeholder when nothing is set). Its panel SHALL present the
palette colour swatches and the curated icon set (as a grid grouped under
its headings), letting the user independently choose an icon, a colour,
both, or neither, and clear either back to unset. It SHALL be usable from
the account form and the category create and edit forms — including when
that form is itself inside a dialog — and its labels SHALL come from i18n
keys.

#### Scenario: The picker opens from its trigger

- **WHEN** the user activates the picker's trigger control
- **THEN** a panel opens showing the colour swatches and the grouped icon
  grid, and dismissing it (choosing "Done" or clicking outside) closes it
  without losing the current selection

#### Scenario: Picking an icon and a colour independently

- **WHEN** the user opens the picker and selects an icon but no colour
- **THEN** the form value has that `icon` token and an unset `color`, and
  the trigger reflects the new selection

#### Scenario: Clearing a selection

- **WHEN** the user has an icon and a colour selected and activates the
  picker's clear control for the icon
- **THEN** the form value's `icon` returns to unset while `color` is
  retained

#### Scenario: Grouped icon grid

- **WHEN** the picker panel is open
- **THEN** its icons are shown under their group headings in the curated
  order

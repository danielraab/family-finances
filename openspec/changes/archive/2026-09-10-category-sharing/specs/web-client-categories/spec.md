## MODIFIED Requirements

### Requirement: The categories page renders the caller's tree, ordered by sibling order

`/categories` SHALL fetch the caller's categories (`GET /api/categories`,
now owned categories plus every category shared with the caller — see
`category-sharing`) and render the caller's own categories as a nested
tree, each node's children ordered by `sort_order`. Every category shared
with the caller SHALL render separately, in a distinct "Shared with me"
section, flat rather than nested (see the new requirement below) — it
SHALL NOT be interleaved into the owned tree. The page SHALL indicate a
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

## ADDED Requirements

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

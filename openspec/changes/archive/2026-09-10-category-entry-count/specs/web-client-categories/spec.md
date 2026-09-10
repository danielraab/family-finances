## MODIFIED Requirements

### Requirement: The categories page renders the caller's tree, ordered by sibling order

`/categories` SHALL fetch the caller's categories (`GET /api/categories`)
and render them as a nested tree, each node's children ordered by
`sort_order`. The page SHALL indicate a category's `disabled` status
distinctly from an enabled one, SHALL show each node's `entry_count` (the
number of the caller's own entries directly categorized under it) as a
distinct, non-interactive indicator on that node, and SHALL render at any
viewport width down to a typical mobile screen without requiring horizontal
scrolling or drag gestures for any control.

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

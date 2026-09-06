## ADDED Requirements

### Requirement: An entry can only be categorized with its owner's own category

Setting `category_id` on an entry (at creation or update) SHALL require the
category to belong to the same owner as the entry — the same rule
`entry-tags` already applies to `tag_id`. Referencing a nonexistent
category id, or a category owned by a different user, SHALL be rejected.

#### Scenario: Categorizing with another user's category rejected

- **WHEN** a user attempts to create or update an entry with a
  `category_id` owned by a different user
- **THEN** the request is rejected (`400`) and the entry is not categorized
  with it

### Requirement: A disabled category cannot be newly set on an entry

Setting `category_id` on an entry — at creation, or on an update that
includes `category_id` in the request — SHALL be rejected when it resolves
to a disabled category. An update that does not include `category_id` SHALL
NOT be rejected on account of the entry's current category having since
become disabled — unlike a disabled account type, a disabled category does
not block unrelated edits to entries that already reference it.

#### Scenario: Creating an entry with a disabled category is rejected

- **WHEN** a user calls `POST /api/entries` with a `category_id` that
  resolves to a disabled category
- **THEN** the request is rejected (`400`) and no entry is created

#### Scenario: Editing an unrelated field does not require reselecting a since-disabled category

- **WHEN** an entry's current category has since been disabled and its
  owner calls `PATCH /api/entries/{id}` changing only an unrelated field
  (no `category_id` in the body)
- **THEN** the update succeeds and the entry keeps its (disabled) category

#### Scenario: Explicitly re-setting a disabled category on update is rejected

- **WHEN** a user calls `PATCH /api/entries/{id}` with a `category_id` that
  resolves to a disabled category
- **THEN** the request is rejected (`400`) and the entry's category is
  unchanged

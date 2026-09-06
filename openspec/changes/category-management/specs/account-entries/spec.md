## ADDED Requirements

### Requirement: An entry can only be categorized with its owner's own category

Setting `category_id` on an entry (at creation or update) SHALL require the
category to belong to the same owner as the entry — the same rule
`entry-tags` already applies to `tag_id`. Referencing a nonexistent
category id, or a category owned by a different user, SHALL be rejected.

#### Scenario: Categorizing with another user's category rejected

- **WHEN** a user attempts to create or update an entry with a
  `category_id` owned by a different user
- **THEN** the request is rejected (`422`) and the entry is not categorized
  with it

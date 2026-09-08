## ADDED Requirements

### Requirement: A disabled tag cannot be newly attached to an entry

Setting `tag_ids` on an entry — at creation, or on an update that includes
`tag_ids` in the request — SHALL be rejected when any id in the request
that is not already among the entry's current tags resolves to a disabled
tag. A tag id already present on the entry before the update SHALL NOT
block that update on account of having since become disabled, even when
the same request resubmits it as part of the full `tag_ids` array —
`tag_ids` is always a full replacement list, so this exemption applies
specifically to ids the update does not actually change the presence of,
not to the request as a whole.

#### Scenario: Creating an entry with a disabled tag is rejected

- **WHEN** a user calls `POST /api/entries` with a `tag_ids` entry that
  resolves to a disabled tag
- **THEN** the request is rejected (`400`) and no entry is created

#### Scenario: Resubmitting an entry's existing, since-disabled tag succeeds

- **WHEN** an entry currently carries a tag that has since been disabled,
  and its owner calls `PATCH /api/entries/{id}` with a `tag_ids` array that
  still includes that same tag id (unchanged) alongside other, unrelated
  changes
- **THEN** the update succeeds and the entry keeps the disabled tag

#### Scenario: Adding a new disabled tag alongside an untouched existing one is rejected

- **WHEN** an entry currently carries tag A (since disabled), and its owner
  calls `PATCH /api/entries/{id}` with `tag_ids` containing both A and a
  different tag B that is also disabled
- **THEN** the request is rejected (`400`) and the entry's tags are
  unchanged

#### Scenario: Editing an unrelated field does not require dropping a since-disabled tag

- **WHEN** an entry's current tags include one that has since been
  disabled, and its owner calls `PATCH /api/entries/{id}` changing only an
  unrelated field (no `tag_ids` in the body)
- **THEN** the update succeeds and the entry keeps every one of its
  existing tags, including the disabled one

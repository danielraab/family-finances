## ADDED Requirements

### Requirement: Tags are private to their owner

Every tag SHALL carry a required `owner_id`, set to the authenticated caller
at creation. Listing, reading, updating, and deleting a tag SHALL be scoped
to `owner_id = <authenticated user>`; a tag belonging to a different user
SHALL behave as if it does not exist (`404`). A tag name SHALL be unique per
owner (case-sensitive), not unique instance-wide.

#### Scenario: A user manages their own tags

- **WHEN** an authenticated user calls `POST /api/tags` with a name they
  have not used before
- **THEN** the response is `201` with the created tag, owned by the caller

#### Scenario: A user cannot see another user's tags

- **WHEN** an authenticated user calls `GET /api/tags/{id}` for a tag owned
  by a different user
- **THEN** the response is `404`

#### Scenario: Duplicate tag name for the same owner rejected

- **WHEN** a user creates a tag with a name identical to one they already
  own
- **THEN** the response is `409` and no new tag is created

#### Scenario: Same tag name across different owners is allowed

- **WHEN** two different users each create a tag named "groceries"
- **THEN** both requests succeed, as two distinct tags

### Requirement: An entry can only be tagged with tags its owner owns

Attaching a tag to an entry SHALL require the tag to belong to the same
owner as the entry. Referencing a nonexistent tag id, or a tag owned by a
different user, SHALL be rejected.

#### Scenario: Tagging with another user's tag rejected

- **WHEN** a user attempts to create or update an entry with a `tag_id`
  owned by a different user
- **THEN** the request is rejected (`422`) and the entry is not tagged with
  it

### Requirement: Deleting a tag detaches it from every entry, without rejecting the delete

Unlike a category or account type, a tag SHALL always be deletable by its
owner regardless of use. Deleting a tag SHALL remove its association from
every entry it was attached to; it SHALL NOT delete or otherwise modify
those entries themselves.

#### Scenario: Deleting a used tag succeeds and detaches it

- **WHEN** a user deletes a tag that is attached to one or more of their
  entries
- **THEN** the response is `204`, the tag no longer appears on any entry,
  and those entries are otherwise unchanged

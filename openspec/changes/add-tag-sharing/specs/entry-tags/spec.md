## MODIFIED Requirements

### Requirement: Tags are visible to their owner and to anyone they're shared with

Every tag SHALL carry a required `owner_id`, set to the authenticated
caller at creation. Creating a tag, renaming it, disabling/enabling it,
and managing its shares SHALL require real ownership — a share never
grants any of these (see `tag-sharing`). `GET /api/tags` SHALL return the
caller's own tags plus every tag currently shared with them, each of the
latter carrying `permission` (`view` or `append`), `shared: true`, and
`owner_name`. A tag the caller owns SHALL carry `permission: owner` and
no `shared`/`owner_name`. `GET /api/tags/{id}` SHALL behave identically
for a single tag: reachable by the real owner or any share recipient,
carrying the same permission/shared/owner_name fields, and behaving as if
it does not exist (`404`) for a caller with no permission on it at all. A
tag name SHALL be unique per owner (case-sensitive), not unique
instance-wide — sharing does not change this, since the constraint is
scoped to `owner_id` and a shared tag keeps its original owner.

#### Scenario: A user manages their own tags

- **WHEN** an authenticated user calls `POST /api/tags` with a name they
  have not used before
- **THEN** the response is `201` with the created tag, owned by the
  caller, carrying `permission: owner`

#### Scenario: A user with no permission cannot see a tag

- **WHEN** an authenticated user calls `GET /api/tags/{id}` for a tag they
  neither own nor have a share on
- **THEN** the response is `404`

#### Scenario: A share recipient can fetch the tag by id

- **WHEN** a user with `view` or `append` permission (via a share) calls
  `GET /api/tags/{id}` for that tag
- **THEN** the response is `200`, carrying that user's permission,
  `shared: true`, and the real owner's name

#### Scenario: Duplicate tag name for the same owner rejected

- **WHEN** a user creates a tag with a name identical to one they already
  own
- **THEN** the response is `409` and no new tag is created

#### Scenario: Same tag name across different owners is allowed

- **WHEN** two different users each create a tag named "groceries"
- **THEN** both requests succeed, as two distinct tags

#### Scenario: A share recipient cannot rename, disable, or delete the tag

- **WHEN** a user with `view` or `append` permission (via a share) calls
  `PATCH /api/tags/{id}`, `POST /api/tags/{id}/disable`, or
  `DELETE /api/tags/{id}`
- **THEN** the request is rejected — only the tag's real owner may perform
  any of these

### Requirement: An entry can be tagged with a tag its owner owns, or has at least append permission on via a share

Attaching a tag to an entry SHALL require the entry's owner to either own
the tag, or hold at least `append` permission on it via a share (see
`tag-sharing`). Referencing a nonexistent tag id, or a tag the entry's
owner has neither ownership nor `append` permission on, SHALL be rejected.

#### Scenario: Tagging with an unrelated tag rejected

- **WHEN** a user attempts to create or update an entry with a `tag_id`
  they neither own nor have `append` permission on
- **THEN** the request is rejected (`422`) and the entry is not tagged
  with it

#### Scenario: Tagging with a tag shared at append permission succeeds

- **WHEN** a user with `append` permission on a tag (via a share) creates
  or updates an entry with that `tag_id`
- **THEN** the request succeeds and the entry carries that tag

#### Scenario: Tagging with a tag shared only at view permission is rejected

- **WHEN** a user with only `view` permission on a tag (via a share)
  attempts to create or update an entry with that `tag_id`
- **THEN** the request is rejected (`422`) and the entry is not tagged
  with it

### Requirement: Deleting a tag detaches it from every entry, unless the tag is currently shared

A tag currently shared with at least one other user SHALL NOT be
deletable — `DELETE /api/tags/{id}` SHALL be rejected (`409`) until every
share on it has been revoked or left. An unshared tag SHALL remain
deletable by its owner exactly as before, regardless of how many of the
owner's own entries carry it: deleting it SHALL remove its association
from every entry it was attached to, without deleting or otherwise
modifying those entries themselves.

#### Scenario: Deleting an unshared, used tag succeeds and detaches it

- **WHEN** a user deletes a tag, not currently shared with anyone, that
  is attached to one or more of their entries
- **THEN** the response is `204`, the tag no longer appears on any entry,
  and those entries are otherwise unchanged

#### Scenario: Deleting a shared tag is rejected

- **WHEN** a user attempts to delete a tag that currently has at least
  one active share
- **THEN** the response is `409` and the tag is not deleted

#### Scenario: Deleting a formerly-shared tag succeeds once every share is gone

- **WHEN** every share on a tag has been revoked or left, and its owner
  then deletes it
- **THEN** the response is `204`, identical to deleting a tag that was
  never shared

### Requirement: Every tag reports how many of the viewing caller's own entries carry it

Every `Tag` returned by the API (from `GET /api/tags` and every other tag
endpoint) SHALL include `entry_count`: the number of the *viewing
caller's own* non-deleted entries currently carrying that tag — scoped to
the caller even when they are viewing a tag shared with them rather than
one they own, mirroring `category-sharing`'s equivalent field.

#### Scenario: A freshly created tag has zero entries

- **WHEN** a user creates a new tag
- **THEN** the response's `entry_count` is `0`

#### Scenario: Entry count reflects current attachment

- **WHEN** a user attaches a tag to two of their entries and then removes
  it from one of them
- **THEN** `GET /api/tags` subsequently reports that tag's `entry_count`
  as `1`

#### Scenario: A deleted entry no longer counts toward its tags

- **WHEN** an entry carrying a tag is (soft-)deleted
- **THEN** that tag's `entry_count` no longer includes it

#### Scenario: A share recipient's entry count reflects only their own entries

- **WHEN** a tag's real owner has five of their own entries carrying it,
  and a user with `append` permission via a share has attached it to one
  of their own entries
- **THEN** the share recipient's `GET /api/tags` response for that tag
  shows `entry_count: 1`, not `6`

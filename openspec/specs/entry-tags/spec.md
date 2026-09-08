# entry-tags Specification

## Purpose

Per-user tags entries can be organized by — private to their owner,
created inline from the entry form and managed (renamed,
disabled/enabled, deleted) from a dedicated Tags settings tab. See
`account-entries` for how tags attach to entries and
`web-client-settings` for the management tab.

## Requirements

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

### Requirement: Disabling a tag blocks new use without touching history

Each tag SHALL carry `disabled` (boolean, `false` by default), toggled via
`POST /api/tags/{id}/disable` and reversed via `POST /api/tags/{id}/enable`.
Disabling a tag SHALL NOT change, hide, or otherwise affect any entry
already carrying it, and SHALL NOT remove it from `GET /api/tags` — its
only effect is that it can no longer be newly attached to an entry (see
`account-entries`). Disabling and delete are independent: a tag need not be
disabled before it can be deleted, and disabling a tag never affects
whether it can subsequently be deleted.

#### Scenario: Disabling does not affect entries already carrying the tag

- **WHEN** a user disables a tag that one or more of their entries
  currently carry
- **THEN** those entries are unchanged, and the tag itself still appears in
  `GET /api/tags` with `disabled: true`

#### Scenario: Enabling reverses it

- **WHEN** a user enables a previously disabled tag
- **THEN** it becomes attachable again to new or edited entries

#### Scenario: A disabled tag can still be deleted, same as an enabled one

- **WHEN** a user deletes a tag that is currently disabled
- **THEN** the response is `204`, identical to deleting a non-disabled tag

### Requirement: Every tag reports how many of the owner's entries carry it

Every `Tag` returned by the API (from `GET /api/tags` and every other tag
endpoint) SHALL include `entry_count`: the number of the caller's own
non-deleted entries currently carrying that tag.

#### Scenario: A freshly created tag has zero entries

- **WHEN** a user creates a new tag
- **THEN** the response's `entry_count` is `0`

#### Scenario: Entry count reflects current attachment

- **WHEN** a user attaches a tag to two of their entries and then removes
  it from one of them
- **THEN** `GET /api/tags` subsequently reports that tag's `entry_count` as
  `1`

#### Scenario: A deleted entry no longer counts toward its tags

- **WHEN** an entry carrying a tag is (soft-)deleted
- **THEN** that tag's `entry_count` no longer includes it

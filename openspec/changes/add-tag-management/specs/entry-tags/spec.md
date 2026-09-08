## ADDED Requirements

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

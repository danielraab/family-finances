## MODIFIED Requirements

### Requirement: A card's config is validated against the caller's actual access and its type's required fields

`internal/dashboard` SHALL validate, at both create and update:

- Any `account_id` in `config` refers to an account the caller has *any*
  permission tier on (owned or shared, `view` and up) — the same bar
  `/reports`'s account filter already allows.
- Any `category_id` refers to a category visible to the caller at `view`
  tier or above (owned, or shared at `view` or `append`).
- Any `tag_id` refers to a tag visible to the caller at `view` tier or
  above (owned, or shared at `view` or `append`).
- The fields required by the card's `type` are present and no field
  foreign to that type is set: `account_stat` requires `account_id` and
  accepts no other field; `bar_chart` requires `unit` (`month` or `day`)
  and accepts only `account_id`/`category_id`/`include_subcategories`/
  `tag_id`/`title`/`show_recurring_preview` besides it; `query_stat`
  accepts `account_id`/`category_id`/`include_subcategories`/`tag_id`/
  `range`/`title`, all optional, and no other field; `entry_list` accepts
  the same fields as `query_stat` plus an optional `columns` (`2`-`4`
  inclusive, default `2` when absent) and an optional
  `show_recurring_preview` (boolean, default `false`) — no other type may
  set `columns`, only `entry_list`/`bar_chart` may set
  `show_recurring_preview`, and only `query_stat`/`entry_list`/
  `bar_chart` may set `title` (free-text, no format constraint).

A reference the caller cannot see at all, a missing required field, or a
field foreign to the given type, SHALL be rejected (`ErrInvalidValue`,
`400`) — nothing is created or updated.

#### Scenario: Referencing an inaccessible account is rejected

- **WHEN** a caller creates a card whose `config.account_id` names an
  account they have no permission on
- **THEN** the request is rejected (`400`) and no card is created

#### Scenario: Referencing a category shared at view tier is accepted

- **WHEN** a caller creates a `query_stat` card whose `config.category_id`
  names a category shared with them at `view` tier only
- **THEN** the card is created successfully

#### Scenario: An account_stat card without account_id is rejected

- **WHEN** a caller creates an `account_stat` card with an empty `config`
- **THEN** the request is rejected (`400`)

#### Scenario: A bar_chart card without unit is rejected

- **WHEN** a caller creates a `bar_chart` card whose `config` has no
  `unit`
- **THEN** the request is rejected (`400`)

#### Scenario: A bar_chart card with an invalid unit is rejected

- **WHEN** a caller creates a `bar_chart` card with `config.unit:
  "week"`
- **THEN** the request is rejected (`400`)

#### Scenario: A field foreign to the card's type is rejected

- **WHEN** a caller creates an `account_stat` card whose `config` also
  sets `tag_id`
- **THEN** the request is rejected (`400`)

#### Scenario: An entry_list card without columns defaults to 2

- **WHEN** a caller creates an `entry_list` card with no `columns` in
  `config`
- **THEN** the card is created successfully

#### Scenario: An entry_list card accepts columns within range

- **WHEN** a caller creates an `entry_list` card with `config.columns`
  set to `2`, `3`, or `4`
- **THEN** the card is created successfully in each case

#### Scenario: An entry_list card rejects columns outside 2-4

- **WHEN** a caller creates an `entry_list` card with `config.columns`
  set to `1` or `5`
- **THEN** the request is rejected (`400`)

#### Scenario: columns is rejected on every other type

- **WHEN** a caller creates an `account_stat`, `query_stat`, or
  `bar_chart` card whose `config` sets `columns`
- **THEN** the request is rejected (`400`)

#### Scenario: title is accepted on query_stat, entry_list, and bar_chart

- **WHEN** a caller creates a `query_stat`, `entry_list`, or `bar_chart`
  card with a `title` set in `config`
- **THEN** the card is created successfully in each case

#### Scenario: title is rejected on account_stat

- **WHEN** a caller creates an `account_stat` card whose `config` also
  sets `title`
- **THEN** the request is rejected (`400`)

#### Scenario: show_recurring_preview is accepted on entry_list and bar_chart

- **WHEN** a caller creates an `entry_list` or `bar_chart` card with
  `config.show_recurring_preview: true`
- **THEN** the card is created successfully in each case

#### Scenario: show_recurring_preview is rejected on account_stat and query_stat

- **WHEN** a caller creates an `account_stat` or `query_stat` card whose
  `config` sets `show_recurring_preview`
- **THEN** the request is rejected (`400`)

#### Scenario: show_recurring_preview defaults to false

- **WHEN** a caller creates an `entry_list` or `bar_chart` card with no
  `show_recurring_preview` in `config`
- **THEN** the card is created successfully and no Upcoming preview is
  shown for it until the visitor explicitly enables it

#### Scenario: Updating a card re-validates its new config the same way

- **WHEN** a caller updates an existing card's `config.account_id` to an
  account they have no permission on
- **THEN** the update is rejected (`400`) and the card's stored config is
  unchanged

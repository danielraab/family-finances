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
  `show_recurring_preview` (boolean, default `false`); `line_chart`
  accepts only `account_id`/`title`/`show_recurring_preview`, all
  optional, and no other field — no category/tag filter or bucketing
  choice to accept. `columns` is settable only on `entry_list`;
  `show_recurring_preview` only on `entry_list`/`bar_chart`/`line_chart`;
  `title` only on `query_stat`/`entry_list`/`bar_chart`/`line_chart`.

A reference the caller cannot see at all, a missing required field, or a
field foreign to the given type, SHALL be rejected (`ErrInvalidValue`,
`400`) — nothing is created or updated.

#### Scenario: show_recurring_preview is accepted on entry_list, bar_chart, and line_chart

- **WHEN** a caller creates an `entry_list`, `bar_chart`, or `line_chart`
  card with `config.show_recurring_preview: true`
- **THEN** the card is created successfully in each case

#### Scenario: show_recurring_preview is rejected on account_stat and query_stat

- **WHEN** a caller creates an `account_stat` or `query_stat` card whose
  `config` sets `show_recurring_preview`
- **THEN** the request is rejected (`400`)

#### Scenario: show_recurring_preview defaults to false

- **WHEN** a caller creates an `entry_list`, `bar_chart`, or `line_chart`
  card with no `show_recurring_preview` in `config`
- **THEN** the card is created successfully and no recurring-assumptions
  overlay is shown for it until the visitor explicitly enables it

#### Scenario: A line_chart card with show_recurring_preview still rejects category/tag/range/unit/columns

- **WHEN** a caller creates a `line_chart` card with
  `config.show_recurring_preview: true` and `config.category_id` also set
- **THEN** the request is rejected (`400`)

# dashboard-cards Specification

## Purpose

A per-user, ordered set of dashboard cards backing the customizable
`/home` dashboard: each card's type, inline filter config, and position,
plus creation, update, deletion, and reordering. `internal/dashboard`
owns this domain. Validating a card's config against the caller's actual
account/category/tag access borrows the same permission model as
`accounts`, `entry-categories`, and `entry-tags`. See `web-client-home`
for how the client renders and edits cards, and `account-entries` for the
entry-summary/flow-summary endpoints a card's data ultimately draws from.

## Requirements

### Requirement: A user's dashboard is an ordered, per-user list of cards

`internal/dashboard` SHALL own a per-user list of dashboard cards, each
with an id, a `type` (`account_stat`, `query_stat`, `entry_list`, or
`bar_chart`, immutable after creation), a `config` object, and a position
among that user's own cards. `GET /api/dashboard/cards` SHALL return only
the authenticated caller's own cards, ordered by position. A card belongs
to exactly one user and is never visible to, or affected by, any other
user, regardless of whether entities it references (an account, category,
or tag) are shared with others.

#### Scenario: Listing returns only the caller's own cards, in order

- **WHEN** an authenticated user with three cards calls
  `GET /api/dashboard/cards`
- **THEN** the response contains exactly those three cards, ordered by
  their position

#### Scenario: A brand-new user has no cards

- **WHEN** a user who has never created a card calls
  `GET /api/dashboard/cards`
- **THEN** the response is an empty list

#### Scenario: One user's cards are invisible to another

- **WHEN** two users each have their own dashboard cards
- **THEN** calling `GET /api/dashboard/cards` as one of them never returns
  a card belonging to the other

### Requirement: A card is created with a type and a config, and its type cannot change afterward

`POST /api/dashboard/cards` SHALL accept `{ type, config }`, create the
card at the end of the caller's own list, and return it. `type` SHALL be
one of `account_stat`, `query_stat`, `entry_list`, `bar_chart`; any other
value is rejected (`ErrInvalidValue`, `400`). `PATCH
/api/dashboard/cards/{id}` SHALL accept only `{ config }` — a `type` field
in the request body, if present, SHALL be ignored rather than applied.
`DELETE /api/dashboard/cards/{id}` SHALL remove the card; a caller may
only create, update, or delete their own cards — acting on another user's
card id SHALL respond `404`, the same not-found-not-forbidden treatment
`internal/category`/`internal/tag` already give a wrong-owner id.

#### Scenario: Creating a card appends it to the end of the caller's list

- **WHEN** a caller with two existing cards creates a third
- **THEN** the new card is positioned after the existing two

#### Scenario: An unknown type is rejected

- **WHEN** `POST /api/dashboard/cards` is called with
  `type: "something_else"`
- **THEN** the request is rejected (`400`) and no card is created

#### Scenario: Updating a card only ever changes its config

- **WHEN** a caller calls `PATCH /api/dashboard/cards/{id}` on their own
  `account_stat` card with `{ "type": "bar_chart", "config": {...} }`
- **THEN** the card's `type` remains `account_stat` and only `config` is
  updated

#### Scenario: Acting on another user's card is not found

- **WHEN** a caller calls `PATCH`, `DELETE`, `/move-up`, or `/move-down`
  on a card id belonging to a different user
- **THEN** the response is `404`

### Requirement: Cards can be reordered via move-up and move-down, with no-op edges

`POST /api/dashboard/cards/{id}/move-up` and `.../move-down` SHALL swap
the target card's position with its immediate previous/next card in the
caller's own list. Calling `move-up` on the first card, or `move-down` on
the last, SHALL succeed (`200`) and leave the order unchanged, mirroring
`POST /api/categories/{id}/move-up`/`/move-down`'s no-op edge behavior.

#### Scenario: Moving a card up swaps it with the previous one

- **WHEN** a caller's cards are ordered `[A, B, C]` and they move `B` up
- **THEN** the order becomes `[B, A, C]`

#### Scenario: Moving the first card up is a no-op

- **WHEN** a caller's first card is moved up
- **THEN** the response is `200` and the order is unchanged

#### Scenario: Moving the last card down is a no-op

- **WHEN** a caller's last card is moved down
- **THEN** the response is `200` and the order is unchanged

#### Scenario: Deleting a card closes the gap in position for the rest

- **WHEN** a caller's cards are ordered `[A, B, C]` and `B` is deleted
- **THEN** the remaining order is `[A, C]` with no gap a subsequent move
  could get stuck on

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
  `tag_id`/`title` besides it; `query_stat` accepts
  `account_id`/`category_id`/`include_subcategories`/`tag_id`/`range`/
  `title`, all optional, and no other field; `entry_list` accepts the
  same fields as `query_stat` plus an optional `columns` (`2`-`4`
  inclusive, default `2` when absent) — no other type may set `columns`,
  and only `query_stat`/`entry_list`/`bar_chart` may set `title`
  (free-text, no format constraint).

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

#### Scenario: Updating a card re-validates its new config the same way

- **WHEN** a caller updates an existing card's `config.account_id` to an
  account they have no permission on
- **THEN** the update is rejected (`400`) and the card's stored config is
  unchanged

### Requirement: A revoked or deleted reference is never re-validated after the fact

Once a card is created or updated with a valid config, `internal/
dashboard` SHALL NOT re-check that config's references on subsequent
reads (`GET`), moves, or unrelated updates — a share revoked or an entity
soft-deleted after the fact does not invalidate, delete, or alter the
card. Resolving whether a referenced id still exists and is still visible
is left entirely to the caller of `GET /api/dashboard/cards` (the web
client), the same way a since-disabled category or tag stays attached to
an entry that already referenced it.

#### Scenario: A share revoked after card creation does not delete the card

- **WHEN** a caller's card references an account shared with them, and
  that share is later revoked
- **THEN** the card still appears in `GET /api/dashboard/cards`, with its
  original `config` unchanged

#### Scenario: Reordering does not re-validate a stale reference

- **WHEN** a caller moves a card whose referenced account is no longer
  accessible to them
- **THEN** the move still succeeds

## MODIFIED Requirements

### Requirement: Entry list card shows the most recent matching entries

An `entry_list` card SHALL show up to 10 of the most recent entries
(`GET /api/entries`, sorted by `booking_timestamp` descending) matching
its configured filter (the same optional `account_id`/`category_id`/
`include_subcategories`/`tag_id`/`range` fields as a `query_stat` card),
each row showing at least its date, title, account (per
`web-client-accounts`' `AccountLabel`, when visible to the caller), and
sign-colored amount. The card SHALL show explanatory empty text when no
entry matches.

Each row's entry title SHALL open that entry's read-only summary modal
(per `web-client-entries`) when activated, rather than being inert text.
Activating it SHALL NOT enter the dashboard's edit mode or trigger the
card's own controls.

#### Scenario: An entry list card shows recent matching entries

- **WHEN** an `entry_list` card's filter matches 15 entries
- **THEN** the card shows the 10 most recent, most-recent first

#### Scenario: An entry list card with no matches shows empty text

- **WHEN** an `entry_list` card's filter matches no entries
- **THEN** the card shows explanatory empty-state text instead of an empty
  list

#### Scenario: A card row's title opens the entry summary

- **WHEN** the visitor activates an entry's title in an `entry_list` card
- **THEN** that entry's read-only summary modal opens, and the dashboard
  is not navigated away from

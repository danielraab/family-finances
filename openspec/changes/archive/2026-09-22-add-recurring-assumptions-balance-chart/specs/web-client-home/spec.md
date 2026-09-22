## MODIFIED Requirements

### Requirement: Edit mode lets the visitor add a card

`/home` SHALL offer an edit-mode toggle. While active, an "Add card"
control SHALL open a type picker (`account_stat`/`query_stat`/
`entry_list`/`bar_chart`/`line_chart`), followed by that type's config
form — reusing `/reports`' existing account/category/tag `<select>`s and
`DateRangeFilter` for the filter-bearing types. For `query_stat`/
`entry_list`/`bar_chart`/`line_chart`, the form additionally offers an
optional free-text title field (absent for `account_stat`, which has
none); `line_chart`'s form offers only the account picker, title field,
and the recurring-preview toggle below, with no category/tag/date-range/
unit controls, since its config accepts none of them. The `entry_list`
form additionally offers a named width choice (e.g. "Narrow"/"Wide"/"Full
width", mapping to `columns` `2`/`3`/`4`) defaulting to the narrowest
option. `entry_list`/`bar_chart`/`line_chart` additionally offer the
`show_recurring_preview` toggle (see the next requirement). Submitting the
form SHALL create the card (`POST /api/dashboard/cards`) at the end of the
visitor's list and render it immediately.

#### Scenario: Adding an account stat card

- **WHEN** a visitor in edit mode picks `account_stat`, selects an
  account, and submits
- **THEN** a new `account_stat` card for that account appears at the end
  of the dashboard

#### Scenario: Adding an entry list card with a chosen width

- **WHEN** a visitor in edit mode picks `entry_list`, chooses the
  "Full width" option, and submits
- **THEN** a new `entry_list` card with `columns: 4` appears at the end of
  the dashboard, spanning the full grid width

#### Scenario: Adding a query stat card with a filter

- **WHEN** a visitor in edit mode picks `query_stat`, sets a category and
  a date-range preset, and submits
- **THEN** a new `query_stat` card with that filter appears at the end of
  the dashboard

#### Scenario: Adding a line chart card scoped to one account

- **WHEN** a visitor in edit mode picks `line_chart`, selects an account,
  and submits
- **THEN** a new `line_chart` card scoped to that account appears at the
  end of the dashboard, with no category/tag/date-range fields ever
  offered for it

#### Scenario: Adding a card with a custom title

- **WHEN** a visitor in edit mode picks `entry_list`, enters a custom
  title, and submits
- **THEN** the new card renders with that title as its heading instead of
  a generated filter summary

#### Scenario: An invalid config form cannot be submitted

- **WHEN** a visitor in edit mode picks `account_stat` and submits without
  selecting an account
- **THEN** the form shows an inline error and no card is created

#### Scenario: Adding a line chart card with recurring assumptions enabled

- **WHEN** a visitor in edit mode picks `line_chart`, enables the
  recurring-preview toggle, and submits
- **THEN** the new card is created with `config.show_recurring_preview:
  true` and immediately renders its projected balance overlay

### Requirement: Line chart card shows a running-balance chart for an account or all accounts

A `line_chart` card SHALL show a running-balance line chart via
`GET /api/entries/balance-series`, scoped to its configured `account_id`
(optional — omitted means every account the visitor owns, summed per
currency) sampled per day over a selected month, with a previous/next-
month pager. The displayed month SHALL default to the current one and is
not persisted across reloads. One chart SHALL render per currency present
in the response, each headed by its currency code when more than one is
present, mirroring how a `bar_chart` card renders one chart per currency.

When the card's `config.show_recurring_preview` is `true`, each currency's
chart additionally overlays a projected balance line, computed and
rendered the same way the account details page's own balance chart does
(see `web-client-accounts`): fetch `GET /api/recurring-transactions/
preview` scoped to the card's `account_id` (or every account the visitor
owns, matching the card's own unscoped case), bounded by the visitor's
`recurring_preview_horizon` setting, excluding overdue occurrences, and
draw the resulting cumulative projection as a dashed continuation of the
real line from today onward.

#### Scenario: A line chart card defaults to the current month

- **WHEN** a `line_chart` card is rendered
- **THEN** it shows the running balance for each day of the current
  calendar month

#### Scenario: A line chart card can be scoped to one account

- **WHEN** a `line_chart` card's config sets `account_id` to a single
  account
- **THEN** its line reflects only that account's running balance

#### Scenario: An unscoped line chart card sums every account per currency

- **WHEN** a `line_chart` card's config has no `account_id`
- **THEN** its line reflects every account the visitor owns, summed per
  currency, with one chart rendered per currency present

#### Scenario: Switching months does not persist across reloads

- **WHEN** a visitor pages a `line_chart` card to a previous month and
  then reloads `/home`
- **THEN** the card again opens on the current month

#### Scenario: A line chart card with recurring preview enabled shows a projected overlay

- **WHEN** a `line_chart` card has `config.show_recurring_preview: true`
  and its account has an upcoming, non-overdue recurring transaction
- **THEN** the card's chart draws a dashed projected line from today's
  balance forward, reflecting that occurrence's amount once its date
  passes

#### Scenario: A line chart card without recurring preview shows only the real line

- **WHEN** a `line_chart` card has no `show_recurring_preview` set (the
  default)
- **THEN** the card's chart shows only the real balance line, with no
  projected overlay or extra legend entry

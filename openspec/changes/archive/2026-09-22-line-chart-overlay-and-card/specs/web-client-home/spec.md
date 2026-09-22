## MODIFIED Requirements

### Requirement: A query stat, entry list, bar chart, or line chart card's heading is a customizable link through to /reports

`query_stat`, `entry_list`, `bar_chart`, and `line_chart` cards SHALL
accept an optional `title` in their config. The card's heading SHALL show
that `title` when set, falling back to the same generated filter summary
shown today (e.g. the category/tag/account name, or "All accounts") when
unset. This heading SHALL always be a link to `/reports`, prefilling its
account/category/include-subcategories/tag/date-range controls from the
card's own filter (a `bar_chart` card's `unit`/`columns` have no
equivalent on `/reports` and are not carried over; a `line_chart` card
carries over only its `account_id`, having no category/tag/range/unit of
its own). Activating it SHALL NOT itself run the report — `/reports`
still requires its own explicit "Generate report" activation, exactly as
it does today for a visitor who arrives there directly. `account_stat`
cards are unaffected — their heading remains the account's own name,
linking to that account's details page, with no `title` option.

#### Scenario: A custom title renders in place of the generated summary

- **WHEN** a `query_stat` card has `config.title` set
- **THEN** the card's heading shows that title, not the generated filter
  summary

#### Scenario: An unset title falls back to the generated filter summary

- **WHEN** a `query_stat`, `entry_list`, `bar_chart`, or `line_chart` card
  has no `config.title`
- **THEN** the card's heading shows the same generated filter summary it
  showed before `title` existed

#### Scenario: Activating the heading opens /reports with the filter prefilled

- **WHEN** a visitor activates a `query_stat` card's heading whose config
  sets a category and a tag
- **THEN** the client navigates to `/reports` with that category and tag
  preselected in its filter controls, and no report yet generated

#### Scenario: account_stat's heading is unaffected

- **WHEN** an `account_stat` card renders
- **THEN** its heading is still the account's own name, linking to that
  account's details page, with no `title` config option offered

### Requirement: Dashboard cards lay out as a grid, with bar chart and line chart cards always full width and entry list width configurable

`bar_chart` and `line_chart` cards SHALL each render alone, full width,
one per row. Every other card type (`account_stat`, `query_stat`,
`entry_list`) SHALL flow in a responsive grid of up to 4 cards per row
depending on viewport width, matching the breakpoint pattern the
account-cards grid already used. `account_stat` and `query_stat` cards
always occupy exactly one column of that grid. An `entry_list` card's
width is configurable via its `columns` config (`2`-`4`, default `2`) —
how many of the grid's columns it spans; every other card type's width is
fixed by its type, and only position among same-row cards is affected by
reordering.

#### Scenario: A bar chart card spans the full row

- **WHEN** `/home` renders a `bar_chart` card
- **THEN** it occupies its own full-width row, with no other card beside
  it

#### Scenario: A line chart card spans the full row

- **WHEN** `/home` renders a `line_chart` card
- **THEN** it occupies its own full-width row, with no other card beside
  it

#### Scenario: Non-chart cards flow multiple per row

- **WHEN** `/home` renders several `account_stat`/`query_stat`/
  `entry_list` cards in sequence
- **THEN** they flow into a grid of multiple cards per row rather than
  one per row

#### Scenario: An entry list card spans its configured width

- **WHEN** `/home` renders an `entry_list` card with `columns: 3`
- **THEN** it spans three of the grid's columns at viewports wide enough
  to offer three, rather than the single column an `account_stat` or
  `query_stat` card occupies

#### Scenario: An entry list card without columns spans the minimum width

- **WHEN** `/home` renders an `entry_list` card with no `columns` set
- **THEN** it spans two of the grid's columns

### Requirement: Edit mode lets the visitor add a card

`/home` SHALL offer an edit-mode toggle. While active, an "Add card"
control SHALL open a type picker (`account_stat`/`query_stat`/
`entry_list`/`bar_chart`/`line_chart`), followed by that type's config
form — reusing `/reports`' existing account/category/tag `<select>`s and
`DateRangeFilter` for the filter-bearing types. For `query_stat`/
`entry_list`/`bar_chart`/`line_chart`, the form additionally offers an
optional free-text title field (absent for `account_stat`, which has
none); `line_chart`'s form offers only the account picker and title
field, with no category/tag/date-range/unit controls, since its config
accepts none of them. The `entry_list` form additionally offers a named
width choice (e.g. "Narrow"/"Wide"/"Full width", mapping to `columns`
`2`/`3`/`4`) defaulting to the narrowest option. Submitting the form
SHALL create the card (`POST /api/dashboard/cards`) at the end of the
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

## ADDED Requirements

### Requirement: Line chart card shows a running-balance chart for an account or all accounts

A `line_chart` card SHALL show a running-balance line chart via
`GET /api/entries/balance-series`, scoped to its configured `account_id`
(optional — omitted means every account the visitor owns, summed per
currency) sampled per day over a selected month, with a previous/next-
month pager. The displayed month SHALL default to the current one and is
not persisted across reloads. One chart SHALL render per currency present
in the response, each headed by its currency code when more than one is
present, mirroring how a `bar_chart` card renders one chart per currency.

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

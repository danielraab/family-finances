## REMOVED Requirements

### Requirement: Home dashboard shows a card per account with its balance

**Reason**: Replaced by the opt-in `account_stat` card type — `/home` no
longer automatically renders every account, since the dashboard is now
user-composed.
**Migration**: A visitor who wants an account represented on `/home` adds
an `account_stat` card for it via edit mode. See the new "Account stat
card shows an account's live balance" requirement.

#### Scenario: Accounts render as cards with their balances

- **WHEN** an authenticated visitor with two accounts opens `/home`
- **THEN** both accounts render as cards, each showing its title,
  financial institute, and current live balance

#### Scenario: A shared account's card shows its real owner

- **WHEN** an authenticated visitor has permission on an account they do
  not really own
- **THEN** its card on `/home` shows a shared indicator and the real
  owner's name

#### Scenario: Activating a card opens the account's details

- **WHEN** an authenticated visitor activates an account card on `/home`
- **THEN** the client navigates to `/accounts/{id}` for that account

#### Scenario: Missing financial institute is handled gracefully

- **WHEN** an authenticated visitor opens `/home` and one of their accounts
  has no `financial_institute` set
- **THEN** that account's card renders without an institute line, rather
  than showing an empty or broken value

### Requirement: Home dashboard shows an all-accounts income/outcome bar chart with a year switcher

**Reason**: Replaced by the opt-in `bar_chart` card type — the dashboard
no longer automatically shows one fixed, all-accounts chart. A visitor who
wants the equivalent view configures it themselves, and can additionally
scope it to one account, category, or tag.
**Migration**: A visitor who wants the previous all-accounts year-view
chart adds a `bar_chart` card with `unit: month` and no account/category/
tag filter — equivalent to what previously rendered automatically. See
the new "Bar chart card shows an income/outcome chart" requirement.

#### Scenario: Chart shows twelve months of the current year by default

- **WHEN** an authenticated visitor with at least one owned or shared
  account opens `/home`
- **THEN** below the account cards, a bar chart shows one income bar and
  one outcome bar for each month of the current year, aggregated across
  all their accounts

#### Scenario: A shared account's entries contribute to the chart

- **WHEN** an authenticated visitor has permission on a shared account with
  entries in the selected year
- **THEN** those entries' amounts are included in the year-overview chart

#### Scenario: One chart per currency

- **WHEN** an authenticated visitor whose accounts span two currencies
  opens `/home`
- **THEN** two bar charts render one above the other, each headed by its
  currency code, and each showing only that currency's income and outcome

#### Scenario: Single-currency visitor sees a single chart

- **WHEN** an authenticated visitor whose accounts all use one currency
  opens `/home`
- **THEN** exactly one bar chart renders for the year overview

#### Scenario: Switching years

- **WHEN** the visitor activates the previous-year or next-year control
- **THEN** every per-currency chart refetches and redraws for the newly
  selected year, with no restriction on navigating past the current year

#### Scenario: A month with no entries still renders

- **WHEN** the selected year includes a month with no matching entries in
  a given currency
- **THEN** that month's bars in that currency's chart render at zero
  rather than being omitted or erroring

#### Scenario: No chart in the empty state

- **WHEN** an authenticated visitor with no owned or shared accounts opens
  `/home`
- **THEN** the empty-state text renders and no year-overview chart or year
  controls are shown

## MODIFIED Requirements

### Requirement: Home dashboard empty state

When an authenticated visitor has no accounts at all, `/home` SHALL show
explanatory empty-state text with a link to `/accounts/new`, instead of
any card grid or "add your first card" prompt — a visitor with nothing to
build a dashboard from is directed to create an account first, not to
configure cards.

#### Scenario: Empty state for a visitor with no accounts

- **WHEN** an authenticated visitor with no accounts opens `/home`
- **THEN** the page shows text explaining there are none, with a link to
  `/accounts/new`, and no "add your first card" prompt

### Requirement: Dashboard card amounts are colored by sign

Every amount a dashboard card renders — an `account_stat` card's balance,
a `query_stat` card's per-currency sum, an `entry_list` card's per-entry
amount — SHALL be colored by sign, matching the rule already applied on
`/accounts` and `/reports`: red when negative, the default/neutral text
color when exactly zero, and green when positive.

#### Scenario: A negative balance is red

- **WHEN** an `account_stat` card's balance is negative
- **THEN** the balance is shown in red

#### Scenario: A positive query sum is green

- **WHEN** a `query_stat` card's sum for a currency is positive
- **THEN** that sum is shown in green

#### Scenario: A zero amount uses the neutral color

- **WHEN** any dashboard card renders an amount that is exactly zero
- **THEN** it is shown in the default/neutral text color, not red or green

### Requirement: An account stat card offers a button to add a new entry for that account

An `account_stat` card for an account on which the visitor holds at least
`append` permission SHALL include a button that navigates to
`/entries/new?account_id={id}`, preselecting that account for a new
entry, mirroring the equivalent per-row action on `/accounts`. A card for
an account where the visitor holds only `view` permission SHALL NOT show
this button. No other card type offers this button — a `query_stat`,
`entry_list`, or `bar_chart` card may span more than one account, so
there is no single account to preselect.

#### Scenario: Adding an entry from an account stat card

- **WHEN** an authenticated visitor with `append`+ permission activates
  the add-entry button on an `account_stat` card
- **THEN** the client navigates to `/entries/new?account_id={id}` for that
  account, with the account preselected

#### Scenario: A view-only account stat card offers no add-entry button

- **WHEN** an authenticated visitor has only `view` permission on the
  account an `account_stat` card references
- **THEN** no add-entry button is shown on that card

## ADDED Requirements

### Requirement: Home dashboard renders the caller's cards in their saved order

`/home` SHALL, for an authenticated visitor with at least one account,
fetch the caller's dashboard cards (`GET /api/dashboard/cards`) and render
each one according to its `type`, in the order returned. Reloading the
page SHALL always reproduce the same order until the visitor changes it
in edit mode.

#### Scenario: Cards render in their saved order

- **WHEN** an authenticated visitor has three cards in a given order
- **THEN** `/home` renders them in that same order

#### Scenario: A freshly reloaded page keeps the saved order

- **WHEN** a visitor reloads `/home` without entering edit mode
- **THEN** their cards render in the same order as before the reload

### Requirement: Account stat card shows an account's live balance

An `account_stat` card SHALL show its referenced account's title, shared
indicator and owner name when applicable (per `web-client-accounts`), and
live balance (`GET /api/accounts/{id}/balance`), formatted at the
visitor's `displayed_decimal_places`. Activating the card SHALL navigate
to that account's details page (`/accounts/{id}`).

#### Scenario: An account stat card shows the account's current balance

- **WHEN** an `account_stat` card references an account with a positive
  balance
- **THEN** the card shows that account's title and its current balance

#### Scenario: Activating an account stat card opens the account's details

- **WHEN** a visitor activates an account stat card
- **THEN** the client navigates to `/accounts/{id}` for the referenced
  account

### Requirement: Query stat card shows a filtered sum

A `query_stat` card SHALL show the per-currency sum and count for its
configured filter (`account_id`, `category_id` + `include_subcategories`,
`tag_id`, `range`, any subset optional), fetched from
`GET /api/entries/summary` with the filter's `range` resolved to concrete
`from`/`to` timestamps at render time, the same resolution `/reports`
already performs for its own filters. An omitted filter field behaves
exactly as it does when left blank on `/reports` — no `account_id` means
every account the visitor can see, and so on. Each currency present in
the response SHALL render as its own sum line.

#### Scenario: A query stat card with no filters sums everything

- **WHEN** a `query_stat` card's config has no `account_id`, `category_id`,
  or `tag_id`
- **THEN** it shows the sum across every account the visitor can see, for
  all time

#### Scenario: A relative range stays current across reloads

- **WHEN** a `query_stat` card's `range` is a relative preset (e.g. "this
  month") and the visitor reloads `/home` on a later day within the same
  month
- **THEN** the card's sum still reflects "this month" as of the reload,
  not the month the card was created in

#### Scenario: Multiple currencies render as separate sum lines

- **WHEN** a `query_stat` card's filter matches entries in two currencies
- **THEN** the card shows two sum lines, one per currency

### Requirement: Entry list card shows the most recent matching entries

An `entry_list` card SHALL show up to 10 of the most recent entries
(`GET /api/entries`, sorted by `booking_timestamp` descending) matching
its configured filter (the same optional `account_id`/`category_id`/
`include_subcategories`/`tag_id`/`range` fields as a `query_stat` card),
each row showing at least its date, title, account (per
`web-client-accounts`' `AccountLabel`, when visible to the caller), and
sign-colored amount. The card SHALL show explanatory empty text when no
entry matches.

#### Scenario: An entry list card shows recent matching entries

- **WHEN** an `entry_list` card's filter matches 15 entries
- **THEN** the card shows the 10 most recent, most-recent first

#### Scenario: An entry list card with no matches shows empty text

- **WHEN** an `entry_list` card's filter matches no entries
- **THEN** the card shows explanatory empty-state text instead of an empty
  list

### Requirement: Bar chart card shows an income/outcome chart for an account or filter

A `bar_chart` card SHALL show an income/outcome bar chart via
`GET /api/entries/flow-summary`, scoped to its configured
`account_id`/`category_id`/`include_subcategories`/`tag_id` (any subset
optional, same semantics as a `query_stat` card's filter, minus `range`)
and its configured `unit`: `month` renders twelve bars per currency for a
year, with a previous/next-year pager; `day` renders one bar per day of a
month, with a previous/next-month pager. The displayed year/month SHALL
default to the current one and is not persisted across reloads. One chart
SHALL render per currency present in the response, exactly as the
previous all-accounts chart did.

#### Scenario: A bar chart card defaults to the current year

- **WHEN** a `bar_chart` card with `unit: month` is rendered
- **THEN** it shows twelve monthly bars for the current calendar year

#### Scenario: A bar chart card can be scoped to one account

- **WHEN** a `bar_chart` card's config sets `account_id` to a single
  account
- **THEN** its bars reflect only that account's entries

#### Scenario: A bar chart card can be scoped to a category or tag

- **WHEN** a `bar_chart` card's config sets `category_id` or `tag_id`
- **THEN** its bars reflect only entries matching that filter

#### Scenario: Switching years does not persist across reloads

- **WHEN** a visitor pages a `bar_chart` card to a previous year and then
  reloads `/home`
- **THEN** the card again opens on the current year

### Requirement: A query stat, entry list, or bar chart card's heading is a customizable link through to /reports

`query_stat`, `entry_list`, and `bar_chart` cards SHALL accept an
optional `title` in their config. The card's heading SHALL show that
`title` when set, falling back to the same generated filter summary shown
today (e.g. the category/tag/account name, or "All accounts") when unset.
This heading SHALL always be a link to `/reports`, prefilling its
account/category/include-subcategories/tag/date-range controls from the
card's own filter (a `bar_chart` card's `unit`/`columns` have no
equivalent on `/reports` and are not carried over). Activating it SHALL
NOT itself run the report — `/reports` still requires its own explicit
"Generate report" activation, exactly as it does today for a visitor who
arrives there directly. `account_stat` cards are unaffected — their
heading remains the account's own name, linking to that account's details
page as before, with no `title` option.

#### Scenario: A custom title renders in place of the generated summary

- **WHEN** a `query_stat` card has `config.title` set
- **THEN** the card's heading shows that title, not the generated filter
  summary

#### Scenario: An unset title falls back to the generated filter summary

- **WHEN** a `query_stat`, `entry_list`, or `bar_chart` card has no
  `config.title`
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

### Requirement: Dashboard cards lay out as a grid, with bar chart cards always full width and entry list width configurable

`bar_chart` cards SHALL each render alone, full width, one per row.
Every other card type (`account_stat`, `query_stat`, `entry_list`) SHALL
flow in a responsive grid of up to 4 cards per row depending on viewport
width, matching the breakpoint pattern the account-cards grid already
used. `account_stat` and `query_stat` cards always occupy exactly one
column of that grid. An `entry_list` card's width is configurable via its
`columns` config (`2`-`4`, default `2`) — how many of the grid's columns
it spans; every other card type's width is fixed by its type, and only
position among same-row cards is affected by reordering.

#### Scenario: A bar chart card spans the full row

- **WHEN** `/home` renders a `bar_chart` card
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

### Requirement: Home dashboard empty state when the visitor has accounts but no cards

When an authenticated visitor has at least one account but zero dashboard
cards, `/home` SHALL show a distinct "add your first card" prompt that
opens edit mode's add-card flow, instead of an empty grid and instead of
the no-accounts empty state.

#### Scenario: A visitor with accounts but no cards sees the add-card prompt

- **WHEN** an authenticated visitor has at least one account and zero
  dashboard cards
- **THEN** `/home` shows a prompt to add their first card, not the
  no-accounts empty state

#### Scenario: Activating the prompt opens the add-card flow

- **WHEN** the visitor activates the "add your first card" prompt
- **THEN** edit mode's type picker opens, the same flow as the ordinary
  "Add card" control

### Requirement: Edit mode lets the visitor add a card

`/home` SHALL offer an edit-mode toggle. While active, an "Add card"
control SHALL open a type picker (`account_stat`/`query_stat`/
`entry_list`/`bar_chart`), followed by that type's config form — reusing
`/reports`' existing account/category/tag `<select>`s and
`DateRangeFilter` for the filter-bearing types. For `query_stat`/
`entry_list`/`bar_chart`, the form additionally offers an optional
free-text title field (absent for `account_stat`, which has none). The
`entry_list` form additionally offers a named width choice (e.g.
"Narrow"/"Wide"/"Full width", mapping to `columns` `2`/`3`/`4`)
defaulting to the narrowest option. Submitting the form SHALL create the
card (`POST /api/dashboard/cards`) at the end of the visitor's list and
render it immediately.

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

#### Scenario: Adding a card with a custom title

- **WHEN** a visitor in edit mode picks `entry_list`, enters a custom
  title, and submits
- **THEN** the new card renders with that title as its heading instead of
  a generated filter summary

#### Scenario: An invalid config form cannot be submitted

- **WHEN** a visitor in edit mode picks `account_stat` and submits without
  selecting an account
- **THEN** the form shows an inline error and no card is created

### Requirement: Edit mode lets the visitor remove a card

While edit mode is active, each card SHALL show a remove action.
Activating it SHALL delete the card (`DELETE /api/dashboard/cards/{id}`)
and remove it from the dashboard immediately, with no confirmation dialog
— cheaply reversible by re-adding an equivalent card, mirroring this
app's existing disable/enable-style interactions rather than its
delete-confirmation ones.

#### Scenario: Removing a card deletes it immediately

- **WHEN** a visitor in edit mode activates a card's remove action
- **THEN** the card is deleted and no longer renders, with no confirmation
  step

### Requirement: Edit mode lets the visitor reorder cards via move-up/move-down

While edit mode is active, each card SHALL show move-up and move-down
controls (`POST /api/dashboard/cards/{id}/move-up`/`/move-down`),
disabled at either end of the whole card list — no drag-and-drop, matching
`/categories`'s existing reorder convention so the dashboard is editable
the same way on a phone as on a desktop.

#### Scenario: Moving a card up reorders the dashboard

- **WHEN** a visitor in edit mode moves a card up
- **THEN** it renders one position earlier, swapping with the card that
  was previously before it

#### Scenario: The first card's move-up control is disabled

- **WHEN** edit mode renders the first card in the list
- **THEN** its move-up control is disabled

#### Scenario: The last card's move-down control is disabled

- **WHEN** edit mode renders the last card in the list
- **THEN** its move-down control is disabled

### Requirement: Edit mode's per-card controls render attached to the card they act on

While edit mode is active, a card's move-up/move-down/edit/remove
controls SHALL render as a toolbar directly attached to that card — no
visible gap, and sharing the card's own border/corner rounding — rather
than as a separate, detached element merely positioned nearby, so which
card a control acts on is visually unambiguous.

#### Scenario: The control toolbar has no gap from its card

- **WHEN** edit mode renders a card's controls
- **THEN** the toolbar sits flush against that card, with no visible gap
  between them

### Requirement: Edit mode lets the visitor edit an existing card's config

While edit mode is active, each card SHALL show an edit action that opens
the same config form edit mode's "Add card" flow uses, pre-filled with
that card's current `type` and `config`. The `type` selector SHALL be
disabled in this mode — a card's type is immutable after creation — with
the rest of the form fully editable. Submitting SHALL update the card
(`PATCH /api/dashboard/cards/{id}`) in place, preserving its position, and
re-render it with the saved config immediately.

#### Scenario: Editing a card pre-fills its current config

- **WHEN** a visitor in edit mode activates a card's edit action
- **THEN** the form opens with that card's type shown (disabled) and every
  other field matching its current config

#### Scenario: Saving an edited card updates it in place

- **WHEN** a visitor changes a filter field on an existing card's edit
  form and submits
- **THEN** the card re-renders with the new filter at its same position in
  the dashboard, without creating a second card

#### Scenario: The type selector is disabled while editing

- **WHEN** a visitor opens the edit form for an existing card
- **THEN** the card-type selector is disabled and cannot be changed

### Requirement: A card whose reference is no longer accessible renders a placeholder

When a card's `config` references an account, category, or tag that no
longer resolves in the visitor's own `GET /api/accounts`/`/categories`/
`/tags` response (revoked share, soft delete), `/home` SHALL render that
card as a "no longer accessible" placeholder — same card footprint in the
grid, no data fetch attempted for the missing reference — instead of
erroring or silently omitting the card. The card SHALL remain removable
via edit mode's ordinary remove action.

#### Scenario: A card with a revoked account share shows a placeholder

- **WHEN** an `account_stat` card references an account whose share was
  since revoked
- **THEN** the card renders a "no longer accessible" placeholder instead
  of a balance or an error

#### Scenario: A placeholder card can still be removed

- **WHEN** a visitor in edit mode activates the remove action on a
  placeholder card
- **THEN** it is deleted like any other card

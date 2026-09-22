# web-client-home Specification

## Purpose

The authenticated `/home` dashboard: an auth gate redirecting anonymous
visitors to the public placeholder, and a customizable, per-user grid of
dashboard cards (`account_stat`, `query_stat`, `entry_list`, `bar_chart`)
backed by `dashboard-cards`, with an edit mode to add, remove, reorder,
and edit cards. See `web-client-shell` for the sidebar's "Home" link,
`web-client-accounts` for the equivalent `/accounts` overview, and
`web-client-reports` for the filter controls several card types reuse.

## Requirements

### Requirement: Home dashboard requires authentication, redirecting to the root placeholder

`/home` SHALL be accessible only to an authenticated visitor. While the
visitor's auth status is resolving, `/home` SHALL render nothing. An
anonymous visitor navigating to `/home` SHALL be redirected to `/` — not
`/login` — since `/home` is reached from the sidebar's "Home" item, which
anonymous visitors also see.

#### Scenario: Anonymous visitor is redirected to the placeholder

- **WHEN** an anonymous visitor navigates to `/home`
- **THEN** the client redirects them to `/`

#### Scenario: No flash while auth status resolves

- **WHEN** a visitor opens `/home` and their auth status has not yet
  resolved
- **THEN** the page renders nothing until the status resolves to
  `anonymous` or `authenticated`

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

### Requirement: An overdue previewed occurrence on an entry_list card is visually distinguished but stays in order

An `entry_list` card's Upcoming block SHALL render an `overdue` row with
the same distinct, muted background tint and the same never-truncated
overdue marker `web-client-entries` defines for its own Upcoming block,
in its normal date-ascending position.

#### Scenario: An overdue row is tinted, not reordered

- **WHEN** an `entry_list` card's Upcoming block includes an overdue row
- **THEN** it renders tinted, with its overdue marker shown in full, in
  its correct chronological position

### Requirement: Each entry_list card's Upcoming row offers the existing Create transaction action

Each row in an `entry_list` card's Upcoming block SHALL offer the same
"Create transaction" action `web-client-entries`'s Upcoming block offers,
navigating to `/entries/new` with that row's `recurring_transaction_id`
and `booking_timestamp`, and rendering as a plus glyph alone below the
`sm` breakpoint exactly as that block does.

#### Scenario: Activating Create transaction on a card's Upcoming row

- **WHEN** a visitor activates "Create transaction" on an `entry_list`
  card's Upcoming block row
- **THEN** the client navigates to `/entries/new` with that row's
  `recurring_transaction_id` and `booking_timestamp`

#### Scenario: The action is icon-only on a phone

- **WHEN** an `entry_list` card's Upcoming block is rendered below the
  `sm` breakpoint
- **THEN** each row's Create transaction action shows the plus glyph
  without its label, and its accessible name is still the translated
  label

### Requirement: An entry_list card's Upcoming row title opens the recurring summary

Each row in an `entry_list` card's Upcoming block SHALL render its title
as an activatable control opening that row's recurring transaction in the
read-only recurring summary modal, the same summary the card's real entry
rows already reach through an entry's own summary.

#### Scenario: Activating the title of a card's Upcoming row

- **WHEN** a visitor activates the title of an `entry_list` card's
  Upcoming block row
- **THEN** that recurring transaction's read-only summary modal opens,
  with no navigation away from the dashboard

### Requirement: An entry_list card's Upcoming block is not drawn as a box inside the card

The Upcoming block rendered inside an `entry_list` card SHALL NOT draw
its own border, rounding or padding inside the card's. It SHALL keep its
heading and SHALL be separated from the card's real entry list by a rule.
The same block rendered on `/entries` and `/reports`, where it stands on
its own, SHALL keep its bordered box.

#### Scenario: The block renders without a nested box on the dashboard

- **WHEN** an `entry_list` card with the preview enabled renders its
  Upcoming block
- **THEN** the block renders with its heading and a rule separating it
  from the card's entry list, and with no border of its own inside the
  card's border

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

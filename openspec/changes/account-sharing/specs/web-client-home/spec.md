## MODIFIED Requirements

### Requirement: Home dashboard shows a card per account with its balance

`/home` SHALL, for an authenticated visitor, fetch and display a card for
every account they own or have any permission on, each card showing at
least its title, its `financial_institute` (when set), and its live
balance (`GET /api/accounts/{id}/balance`, formatted at the visitor's
`displayed_decimal_places`). A card for an account the visitor does not
really own SHALL additionally show a shared indicator and the real owner's
name, per `web-client-accounts`. Activating a card SHALL navigate to that
account's details page (`/accounts/{id}`).

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

### Requirement: Each card offers a button to add a new entry for that account

Each account card on `/home` for which the visitor holds at least `append`
permission SHALL include a button that navigates to
`/entries/new?account_id={id}`, preselecting that account for a new entry,
mirroring the equivalent per-row action already on `/accounts`. A card for
an account where the visitor holds only `view` permission SHALL NOT show
this button.

#### Scenario: Adding an entry from a card

- **WHEN** an authenticated visitor with `append`+ permission activates the
  add-entry button on one of their account cards
- **THEN** the client navigates to `/entries/new?account_id={id}` for that
  account, with the account preselected

#### Scenario: A view-only card offers no add-entry button

- **WHEN** an authenticated visitor with only `view` permission on a shared
  account views its card on `/home`
- **THEN** no add-entry button is shown on that card

### Requirement: Home dashboard shows an all-accounts income/outcome bar chart with a year switcher

`/home` SHALL, below the account-cards grid, display an income/outcome bar
chart covering **every** account the visitor owns or has any permission on
(disabled accounts included, matching the cards grid), fetched from
`GET /api/entries/flow-summary?unit=month&year={year}` with no
`account_id` parameter. The chart SHALL show two bars — income and
outcome — per calendar month of a selected year.

The response groups its income and outcome totals per currency. `/home`
SHALL render one bar chart per currency present in the response, stacked
vertically, each headed by that currency's code, with currency codes
ordered alphabetically. A visitor whose accounts (owned and shared) all
share one currency therefore sees exactly one chart.

The section SHALL offer previous-year and next-year controls that refetch
and redraw every per-currency chart for the newly selected year,
defaulting to the current calendar year, with no upper or lower bound on
how far the visitor may navigate. Amounts SHALL be formatted at the
visitor's `displayed_decimal_places` setting.

This chart is the shared `FlowChart` component also used on
`/accounts/{id}` — see `web-client-accounts`.

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

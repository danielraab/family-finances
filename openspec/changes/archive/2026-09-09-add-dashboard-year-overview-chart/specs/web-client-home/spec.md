## ADDED Requirements

### Requirement: Home dashboard shows an all-accounts income/outcome bar chart with a year switcher

`/home` SHALL, below the account-cards grid, display an income/outcome bar
chart covering **every** account the visitor owns (disabled accounts
included, matching the cards grid), fetched from
`GET /api/entries/flow-summary?unit=month&year={year}` with no
`account_id` parameter. The chart SHALL show two bars — income and
outcome — per calendar month of a selected year.

The response groups its income and outcome totals per currency. `/home`
SHALL render one bar chart per currency present in the response, stacked
vertically, each headed by that currency's code, with currency codes
ordered alphabetically. A visitor whose accounts all share one currency
therefore sees exactly one chart.

The section SHALL offer previous-year and next-year controls that refetch
and redraw every per-currency chart for the newly selected year,
defaulting to the current calendar year, with no upper or lower bound on
how far the visitor may navigate. Amounts SHALL be formatted at the
visitor's `displayed_decimal_places` setting.

This chart is the shared `FlowChart` component also used on
`/accounts/{id}` — see `web-client-accounts`.

#### Scenario: Chart shows twelve months of the current year by default

- **WHEN** an authenticated visitor with at least one account opens
  `/home`
- **THEN** below the account cards, a bar chart shows one income bar and
  one outcome bar for each month of the current year, aggregated across
  all their accounts

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

- **WHEN** an authenticated visitor with no accounts opens `/home`
- **THEN** the empty-state text renders and no year-overview chart or year
  controls are shown

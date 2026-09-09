## MODIFIED Requirements

### Requirement: Account details page shows an income/outcome bar chart with a year switcher

`/accounts/{id}` SHALL, above "Recent entries," display a bar chart with
two bars per month of a selected year — income and outcome — for that
account, fetched from
`GET /api/entries/flow-summary?account_id={id}&unit=month&year={year}`.
The page SHALL offer previous-year and next-year controls that refetch and
redraw the chart for the newly selected year, with no upper or lower bound
on how far the visitor may navigate. Only the buckets' totals in the
account's own currency are shown.

This chart SHALL be rendered by a reusable `FlowChart` component that owns
the flow-summary fetch, the year pager, and the `FlowBucket`→bar-data
mapping (with localized month labels), and composes the presentational
`BarChart`. The component takes an optional list of account ids (omitted
means every account the caller owns) and an optional currency (given means
render a single chart filtered to it; omitted means one chart per currency
in the response). The account details page passes this account's id and
its currency. The same component renders the all-accounts year overview on
`/home` (see `web-client-home`). Its user-facing strings live under a
shared `flowChart.*` i18n namespace.

#### Scenario: Chart shows twelve months of the current year by default

- **WHEN** an authenticated visitor opens an account's details page
- **THEN** the chart shows one income bar and one outcome bar for each
  month of the current year

#### Scenario: Switching to the previous year

- **WHEN** an authenticated visitor activates the previous-year control
- **THEN** the chart refetches and redraws for the prior year

#### Scenario: Switching to the next year

- **WHEN** an authenticated visitor activates the next-year control
- **THEN** the chart refetches and redraws for the following year, with no
  restriction on navigating past the current year

#### Scenario: A month with no entries still renders

- **WHEN** the selected year includes a month with no matching entries
- **THEN** that month's bars render at zero rather than being omitted or
  erroring

#### Scenario: Only the account's currency is charted

- **WHEN** an account's details page renders its flow chart
- **THEN** the bars reflect only the flow-summary totals in that account's
  currency

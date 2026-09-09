## ADDED Requirements

### Requirement: Account details page shows a running-balance line chart with a month switcher

`/accounts/{id}` SHALL, below the income/outcome bar chart, display a line
chart of the account's running balance sampled per day over a selected
month, fetched from
`GET /api/entries/balance-series?account_id={id}&unit=day&year={year}&month={month}`.
The x-axis SHALL span the days of the selected month; the y-axis SHALL be
signed, with a visible zero rule line whenever the plotted range crosses
zero. The line SHALL be drawn as a step (the balance holds flat between the
entries that move it) rather than a straight interpolation between sample
points.

The page SHALL offer previous-month and next-month controls that refetch
and redraw the chart for the newly selected month, with no upper or lower
bound on how far the visitor may navigate. The chart SHALL default to the
current month.

The line chart SHALL be a reusable, presentational, hand-rolled SVG
component (`LineChart`) that takes its data and series definitions as props
and fetches nothing itself — the same convention as the existing
`BarChart` — so it can be reused by a future chart on another page.

#### Scenario: Chart shows the current month by default

- **WHEN** an authenticated visitor opens an account's details page
- **THEN** the line chart shows the account's daily balance for each day of
  the current month, in the account's currency

#### Scenario: Switching to the previous month

- **WHEN** the visitor activates the previous-month control
- **THEN** the chart refetches and redraws for the prior month, rolling the
  year back when moving from January to December

#### Scenario: Switching to the next month

- **WHEN** the visitor activates the next-month control
- **THEN** the chart refetches and redraws for the following month, with no
  restriction on navigating past the current month

#### Scenario: A month with no activity renders a flat line

- **WHEN** the selected month has no entries that move the balance
- **THEN** the line renders flat at the month's opening balance rather than
  being omitted or erroring

#### Scenario: A balance that goes negative shows a zero line

- **WHEN** the account's balance is above zero on some days of the selected
  month and below zero on others
- **THEN** the chart draws a zero rule line and plots the line on both
  sides of it

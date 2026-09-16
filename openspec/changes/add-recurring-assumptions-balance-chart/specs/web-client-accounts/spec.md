## MODIFIED Requirements

### Requirement: Account details page shows a running-balance line chart with a month switcher

`/accounts/{id}` SHALL, below the income/outcome bar chart, display a line
chart of the account's running balance sampled per day over a selected
month, fetched from
`GET /api/entries/balance-series?account_id={id}&unit=day&year={year}&month={month}`.
The x-axis SHALL span the days of the selected month; the y-axis SHALL be
signed and framed to the plotted data, with a visible zero rule line
whenever the plotted range crosses zero. The line SHALL be drawn as a step
(the balance holds flat between the entries that move it) rather than a
straight interpolation between sample points.

The page SHALL offer previous-month and next-month controls that refetch
and redraw the chart for the newly selected month, with no upper or lower
bound on how far the visitor may navigate. The chart SHALL default to the
current month.

Alongside the month switcher, the page SHALL offer a "Show recurring
assumptions" toggle, off by default and not persisted across reloads. When
on, the page SHALL fetch `GET /api/recurring-transactions/preview` scoped
to this one account, bounded by the visitor's `recurring_preview_horizon`
setting (the same cutoff resolution `bar_chart` dashboard cards already
use — the chart's own displayed month is a display window, not a query
filter, so it plays no part in the cutoff), and render the resulting
projection as a second, visually distinct (dashed) line continuing from
today's real balance through the end of the resolved horizon, using the
same cumulative-delta computation `line_chart` dashboard cards use. A
recurring transaction's occurrence already overdue (its next suggested
date in the past) SHALL NOT be included in the projection, mirroring how
it's excluded from `bar_chart`'s own projected segment.

The line chart SHALL be a reusable, presentational, hand-rolled SVG
component (`LineChart`) that takes its data and series definitions as
props and fetches nothing itself — the same convention as the existing
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

#### Scenario: Enabling the toggle overlays a projected balance line

- **WHEN** a visitor enables "Show recurring assumptions" on an account
  with an upcoming recurring transaction due before the resolved horizon
- **THEN** the chart draws a dashed line, starting at today's real balance
  point, that adds each occurrence's amount onto the running total as its
  date passes

#### Scenario: The projected line starts exactly where the real line is today

- **WHEN** the toggle is on and the displayed month includes today
- **THEN** the dashed projected line and the solid real line coincide
  exactly at today's point, diverging only afterward

#### Scenario: An overdue recurring transaction is not projected

- **WHEN** the toggle is on and a recurring transaction's next occurrence
  is already overdue
- **THEN** that occurrence's amount is not added to the projected line

#### Scenario: Disabling the toggle removes the projected line

- **WHEN** a visitor turns "Show recurring assumptions" back off
- **THEN** the chart reverts to showing only the real balance line, with no
  legend entry or tooltip row for a projected value

#### Scenario: The toggle does not persist across reloads

- **WHEN** a visitor enables the toggle and then reloads `/accounts/{id}`
- **THEN** the toggle is off again, mirroring the chart's own month
  resetting to the current one

## MODIFIED Requirements

### Requirement: Account balances and recent-entry amounts are colored by sign

The accounts overview's per-account balance, an account's detail-page
balance, and each amount in an account's recent-entries list SHALL be
rendered in a color reflecting sign: red when negative, the default/
neutral text color when exactly zero, and green when positive. Each row of
the recent-entries list, being backed by a single entry, SHALL additionally
render its amount underlined when that entry's `kind` is
`balance_adjustment`. The accounts overview and account-detail balance
figures are computed totals, not a single entry, and are therefore never
underlined. A recent-entries row for a `balance_adjustment` entry SHALL
additionally show its computed delta (`amount`) as a small, gray annotation
next to its reading, rendered without sign-based coloring.

#### Scenario: Negative account balance is red

- **WHEN** the accounts overview or an account's detail page renders a
  negative balance
- **THEN** the balance is shown in red

#### Scenario: Zero account balance stays neutral

- **WHEN** the accounts overview or an account's detail page renders a
  balance of exactly zero
- **THEN** the balance is shown in the default text color, neither red nor
  green

#### Scenario: Positive account balance is green

- **WHEN** the accounts overview or an account's detail page renders a
  positive balance
- **THEN** the balance is shown in green

#### Scenario: A balance adjustment in recent entries is underlined

- **WHEN** an account's recent-entries list renders an entry whose `kind`
  is `balance_adjustment`
- **THEN** its amount is rendered underlined, in addition to its sign color

#### Scenario: A transaction in recent entries is not underlined

- **WHEN** an account's recent-entries list renders an entry whose `kind`
  is `transaction`
- **THEN** its amount is rendered without an underline

#### Scenario: Balance figures are never underlined

- **WHEN** the accounts overview or an account's detail page renders its
  balance figure
- **THEN** the figure is never underlined, regardless of sign

#### Scenario: A balance adjustment shows its delta alongside its reading

- **WHEN** an account's recent-entries list renders an entry whose `kind`
  is `balance_adjustment`
- **THEN** its computed delta is shown next to its reading, in a smaller,
  gray typeface, not colored by sign

## ADDED Requirements

### Requirement: Account details page shows an income/outcome bar chart with a year switcher

`/accounts/{id}` SHALL, above "Recent entries," display a bar chart with
two bars per month of a selected year — income and outcome — for that
account, fetched from `GET /api/entries/flow-summary?account_id={id}&unit=month&year={year}`.
The page SHALL offer previous-year and next-year controls that refetch and
redraw the chart for the newly selected year, with no upper or lower bound
on how far the visitor may navigate.

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

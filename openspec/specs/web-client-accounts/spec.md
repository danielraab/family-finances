# web-client-accounts Specification

## Purpose

The authenticated `/accounts` pages: the sidebar link, the auth
gate, the accounts overview (balance and status per account), the
account details page (fields plus recent entries), and the
create/edit form including disable/enable and soft delete. See
`accounts` for the backend capability and `web-client-entries` for
the linked entry ledger.

## Requirements

### Requirement: Accounts link in the sidebar

The `Sidebar` navigation SHALL contain an "Accounts" item, visible to an
authenticated visitor, that navigates to `/accounts` and is shown as active
for `/accounts` and every route nested under it.

#### Scenario: Navigating to accounts from the sidebar

- **WHEN** an authenticated visitor activates "Accounts" in the sidebar
- **THEN** the client navigates to `/accounts`

#### Scenario: Nested account routes still highlight the sidebar item

- **WHEN** an authenticated visitor is on `/accounts/{id}/edit`
- **THEN** the "Accounts" sidebar item is shown as active

### Requirement: Accounts routes require authentication

`/accounts` and every route nested under it SHALL be accessible only to an
authenticated visitor. An anonymous visitor navigating to any accounts
route SHALL be redirected to `/login`.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/accounts`
- **THEN** the client redirects them to `/login`

### Requirement: Accounts overview lists every account with its balance and status

`/accounts` SHALL, on mount, fetch and display every account the visitor
owns (`GET /api/accounts`), each row showing at least its title, type,
currency, live balance (`GET /api/accounts/{id}/balance`, formatted at the
visitor's `displayed_decimal_places`), and a status indicator reflecting
whether it is disabled and/or closed. When the visitor has no accounts, the
page SHALL show explanatory empty-state text with a way to create one
instead of an empty list.

#### Scenario: Accounts and balances are listed

- **WHEN** an authenticated visitor with two accounts opens `/accounts`
- **THEN** both accounts are shown, each with its current live balance

#### Scenario: Disabled account is visually distinguished

- **WHEN** one of the visitor's accounts is disabled
- **THEN** its row indicates the disabled state

#### Scenario: Empty state

- **WHEN** an authenticated visitor with no accounts opens `/accounts`
- **THEN** the page shows text explaining there are none, with a way to
  create one

### Requirement: Account details page shows account fields and recent entries

`/accounts/{id}` SHALL fetch and display the account's full details
(`GET /api/accounts/{id}`) and its most recent entries
(`GET /api/entries` filtered to that account, sorted by booking timestamp
descending, limited to a small fixed page), each linking to that entry's
edit page. The page SHALL offer a link to `/entries?account_id={id}` for
the account's complete, filterable entry list.

#### Scenario: Recent entries link to the full filtered list

- **WHEN** an authenticated visitor opens an account's details page
- **THEN** they see its most recent entries and a link that navigates to
  `/entries?account_id={id}`

#### Scenario: Cross-owner access is not found

- **WHEN** an authenticated visitor navigates to `/accounts/{id}` for an
  account they do not own
- **THEN** the page reflects the backend's `404` (not found), not the
  account's details

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

### Requirement: Account details page shows an income/outcome bar chart with a year switcher

`/accounts/{id}` SHALL, above "Recent entries," display a bar chart with
two bars per month of a selected year — income and outcome — for that
account, fetched from `GET /api/entries/flow-summary?account_id={id}&unit=month&year={year}`.
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

### Requirement: Creating and editing an account

`/accounts/new` SHALL offer a form for `title`, `description`, `type_id`
(populated from `GET /api/account-types`), `currency`, `financial_institute`,
`opening_date`, and `closing_date`, submitting `POST /api/accounts` on
success and navigating to the new account's details page.
`/accounts/{id}/edit` SHALL offer the same fields pre-populated from
`GET /api/accounts/{id}`, submitting `PATCH /api/accounts/{id}` (or
equivalent update) on save. Both forms SHALL validate client-side to the
same shape the backend enforces (currency as three letters, closing date
not before opening date) and surface the backend's validation error when a
submission is rejected.

The `financial_institute` field SHALL remain free text, and SHALL offer
suggestions drawn from the distinct, non-empty `financial_institute` values
already present on the visitor's own accounts (fetched via
`GET /api/accounts`), deduplicated by exact string match and sorted
alphabetically. Suggestions SHALL be shown, as clickable chips, whenever
the field has focus — every suggestion when the field is empty, narrowed to
a case-insensitive substring match against the field's current value as the
visitor types — and hidden when the field loses focus. Activating a chip
SHALL set the field to that chip's exact value. Typing a value that matches
no suggestion SHALL remain valid and submittable, unchanged from today.

#### Scenario: Creating an account

- **WHEN** an authenticated visitor submits the create form with valid
  fields
- **THEN** `POST /api/accounts` is called and, on success, the visitor is
  taken to the new account's details page

#### Scenario: Invalid closing date is caught before submission

- **WHEN** an authenticated visitor sets a closing date earlier than the
  opening date on either form
- **THEN** the form shows a validation error and does not submit

#### Scenario: Financial institute suggestions appear on focus

- **WHEN** an authenticated visitor with at least one existing account
  carrying a `financial_institute` value focuses the financial institute
  field on the create or edit form
- **THEN** that value appears as a clickable suggestion chip, alongside
  every other distinct value already used across the visitor's own
  accounts, sorted alphabetically

#### Scenario: Typing narrows the suggestions

- **WHEN** an authenticated visitor types into the financial institute
  field while suggestions are shown
- **THEN** only suggestions containing the typed text (case-insensitive)
  remain visible

#### Scenario: Selecting a suggestion fills the field

- **WHEN** an authenticated visitor activates a financial institute
  suggestion chip
- **THEN** the field's value becomes exactly that chip's text

#### Scenario: A new institute name is still accepted

- **WHEN** an authenticated visitor types a financial institute value that
  matches none of their existing accounts' values and submits the form
- **THEN** the account is created (or updated) with that value, unchanged
  from today's free-text behavior

#### Scenario: No suggestions when the visitor has none to offer

- **WHEN** an authenticated visitor with no accounts, or none carrying a
  `financial_institute` value, focuses the field
- **THEN** no suggestion chips are shown

### Requirement: Disabling, enabling, and soft-deleting an account require confirmation

The edit page SHALL offer a Disable action when the account is enabled
(`POST /api/accounts/{id}/disable`) or an Enable action when it is disabled
(`POST /api/accounts/{id}/enable`), and a (soft) delete action
(`DELETE /api/accounts/{id}`). Each SHALL require an explicit confirmation
step before the request is sent, mirroring the confirmation pattern
`/settings/users` uses for user lifecycle actions; the delete confirmation's
copy SHALL state that the action cannot be undone.

#### Scenario: Disabling requires confirmation

- **WHEN** an authenticated visitor activates "Disable" on an enabled
  account's edit page
- **THEN** a confirmation step appears and
  `POST /api/accounts/{id}/disable` is not called until it is confirmed

#### Scenario: Soft delete confirmation warns it is permanent

- **WHEN** an authenticated visitor activates "Delete" on an account's edit
  page
- **THEN** the confirmation step's copy states the action cannot be undone,
  and `DELETE /api/accounts/{id}` is not called until confirmed

#### Scenario: Deleted account disappears from the overview

- **WHEN** an authenticated visitor confirms deleting an account
- **THEN** it no longer appears in `/accounts`

### Requirement: The account form offers an icon and colour picker

The account create form and the account edit form SHALL include the shared
`IconColorPicker`, letting the user set, change, or clear the account's
`icon` and `color` independently. On submit, the chosen values SHALL be sent
on the `POST /api/accounts` / `PATCH /api/accounts/{id}` body; an unset icon
or colour SHALL be sent such that the field is created without it, and
clearing a previously set field on edit SHALL send the empty string so the
backend clears it. The picker's presence SHALL NOT make either field
required — an account can still be created and saved with neither set.

#### Scenario: Setting an icon and colour while creating an account

- **WHEN** a visitor fills the create form, picks an icon and a colour, and
  submits
- **THEN** `POST /api/accounts` is called with that `icon` and `color`, and
  the new account carries them

#### Scenario: Clearing an account's colour from the edit form

- **WHEN** a visitor opens the edit form for an account that has a `color`,
  clears the colour in the picker, and submits
- **THEN** `PATCH /api/accounts/{id}` is called with `color` as `""` and the
  account's colour becomes unset

#### Scenario: Creating an account without touching the picker

- **WHEN** a visitor completes the create form leaving the icon/colour
  picker untouched and submits
- **THEN** the account is created with no `icon` and no `color`

### Requirement: Account surfaces render the account's icon and colour before its title

The accounts overview, the account detail page, and the home account cards
SHALL render each account's `EntityIcon` badge immediately before its title,
via the shared `AccountLabel` component, when the account has an `icon`
and/or `color` set. An account with neither set SHALL render exactly as
before.

#### Scenario: The account detail header shows the badge

- **WHEN** an authenticated visitor opens the detail page of an account that
  has an `icon` and a `color`
- **THEN** the header shows the icon/colour badge immediately before the
  account title

#### Scenario: A home account card shows the badge

- **WHEN** the home page renders an account card for an account that has an
  `icon`
- **THEN** the card shows that icon immediately before the account title

#### Scenario: An account with no icon or colour is unchanged

- **WHEN** the accounts overview lists an account with neither `icon` nor
  `color`
- **THEN** that row shows just the title, with no badge and no layout shift

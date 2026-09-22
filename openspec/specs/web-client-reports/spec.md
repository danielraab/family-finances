# web-client-reports Specification

## Purpose

The authenticated `/reports` page: optionally filter by a category
(optionally including its subcategories), a tag, an account, and a date
range — any combination, or none — then explicitly generate a report
showing the matching transaction entries and their sum per currency,
without ever fetching automatically as filters change.

## Requirements

### Requirement: Reports link in the sidebar

The `Sidebar` navigation SHALL contain a "Reports" item, visible to an
authenticated visitor, that navigates to `/reports` and is shown as active
for `/reports` and every route nested under it.

#### Scenario: Navigating to reports from the sidebar

- **WHEN** an authenticated visitor activates "Reports" in the sidebar
- **THEN** the client navigates to `/reports`

### Requirement: The reports route requires authentication

`/reports` SHALL be accessible only to an authenticated visitor. An
anonymous visitor navigating to `/reports` SHALL be redirected to
`/login`.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/reports`
- **THEN** the client redirects them to `/login`

### Requirement: A selected category offers an "include subcategories" checkbox

When a category is selected on `/reports`, an "include subcategories"
checkbox SHALL be shown, defaulting to checked. Checked SHALL request
`GET /api/entries` and `GET /api/entries/summary` with `category_mode`
omitted or `subtree`; unchecked SHALL request them with
`category_mode=exact`. Whether a tag is also selected SHALL NOT affect
whether this checkbox is shown — it is governed solely by whether a
category is selected.

#### Scenario: Checkbox defaults to checked

- **WHEN** an authenticated visitor selects a category on `/reports`
- **THEN** the "include subcategories" checkbox is shown, checked

#### Scenario: Unchecking narrows the report to the exact category

- **WHEN** an authenticated visitor unchecks "include subcategories" and
  generates the report
- **THEN** the request uses `category_mode=exact`, and entries carrying a
  descendant category are excluded from both the list and the sum

#### Scenario: Checkbox is still shown when a tag is also selected

- **WHEN** an authenticated visitor has both a category and a tag selected
  on `/reports`
- **THEN** the "include subcategories" checkbox is shown

### Requirement: Account and date-range filters narrow the report

`/reports` SHALL offer an account filter and a date-range filter (the shared
date-range filter — see `web-client-date-range-filter`), applied in addition
to the category-or-tag selection, the same way `/entries`'s equivalent
filters narrow its list. The account filter's choices SHALL include every
non-deleted account the visitor owns or has any permission on (view and
up), since reports are read-only.

#### Scenario: Narrowing by account

- **WHEN** an authenticated visitor sets an account filter alongside a
  category or tag selection and generates the report
- **THEN** only entries on that account are included in the results and
  sum

#### Scenario: Narrowing by date range

- **WHEN** an authenticated visitor sets a date range alongside a category
  or tag selection and generates the report
- **THEN** only entries whose `booking_timestamp` falls in that range are
  included in the results and sum

#### Scenario: A shared account appears in the account filter

- **WHEN** an authenticated visitor has any permission (including `view`)
  on an account they do not really own
- **THEN** that account appears among the account filter's choices, and a
  report narrowed to it includes its matching entries

### Requirement: The report has no default date range

The report SHALL apply no date filter when `/reports` is opened with no
date-range parameter (`range`, `from`, or `to`) present in the URL and no
filter state is restored from browser-local storage (see "Returning to the
report restores the last-applied filters") — the date-range filter's
dropdown SHALL show "All time" selected with both bounds empty and
disabled, naming the unrestricted range that is in effect.

#### Scenario: Opening reports with no filters and nothing persisted applies no date restriction

- **WHEN** an authenticated visitor with no previously persisted filter
  state opens `/reports` with no `range`, `from`, or `to` parameter and
  generates a report
- **THEN** the results are not restricted by date, and the date-range
  filter shows "All time" selected with both fields empty

### Requirement: The report's filter state lives in the URL

`/reports` SHALL represent its category-or-tag selection (including the
subcategories checkbox), account, and date-range filters as typed URL
search parameters, readable and writable through TanStack Router's
search-param APIs. Changing a control SHALL update the URL immediately, and
SHALL also write the same state to browser-local storage, keyed per
visitor's browser (not synced to the account or the backend) — see
"Returning to the report restores the last-applied filters" for when that
stored state is read back. Reloading a URL with search parameters SHALL
restore the same filter selections without generating a report.

#### Scenario: Changing a filter updates the URL without generating a report

- **WHEN** an authenticated visitor changes the category, tag, account, or
  date-range control
- **THEN** the corresponding URL search parameter changes to match, the
  same state is written to browser-local storage, and no request is sent

#### Scenario: A filtered view's controls survive a reload

- **WHEN** an authenticated visitor sets filters, generates a report, then
  reloads the page
- **THEN** the same filter selections are shown in the controls, but the
  report is not automatically regenerated

### Requirement: Returning to the report restores the last-applied filters

`/reports` SHALL restore the visitor's most recently persisted filter state
into its draft controls when it is arrived at with a completely bare URL —
no `category_id`, `include_subcategories`, `tag_id`, `account_id`, `range`,
`from`, `to`, `show_recurring`, `sort`, or `dir` parameter present at all —
replacing the URL rather than leaving the bare arrival in browser history,
if any state has been persisted. This applies to any ordinary navigation to
bare `/reports` (a sidebar link, a link from elsewhere in the app).
Arriving with any explicit parameter already present SHALL NOT be
overridden by persisted state. Restoring persisted filters SHALL NOT itself
generate a report — "A report is only generated on explicit action"
applies exactly as it does to a bookmarked or reloaded URL. When nothing
has been persisted yet, a bare arrival SHALL fall back to the default view
(see "The report has no default date range").

#### Scenario: A sidebar click restores the last-applied filters

- **WHEN** an authenticated visitor sets a category filter and a date range
  on `/reports`, navigates elsewhere in the app, then clicks "Reports" in
  the sidebar
- **THEN** they land on `/reports` with the same category filter and date
  range shown in the controls, but no report is automatically generated

#### Scenario: No persisted state falls back to the default view

- **WHEN** an authenticated visitor with nothing yet persisted opens a bare
  `/reports`
- **THEN** the default view applies, as described in "The report has no
  default date range"

#### Scenario: An explicit link is not overridden by persisted state

- **WHEN** an authenticated visitor with a persisted account filter follows
  a link to `/reports?category_id={id}`
- **THEN** the report's controls are filtered only by that category, not
  also by the previously persisted account

### Requirement: A report is only generated on explicit action

`/reports` SHALL NOT fetch entries or a sum automatically — not on initial
load, not when arriving with filters already present in the URL, and not
as any filter control changes. It SHALL offer a "Generate report" action
that is always enabled and that, when activated, fetches
`GET /api/entries` (for display) and `GET /api/entries/summary` (for the
sum) using the currently selected filters, whatever they are — including
no category, no tag, and no account or date filter, in which case the
report covers every matching transaction entry. Changing a filter after a
report has been generated SHALL leave the previously generated results on
screen until "Generate report" is activated again — but SHALL show text
indicating the displayed results no longer match the current filter
selection, distinguishing it from a freshly generated, up-to-date report.
That indication SHALL disappear as soon as the filter controls again match
the filters the displayed report was generated with (for example, undoing
the change that caused it), even without activating "Generate report"
again.

#### Scenario: Arriving with filters in the URL does not auto-generate

- **WHEN** an authenticated visitor navigates directly to `/reports` with
  a category, tag, account, or date filter already set in the URL
- **THEN** no request is sent until "Generate report" is activated

#### Scenario: Generating a report fetches the list and the sum

- **WHEN** an authenticated visitor activates "Generate report"
- **THEN** both the matching entries and their per-currency sum are
  fetched and displayed using the currently selected filters

#### Scenario: Changing a filter after generating leaves the prior report visible

- **WHEN** an authenticated visitor changes a filter after a report has
  already been generated
- **THEN** the previously generated entries and sum remain displayed
  unchanged until "Generate report" is activated again

#### Scenario: Changing a filter after generating shows a stale-results hint

- **WHEN** an authenticated visitor changes a filter after a report has
  already been generated
- **THEN** text is shown indicating the displayed results no longer match
  the current filters and that "Generate report" should be activated again

#### Scenario: Reverting a filter change clears the stale-results hint

- **WHEN** an authenticated visitor changes a filter (triggering the
  stale-results hint) and then changes it back to the value the displayed
  report was generated with
- **THEN** the stale-results hint disappears, without "Generate report"
  having been activated again

#### Scenario: Generating with neither category nor tag selected

- **WHEN** an authenticated visitor activates "Generate report" without
  having selected a category or a tag
- **THEN** `GET /api/entries` and `GET /api/entries/summary` are requested
  with no `category_id` or `tag_id`, and the matching transaction entries
  and their per-currency sum are displayed, narrowed only by any account
  or date-range filter that is set

### Requirement: The report shows matching entries and a per-currency sum

Once generated, `/reports` SHALL display the matching entries (as
transaction entries — `balance_adjustment` entries are never included) via
the same cursor-based infinite scroll behavior as `/entries`, and SHALL
display the sum for each currency present among the results, from
`GET /api/entries/summary`'s response. When the generated report matches
no entries, the page SHALL show text indicating that, distinct from the
pre-generation empty state.

Each result row's entry title SHALL open that entry's read-only summary
modal (per `web-client-entries`) when activated, rather than being
inert text.

#### Scenario: A single-currency report shows one sum

- **WHEN** a generated report's matching entries all belong to
  same-currency accounts
- **THEN** exactly one sum is displayed, in that currency

#### Scenario: A multi-currency report shows a sum per currency

- **WHEN** a generated report's matching entries belong to accounts of two
  different currencies
- **THEN** a separate sum is displayed for each currency, never a single
  combined figure

#### Scenario: A generated report with no matches

- **WHEN** a generated report's filters match no entries
- **THEN** the page shows text indicating no entries match the report's
  filters, distinct from the pre-generation prompt shown before "Generate
  report" has ever been activated

#### Scenario: A result row's title opens the entry summary

- **WHEN** the visitor activates an entry's title in the results table
- **THEN** that entry's read-only summary modal opens and the report's
  results are not navigated away from

### Requirement: A linked entry shows the same recurring-transaction badge in report results

The `/reports` results table SHALL show the same recurring-transaction
badge `web-client-entries` defines for the ledger, on any result entry
whose `recurring_transaction_id` is set, opening that recurring
transaction's read-only summary modal when activated.

#### Scenario: Report result shows the badge

- **WHEN** a generated report's results include an entry with a non-null
  `recurring_transaction_id`
- **THEN** its row in the results table shows the recurring-transaction
  badge

#### Scenario: The badge opens the recurring summary

- **WHEN** the visitor activates that badge
- **THEN** the recurring transaction's read-only summary modal opens and
  the browser does not navigate

### Requirement: An overdue previewed occurrence is visually distinguished but stays in chronological order

A row in the report's Upcoming block whose `overdue` is `true` SHALL
render with a distinct, muted background tint from a non-overdue row, and
SHALL carry the same legible, never-truncated overdue marker
`web-client-entries` defines for its own Upcoming block. It SHALL NOT be
moved out of its normal position in the block's date-ascending order.

#### Scenario: An overdue row is tinted, not reordered

- **WHEN** the report's Upcoming block includes one overdue row among
  several non-overdue rows
- **THEN** the overdue row renders with the distinct background tint and
  its overdue marker in full, in its correct chronological position, not
  pulled to the top or bottom

### Requirement: Each Upcoming row offers the existing Create transaction action, prefilled to that occurrence's date

Each row in the report's Upcoming block SHALL offer a "Create transaction"
action, navigating to `/entries/new?recurring_transaction_id={id}
&booking_timestamp={that row's booking_timestamp}` (see
`web-client-recurring-transactions`'s date-override requirement), and
rendering as a plus glyph alone below the `sm` breakpoint exactly as
`web-client-entries`' Upcoming block does.

#### Scenario: Activating Create transaction on a report's Upcoming row

- **WHEN** an authenticated visitor activates "Create transaction" on a
  report's Upcoming block row
- **THEN** the client navigates to `/entries/new` with that row's
  `recurring_transaction_id` and `booking_timestamp`, and the form
  prefills accordingly

### Requirement: A report's Upcoming row title opens the recurring summary

Each row in the report's Upcoming block SHALL render its title as an
activatable control opening that row's recurring transaction in the
read-only recurring summary modal — the same summary the recurring badge
in the report's results table already opens.

#### Scenario: Activating the title of a report's Upcoming row

- **WHEN** an authenticated visitor activates the title of a report's
  Upcoming block row
- **THEN** that recurring transaction's read-only summary modal opens,
  with no navigation away from the report

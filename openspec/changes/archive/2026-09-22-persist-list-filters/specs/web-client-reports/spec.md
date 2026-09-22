## MODIFIED Requirements

### Requirement: The report has no default date range

The report SHALL apply no date filter when `/reports` is opened with no
date-range parameter (`range`, `from`, or `to`) present in the URL and no
filter state is restored from browser-local storage (see "Returning to the
report restores the last-applied filters") — the date-range filter's
dropdown SHALL show "Custom" selected with both bounds empty, matching its
behavior before the shared date-range filter existed.

#### Scenario: Opening reports with no filters and nothing persisted applies no date restriction

- **WHEN** an authenticated visitor with no previously persisted filter
  state opens `/reports` with no `range`, `from`, or `to` parameter and
  generates a report
- **THEN** the results are not restricted by date, and the date-range
  filter shows "Custom" selected with both fields empty

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

## ADDED Requirements

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

# web-client-reports Specification

## Purpose

The authenticated `/reports` page: pick a category (optionally including
its subcategories) or a tag, narrow by account and date, and explicitly
generate a report showing the matching transaction entries and their sum
per currency, without ever fetching automatically as filters change.

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

### Requirement: Category and tag are an exclusive selection

`/reports` SHALL offer a choice between filtering by category or by tag,
never both at once. Selecting a category SHALL clear any selected tag, and
selecting a tag SHALL clear any selected category. Neither is required to
be set to interact with the rest of the page's controls.

#### Scenario: Selecting a tag clears the category selection

- **WHEN** an authenticated visitor has a category selected and then
  selects a tag
- **THEN** the category selection is cleared, leaving only the tag
  selected

#### Scenario: Selecting a category clears the tag selection

- **WHEN** an authenticated visitor has a tag selected and then selects a
  category
- **THEN** the tag selection is cleared, leaving only the category
  selected

### Requirement: A selected category offers an "include subcategories" checkbox

When a category is selected on `/reports`, an "include subcategories"
checkbox SHALL be shown, defaulting to checked. Checked SHALL request
`GET /api/entries` and `GET /api/entries/summary` with `category_mode`
omitted or `subtree`; unchecked SHALL request them with
`category_mode=exact`. This checkbox SHALL NOT be shown while a tag is
selected instead.

#### Scenario: Checkbox defaults to checked

- **WHEN** an authenticated visitor selects a category on `/reports`
- **THEN** the "include subcategories" checkbox is shown, checked

#### Scenario: Unchecking narrows the report to the exact category

- **WHEN** an authenticated visitor unchecks "include subcategories" and
  generates the report
- **THEN** the request uses `category_mode=exact`, and entries carrying a
  descendant category are excluded from both the list and the sum

### Requirement: Account and date-range filters narrow the report

`/reports` SHALL offer an account filter and a `from`/`to` date-range
filter, applied in addition to the category-or-tag selection, the same way
`/entries`'s equivalent filters narrow its list.

#### Scenario: Narrowing by account

- **WHEN** an authenticated visitor sets an account filter alongside a
  category or tag selection and generates the report
- **THEN** only entries on that account are included in the results and
  sum

#### Scenario: Narrowing by date range

- **WHEN** an authenticated visitor sets a `from`/`to` date range alongside
  a category or tag selection and generates the report
- **THEN** only entries whose `booking_timestamp` falls in that range are
  included in the results and sum

### Requirement: The report's filter state lives in the URL

`/reports` SHALL represent its category-or-tag selection (including the
subcategories checkbox), account, and date-range filters as typed URL
search parameters, readable and writable through TanStack Router's
search-param APIs. Changing a control SHALL update the URL immediately.
Reloading a URL with search parameters SHALL restore the same filter
selections without generating a report.

#### Scenario: Changing a filter updates the URL without generating a report

- **WHEN** an authenticated visitor changes the category, tag, account, or
  date-range control
- **THEN** the corresponding URL search parameter changes to match, and no
  request is sent

#### Scenario: A filtered view's controls survive a reload

- **WHEN** an authenticated visitor sets filters, generates a report, then
  reloads the page
- **THEN** the same filter selections are shown in the controls, but the
  report is not automatically regenerated

### Requirement: A report is only generated on explicit action

`/reports` SHALL NOT fetch entries or a sum automatically — not on initial
load, not when arriving with filters already present in the URL, and not
as any filter control changes. It SHALL offer a "Generate report" action
that, when activated, fetches `GET /api/entries` (for display) and
`GET /api/entries/summary` (for the sum) using the currently selected
filters. Changing a filter after a report has been generated SHALL leave
the previously generated results on screen until "Generate report" is
activated again — but SHALL show text indicating the displayed results no
longer match the current filter selection, distinguishing it from a
freshly generated, up-to-date report. That indication SHALL disappear as
soon as the filter controls again match the filters the displayed report
was generated with (for example, undoing the change that caused it), even
without activating "Generate report" again.

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

#### Scenario: Neither category nor tag selected

- **WHEN** an authenticated visitor activates "Generate report" without
  having selected a category or a tag
- **THEN** no request is sent, and the page indicates that a category or
  tag must be selected first

### Requirement: The report shows matching entries and a per-currency sum

Once generated, `/reports` SHALL display the matching entries (as
transaction entries — `balance_adjustment` entries are never included) via
the same cursor-based infinite scroll behavior as `/entries`, and SHALL
display the sum for each currency present among the results, from
`GET /api/entries/summary`'s response. When the generated report matches
no entries, the page SHALL show text indicating that, distinct from the
pre-generation empty state.

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
  filters, distinct from the pre-generation prompt to select a category or
  tag

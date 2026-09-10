## REMOVED Requirements

### Requirement: Category and tag are an exclusive selection

**Reason**: The category and tag filters are now independent. Both may be
set at once and are applied together as an AND filter, matching the
backend's `/api/entries` and `/api/entries/summary` behavior, which has
always accepted `category_id` and `tag_id` together.

**Migration**: None. Existing `/reports` URLs that carry only a
`category_id` or only a `tag_id` continue to behave exactly as before; the
change only permits URLs and control states that set both, or neither.

## MODIFIED Requirements

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

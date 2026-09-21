## MODIFIED Requirements

### Requirement: The report shows matching entries and a per-currency sum

Once generated, `/reports` SHALL display the matching entries (as
transaction entries — `balance_adjustment` entries are never included) via
the same cursor-based infinite scroll behavior as `/entries`, and SHALL
display, for each currency present among the results, its net sum plus its
Income and Outcome totals, from `GET /api/entries/summary`'s response.
When the generated report matches no entries, the page SHALL show text
indicating that, distinct from the pre-generation empty state.

#### Scenario: A single-currency report shows one sum

- **WHEN** a generated report's matching entries all belong to
  same-currency accounts
- **THEN** exactly one currency's net sum, Income, and Outcome are
  displayed, in that currency

#### Scenario: A multi-currency report shows a sum per currency

- **WHEN** a generated report's matching entries belong to accounts of two
  different currencies
- **THEN** a separate net sum, Income, and Outcome are displayed for each
  currency, never combined into a single figure

#### Scenario: A generated report with no matches

- **WHEN** a generated report's filters match no entries
- **THEN** the page shows text indicating no entries match the report's
  filters, distinct from the pre-generation prompt shown before "Generate
  report" has ever been activated

### Requirement: The report result table is sortable by date or amount

`/reports`' results table SHALL offer clickable Date and Amount column
headers that sort the displayed entries via `GET /api/entries`'s `sort`
and `dir` parameters, matching `/entries`' identical column-header sort
control. Activating a column header not currently sorted SHALL sort by
that column, descending; activating the currently-sorted column again
SHALL reverse its direction. The active sort column and direction SHALL
be reflected in the URL and SHALL survive a reload. Sorting SHALL take
effect immediately once a report has been generated — it SHALL NOT
require "Generate report" to be activated again, and SHALL NOT trigger
the "results are stale" hint described in "A report is only generated on
explicit action," since it reorders the same generated result set rather
than changing which entries are included or their sum.

#### Scenario: Sorting by date, descending by default

- **WHEN** an authenticated visitor generates a report with no explicit
  sort selected
- **THEN** results are ordered by `booking_timestamp`, newest first, and
  the Date column header shows the active-sort indicator

#### Scenario: Activating the Amount column sorts by amount

- **WHEN** an authenticated visitor activates the Amount column header on
  a generated report
- **THEN** the results re-order by `amount`, descending, without
  requiring "Generate report" to be activated again

#### Scenario: Activating the same column again reverses direction

- **WHEN** an authenticated visitor activates a column header that is
  already the active sort
- **THEN** the results re-order in the opposite direction

#### Scenario: Sorting does not trigger the stale-results hint

- **WHEN** an authenticated visitor activates a column header on an
  already-generated report
- **THEN** the results re-order in place and no "results are stale" hint
  is shown

#### Scenario: The sort selection survives a reload

- **WHEN** an authenticated visitor sorts a generated report by amount and
  then reloads the page
- **THEN** the same sort column and direction are shown in the URL and
  applied when a report is next generated

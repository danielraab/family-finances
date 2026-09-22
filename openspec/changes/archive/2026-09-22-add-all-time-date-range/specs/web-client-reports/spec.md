## MODIFIED Requirements

### Requirement: The report has no default date range

When `/reports` is opened with no date-range parameter (`range`, `from`, or
`to`) present in the URL, the report SHALL apply no date filter — the
date-range filter's dropdown SHALL show "All time" selected with both
bounds empty and disabled, naming the unrestricted range that is in effect.
The applied filter is unchanged from before this change; only the dropdown's
displayed selection differs, and the URL is still not modified.

#### Scenario: Opening reports with no filters applies no date restriction

- **WHEN** an authenticated visitor opens `/reports` with no `range`,
  `from`, or `to` parameter and generates a report
- **THEN** the results are not restricted by date, and the date-range
  filter shows "All time" selected with both fields empty

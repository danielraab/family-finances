## MODIFIED Requirements

### Requirement: Account and date-range filters narrow the report

`/reports` SHALL offer an account filter and a `from`/`to` date-range
filter, applied in addition to the category-or-tag selection, the same way
`/entries`'s equivalent filters narrow its list. The account filter's
choices SHALL include every non-deleted account the visitor owns or has
any permission on (view and up), since reports are read-only.

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

#### Scenario: A shared account appears in the account filter

- **WHEN** an authenticated visitor has any permission (including `view`)
  on an account they do not really own
- **THEN** that account appears among the account filter's choices, and a
  report narrowed to it includes its matching entries

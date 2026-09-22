## MODIFIED Requirements

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

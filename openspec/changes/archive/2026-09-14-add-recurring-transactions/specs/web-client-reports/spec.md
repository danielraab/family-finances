## ADDED Requirements

### Requirement: A linked entry shows the same recurring-transaction badge in report results

The `/reports` results table SHALL show the same recurring-transaction
badge `web-client-entries` defines for the ledger, on any result entry
whose `recurring_transaction_id` is set, navigating to that recurring
transaction's edit page when activated.

#### Scenario: Report result shows the badge

- **WHEN** a generated report's results include an entry with a non-null
  `recurring_transaction_id`
- **THEN** its row in the results table shows the recurring-transaction
  badge, linking to that recurring transaction's edit page

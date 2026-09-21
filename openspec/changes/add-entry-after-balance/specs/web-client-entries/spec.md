## MODIFIED Requirements

### Requirement: An entry's title opens a read-only summary modal

The read-only entry summary SHALL show the server-computed balance immediately
after the entry for every entry kind. A balance adjustment's distinct absolute
reading SHALL remain visible when present and SHALL be labelled as a balance
reading rather than as the general after-entry balance.

#### Scenario: Transaction summary shows its resulting balance

- **WHEN** a visitor opens a transaction's read-only summary
- **THEN** the summary displays the Entry's `after_balance` as “Balance after
  entry”

#### Scenario: Adjustment distinguishes reading and result

- **WHEN** a visitor opens a balance adjustment's read-only summary
- **THEN** the summary separately labels its server-computed after-entry balance
  and its recorded balance reading

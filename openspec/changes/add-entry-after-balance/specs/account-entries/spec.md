## ADDED Requirements

### Requirement: Entry responses expose the balance after the entry

Every Entry response SHALL include `after_balance`, the represented account's
running balance immediately after that entry occurrence is applied. Entries
sharing a booking timestamp SHALL be applied in their stable ledger order.

For the receiving occurrence of a self-transfer, `after_balance` SHALL describe
the receiving account identified by that occurrence's `account_id` and include
the occurrence's account-facing (negated) amount.

#### Scenario: Transaction reports its resulting balance

- **WHEN** an account has a balance of 100 before a transaction of -25
- **THEN** that Entry response has an `after_balance` of 75

#### Scenario: Filtered listing still uses the complete ledger

- **WHEN** an Entry is returned by a filtered or paginated listing
- **THEN** its `after_balance` includes every earlier account entry whether or
  not those entries matched the listing filter or page

#### Scenario: Receiving transfer occurrence uses its account perspective

- **WHEN** a self-transfer's receiving occurrence is returned
- **THEN** its `after_balance` is the receiving account balance after applying
  that occurrence's amount

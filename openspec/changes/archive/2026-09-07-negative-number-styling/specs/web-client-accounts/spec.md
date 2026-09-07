## ADDED Requirements

### Requirement: Account balances and recent-entry amounts are colored by sign

The accounts overview's per-account balance, an account's detail-page
balance, and each amount in an account's recent-entries list SHALL be
rendered in a color reflecting sign: red when negative, the default/
neutral text color when exactly zero, and green when positive. Each row of
the recent-entries list, being backed by a single entry, SHALL additionally
render its amount underlined when that entry's `kind` is
`balance_adjustment`. The accounts overview and account-detail balance
figures are computed totals, not a single entry, and are therefore never
underlined.

#### Scenario: Negative account balance is red

- **WHEN** the accounts overview or an account's detail page renders a
  negative balance
- **THEN** the balance is shown in red

#### Scenario: Zero account balance stays neutral

- **WHEN** the accounts overview or an account's detail page renders a
  balance of exactly zero
- **THEN** the balance is shown in the default text color, neither red nor
  green

#### Scenario: Positive account balance is green

- **WHEN** the accounts overview or an account's detail page renders a
  positive balance
- **THEN** the balance is shown in green

#### Scenario: A balance adjustment in recent entries is underlined

- **WHEN** an account's recent-entries list renders an entry whose `kind`
  is `balance_adjustment`
- **THEN** its amount is rendered underlined, in addition to its sign color

#### Scenario: A transaction in recent entries is not underlined

- **WHEN** an account's recent-entries list renders an entry whose `kind`
  is `transaction`
- **THEN** its amount is rendered without an underline

#### Scenario: Balance figures are never underlined

- **WHEN** the accounts overview or an account's detail page renders its
  balance figure
- **THEN** the figure is never underlined, regardless of sign

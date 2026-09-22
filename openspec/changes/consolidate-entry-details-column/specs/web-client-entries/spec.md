# Spec Delta

## ADDED Requirements

### Requirement: The entry ledger groups entry metadata in a Details column

The `/entries` ledger SHALL render a localized **Details** column immediately
after Date and before Title. It SHALL replace the separate Account, Category,
and Tags columns.

Within each row, Details SHALL render the entry's account first, its category
second when present, and its tags last when present. Account and category
SHALL retain their existing icon, shared-owner, and unavailable-entity
presentation. Tags SHALL retain their existing label, shared indicator, and
unavailable-entity presentation, and SHALL wrap within the Details cell when
necessary. An entry without a category or tags SHALL omit the corresponding
part of the Details stack.

#### Scenario: A fully attributed entry uses the Details column

- **WHEN** the ledger renders an entry with an account, a category, and tags
- **THEN** the table places Details between Date and Title and renders the
  account, category, and tag labels in that order within the Details cell

#### Scenario: An entry has no optional metadata

- **WHEN** the ledger renders an entry without a category and without tags
- **THEN** its Details cell renders only its account and does not show empty
  category or tag placeholders

#### Scenario: Existing shared and unavailable presentation is retained

- **WHEN** the ledger renders a shared or unavailable account, category, or tag
- **THEN** the corresponding item in Details keeps the same shared indicator or
  unavailable-entity fallback that the ledger displayed before the column was
  consolidated

## ADDED Requirements

### Requirement: The recurring list filters by account, category and tag

`/recurring` SHALL offer an **Account**, a **Category** and a **Tag**
control above the list, alongside the existing self-transfer checkboxes.
Each SHALL be a single-choice control whose unset option ("All accounts",
"All categories", "All tags") is its default, and each SHALL map to the
matching `account_id`, `category_id` and `tag_id` parameter on both the
list request and the summary request, so the per-currency totals below
the list always total the rows shown above it.

The account control SHALL list the accounts the visitor can see
(`GET /api/accounts`); the category control SHALL list their categories
as a flattened tree, indicating a shared category's owner the way
`/entries` does; the tag control SHALL list their tags
(`GET /api/tags`), indicating a shared tag's owner likewise.

Every control's state SHALL live in the URL's search parameters, as the
self-transfer flags already do, so a filtered list can be bookmarked,
shared and restored by the browser's back button.

#### Scenario: Filtering by account

- **WHEN** the visitor selects an account
- **THEN** the list shows only that account's recurring transactions, the
  totals below are recomputed for them, and `account_id` appears in the
  URL

#### Scenario: Filtering by category

- **WHEN** the visitor selects a category
- **THEN** the list shows only recurring transactions in that category or
  one of its descendants, and `category_id` appears in the URL

#### Scenario: Filtering by tag

- **WHEN** the visitor selects a tag
- **THEN** the list shows only recurring transactions carrying that tag,
  and `tag_id` appears in the URL

#### Scenario: Filters combine

- **WHEN** the visitor sets an account filter and a category filter
  together
- **THEN** only recurring transactions matching both are listed

#### Scenario: A filtered list survives a reload

- **WHEN** the visitor applies a category filter and reloads the page
- **THEN** the same filter is applied after the reload

#### Scenario: The totals follow the filters

- **WHEN** the visitor changes any filter
- **THEN** the summary is refetched with the same parameters as the list,
  so the per-currency totals are the totals of the listed rows

### Requirement: The recurring list's filters live in a collapsible panel with a clear-all action

`/recurring`'s filter controls SHALL be presented in the same panel
`/entries` uses: below the `sm` breakpoint the controls SHALL be
collapsed behind a toggle that reports how many filters are currently
applied and whose expanded state is conveyed to assistive technology;
from `sm` up they SHALL always be shown. While at least one filter is
applied, the panel SHALL offer a single action that clears every filter
at once.

One applied filter SHALL count once towards that number, and a control
left at its default SHALL not count.

#### Scenario: Controls are collapsed on a phone

- **WHEN** the visitor opens `/recurring` at a viewport narrower than
  `sm`
- **THEN** the filter controls are hidden behind a toggle, and activating
  it reveals them

#### Scenario: The toggle reports how many filters are applied

- **WHEN** the visitor has applied an account filter and a tag filter
- **THEN** the collapsed toggle indicates that two filters are applied

#### Scenario: Clearing every filter at once

- **WHEN** the visitor has applied one or more filters and activates the
  clear-all action
- **THEN** every filter returns to its default, the corresponding search
  parameters are removed from the URL, and the unfiltered list and totals
  are shown

#### Scenario: Nothing to clear

- **WHEN** no filter is applied
- **THEN** the clear-all action is not offered

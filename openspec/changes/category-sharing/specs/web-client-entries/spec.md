## MODIFIED Requirements

### Requirement: The entry form's category picker excludes disabled categories from new selections

The category picker on `/entries/new` and `/entries/{id}/edit` SHALL
offer the caller's own categories plus every category shared with them
at `append` tier (flat, top-level options — see `category-sharing`'s
no-cascade rule), excluding disabled categories (owned or shared) and
excluding any category shared with the caller at `view` tier only, from
the choices offered when picking a category. If the entry being edited
currently holds a category that has since been disabled, or that has
since been unshared/downgraded below `append`, that category SHALL still
render as the field's current, selected value (labeled distinctly, e.g.
as disabled) rather than disappearing — but it SHALL NOT appear as a
choosable option, so it cannot be re-selected once cleared. Leaving the
field untouched and saving the rest of the form SHALL succeed normally.

#### Scenario: Creating an entry only offers live, sufficiently-permitted categories

- **WHEN** an authenticated visitor opens the new-entry form
- **THEN** the category picker lists only non-disabled categories the
  visitor owns or holds `append` permission on

#### Scenario: An append-shared category is offered flat

- **WHEN** a category shared with the visitor at `append` tier is a child
  category in its real owner's tree
- **THEN** it appears in the category picker as a top-level option, not
  nested under any other entry

#### Scenario: A view-only shared category is not offered

- **WHEN** a category is shared with the visitor at `view` tier only
- **THEN** it does not appear among the category picker's choosable
  options

#### Scenario: A disabled current category still renders, but isn't re-selectable

- **WHEN** an authenticated visitor edits an entry whose current category
  is disabled
- **THEN** the form shows that category as the current value, distinctly
  labeled as disabled, and it does not appear among the selectable
  options

#### Scenario: Saving other changes does not require reselecting a disabled category

- **WHEN** an authenticated visitor edits an entry whose current category
  is disabled and changes only an unrelated field, without touching the
  category
- **THEN** the save succeeds and the entry keeps its (disabled) category

## ADDED Requirements

### Requirement: The entry filter's category dropdown includes categories shared with the caller

The category filter offered on `/entries` and `/reports` SHALL include
every category the caller owns plus every category shared with them at
`view` or `append` tier, flat for the shared ones — a `view`-tier share
is sufficient here, unlike the entry form's category picker, since
filtering only requires seeing by the category, not selecting it for a
new entry.

#### Scenario: A view-shared category is filterable

- **WHEN** a category is shared with the caller at `view` tier only
- **THEN** it appears among the choosable options in the entry-list and
  report category filters

### Requirement: A category shown on an entry displays a shared badge when it isn't the viewer's own

Wherever a category is rendered by name on an entry — the ledger
(`/entries`), an account's recent-entries list, and the reports results
table — the shared badge and real owner's name (per `CategoryLabel`, the
same treatment `AccountLabel` gives a shared account) SHALL be shown
whenever that category isn't the current viewer's own, i.e. its
`shared` field is `true`. A category the viewer owns SHALL render exactly
as before this capability, with no such badge.

#### Scenario: An entry's shared category shows its owner

- **WHEN** the ledger, an account's recent entries, or the reports table
  renders an entry categorized under a category shared with (not owned
  by) the current viewer
- **THEN** that row shows the shared badge and the category's real
  owner's name

#### Scenario: An owned category shows no badge

- **WHEN** any of those views renders an entry categorized under a
  category the current viewer owns
- **THEN** no shared badge is shown, unchanged from before this
  capability existed

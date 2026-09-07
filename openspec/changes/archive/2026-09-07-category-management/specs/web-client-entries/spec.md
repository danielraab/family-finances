## ADDED Requirements

### Requirement: The entry form's category picker excludes disabled categories from new selections

The category picker on `/entries/new` and `/entries/{id}/edit` SHALL
exclude disabled categories from the choices offered when picking a
category. If the entry being edited currently holds a category that has
since been disabled, that category SHALL still render as the field's
current, selected value (labeled distinctly, e.g. as disabled) rather than
disappearing — but it SHALL NOT appear as a choosable option, so it cannot
be re-selected once cleared. Leaving the field untouched and saving the
rest of the form SHALL succeed normally; the entry is not forced to change
its category just because that category was disabled.

#### Scenario: Creating an entry only offers live categories

- **WHEN** an authenticated visitor opens the new-entry form
- **THEN** the category picker lists only non-disabled categories

#### Scenario: A disabled current category still renders, but isn't re-selectable

- **WHEN** an authenticated visitor edits an entry whose current category
  is disabled
- **THEN** the form shows that category as the current value, distinctly
  labeled as disabled, and it does not appear among the selectable options

#### Scenario: Saving other changes does not require reselecting a disabled category

- **WHEN** an authenticated visitor edits an entry whose current category
  is disabled and changes only an unrelated field, without touching the
  category
- **THEN** the save succeeds and the entry keeps its (disabled) category

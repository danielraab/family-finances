## ADDED Requirements

### Requirement: The entry form's category picker excludes disabled categories from new selections

The category picker on `/entries/new` and `/entries/{id}/edit` SHALL
exclude disabled categories when choosing a category for a new entry or
changing an existing entry's category. If the entry being edited currently
holds a category that has since been disabled, that category SHALL still
be shown (labeled distinctly, e.g. as disabled) so the form does not appear
to have lost the entry's data, but it SHALL NOT be resubmittable as the
entry's category — a different, non-disabled category (or, for a balance
adjustment, no category) must be chosen before the field is valid again.
This mirrors how the account form already treats a disabled account type.

#### Scenario: Creating an entry only offers live categories

- **WHEN** an authenticated visitor opens the new-entry form
- **THEN** the category picker lists only non-disabled categories

#### Scenario: Editing an entry on a disabled category forces reselection

- **WHEN** an authenticated visitor edits an entry whose current category
  is disabled
- **THEN** the form shows that category as the current, non-selectable
  value and blocks saving until a different selection is made

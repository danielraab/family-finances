## ADDED Requirements

### Requirement: Account details page links to import

`/accounts/{id}` SHALL offer an "Import" action alongside its "New Entry"
action, shown under the same condition (`permission !== "view"`),
navigating to `/entries/import?account_id={id}` with that account preset.

#### Scenario: Navigating to import from an account's details page

- **WHEN** an authenticated visitor with `append`+ permission on an
  account opens its details page and activates "Import"
- **THEN** the client navigates to `/entries/import?account_id={id}` with
  that account preselected

#### Scenario: View-only permission does not offer import

- **WHEN** an authenticated visitor with only `view` permission on an
  account opens its details page
- **THEN** no "Import" action is shown

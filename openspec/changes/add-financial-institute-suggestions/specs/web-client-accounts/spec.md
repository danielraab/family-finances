## MODIFIED Requirements

### Requirement: Creating and editing an account

`/accounts/new` SHALL offer a form for `title`, `description`, `type_id`
(populated from `GET /api/account-types`), `currency`, `financial_institute`,
`opening_date`, and `closing_date`, submitting `POST /api/accounts` on
success and navigating to the new account's details page.
`/accounts/{id}/edit` SHALL offer the same fields pre-populated from
`GET /api/accounts/{id}`, submitting `PATCH /api/accounts/{id}` (or
equivalent update) on save. Both forms SHALL validate client-side to the
same shape the backend enforces (currency as three letters, closing date
not before opening date) and surface the backend's validation error when a
submission is rejected.

The `financial_institute` field SHALL remain free text, and SHALL offer
suggestions drawn from the distinct, non-empty `financial_institute` values
already present on the visitor's own accounts (fetched via
`GET /api/accounts`), deduplicated by exact string match and sorted
alphabetically. Suggestions SHALL be shown, as clickable chips, whenever
the field has focus — every suggestion when the field is empty, narrowed to
a case-insensitive substring match against the field's current value as the
visitor types — and hidden when the field loses focus. Activating a chip
SHALL set the field to that chip's exact value. Typing a value that matches
no suggestion SHALL remain valid and submittable, unchanged from today.

#### Scenario: Creating an account

- **WHEN** an authenticated visitor submits the create form with valid
  fields
- **THEN** `POST /api/accounts` is called and, on success, the visitor is
  taken to the new account's details page

#### Scenario: Invalid closing date is caught before submission

- **WHEN** an authenticated visitor sets a closing date earlier than the
  opening date on either form
- **THEN** the form shows a validation error and does not submit

#### Scenario: Financial institute suggestions appear on focus

- **WHEN** an authenticated visitor with at least one existing account
  carrying a `financial_institute` value focuses the financial institute
  field on the create or edit form
- **THEN** that value appears as a clickable suggestion chip, alongside
  every other distinct value already used across the visitor's own
  accounts, sorted alphabetically

#### Scenario: Typing narrows the suggestions

- **WHEN** an authenticated visitor types into the financial institute
  field while suggestions are shown
- **THEN** only suggestions containing the typed text (case-insensitive)
  remain visible

#### Scenario: Selecting a suggestion fills the field

- **WHEN** an authenticated visitor activates a financial institute
  suggestion chip
- **THEN** the field's value becomes exactly that chip's text

#### Scenario: A new institute name is still accepted

- **WHEN** an authenticated visitor types a financial institute value that
  matches none of their existing accounts' values and submits the form
- **THEN** the account is created (or updated) with that value, unchanged
  from today's free-text behavior

#### Scenario: No suggestions when the visitor has none to offer

- **WHEN** an authenticated visitor with no accounts, or none carrying a
  `financial_institute` value, focuses the field
- **THEN** no suggestion chips are shown

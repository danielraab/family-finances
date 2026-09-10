## MODIFIED Requirements

### Requirement: Creating and editing an account

`/accounts/new` SHALL offer a form for `title`, `description`, `type`,
`currency`, `financial_institute`, `opening_date`, and `closing_date`,
submitting `POST /api/accounts` on success and navigating to the new
account's details page. `/accounts/{id}/edit` SHALL offer the same fields
pre-populated from `GET /api/accounts/{id}`, submitting
`PATCH /api/accounts/{id}` (or equivalent update) on save. Both forms SHALL
validate client-side to the same shape the backend enforces (`type`
non-empty after trimming, currency as three letters, closing date not
before opening date) and surface the backend's validation error when a
submission is rejected.

The `type` field SHALL be a required free-text input, and SHALL offer
suggestions combining a fixed client-side list of default labels
(Checking, Savings, Cash, Credit Card, Loan, Investment — English only,
not translated) with the distinct in-use `type` values on the visitor's
own accounts fetched from `GET /api/account-types`, deduplicated by exact
string match. Typing a value that matches no suggestion SHALL remain valid
and submittable. The form SHALL NOT fetch or render a managed list of
account types, offer a disabled/enabled distinction, or force reselection
of a previously chosen type.

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
  fields, including a non-empty `type`
- **THEN** `POST /api/accounts` is called and, on success, the visitor is
  taken to the new account's details page

#### Scenario: The type field offers default labels and in-use values

- **WHEN** an authenticated visitor focuses the `type` field on the create
  or edit form
- **THEN** the default labels are offered as suggestions, together with any
  distinct `type` values already used on the visitor's own accounts, with
  duplicates collapsed

#### Scenario: A new type name is still accepted

- **WHEN** an authenticated visitor types a `type` value that matches none
  of the suggestions and submits the form
- **THEN** the account is created (or updated) with that value

#### Scenario: A blank type blocks submission

- **WHEN** an authenticated visitor clears the `type` field and submits
  either form
- **THEN** the form shows a validation error and does not submit

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

### Requirement: Account management affordances are gated by the visitor's permission tier

`/accounts/{id}/edit`, and the account detail page's link to it, SHALL be
offered only to a visitor with `owner`-tier permission on the account (the
real owner or a shared owner) — every other tier's detail page SHALL omit
the edit link, the disable/enable action, and the delete action entirely. A
visitor with a lower tier who navigates directly to `/accounts/{id}/edit`
SHALL be redirected to the account's detail page. Within the edit form,
every field — `type` included — SHALL be editable by any `owner`-tier
visitor, whether the real owner or a shared owner.

#### Scenario: A view or append visitor sees no edit affordance

- **WHEN** a visitor with `view`, `append`, or `entry_admin` permission
  opens an account's detail page
- **THEN** no edit link, disable/enable action, or delete action is shown

#### Scenario: Direct navigation to edit is redirected for a non-owner tier

- **WHEN** a visitor without `owner`-tier permission navigates directly to
  `/accounts/{id}/edit`
- **THEN** the client redirects them to the account's detail page

#### Scenario: A shared owner can edit every field including type

- **WHEN** a shared `owner`-tier visitor opens `/accounts/{id}/edit`
- **THEN** every field, `type` included, is editable, and saving a changed
  `type` succeeds

# web-client-accounts Specification

## Purpose

The authenticated `/accounts` pages: the sidebar link, the auth
gate, the accounts overview (balance and status per account), the
account details page (fields plus recent entries), and the
create/edit form including disable/enable and soft delete. See
`accounts` for the backend capability and `web-client-entries` for
the linked entry ledger.

## Requirements

### Requirement: Accounts link in the sidebar

The `Sidebar` navigation SHALL contain an "Accounts" item, visible to an
authenticated visitor, that navigates to `/accounts` and is shown as active
for `/accounts` and every route nested under it.

#### Scenario: Navigating to accounts from the sidebar

- **WHEN** an authenticated visitor activates "Accounts" in the sidebar
- **THEN** the client navigates to `/accounts`

#### Scenario: Nested account routes still highlight the sidebar item

- **WHEN** an authenticated visitor is on `/accounts/{id}/edit`
- **THEN** the "Accounts" sidebar item is shown as active

### Requirement: Accounts routes require authentication

`/accounts` and every route nested under it SHALL be accessible only to an
authenticated visitor. An anonymous visitor navigating to any accounts
route SHALL be redirected to `/login`.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/accounts`
- **THEN** the client redirects them to `/login`

### Requirement: Accounts overview lists every account with its balance and status

`/accounts` SHALL, on mount, fetch and display every account the visitor
owns (`GET /api/accounts`), each row showing at least its title, type,
currency, live balance (`GET /api/accounts/{id}/balance`, formatted at the
visitor's `displayed_decimal_places`), and a status indicator reflecting
whether it is disabled and/or closed. When the visitor has no accounts, the
page SHALL show explanatory empty-state text with a way to create one
instead of an empty list.

#### Scenario: Accounts and balances are listed

- **WHEN** an authenticated visitor with two accounts opens `/accounts`
- **THEN** both accounts are shown, each with its current live balance

#### Scenario: Disabled account is visually distinguished

- **WHEN** one of the visitor's accounts is disabled
- **THEN** its row indicates the disabled state

#### Scenario: Empty state

- **WHEN** an authenticated visitor with no accounts opens `/accounts`
- **THEN** the page shows text explaining there are none, with a way to
  create one

### Requirement: Account details page shows account fields and recent entries

`/accounts/{id}` SHALL fetch and display the account's full details
(`GET /api/accounts/{id}`) and its most recent entries
(`GET /api/entries` filtered to that account, sorted by booking timestamp
descending, limited to a small fixed page), each linking to that entry's
edit page. The page SHALL offer a link to `/entries?account_id={id}` for
the account's complete, filterable entry list.

#### Scenario: Recent entries link to the full filtered list

- **WHEN** an authenticated visitor opens an account's details page
- **THEN** they see its most recent entries and a link that navigates to
  `/entries?account_id={id}`

#### Scenario: Cross-owner access is not found

- **WHEN** an authenticated visitor navigates to `/accounts/{id}` for an
  account they do not own
- **THEN** the page reflects the backend's `404` (not found), not the
  account's details

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

### Requirement: Disabling, enabling, and soft-deleting an account require confirmation

The edit page SHALL offer a Disable action when the account is enabled
(`POST /api/accounts/{id}/disable`) or an Enable action when it is disabled
(`POST /api/accounts/{id}/enable`), and a (soft) delete action
(`DELETE /api/accounts/{id}`). Each SHALL require an explicit confirmation
step before the request is sent, mirroring the confirmation pattern
`/settings/users` uses for user lifecycle actions; the delete confirmation's
copy SHALL state that the action cannot be undone.

#### Scenario: Disabling requires confirmation

- **WHEN** an authenticated visitor activates "Disable" on an enabled
  account's edit page
- **THEN** a confirmation step appears and
  `POST /api/accounts/{id}/disable` is not called until it is confirmed

#### Scenario: Soft delete confirmation warns it is permanent

- **WHEN** an authenticated visitor activates "Delete" on an account's edit
  page
- **THEN** the confirmation step's copy states the action cannot be undone,
  and `DELETE /api/accounts/{id}` is not called until confirmed

#### Scenario: Deleted account disappears from the overview

- **WHEN** an authenticated visitor confirms deleting an account
- **THEN** it no longer appears in `/accounts`

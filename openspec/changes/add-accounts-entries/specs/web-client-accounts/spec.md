## ADDED Requirements

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

#### Scenario: Creating an account

- **WHEN** an authenticated visitor submits the create form with valid
  fields
- **THEN** `POST /api/accounts` is called and, on success, the visitor is
  taken to the new account's details page

#### Scenario: Invalid closing date is caught before submission

- **WHEN** an authenticated visitor sets a closing date earlier than the
  opening date on either form
- **THEN** the form shows a validation error and does not submit

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

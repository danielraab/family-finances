# web-client-recurring-transactions Specification

## Purpose

The `/recurring` list, create, and edit pages for recurring transaction
templates, and the manual "create a transaction from this template" flow
into the existing entry-create form.

## Requirements

### Requirement: Recurring link in the sidebar

The `Sidebar` navigation SHALL contain a "Recurring" item, visible to an
authenticated visitor, at the same level as (not nested under) the
"Entries" item. It SHALL navigate to `/recurring` and SHALL be shown as
active for `/recurring` and every route nested under it.

#### Scenario: Navigating to Recurring

- **WHEN** the visitor activates the "Recurring" navigation item
- **THEN** the browser navigates to `/recurring` and the "Recurring" item is
  shown as active

#### Scenario: Recurring and Entries are peers

- **WHEN** the sidebar renders
- **THEN** "Recurring" appears as its own top-level item alongside
  "Entries", not as a child/nested item under it

### Requirement: /recurring requires authentication

`/recurring` and its `/recurring/new` and `/recurring/{id}/edit` routes
SHALL be accessible only to an authenticated visitor. An anonymous visitor
navigating to any of them SHALL be redirected to `/login`.

#### Scenario: Anonymous visitor redirected

- **WHEN** an anonymous visitor navigates to `/recurring`
- **THEN** they are redirected to `/login`

### Requirement: The recurring transaction list shows entered and per-year amounts, and a per-currency total

`/recurring` SHALL list the caller's visible recurring transactions
(`GET /api/recurring-transactions`), each row showing at least its title,
category, entered `amount`, and computed `per_year_amount`. A recurring
transaction whose `ended` is `true` SHALL be visually distinguished (e.g.
muted styling or a badge) from an active one. Below the list, the page
SHALL show the total per-year amount per currency
(`GET /api/recurring-transactions/summary`), matching the same per-currency
grouping the API returns — never a single combined figure across
currencies.

#### Scenario: Row shows both amounts

- **WHEN** a recurring transaction with `amount: -80000` and
  `per_year_amount: -960000` is listed
- **THEN** its row shows both the entered amount and the per-year amount

#### Scenario: Ended template is visually distinguished

- **WHEN** a listed recurring transaction's `ended` is `true`
- **THEN** its row is rendered with a distinct, muted treatment from active
  rows

#### Scenario: Total is per currency

- **WHEN** the caller's recurring transactions span two account
  currencies
- **THEN** the page shows two separate per-year totals, one per currency

### Requirement: Creating and editing a recurring transaction

`/recurring/new` and `/recurring/{id}/edit` SHALL present the same content
fields as the entry create/edit form (account, title, description,
category, counterparty, location, tags, signed amount) with no `kind`
selector, plus the recurrence rule: a preset picker (at least Weekly, Every
2 weeks, Monthly, Every 2 months, Quarterly, Every 6 months, Yearly, and a
Custom option) that maps to the underlying `interval_unit`/`interval_count`
pair, a `starts_on` date, and an optional `ends_on` date. Selecting
"Custom" SHALL reveal direct `interval_unit`/`interval_count` inputs for a
combination not covered by a preset (e.g. every 10 days).

#### Scenario: Preset maps to the underlying fields

- **WHEN** the visitor picks "Quarterly" on the create form
- **THEN** the submitted request has `interval_unit: month`,
  `interval_count: 3`

#### Scenario: Custom reveals raw inputs

- **WHEN** the visitor picks "Custom"
- **THEN** direct `interval_unit` and `interval_count` inputs are shown,
  editable to any valid combination

### Requirement: Deleting a recurring transaction with linked entries is blocked in the UI

The edit page's delete action SHALL be disabled (with an explanatory hint)
whenever the recurring transaction has one or more linked entries, mirroring
how the categories page disables delete for an in-use category. A `409`
response from `DELETE /api/recurring-transactions/{id}` (not knowable
client-side in every case, e.g. a race with another visitor) SHALL surface
as an inline error rather than removing the row.

#### Scenario: Delete disabled with linked entries

- **WHEN** the visitor opens the edit page for a recurring transaction that
  has at least one linked entry
- **THEN** the delete action is disabled and shows why

### Requirement: Creating a transaction from a recurring transaction is always a manual action

Each recurring transaction (on the list and on its edit page) SHALL offer a
"Create transaction" action. Activating it SHALL navigate to
`/entries/new` prefilled from the template — account, title, description,
category, counterparty, location, tags, and signed amount — with
`booking_timestamp` prefilled to the recurring transaction's
`next_suggested_date`, and with `recurring_transaction_id` carried through
so the entry is linked once submitted. Every prefilled field SHALL remain
editable before submission, and no entry SHALL be created without the
visitor explicitly submitting that form — there is no automatic or
scheduled creation.

#### Scenario: Create transaction prefills and links

- **WHEN** the visitor activates "Create transaction" on a recurring
  transaction and submits the prefilled form unchanged
- **THEN** a new entry is created matching the template's fields, booked on
  the template's `next_suggested_date`, with `recurring_transaction_id` set
  to that recurring transaction

#### Scenario: Prefilled fields remain editable

- **WHEN** the visitor activates "Create transaction" and changes the
  amount or booking date before submitting
- **THEN** the created entry reflects the edited values, still linked to
  the recurring transaction

#### Scenario: No background or scheduled creation

- **WHEN** a recurring transaction's `next_suggested_date` has passed with
  no visitor action taken
- **THEN** no entry is created automatically — the list still shows the
  template with its (past) `next_suggested_date`, unchanged until the
  visitor manually acts

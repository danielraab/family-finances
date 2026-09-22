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

### Requirement: The recurring transaction list shows entered, per-year and per-month amounts, and a per-currency total

`/recurring` SHALL list the caller's visible recurring transactions
(`GET /api/recurring-transactions`), each row showing at least its title,
category, entered `amount`, computed `per_year_amount`, and a per-month
amount derived from `per_year_amount` by dividing it by twelve. A
recurring transaction whose `ended` is `true` SHALL be visually
distinguished (e.g. muted styling or a badge) from an active one. Below
the list, the page SHALL show both the total per-year amount per currency
(`GET /api/recurring-transactions/summary`) and a per-month total derived
from each of those per-currency totals by dividing it by twelve, matching
the same per-currency grouping the API returns — never a single combined
figure across currencies.

Every per-month figure SHALL be rendered in the same currency, at the
same display precision, and with the same sign colouring as the per-year
figure it is derived from.

Every amount the list renders — entered, per year, per month, and each
per-currency total — SHALL be rendered on a single line at every viewport
width, with its sign attached to its digits.

#### Scenario: Row shows entered, per-year and per-month amounts

- **WHEN** a recurring transaction with `amount: -80000` and
  `per_year_amount: -960000` is listed
- **THEN** its row shows the entered amount, the per-year amount, and a
  per-month amount of `-80000`

#### Scenario: A non-monthly row's per-month amount is the annual figure divided by twelve

- **WHEN** a yearly recurring transaction with `per_year_amount: -6000000`
  is listed
- **THEN** its row shows a per-month amount of `-500000`

#### Scenario: A negative amount keeps its sign on one line

- **WHEN** a recurring transaction with a negative amount is listed
- **THEN** each of its amount cells renders the sign and the digits on
  the same line, at every viewport width

#### Scenario: Ended template is visually distinguished

- **WHEN** a listed recurring transaction's `ended` is `true`
- **THEN** its row is rendered with a distinct, muted treatment from active
  rows

#### Scenario: Total is per currency

- **WHEN** the caller's recurring transactions span two account
  currencies
- **THEN** the page shows two separate per-year totals and two separate
  per-month totals, one of each per currency

#### Scenario: The per-month total is the per-year total divided by twelve

- **WHEN** the summary endpoint returns a per-year total of `-15000000`
  for a currency
- **THEN** the page shows a per-month total of `-1250000` for that
  currency

### Requirement: The recurring list has a self-transfer filter with a revealed second flag

`/recurring` SHALL offer a filter control above the list carrying two
checkboxes. The first, "Show self-transfers", SHALL be unchecked by
default and SHALL map to `include_self_transfer` on both the list and the
summary request. The second, "Show both sides (income and outcome)", SHALL
be rendered only while the first is checked, SHALL itself be unchecked by
default, and SHALL map to `self_transfer_both_legs` on the same two
requests. Unchecking the first SHALL hide the second and clear it, so the
two can never be left in the meaningless "both legs, transfers hidden"
combination. Both flags SHALL live in the URL's search parameters, the way
`/entries` keeps its own filter state, so a filtered list can be
bookmarked, shared, and restored by the browser's back button. The
per-currency total below the list SHALL always reflect the same two flags
as the list above it.

#### Scenario: Transfers hidden by default

- **WHEN** the visitor opens `/recurring` with a `self_transfer` recurring
  transaction among their templates
- **THEN** it is not listed, and the second checkbox is not rendered

#### Scenario: Checking the first flag reveals the second

- **WHEN** the visitor checks "Show self-transfers"
- **THEN** self-transfer templates appear in the list, each once, and the
  "Show both sides" checkbox becomes visible, unchecked

#### Scenario: Both sides shown

- **WHEN** the visitor additionally checks "Show both sides (income and
  outcome)" with both of a transfer's accounts visible to them
- **THEN** that template is listed twice, once as an outgoing row on the
  sending account and once as an incoming row on the receiving account

#### Scenario: Unchecking the first flag clears the second

- **WHEN** the visitor has both checkboxes checked and unchecks "Show
  self-transfers"
- **THEN** the second checkbox is hidden, its value is cleared from the
  URL, and no self-transfer rows are listed

#### Scenario: The total follows the flags

- **WHEN** the visitor toggles either flag
- **THEN** the per-currency per-year total below the list is refetched with
  the same flags, so it always totals the rows shown

### Requirement: A self-transfer row identifies the other account and its direction

A listed `self_transfer` recurring transaction SHALL show the other
account it moves money to or from, visually distinguished from a plain
transaction row the way a self-transfer entry already is in the ledger, and
SHALL make clear which side the row represents when both sides are shown.
Its category cell SHALL render as empty rather than broken when the
template has no category, and its counterparty and location cells SHALL be
empty, since a self-transfer carries neither.

#### Scenario: The other account is named on the row

- **WHEN** a `self_transfer` recurring transaction is listed
- **THEN** its row names the account on the other side of the transfer

#### Scenario: The two sides are distinguishable

- **WHEN** both sides of one transfer are listed together
- **THEN** each row shows its own account and an opposite-signed amount,
  and the two are not mistakable for two separate templates

#### Scenario: A category-less transfer renders cleanly

- **WHEN** a `self_transfer` recurring transaction with no category is
  listed
- **THEN** its category cell is empty and the row renders normally

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

### Requirement: Returning to the recurring list restores the last-applied filters

`/recurring` SHALL write its current filter state (`account_id`,
`category_id`, `tag_id`, `include_self_transfer`,
`self_transfer_both_legs`) to browser-local storage, keyed per visitor's
browser (not synced to the account or the backend), whenever it changes.
Arriving at `/recurring` with a completely bare URL — none of those
parameters present at all — SHALL restore the visitor's most recently
persisted filter state from that storage, replacing the URL rather than
leaving the bare arrival in browser history, if any state has been
persisted. This applies to any ordinary navigation to bare `/recurring` (a
sidebar link, a link from elsewhere in the app). Arriving with any explicit
parameter already present SHALL NOT be overridden by persisted state — the
explicit parameters apply exactly as they would with nothing persisted.
When nothing has been persisted yet, a bare arrival SHALL show the
unfiltered list, as today.

#### Scenario: A sidebar click restores the last-applied filters

- **WHEN** an authenticated visitor applies an account filter and enables
  "include self-transfers" on `/recurring`, navigates elsewhere in the
  app, then clicks "Recurring" in the sidebar
- **THEN** they land on `/recurring` with the same account filter and
  self-transfer toggle applied, not the unfiltered list

#### Scenario: No persisted state shows the unfiltered list

- **WHEN** an authenticated visitor with nothing yet persisted opens a bare
  `/recurring`
- **THEN** the unfiltered list is shown, as today

#### Scenario: An explicit link is not overridden by persisted state

- **WHEN** an authenticated visitor with a persisted category filter
  follows a link to `/recurring?account_id={id}`
- **THEN** the recurring list is filtered only by that account, not also by
  the previously persisted category

#### Scenario: Changing a filter persists it

- **WHEN** an authenticated visitor changes a filter control on
  `/recurring`
- **THEN** the corresponding URL search parameter changes to match, and the
  same state is written to browser-local storage

### Requirement: Creating and editing a recurring transaction

`/recurring/new` and `/recurring/{id}/edit` SHALL present a kind selector
offering "Transaction" and "Self-transfer", plus the content fields the
entry create/edit form presents for that kind (account, title,
description, category, counterparty, location, tags, signed amount), plus
the recurrence rule: a preset picker (at least Weekly, Every 2 weeks,
Monthly, Every 2 months, Quarterly, Every 6 months, Yearly, and a Custom
option) that maps to the underlying `interval_unit`/`interval_count` pair,
a `starts_on` date, and an optional `ends_on` date. Selecting "Custom"
SHALL reveal direct `interval_unit`/`interval_count` inputs for a
combination not covered by a preset (e.g. every 10 days).

Selecting "Self-transfer" SHALL reveal a "To account" picker and hide
counterparty and location, and SHALL stop requiring a category — mirroring
what `/entries/new` already does for the same kind. The "To account"
picker SHALL offer only accounts the visitor holds at least `append`
permission on that are not disabled, share the selected source account's
currency, and are not the source account itself. On `/recurring/{id}/edit`
the kind selector and the "To account" picker SHALL be shown as
immutable — the backend rejects changing either — and the whole form SHALL
be read-only when the visitor no longer holds `append`+ on both accounts,
with an explanation rather than a save that fails.

#### Scenario: Preset maps to the underlying fields

- **WHEN** the visitor picks "Quarterly" on the create form
- **THEN** the submitted request has `interval_unit: month`,
  `interval_count: 3`

#### Scenario: Custom reveals raw inputs

- **WHEN** the visitor picks "Custom"
- **THEN** direct `interval_unit` and `interval_count` inputs are shown,
  editable to any valid combination

#### Scenario: Self-transfer reveals the to-account picker

- **WHEN** the visitor selects "Self-transfer" on `/recurring/new`
- **THEN** a "To account" picker appears, counterparty and location are
  hidden, and the form submits without a category

#### Scenario: The to-account picker excludes unusable accounts

- **WHEN** the visitor has selected a source account in EUR and opens the
  "To account" picker
- **THEN** the source account itself, any disabled account, any account
  they hold only `view` on, and any account in another currency are not
  offered

#### Scenario: Kind is not editable

- **WHEN** the visitor opens `/recurring/{id}/edit` for a `self_transfer`
  recurring transaction
- **THEN** the kind and "To account" are shown but cannot be changed

### Requirement: A recurring transaction's edit page lists its linked entries, cursor-paginated

`/recurring/{id}/edit` SHALL show a "Linked transactions" section listing
the entries whose `recurring_transaction_id` is that template's id
(`GET /api/entries?recurring_transaction_id={id}`), newest booking
timestamp first, each row showing at least the entry's booking date,
title, category, and signed amount, and linking to that entry's own
`/entries/{entryId}/edit` page. When the template's `linked_entry_count`
is `0`, the section SHALL show an empty state without issuing the list
request. When more results remain (`next_cursor` non-null), the section
SHALL offer a "Load more" action that appends the next page; it SHALL NOT
fetch further pages automatically on scroll.

#### Scenario: No linked entries shows an empty state, no request

- **WHEN** the visitor opens the edit page for a recurring transaction
  whose `linked_entry_count` is `0`
- **THEN** an empty state is shown and `GET /api/entries` is not called
  with `recurring_transaction_id` for it

#### Scenario: Linked entries list, newest first

- **WHEN** the visitor opens the edit page for a recurring transaction
  with linked entries
- **THEN** the section lists them ordered by booking timestamp, newest
  first

#### Scenario: A row links to its entry's edit page

- **WHEN** the visitor activates a row in the linked-transactions list
- **THEN** the browser navigates to that entry's `/entries/{entryId}/edit`
  page

#### Scenario: Load more appends the next page

- **WHEN** the visitor activates "Load more" and a `next_cursor` was
  present
- **THEN** the next page's entries are appended to the visible list
  without replacing it, using that cursor as `after`

#### Scenario: No load-more action on the last page

- **WHEN** the loaded page's `next_cursor` is `null`
- **THEN** no "Load more" action is shown

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

### Requirement: Creating a recurring transaction from an existing entry prefills and auto-links it

`/recurring/new` SHALL accept an optional `from_entry_id` search
parameter. When present, it SHALL fetch that entry and prefill the create
form's account (locked, not changeable in this flow — mirroring
`/entries/new`'s `?account_id=` lock), title, description, category,
counterparty, location, tags, and signed amount from it, and SHALL default
`starts_on` to the entry's booking date (date portion only, no
time-of-day). Every prefilled field SHALL remain editable before
submission except the locked account.

On successful creation from this flow, the new recurring transaction SHALL
always be linked back to the originating entry — `PATCH
/api/entries/{id}` setting `recurring_transaction_id` to the newly created
template's id — with no visitor-facing opt-out. The visitor SHALL land on
`/recurring` after creation, the same destination `/recurring/new` already
navigates to on a normal (non-`from_entry_id`) successful creation.

#### Scenario: Prefilled from the entry

- **WHEN** the visitor arrives at `/recurring/new?from_entry_id=<id>`
- **THEN** the form is populated with that entry's title, description,
  category, counterparty, location, tags, and signed amount, the account
  field is locked to the entry's account, and `starts_on` defaults to the
  entry's booking date

#### Scenario: Prefilled fields remain editable

- **WHEN** the visitor arrives via `from_entry_id` and changes the title or
  amount before submitting
- **THEN** the created recurring transaction reflects the edited values

#### Scenario: Successful creation links back to the originating entry

- **WHEN** the visitor submits the form reached via
  `?from_entry_id=<id>` and creation succeeds
- **THEN** `PATCH /api/entries/{id}` is called with `recurring_transaction_id`
  set to the newly created recurring transaction's id, and the visitor
  lands on `/recurring`

#### Scenario: No prefill without from_entry_id

- **WHEN** the visitor opens `/recurring/new` with no `from_entry_id`
  parameter
- **THEN** the form starts empty as it does today, and no entry is
  fetched or linked

### Requirement: Creating a transaction from a recurring transaction is always a manual action

Each recurring transaction (on the list and on its edit page) SHALL offer a
"Create transaction" action. Activating it SHALL navigate to
`/entries/new` prefilled from the template — account, kind, title,
description, category, counterparty, location, tags, signed amount, and,
for a `self_transfer`, the receiving account — with `booking_timestamp`
prefilled to the recurring transaction's `next_suggested_date`, and with
`recurring_transaction_id` carried through so the entry is linked once
submitted. The prefill SHALL always resolve the template in its stored
orientation, so activating the action on an incoming (receiving-side) row
still creates an entry whose `account_id` is the template's sending
account — the orientation the link is validated against. Every prefilled
field SHALL remain editable before submission, and no entry SHALL be
created without the visitor explicitly submitting that form — there is no
automatic or scheduled creation.

On the list, this action SHALL render as a single unbroken control whose
label never wraps across lines at any viewport width. Below the `sm`
breakpoint it SHALL render as a plus glyph with no visible label; at `sm`
and above it SHALL render its translated label. At every width its
accessible name SHALL be that same translated label and its destination
SHALL be unchanged.

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

#### Scenario: Creating from a self-transfer template

- **WHEN** the visitor activates "Create transaction" on a `self_transfer`
  recurring transaction
- **THEN** the entry form opens with kind "Self-transfer", both accounts
  prefilled, and submitting it creates a linked `self_transfer` entry

#### Scenario: Creating from the incoming side of a transfer

- **WHEN** the visitor activates "Create transaction" on the receiving-side
  row of a `self_transfer` recurring transaction shown with both sides
- **THEN** the created entry's `account_id` is the template's sending
  account, its `to_account_id` the receiving one, and the link is accepted

#### Scenario: No background or scheduled creation

- **WHEN** a recurring transaction's `next_suggested_date` has passed with
  no visitor action taken
- **THEN** no entry is created automatically — the list still shows the
  template with its (past) `next_suggested_date`, unchanged until the
  visitor manually acts

#### Scenario: The row action is a plus glyph on a phone

- **WHEN** an authenticated visitor opens `/recurring` on a viewport
  narrower than `sm`
- **THEN** each row's create-transaction action shows a plus glyph and no
  visible label, occupies a single unbroken box, and still exposes its
  translated label as its accessible name

#### Scenario: The row action keeps its label on a wide viewport

- **WHEN** an authenticated visitor opens `/recurring` on a viewport at or
  above `sm`
- **THEN** each row's create-transaction action shows its translated
  label on one line

### Requirement: The recurring list's create action is icon-only on a narrow viewport

`/recurring`'s "New recurring transaction" action SHALL render as a plus
glyph with no visible label on a viewport narrower than the `sm`
breakpoint. At `sm` and above it SHALL render its translated label as
before. At every width its accessible name SHALL be that same translated
label, and its destination SHALL be unchanged — `/recurring/new`.

#### Scenario: A phone-width visitor sees a plus button

- **WHEN** an authenticated visitor opens `/recurring` on a viewport
  narrower than `sm`
- **THEN** the create action shows a plus glyph and no visible label,
  while still exposing its translated label as its accessible name

#### Scenario: A wide viewport keeps the label

- **WHEN** an authenticated visitor opens `/recurring` on a viewport at or
  above `sm`
- **THEN** the create action shows its translated label

### Requirement: A recurring transaction has a read-only summary modal

A recurring transaction SHALL have a read-only summary modal, opened from
its badge on a linked entry in the ledger and the `/reports` results
table, from a `/recurring` list row, and from an Upcoming block row's
title. The summary SHALL show the template's title, account, signed
amount, interval (count and unit), start date, end date when set, whether
it has ended, next suggested date, per-year amount, category, tags,
counterparty, and linked entry count, each omitted when not carried.
Nothing in the summary SHALL be editable.

The summary SHALL offer an Edit action navigating to
`/recurring/{id}/edit`.

The summary SHALL also offer a "Create transaction" action navigating to
`/entries/new?recurring_transaction_id={id}`, the same action
`/recurring`'s own rows offer. When the summary was opened for a specific
projected occurrence, that action SHALL additionally carry that
occurrence's `booking_timestamp`; when it was not, it SHALL send no
`booking_timestamp` and the entry form's own default date applies.

#### Scenario: The badge opens the summary

- **WHEN** a visitor activates the recurring badge on a linked entry
- **THEN** that recurring transaction's read-only summary modal opens

#### Scenario: The summary offers Edit

- **WHEN** the recurring summary modal is open
- **THEN** it offers an Edit action navigating to `/recurring/{id}/edit`

#### Scenario: The summary offers Create transaction

- **WHEN** the recurring summary modal is open
- **THEN** it offers a Create transaction action navigating to
  `/entries/new` with that recurring transaction's id

#### Scenario: A summary opened from an occurrence books that occurrence's date

- **WHEN** a visitor opens the summary from an Upcoming block row
  projected for 2026-12-01 and activates Create transaction
- **THEN** the client navigates to `/entries/new` with that recurring
  transaction's id and `booking_timestamp=2026-12-01`

#### Scenario: A summary opened from a badge sends no date

- **WHEN** a visitor opens the summary from a linked entry's badge or a
  `/recurring` row and activates Create transaction
- **THEN** the client navigates to `/entries/new` with the recurring
  transaction's id and no `booking_timestamp`, and the form prefills its
  own default date

#### Scenario: An ended template is shown as ended

- **WHEN** the summary opens for a recurring transaction whose `ended` is
  true
- **THEN** the summary indicates that it has ended

### Requirement: The entry and recurring summaries cross-link in place

An entry summary for an entry with a `recurring_transaction_id` SHALL
offer a way to open that recurring transaction's summary, and the
recurring summary SHALL offer a way back to the entry summary it was
opened from. Following either SHALL replace the open modal's content
rather than opening a second modal on top of it.

#### Scenario: Opening the recurring summary from an entry summary

- **WHEN** the visitor follows the recurring-transaction link inside an
  entry's summary
- **THEN** the same modal's content is replaced by the recurring
  transaction's summary, with no second modal stacked above it

#### Scenario: Returning to the entry summary

- **WHEN** the visitor activates the back affordance on a recurring
  summary reached from an entry summary
- **THEN** the modal's content returns to that entry's summary

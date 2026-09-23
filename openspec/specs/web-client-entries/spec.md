# web-client-entries Specification

## Purpose

The authenticated `/entries` pages: the sidebar link, the auth
gate, the filterable/searchable/sortable, infinitely-scrolling entry
ledger with its filter/search/sort state kept in the URL, and the
create/edit form including inline tag creation. See
`account-entries` for the backend capability.

## Requirements

### Requirement: Entries link in the sidebar

The `Sidebar` navigation SHALL contain an "Entries" item, visible to an
authenticated visitor, that navigates to `/entries` and is shown as active
for `/entries` and every route nested under it.

#### Scenario: Navigating to entries from the sidebar

- **WHEN** an authenticated visitor activates "Entries" in the sidebar
- **THEN** the client navigates to `/entries`

### Requirement: Entries routes require authentication

`/entries` and every route nested under it SHALL be accessible only to an
authenticated visitor. An anonymous visitor navigating to any entries route
SHALL be redirected to `/login`.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/entries`
- **THEN** the client redirects them to `/login`

### Requirement: The entry ledger's filter, search, and sort state lives in the URL

`/entries` SHALL represent its current account, category, tag, kind, date range, signed amount range, and free-text search filters, and its sort field and direction, as typed URL search parameters, readable and writable through TanStack Router's search-param APIs. The amount range SHALL use optional `amount_from` and `amount_to` stored-scale integer values. Reloading a URL with search parameters SHALL reproduce the same filtered/sorted view. Arriving at `/entries` with an `account_id` parameter already set (for example, via the link from an account's details page) SHALL apply that filter immediately on load. Every change to this state SHALL also be written to browser-local storage, keyed per visitor's browser (not synced to the account or the backend) — see "Returning to the entry ledger restores the last-applied filters" for when that stored state is read back.

#### Scenario: A filtered view survives a reload

- **WHEN** an authenticated visitor applies a category filter and a sort order, then reloads the page
- **THEN** the same category filter and sort order are applied after the reload

#### Scenario: An amount range survives a reload

- **WHEN** an authenticated visitor applies an amount start and/or end bound, then reloads the page
- **THEN** the same amount bounds are present in the URL and applied to the ledger

#### Scenario: Arriving with a preset account filter

- **WHEN** an authenticated visitor follows a link to `/entries?account_id={id}`
- **THEN** the entry list is immediately filtered to that account

#### Scenario: Changing a filter updates the URL

- **WHEN** an authenticated visitor changes the search text or a filter control
- **THEN** the corresponding URL search parameter changes to match, and the same state is written to browser-local storage

### Requirement: The entry ledger loads via cursor-based infinite scroll

`/entries` SHALL fetch its first page from `GET /api/entries` using the
current filter/search/sort parameters, and SHALL fetch subsequent pages
using the previous response's `next_cursor` as new entries are needed (for
example, on reaching the bottom of the currently loaded list), appending
them to what is already shown. Changing any filter, search, or sort
parameter SHALL discard the currently loaded pages and start again from the
first page with the new parameters. A page fetch in progress SHALL be
indicated to the visitor, and reaching a `next_cursor` of `null` SHALL stop
further fetching.

#### Scenario: Scrolling loads more entries

- **WHEN** an authenticated visitor scrolls to the bottom of the currently
  loaded entries and more match the current filters
- **THEN** the next page is fetched using the previous page's cursor and
  appended to the list

#### Scenario: Changing a filter resets the loaded list

- **WHEN** an authenticated visitor has scrolled through several pages and
  then changes a filter
- **THEN** the previously loaded entries are discarded and the list
  restarts from the first page under the new filter

#### Scenario: Reaching the end stops fetching

- **WHEN** a fetched page's `next_cursor` is `null`
- **THEN** no further page fetch is triggered by additional scrolling

### Requirement: Entry list filters, search, and sort controls

`/entries` SHALL offer controls for every backend filter (`account_id`, `category_id`, `tag_id`, `kind`, a date range via the shared date-range filter — see `web-client-date-range-filter`, and optional signed amount start and end bounds), a free-text search input, and a way to sort by booking timestamp or amount in either direction. The amount controls SHALL accept full stored precision in major units and convert to/from the API's fixed stored scale; they SHALL permit either bound independently. When no entries match the current filters, the page SHALL show text distinguishing "no entries match these filters" from "no entries exist yet."

Those controls SHALL be laid out as a responsive grid that fills the available width — one column below the `sm` breakpoint, two from `sm`, and three from `lg` — with each control sized to its cell rather than to its content, and the free-text search input spanning the grid's full width.

#### Scenario: Combining filters narrows the results

- **WHEN** an authenticated visitor sets an account filter, a category filter, and a date range together
- **THEN** only entries matching all three narrow the results

#### Scenario: An open amount bound narrows the results

- **WHEN** an authenticated visitor sets only an amount end bound
- **THEN** the ledger requests and shows only entries at or below that signed amount

#### Scenario: No matches under the current filters

- **WHEN** the current filters match no entries but entries exist on the account
- **THEN** the page shows text indicating no entries match the current filters, not that none exist at all

#### Scenario: Controls stack in a single column on a phone

- **WHEN** an authenticated visitor views the ledger's filter controls on a viewport narrower than `sm`
- **THEN** each control occupies its own full-width row

### Requirement: The ledger's create action is icon-only on a narrow viewport

`/entries`' "New entry" action SHALL render as a plus glyph with no
visible label on a viewport narrower than the `sm` breakpoint. At `sm` and
above it SHALL render its translated label as before. At every width its
accessible name SHALL be that same translated label, and its destination
SHALL be unchanged — `/entries/new`, carrying the current `account_id`
filter when one is applied.

#### Scenario: A phone-width visitor sees a plus button

- **WHEN** an authenticated visitor opens `/entries` on a viewport
  narrower than `sm`
- **THEN** the create action shows a plus glyph and no visible label,
  while still exposing its translated label as its accessible name

#### Scenario: A wide viewport keeps the label

- **WHEN** an authenticated visitor opens `/entries` on a viewport at or
  above `sm`
- **THEN** the create action shows its translated label

#### Scenario: The icon-only action carries the account filter

- **WHEN** a visitor on a viewport narrower than `sm` is viewing
  `/entries?account_id={id}` and activates the create action
- **THEN** the client navigates to `/entries/new?account_id={id}`

### Requirement: The ledger's search input can be cleared in one action

The ledger's free-text search input SHALL offer a clear action inside the
field whenever the field holds text, and SHALL NOT render it when the
field is empty. Activating it SHALL empty the field, remove `q` from the
URL without waiting for the input's debounce interval, and leave keyboard
focus on the search input. The clear action SHALL be the only clear
affordance the field presents, including in browsers that render a native
search-cancel control.

#### Scenario: Clearing a search empties the field and the URL

- **WHEN** an authenticated visitor with search text applied activates the
  search field's clear action
- **THEN** the field is empty, `q` is absent from the URL, and the
  unfiltered-by-text results are shown

#### Scenario: An empty search field offers no clear action

- **WHEN** the ledger's search field holds no text
- **THEN** no clear action is rendered inside it

#### Scenario: Clearing does not get undone by a pending debounce

- **WHEN** a visitor types search text and activates the clear action
  before the input's debounce interval elapses
- **THEN** `q` stays absent from the URL and the typed text is not
  re-applied

### Requirement: The ledger offers a clear-all-filters action

`/entries` SHALL offer a single action clearing every filter at once, rendered only while at least one filter is active. Activating it SHALL remove the account, category, tag, kind, date-range, amount-range, free-text search, and upcoming-recurring parameters from the URL in one navigation, and SHALL leave the sort field and direction untouched. With no date-range parameter present the ledger's own default range applies again, per "The entry ledger defaults to the last 2 weeks".

A filter counts as active when its parameter is present in the URL; the date range counts as one active filter when any of `range`, `from`, or `to` is present, the amount range counts as one active filter when either `amount_from` or `amount_to` is present, and the implicitly applied default range does not count.

#### Scenario: Clearing every filter at once

- **WHEN** an authenticated visitor with an account filter, a tag filter, an amount bound, and search text applied activates the clear-all-filters action
- **THEN** all four parameters are removed from the URL and the ledger shows the default unfiltered view

#### Scenario: Clearing filters preserves the sort

- **WHEN** a visitor sorted by amount ascending activates the clear-all action
- **THEN** the results remain sorted by amount ascending

#### Scenario: No action is offered when nothing is filtered

- **WHEN** a visitor opens `/entries` with no filter parameters in the URL
- **THEN** no clear-all-filters action is rendered

### Requirement: The ledger's filter controls collapse on a narrow viewport

The ledger's filter controls SHALL be presented in a panel with a header.
On a viewport narrower than the `sm` breakpoint the panel's controls
SHALL be hidden behind a toggle in that header, closed on load; from `sm`
up the controls SHALL always be shown and no toggle SHALL be rendered.
While at least one filter is active the header SHALL show the number of
active filters, counted as defined above, so a closed panel never hides
that the ledger is filtered.

#### Scenario: Filters start collapsed on a phone

- **WHEN** an authenticated visitor opens `/entries` on a viewport
  narrower than `sm`
- **THEN** the filter controls are not shown, and a toggle for them is

#### Scenario: Expanding the panel reveals every control

- **WHEN** that visitor activates the toggle
- **THEN** the account, category, tag, kind, date-range,
  upcoming-recurring, and search controls are all shown

#### Scenario: A collapsed panel still reports active filters

- **WHEN** a visitor on a viewport narrower than `sm` follows a link to
  `/entries?account_id={id}`
- **THEN** the filter panel is closed and its header shows that one
  filter is active

#### Scenario: A wide viewport shows the controls unconditionally

- **WHEN** an authenticated visitor opens `/entries` on a viewport at or
  above `sm`
- **THEN** the filter controls are shown and no toggle is rendered

### Requirement: The entry ledger defaults to the last 2 weeks

The entry ledger SHALL apply the "Last 2 weeks" preset as its effective
date filter when `/entries` is opened with no date-range parameter
(`range`, `from`, or `to`) present in the URL and no filter state is
restored from browser-local storage (see "Returning to the entry ledger
restores the last-applied filters"), without writing that preset into the
URL. The date-range filter's dropdown SHALL show "Last 2 weeks" selected in
this state.

#### Scenario: Opening the ledger with no filters and nothing persisted applies a two-week default

- **WHEN** an authenticated visitor with no previously persisted filter
  state opens `/entries` with no `range`, `from`, or `to` parameter
- **THEN** the list shows only entries within the last 2 weeks, the
  date-range filter shows "Last 2 weeks" selected, and the URL is not
  modified

#### Scenario: An explicit date filter overrides the default

- **WHEN** an authenticated visitor opens `/entries?range=this_month`
- **THEN** the list shows entries within the current month instead of the
  last-2-weeks default

### Requirement: Returning to the entry ledger restores the last-applied filters

`/entries` SHALL restore the visitor's most recently persisted
filter/search/sort state from browser-local storage when it is arrived at
with a completely bare URL — no `account_id`, `category_id`, `tag_id`,
`kind`, `range`, `from`, `to`, `q`, `sort`, `dir`, or `show_recurring`
parameter present at all — replacing the URL rather than leaving the bare
arrival in browser history, if any state has been persisted. This applies
to every ordinary navigation to bare `/entries` (a sidebar link, a saved
bookmark to `/entries` with no parameters, a link from elsewhere in the
app), not only a specific round trip. Arriving with any explicit parameter
already present SHALL NOT be overridden by persisted state — the explicit
parameters apply exactly as they would with nothing persisted. When
nothing has been persisted yet, a bare arrival SHALL fall back to the
default view (see "The entry ledger defaults to the last 2 weeks").

#### Scenario: A sidebar click restores the last-applied filters

- **WHEN** an authenticated visitor applies a tag filter and a custom date
  range on `/entries`, navigates elsewhere in the app, then clicks
  "Entries" in the sidebar
- **THEN** they land on `/entries` with the same tag filter and date range
  applied, not the default view

#### Scenario: Filters persist across a save-and-return round trip

- **WHEN** an authenticated visitor applies a tag filter and a custom date
  range on `/entries`, opens an entry, and saves it
- **THEN** they return to `/entries` with the same tag filter and date
  range applied, not just the account

#### Scenario: No persisted state falls back to the default view

- **WHEN** an authenticated visitor with nothing yet persisted opens a bare
  `/entries`
- **THEN** the default "Last 2 weeks" view applies, as described in "The
  entry ledger defaults to the last 2 weeks"

#### Scenario: An explicit link is not overridden by persisted state

- **WHEN** an authenticated visitor with a persisted category filter
  follows a link to `/entries?account_id={id}`
- **THEN** the entry list is filtered only by that account, not also by the
  previously persisted category

### Requirement: The entry ledger supports selecting loaded entries for a bulk action

The entry ledger (`/entries`) SHALL show a checkbox on every row, always
enabled regardless of the visitor's write permission on that entry, plus a
header checkbox that selects or deselects every entry currently loaded in
the list (the infinite-scroll window fetched so far — not every entry
matching the active filter). Selection SHALL persist as more entries load
via scrolling, and SHALL be cleared whenever a filter/search/sort change
resets the loaded list or the visitor explicitly clears it.

#### Scenario: Selecting individual rows

- **WHEN** an authenticated visitor checks two rows in the ledger
- **THEN** both entries are selected and a bulk-action toolbar appears
  showing a count of 2

#### Scenario: Header checkbox selects only loaded rows

- **WHEN** a visitor activates the header checkbox with 30 entries loaded
  and more available via scrolling
- **THEN** exactly the 30 loaded entries become selected; entries not yet
  scrolled into view are not

#### Scenario: Changing a filter clears the selection

- **WHEN** a visitor has entries selected and then changes any filter,
  search term, or sort
- **THEN** the selection is cleared along with the loaded list resetting

#### Scenario: A row the visitor cannot edit is still selectable

- **WHEN** a visitor with only `view` permission on a shared account, or
  `append` permission on an entry created by someone else, views that
  entry in the ledger
- **THEN** its row checkbox is enabled and selectable like any other row

### Requirement: Every bulk action requires an explicit confirmation before it runs

Selecting a bulk-action toolbar action SHALL open a dialog to choose the
action's value (a category, one or more tags, a recurring transaction, or
— for delete — a destructive-confirmation prompt with no value to choose).
No request SHALL be sent to the backend until the visitor confirms within
that dialog.

#### Scenario: Picking a value does not run the action

- **WHEN** a visitor opens the "Set category" dialog and selects a
  category, without pressing Apply
- **THEN** no request has been sent and no entry has been changed

#### Scenario: Confirming starts the run

- **WHEN** a visitor presses Apply (or, for delete, confirms the
  destructive prompt)
- **THEN** the bulk-run modal opens and begins applying the action to
  every selected entry

### Requirement: A bulk action applies sequentially to every selected entry and reports per-entry results

Confirming a bulk action SHALL run one request per selected entry,
sequentially, showing live progress. On completion it SHALL report how
many entries succeeded and SHALL list, individually, every entry that
failed with a short reason (e.g. forbidden, not found, invalid value). An
entry whose current state already matches the action's chosen value SHALL
be counted as succeeded without a request being sent for it. The
completion view SHALL offer a "Reload list" action, which re-fetches the
ledger under the current filters and clears the selection, and a "Close"
action, which dismisses the modal without refetching.

#### Scenario: A forbidden entry is reported, not silently skipped

- **WHEN** a bulk action's selection includes an entry the visitor lacks
  write permission on
- **THEN** the run modal completes with that entry listed as failed
  (forbidden), and every other selected entry the visitor could act on is
  still applied

#### Scenario: An already-matching entry counts as succeeded with no request sent

- **WHEN** a bulk "Set category" action is applied to a selection where
  one entry already has the chosen category
- **THEN** that entry is counted among the succeeded entries and no update
  request is sent for it

#### Scenario: Reload list refreshes the ledger

- **WHEN** a visitor presses "Reload list" on the run modal's completion
  view
- **THEN** the ledger re-fetches its current filtered view from the start
  and the selection is cleared

#### Scenario: Close leaves the ledger as-is

- **WHEN** a visitor presses "Close" on the run modal's completion view
- **THEN** the modal dismisses and the ledger's currently loaded rows are
  left unchanged until the visitor reloads or navigates

### Requirement: Bulk "Set category" applies one category to every selected entry

The "Set category" action SHALL offer the same category choices as the
entry edit form (the visitor's own non-disabled categories plus any
shared to them at `append`+ permission), including an option to clear the
category. Confirming SHALL update each selected entry's `category_id` to
the chosen value.

#### Scenario: Setting a category across a mixed selection

- **WHEN** a visitor selects entries with different current categories and
  applies "Set category" with a chosen category
- **THEN** every selected entry the visitor can edit ends up with that
  category

#### Scenario: Clearing the category is rejected for a transaction entry

- **WHEN** a visitor applies "Set category" with "No category" chosen to a
  selection that includes a `transaction`-kind entry
- **THEN** that entry appears in the run modal's failure list (a
  transaction requires a category), while any `balance_adjustment`-kind
  entries in the selection are cleared successfully

### Requirement: Bulk "Add tags" and "Set tags" reuse the entry form's tag input

"Add tags" and "Set tags" SHALL each open the same tag-input control the
entry create/edit form uses, offering the visitor's existing non-disabled,
non-view-tier tags as suggestions and allowing a new tag name to be typed.
"Add tags" SHALL add the chosen tags to each selected entry's existing tags
without removing any tag already present. "Set tags" SHALL replace each
selected entry's tag set with exactly the chosen tags.

#### Scenario: Adding tags preserves existing tags

- **WHEN** a visitor applies "Add tags" with tag "Reviewed" chosen to an
  entry that already carries tag "Groceries"
- **THEN** that entry ends up carrying both "Groceries" and "Reviewed"

#### Scenario: Setting tags replaces the existing set

- **WHEN** a visitor applies "Set tags" with only tag "Reviewed" chosen to
  an entry that currently carries "Groceries" and "Business"
- **THEN** that entry ends up carrying only "Reviewed"

### Requirement: Bulk "Remove tags" only offers tags present on the selected entries

The "Remove tags" action SHALL offer, as choosable tags, only those
currently present on at least one selected entry — not the visitor's full
tag list. Confirming SHALL remove each chosen tag from every selected
entry that carries it, leaving entries that don't carry a chosen tag
unaffected by it.

#### Scenario: Only in-use tags are offered

- **WHEN** a visitor opens "Remove tags" for a selection where the only
  tags present across those entries are "Groceries" and "Business"
- **THEN** the tag picker offers only "Groceries" and "Business", even if
  the visitor has other tags defined elsewhere

#### Scenario: Removing a tag only affects entries that carry it

- **WHEN** a visitor applies "Remove tags" with "Groceries" chosen to a
  selection where only some entries carry that tag
- **THEN** entries carrying "Groceries" have it removed, and entries that
  never carried it are counted as succeeded with no change

### Requirement: Bulk "Link to recurring transaction" is restricted to a single-account selection

The "Link to recurring transaction" action SHALL be disabled, with an
explanatory hint, whenever the current selection's entries do not all
share the same account. When enabled, it SHALL offer the recurring
transactions defined on that one shared account, plus an option to unlink.
Confirming SHALL set (or clear) each selected entry's linked recurring
transaction accordingly.

#### Scenario: Action is disabled across accounts

- **WHEN** the current selection includes entries from two different
  accounts
- **THEN** "Link to recurring transaction" appears disabled with a hint
  explaining why

#### Scenario: Action is enabled and scoped for a single-account selection

- **WHEN** every entry in the current selection belongs to the same
  account
- **THEN** "Link to recurring transaction" is enabled and its picker
  offers only that account's recurring transactions

#### Scenario: Unlinking clears the link on every selected entry

- **WHEN** a visitor applies "Link to recurring transaction" with "Not
  linked" chosen
- **THEN** every selected entry's recurring-transaction link is cleared

### Requirement: Bulk "Delete" soft-deletes every selected entry after a destructive confirmation

The "Delete" action SHALL show a destructive-confirmation prompt (no value
to choose) before running. Confirming SHALL soft-delete every selected
entry the visitor has permission to delete, one at a time, reporting the
same succeeded/failed breakdown as any other bulk action.

#### Scenario: Deleting a selection requires confirmation

- **WHEN** a visitor activates "Delete" on a bulk selection
- **THEN** a confirmation prompt appears and no entry is deleted until it
  is confirmed

#### Scenario: Deleted entries no longer appear after reloading

- **WHEN** a visitor confirms bulk delete and then presses "Reload list" on
  the run modal's completion view
- **THEN** the successfully deleted entries no longer appear in the ledger

### Requirement: Entries link in the toolbar to import

`/entries`' toolbar SHALL offer an "Import" action alongside "New Entry",
navigating to `/entries/import`. When the ledger's current view has an
`account_id` filter applied, the Import action SHALL carry it as
`/entries/import?account_id={id}`, the same way "New Entry" already
carries the current filter.

#### Scenario: Navigating to import from the ledger

- **WHEN** an authenticated visitor activates "Import" on `/entries`
- **THEN** the client navigates to `/entries/import`

#### Scenario: The current account filter is carried over

- **WHEN** an authenticated visitor is viewing `/entries?account_id={id}`
  and activates "Import"
- **THEN** the client navigates to `/entries/import?account_id={id}`

### Requirement: Entry amounts are colored by sign, with a distinct treatment for balance adjustments

The entries list SHALL render each entry's amount in a color reflecting its
sign: red when negative, the default/neutral text color when exactly zero,
and green when positive. An entry whose `kind` is `balance_adjustment`
SHALL additionally have its amount rendered underlined, distinguishing it
from a `transaction`'s amount at a glance, and SHALL additionally show its
computed delta (`amount`) as a small, gray annotation next to its reading,
rendered without sign-based coloring.

#### Scenario: Negative amount is red

- **WHEN** the entries list renders an entry with a negative amount
- **THEN** the amount is shown in red

#### Scenario: Zero amount stays neutral

- **WHEN** the entries list renders an entry with an amount of exactly zero
- **THEN** the amount is shown in the default text color, neither red nor
  green

#### Scenario: Positive amount is green

- **WHEN** the entries list renders an entry with a positive amount
- **THEN** the amount is shown in green

#### Scenario: A balance adjustment's amount is underlined

- **WHEN** the entries list renders an entry whose `kind` is
  `balance_adjustment`
- **THEN** its amount is rendered underlined, in addition to its sign color

#### Scenario: A transaction's amount is not underlined

- **WHEN** the entries list renders an entry whose `kind` is `transaction`
- **THEN** its amount is rendered without an underline

#### Scenario: A balance adjustment shows its delta alongside its reading

- **WHEN** the entries list renders an entry whose `kind` is
  `balance_adjustment`
- **THEN** its computed delta is shown next to its reading, in a smaller,
  gray typeface, not colored by sign

### Requirement: Creating and editing an entry

`/entries/new` SHALL offer a form for account (pre-selected and, when
arrived at via `?account_id=`, not changeable in that flow; otherwise a
choice among the visitor's own non-deleted accounts *and* every
non-deleted shared account the visitor holds at least `append` permission
on), kind, amount (entered and displayed at full stored precision,
independent of the visitor's display-rounding preference), booking
timestamp, title, description, a category (required unless kind is
`balance_adjustment` or `self_transfer`), and tags. When kind is
`transaction`, the form SHALL additionally offer a `counterparty` text
input (suggested values from `GET /api/entries/counterparties` via a
`<datalist>`) and a `location` field (a plain text input, editable
directly, alongside a "Use GPS" button and a "Pick on map" button — see the
location requirements below); both are hidden when kind is
`balance_adjustment` or `self_transfer`, mirroring how category is treated.
When kind is `self_transfer`, the form SHALL additionally offer a "To
account" `<select>`, offering only non-deleted, non-disabled accounts the
visitor holds at least `append` permission on, sharing the same `currency`
as the currently selected "from" account, and excluding whichever account
is currently selected as "from" — narrowed live as the "from" account
changes. Submitting calls `POST /api/entries`.

`/entries/{id}/edit` SHALL offer the same form pre-populated from the
existing entry, with kind rendered read-only (immutable per the backend)
and account rendered locked by default, changeable only through the
unlock interaction described below, submitting an update on save. The edit
page SHALL offer its full form, including the delete action behind a
confirmation step, only to a visitor with `entry_admin`+ permission on the
entry's account, or with `append` permission and `created_by` matching
them; a visitor with `view` permission, or `append` permission on an entry
they did not create, SHALL instead see the entry's details as read-only,
with no edit or delete action. For a `self_transfer` entry specifically,
the full editable form additionally requires the visitor to currently hold
at least `append` permission on both `account_id` and `to_account_id` — a
visitor who qualifies under the general rule above but has since lost
access to the other account SHALL still see the entry read-only, with no
edit action; the delete action, when otherwise available, is unaffected by
this narrower rule (see "A self-transfer's delete action does not require
the other account").

#### Scenario: Creating a transaction requires a category

- **WHEN** an authenticated visitor submits the create form with
  `kind: transaction` and no category selected
- **THEN** the form shows a validation error and does not submit

#### Scenario: Creating a balance adjustment allows an empty category

- **WHEN** an authenticated visitor submits the create form with
  `kind: balance_adjustment` and no category selected
- **THEN** the entry is created successfully

#### Scenario: Creating a self-transfer allows an empty category

- **WHEN** an authenticated visitor submits the create form with
  `kind: self_transfer`, a "to account" selected, and no category selected
- **THEN** the entry is created successfully

#### Scenario: Counterparty and location are hidden for a balance adjustment

- **WHEN** an authenticated visitor selects `kind: balance_adjustment` on
  either the create or edit form
- **THEN** the counterparty and location fields are not shown, and neither
  is included in the submitted request

#### Scenario: Counterparty and location are hidden for a self-transfer

- **WHEN** an authenticated visitor selects `kind: self_transfer` on
  either the create or edit form
- **THEN** the counterparty and location fields are not shown, and neither
  is included in the submitted request

#### Scenario: Counterparty input suggests the visitor's own history

- **WHEN** an authenticated visitor focuses the counterparty field on a
  `kind: transaction` form
- **THEN** the suggestion list is populated from
  `GET /api/entries/counterparties`

#### Scenario: Account and kind are not editable

- **WHEN** an authenticated visitor opens `/entries/{id}/edit`
- **THEN** the account and kind fields are shown but cannot be changed —
  the account field's unlock interaction (see below) is a separate,
  deliberate action, not a default-editable state

#### Scenario: Deleting an entry requires confirmation

- **WHEN** an authenticated visitor with sufficient permission activates
  "Delete" on the edit page
- **THEN** a confirmation step appears and the delete request is not sent
  until it is confirmed

#### Scenario: The new-entry account picker includes shared accounts

- **WHEN** a visitor with `append`+ permission on a shared account opens
  `/entries/new` with no preset `account_id`
- **THEN** that account appears among the choosable accounts

#### Scenario: A view-tier visitor sees an entry read-only

- **WHEN** a visitor with only `view` permission on a shared account opens
  one of its entries
- **THEN** the entry's details render read-only, with no edit or delete
  action

#### Scenario: An append-tier visitor cannot edit another user's entry

- **WHEN** a visitor with `append` permission opens an entry on the same
  account that a different user created
- **THEN** the entry's details render read-only, with no edit or delete
  action

#### Scenario: Selecting self-transfer reveals the "to account" picker

- **WHEN** an authenticated visitor selects `kind: self_transfer` on
  `/entries/new`
- **THEN** a "To account" `<select>` appears, offering only append+,
  non-disabled accounts sharing the "from" account's currency, excluding
  the "from" account itself

#### Scenario: The "to account" picker excludes different-currency accounts

- **WHEN** an authenticated visitor has `append`+ permission on an account
  of a different currency than the selected "from" account
- **THEN** that account does not appear among the "to account" choices

#### Scenario: A self-transfer missing permission on the other account is read-only

- **WHEN** a visitor who meets the general edit-permission rule for an
  entry's `account_id` opens a `kind: self_transfer` entry whose
  `to_account_id` they no longer have `append`+ permission on
- **THEN** the entry's details render read-only, with no edit action

### Requirement: A self-transfer's delete action does not require the other account

The edit page's delete action for a `kind: self_transfer` entry SHALL
follow the same permission rule as any other entry's delete action
(`entry_admin`/`owner` on the accessible account, or `append` with
`created_by` matching the visitor), evaluated only against whichever of
the entry's two accounts the visitor currently has permission on. Losing
access to the *other* account SHALL NOT hide or disable the delete action,
even though it does hide the edit action (see "Creating and editing an
entry").

#### Scenario: Delete remains available after losing access to the other account

- **WHEN** a visitor holds `entry_admin`/`owner` permission on one account
  of a `self_transfer` entry and has since lost all permission on the
  other
- **THEN** the edit page still shows the entry's fields read-only (no edit
  action) but keeps the Delete action available

### Requirement: The location field accepts a typed address, device GPS, or a dropped map pin, all as one text value

The `location` field on a `kind: transaction` entry form SHALL be a plain
text input the visitor can type into directly (e.g. a street address),
alongside two buttons: **Use GPS**, which calls the browser's geolocation
API and, on success, writes the coordinate into the field as a JSON string
`{"lat":<number>,"lng":<number>}`; and **Pick on map**, which opens an
interactive map modal with a click-to-place, draggable pin, and on
confirmation writes the same JSON shape into the field. The field SHALL
always display its literal current value, including raw JSON when that is
what is stored — there is no separate, prettified display distinct from
the field's actual content. The map modal SHALL center on the device's
current GPS position when available, otherwise a world view, and SHALL
NOT offer address search.

#### Scenario: Typing an address sets the field to plain text

- **WHEN** a visitor types "123 Main St" into the location field with
  neither button used
- **THEN** the field's value is the literal string "123 Main St"

#### Scenario: Use GPS writes a coordinate JSON string

- **WHEN** a visitor activates "Use GPS" and the browser reports a
  position successfully
- **THEN** the location field's value becomes
  `{"lat":<latitude>,"lng":<longitude>}`, shown verbatim in the field

#### Scenario: GPS failure shows an inline error, not a silent no-op

- **WHEN** a visitor activates "Use GPS" and the browser denies permission
  or the request times out
- **THEN** an inline error is shown and the location field is left
  unchanged

#### Scenario: Pick on map writes the confirmed pin as a coordinate JSON string

- **WHEN** a visitor opens "Pick on map", places or drags the pin, and
  confirms
- **THEN** the location field's value becomes the pin's
  `{"lat":<latitude>,"lng":<longitude>}`, shown verbatim in the field

#### Scenario: The map modal centers on the device position when available

- **WHEN** a visitor opens "Pick on map" and the browser can report a
  current position
- **THEN** the map initially centers on that position

#### Scenario: The map modal falls back to a world view without GPS

- **WHEN** a visitor opens "Pick on map" and no current position is
  available (denied or unsupported)
- **THEN** the map initially shows a world view rather than failing to
  open

### Requirement: Linking or unlinking an existing entry to a recurring transaction happens only on the entry edit page

`/entries/{id}/edit` SHALL offer a field to set or clear the entry's
`recurring_transaction_id`, offering only recurring transactions on the
entry's current account as choices. `/entries/new` SHALL NOT expose this
field as a normal, visitor-facing control — the only way a newly created
entry gets linked is via the recurring transaction's own "Create
transaction" prefill flow (see `web-client-recurring-transactions`), which
carries the id through without a picker.

#### Scenario: Linking an existing entry from its edit page

- **WHEN** the visitor opens `/entries/{id}/edit` and selects a recurring
  transaction on the entry's account, then saves
- **THEN** `PATCH /api/entries/{id}` is called with that
  `recurring_transaction_id`, and the entry is now linked

#### Scenario: Unlinking

- **WHEN** the visitor clears the field on `/entries/{id}/edit` and saves
- **THEN** `PATCH /api/entries/{id}` is called with
  `recurring_transaction_id: null`

#### Scenario: New entry form has no link picker

- **WHEN** the visitor opens `/entries/new` directly (not via a recurring
  transaction's "Create transaction" action)
- **THEN** no recurring-transaction picker is shown on the form

### Requirement: A transaction entry offers a shortcut to create a recurring transaction from it

`/entries/{id}/edit` SHALL offer a compact, icon-only action that starts a
new recurring transaction template prefilled from the current entry,
visible only when the entry's `kind` is `transaction` and the visitor has
the same edit permission the rest of the form requires (`entry_admin`+ on
the account, or `append` and `created_by` matching the visitor). It SHALL
NOT be shown for a `kind: balance_adjustment` entry, and SHALL NOT be shown
to a visitor who only sees the entry read-only.

The action's accessible name (and visible label, if any) SHALL depend on
whether the form has unsaved edits since it was loaded or last saved:

- With no unsaved edits, activating it SHALL navigate directly to
  `/recurring/new?from_entry_id=<id>` without submitting anything.
- With unsaved edits, its label SHALL indicate that saving happens first
  (e.g. "Save and create recurring transaction"). Activating it SHALL run
  the same validation and save request the entry form's normal Save action
  runs — including the account-change confirmation step when the account
  was changed — and, only on a successful save, SHALL navigate to
  `/recurring/new?from_entry_id=<id>` instead of the normal Save action's
  redirect to `/entries`.

#### Scenario: Action hidden for a balance adjustment

- **WHEN** the visitor opens `/entries/{id}/edit` for an entry with
  `kind: balance_adjustment`
- **THEN** the "create recurring transaction" action is not shown

#### Scenario: Action hidden for a read-only visitor

- **WHEN** a visitor with only `view` permission, or `append` permission on
  an entry they did not create, opens the entry
- **THEN** the "create recurring transaction" action is not shown

#### Scenario: Clean form navigates directly

- **WHEN** the visitor opens `/entries/{id}/edit` for a transaction entry,
  makes no changes, and activates the action
- **THEN** the browser navigates to `/recurring/new?from_entry_id=<id>`
  with no save request sent

#### Scenario: Dirty form saves first, then navigates

- **WHEN** the visitor changes a field on the entry form and activates the
  action
- **THEN** `PATCH /api/entries/{id}` is called with the changed values, and
  only after it succeeds does the browser navigate to
  `/recurring/new?from_entry_id=<id>`

#### Scenario: A failed save from this action does not navigate

- **WHEN** the visitor activates the action on a dirty form and the save
  request fails
- **THEN** the entry form shows its normal save error and the browser does
  not navigate to `/recurring/new`

#### Scenario: Account change confirmation still applies

- **WHEN** the visitor changes the entry's account and activates the
  action
- **THEN** the same account-change confirmation dialog the normal Save
  action shows is presented before the save request is sent

### Requirement: An entry's title opens a read-only summary modal

Activating an entry's title SHALL open a read-only summary modal for that
entry rather than navigating, wherever an entry is listed by title — the
ledger (`/entries`), an account's recent-entries list, the `/reports`
results table, and the dashboard's `entry_list` card. The summary SHALL
show the entry's booking timestamp, account, title,
description, category, tags, counterparty, location, signed amount, and
running balance, each omitted when the entry does not carry it, plus the
creator's name whenever `created_by` differs from the visitor. For a
`self_transfer` entry it SHALL additionally name the counterpart account.
Nothing in the summary SHALL be editable.

The summary SHALL render from the entry data the listing already holds,
issuing no further request when opened.

#### Scenario: Activating an entry title opens the summary

- **WHEN** an authenticated visitor activates an entry's title in the
  ledger
- **THEN** a read-only summary modal for that entry opens, and the browser
  does not navigate away from the ledger

#### Scenario: The summary opens without a request

- **WHEN** the summary modal opens for an entry already present in the
  listing
- **THEN** no additional entry request is issued

#### Scenario: A self-transfer summary names both accounts

- **WHEN** the summary opens for an entry whose `kind` is `self_transfer`
- **THEN** it names both the entry's account and its counterpart account

#### Scenario: Absent fields are omitted

- **WHEN** the summary opens for an entry with no description, no
  counterparty, and no location
- **THEN** those rows are omitted rather than shown empty

### Requirement: The entry summary's Edit action follows the entry edit permission rule

The entry summary modal SHALL offer an Edit action navigating to
`/entries/{id}/edit`, shown only to a visitor permitted to edit that
entry under the same rule the edit page applies: `entry_admin` or `owner`
permission on the entry's account, or `append` permission with
`created_by` matching the visitor; and for a `self_transfer`, at least
`append` on *both* accounts with no `created_by` exemption. That rule
SHALL be evaluated by one shared implementation used by both the summary
and the edit page.

#### Scenario: A permitted visitor sees Edit

- **WHEN** a visitor with `owner` permission on the entry's account opens
  the summary
- **THEN** the summary offers an Edit action to that entry's edit page

#### Scenario: A view-tier visitor sees no Edit action

- **WHEN** a visitor with only `view` permission on a shared account opens
  the summary for one of its entries
- **THEN** the summary shows the entry's details with no Edit action

#### Scenario: An append-tier visitor sees no Edit action on another user's entry

- **WHEN** a visitor with `append` permission opens the summary for an
  entry on the same account that a different user created
- **THEN** the summary shows no Edit action

#### Scenario: A self-transfer needs append on both accounts

- **WHEN** a visitor holds `append` on a self-transfer's own account but
  only `view` on its counterpart account
- **THEN** the summary shows no Edit action

### Requirement: A linked entry shows a badge to its recurring transaction in the ledger

`/entries` SHALL show a small icon/badge on any entry whose
`recurring_transaction_id` is set, distinct from the row's other content,
that opens that recurring transaction's read-only summary modal when
activated. An entry with no `recurring_transaction_id` SHALL show no such
badge. Activating the badge SHALL NOT also trigger the surrounding row's
own click behaviour.

#### Scenario: Linked entry shows the badge

- **WHEN** the ledger lists an entry with a non-null
  `recurring_transaction_id`
- **THEN** its row shows the recurring-transaction badge

#### Scenario: Badge opens the recurring transaction's summary

- **WHEN** the visitor activates a linked entry's badge
- **THEN** that recurring transaction's read-only summary modal opens and
  the browser does not navigate

#### Scenario: Activating the badge does not trigger the row

- **WHEN** the visitor activates the badge on a row that itself responds
  to clicks
- **THEN** only the recurring summary opens; the row's own behaviour does
  not fire

#### Scenario: Unlinked entry shows no badge

- **WHEN** the ledger lists an entry with a null `recurring_transaction_id`
- **THEN** no recurring-transaction badge is shown on its row

### Requirement: A self-transfer entry shows a badge linking to its other account's leg

`/entries` and the `/reports` results table SHALL show a small icon/badge
on any `kind: self_transfer` entry row, distinct from the row's other
content, that navigates to `/entries/{id}/edit` for that same entry when
activated — the same entry, viewed from its other account. A `self_transfer`
rendered once (only one of its two accounts in the current view's scope)
still shows the badge; the badge does not depend on the entry appearing
twice.

#### Scenario: A self-transfer row shows the badge

- **WHEN** the ledger or reports table lists a `kind: self_transfer` entry
- **THEN** its row shows the self-transfer badge, naming the other account

#### Scenario: Activating the badge opens the entry's edit page

- **WHEN** the visitor activates a self-transfer row's badge
- **THEN** the browser navigates to that entry's own `/entries/{id}/edit`
  page

#### Scenario: A non-transfer entry shows no self-transfer badge

- **WHEN** the ledger or reports table lists a `transaction` or
  `balance_adjustment` entry
- **THEN** no self-transfer badge is shown on its row

### Requirement: Each Upcoming row offers the existing Create transaction action, prefilled to that occurrence's date

Each row in the Upcoming block SHALL offer a "Create transaction" action,
mirroring the one `/recurring` already offers, navigating to
`/entries/new?recurring_transaction_id={id}&booking_timestamp={that row's
booking_timestamp}` (see `web-client-recurring-transactions`'s date-override
requirement).

Below the `sm` breakpoint the action SHALL render as a plus glyph alone,
and from `sm` up as that glyph beside its translated label. Its
accessible name SHALL be that same translated label at every width.

#### Scenario: Activating Create transaction on an Upcoming row

- **WHEN** an authenticated visitor activates "Create transaction" on an
  Upcoming block row projected for 2026-12-01
- **THEN** the client navigates to `/entries/new` with that row's
  `recurring_transaction_id` and `booking_timestamp=2026-12-01`, and the
  form prefills accordingly

#### Scenario: The action is icon-only on a phone

- **WHEN** an Upcoming block row is rendered below the `sm` breakpoint
- **THEN** its Create transaction action shows the plus glyph without its
  label, and its accessible name is still the translated label

### Requirement: An overdue previewed occurrence is visually distinguished but stays in chronological order

A row in the Upcoming block whose `overdue` is `true` SHALL render with a
distinct, muted background tint from a non-overdue row, and SHALL carry a
legible overdue marker beside its title that is never truncated or
ellipsised, however long the title is. It SHALL NOT be moved out of its
normal position in the block's date-ascending order.

#### Scenario: An overdue row is tinted, not reordered

- **WHEN** the Upcoming block includes one overdue row and several
  non-overdue rows
- **THEN** the overdue row renders with the distinct background tint in
  its correct chronological position among the others, not pulled to the
  top or bottom

#### Scenario: A long title never hides the overdue marker

- **WHEN** an overdue row's title is too long for the row and is
  truncated
- **THEN** the overdue marker beside it is still shown in full

### Requirement: An Upcoming row's title opens the recurring transaction's summary

Each row in the Upcoming block SHALL render its title as an activatable
control that opens that row's recurring transaction in the read-only
recurring summary modal (see `web-client-modals`), the same summary
`/recurring`'s rows and a linked entry's badge already open.

#### Scenario: Activating an Upcoming row's title

- **WHEN** an authenticated visitor activates the title of an Upcoming
  block row
- **THEN** that row's recurring transaction's read-only summary modal
  opens, with no navigation away from the ledger

### Requirement: An Upcoming row lays out so that no part of it overlaps another

An Upcoming row SHALL render its title and amount on one line and its
date, account and Create transaction action on a second, with each
shrinkable part truncating within its own bounds. At every width the
block is rendered at, no part of a row SHALL overlap another, and the
date and its separator SHALL stay on one line.

#### Scenario: A narrow row truncates rather than overlapping

- **WHEN** an Upcoming row with a long title and a long account name is
  rendered in the narrowest container the block is used in
- **THEN** the title and the account name truncate within their own
  bounds, and neither is drawn over the amount or the action

### Requirement: An entry not created by the current visitor shows who created it

Wherever an entry is rendered — the ledger (`/entries`), an account's
recent-entries list, and a single entry's details — a small annotation
naming the creator SHALL be shown whenever that entry's `created_by`
differs from the current visitor, using the `created_by_name` the backend
already resolves. An entry the visitor created themselves SHALL render
exactly as before this capability, with no such annotation.

#### Scenario: An entry created by someone else shows their name

- **WHEN** the entry ledger, an account's recent entries, or an entry's
  detail view renders an entry whose `created_by` differs from the current
  visitor
- **THEN** that row shows the creator's name

#### Scenario: A self-created entry shows no creator annotation

- **WHEN** any of those views renders an entry the current visitor created
  themselves
- **THEN** no creator annotation is shown, unchanged from before this
  capability existed

### Requirement: A valid coordinate location shows a globe icon in the ledger, opening a read-only map preview

In the entry ledger (`/entries`), any entry whose `location` value parses
as a JSON object with numeric `lat`/`lng` fields within valid coordinate
ranges SHALL show a small globe icon next to its title. Activating it
SHALL open a read-only map modal centered on that coordinate with a static
(non-draggable) pin and no editing controls. An entry whose `location` is
empty, or does not parse as such a coordinate object (including a plain
typed address), SHALL show no globe icon.

#### Scenario: A coordinate location shows the globe icon

- **WHEN** the ledger renders an entry whose `location` is
  `{"lat":48.2082,"lng":16.3738}`
- **THEN** a globe icon appears next to that entry's title

#### Scenario: Clicking the globe icon opens a read-only map centered on the point

- **WHEN** a visitor clicks the globe icon on an entry with a coordinate
  location
- **THEN** a modal opens showing a map centered on that coordinate with a
  static pin, and no control to move or confirm a different position

#### Scenario: A plain address location shows no globe icon

- **WHEN** the ledger renders an entry whose `location` is the plain
  string "123 Main St"
- **THEN** no globe icon appears for that entry

#### Scenario: An entry with no location shows no globe icon

- **WHEN** the ledger renders an entry whose `location` is empty or absent
- **THEN** no globe icon appears for that entry

### Requirement: A counterparty is shown as a second line under the entry's title in the ledger

Wherever an entry's title is rendered in the ledger (`/entries`), a
non-empty `counterparty` SHALL be shown as a second line beneath it, the
same slot/style used for the creator annotation (`created_by_name`) — both
may appear together when both apply. An entry with no counterparty SHALL
render exactly as before this capability, with no such line.

#### Scenario: An entry with a counterparty shows it under the title

- **WHEN** the ledger renders an entry whose `counterparty` is "Rewe"
- **THEN** "Rewe" appears on a line beneath that entry's title

#### Scenario: An entry with no counterparty is unaffected

- **WHEN** the ledger renders an entry with no `counterparty`
- **THEN** no counterparty line is shown, unchanged from before this
  capability existed

### Requirement: The edit form's account field is locked by default and unlocked via a button

On `/entries/{id}/edit`, the account field SHALL render, by default, as a
disabled display of the entry's current account with an adjacent icon
button (a pencil, labeled for assistive technology as changing the
account) beside it. Activating that button SHALL replace the disabled
display with a `<select>` listing the visitor's own non-deleted,
non-disabled accounts, alongside a second button that discards any pending
selection and returns the field to its locked, disabled display showing
the entry's original account. If the entry's current account has since
been disabled, it SHALL still render as the unlocked field's current,
selected value (mirroring the category picker's handling of a disabled
current category) but SHALL NOT appear as a choosable option once a
different account has been selected.

#### Scenario: Unlocking the account field

- **WHEN** an authenticated visitor activates the account field's unlock
  button on `/entries/{id}/edit`
- **THEN** the field becomes a `<select>` of the visitor's own non-deleted,
  non-disabled accounts, defaulted to the entry's current account

#### Scenario: Canceling a pending account change

- **WHEN** an authenticated visitor has unlocked the account field,
  selected a different account, and then activates the cancel button
  instead of saving
- **THEN** the field reverts to its locked, disabled display showing the
  entry's original account, and no request is sent

#### Scenario: A disabled current account still renders while unlocked

- **WHEN** an authenticated visitor unlocks the account field on an entry
  whose current account has since been disabled
- **THEN** that account renders as the field's current, selected value,
  but does not appear among the selectable options

### Requirement: Selecting a different-currency account shows a warning

On `/entries/{id}/edit`, once the account field is unlocked, selecting an
account whose `currency` differs from the entry's original account SHALL
show an inline warning next to the field, without preventing the
selection.

#### Scenario: Selecting a same-currency account shows no warning

- **WHEN** an authenticated visitor selects an account whose `currency`
  matches the entry's original account
- **THEN** no currency warning is shown

#### Scenario: Selecting a different-currency account shows a warning

- **WHEN** an authenticated visitor selects an account whose `currency`
  differs from the entry's original account
- **THEN** an inline warning is shown next to the account field, and the
  selection is not prevented

### Requirement: Saving a changed account requires confirmation

On `/entries/{id}/edit`, submitting the form with the account field's
selection unchanged from the entry's original `account_id` SHALL submit
the update immediately, the same as any other field change. Submitting
with a changed `account_id` SHALL first show a confirmation dialog naming
the destination account — repeating the currency-mismatch warning when
applicable — before any request is sent. Confirming SHALL submit the
update, including any other fields edited in the same visit; canceling
SHALL return to the form without sending a request.

#### Scenario: Saving without an account change skips confirmation

- **WHEN** an authenticated visitor saves the edit form without having
  changed the account field
- **THEN** the update is submitted immediately, with no confirmation
  dialog shown

#### Scenario: Saving a changed account shows a confirmation dialog

- **WHEN** an authenticated visitor saves the edit form after selecting a
  different account
- **THEN** a confirmation dialog appears naming the destination account
  before any request is sent

#### Scenario: Confirming submits the whole form

- **WHEN** an authenticated visitor confirms the account-change dialog
- **THEN** the update is submitted with the new `account_id` and any other
  fields edited in the same visit

#### Scenario: Canceling the confirmation submits nothing

- **WHEN** an authenticated visitor dismisses the account-change
  confirmation dialog without confirming
- **THEN** no request is sent and the form remains as it was, with the
  changed account selection still pending

### Requirement: A transaction's amount is entered via a sign toggle and cannot be zero

On `/entries/new` and `/entries/{id}/edit`, when the entry's `kind` is
`transaction`, the amount field SHALL present a sign toggle control
alongside a magnitude-only input, rather than a single free-typed signed
value. Typing or pasting a `-` character into the magnitude input SHALL
never insert that character into the field; it SHALL instead set the
toggle to minus, whether or not it was already set to minus. A newly
created transaction SHALL start with the toggle defaulted to minus.
Submitting the form with a magnitude that resolves to exactly zero SHALL
be rejected with a validation error and SHALL NOT submit the request — a
transaction's amount must be strictly positive or strictly negative.

The amount field's background and text color SHALL reflect the current
sign and magnitude the same way the read-only display does: the default
neutral styling while the magnitude is zero, a red-tinted styling while
the toggle is set to minus and the magnitude is non-zero, and a
green-tinted styling while the toggle is set to plus and the magnitude is
non-zero. The toggle control itself SHALL use a more saturated accent of
the same red/green colors, distinct from the field's lighter tint.

This requirement does not apply when the entry's `kind` is
`balance_adjustment` — see the following requirement.

#### Scenario: Typing a minus sets the toggle and is not inserted

- **WHEN** an authenticated visitor types `-` into the transaction amount
  field, regardless of the toggle's current state
- **THEN** the toggle is set to minus and the `-` character does not appear
  in the field's text

#### Scenario: A new transaction defaults to minus

- **WHEN** an authenticated visitor opens `/entries/new` and selects
  `kind: transaction`
- **THEN** the amount field's sign toggle starts set to minus

#### Scenario: Toggling the sign updates the field's styling

- **WHEN** an authenticated visitor has entered a non-zero magnitude and
  activates the sign toggle
- **THEN** the resulting signed amount, the field's background/text tint,
  and the toggle's own accent color all switch to match the new sign

#### Scenario: A zero-magnitude transaction cannot be submitted

- **WHEN** an authenticated visitor submits the transaction amount field
  with a magnitude of zero
- **THEN** the form shows a validation error and does not submit

### Requirement: A balance adjustment's amount field is unaffected by the sign toggle

On `/entries/new` and `/entries/{id}/edit`, when the entry's `kind` is
`balance_adjustment`, the amount field SHALL remain a single free-typed
input with no sign toggle control and no sign-based coloring, and SHALL
continue to accept a magnitude that resolves to exactly zero. This field
edits the entry's `balance` reading, not its `amount` — submitting the form
for a `balance_adjustment` SHALL send the entered value as `balance`, and
the computed `amount` (delta) returned by the backend is never edited
directly.

#### Scenario: Balance adjustment amount field has no toggle

- **WHEN** an authenticated visitor opens the amount field for an entry
  (new or existing) whose `kind` is `balance_adjustment`
- **THEN** no sign toggle button is shown and the field is not colored by
  sign

#### Scenario: A zero-amount balance adjustment can still be submitted

- **WHEN** an authenticated visitor submits a `balance_adjustment` entry
  with an amount of zero
- **THEN** the entry is saved successfully

#### Scenario: Submitting a balance adjustment sends balance, not amount

- **WHEN** an authenticated visitor submits the create or edit form for a
  `kind: balance_adjustment` entry
- **THEN** the request body carries the entered value as `balance`, and the
  form does not submit an `amount` field for that entry

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
  labeled as disabled, and it does not appear among the selectable options

#### Scenario: Saving other changes does not require reselecting a disabled category

- **WHEN** an authenticated visitor edits an entry whose current category
  is disabled and changes only an unrelated field, without touching the
  category
- **THEN** the save succeeds and the entry keeps its (disabled) category

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
(`/entries`) and an account's recent-entries list — the shared badge and
real owner's name (per `CategoryLabel`, the same treatment `AccountLabel`
gives a shared account) SHALL be shown whenever that category isn't the
current viewer's own, i.e. its `shared` field is `true`. A category the
viewer owns SHALL render exactly as before this capability, with no such
badge.

#### Scenario: An entry's shared category shows its owner

- **WHEN** the ledger or an account's recent entries renders an entry
  categorized under a category shared with (not owned by) the current
  viewer
- **THEN** that row shows the shared badge and the category's real
  owner's name

#### Scenario: An owned category shows no badge

- **WHEN** any of those views renders an entry categorized under a
  category the current viewer owns
- **THEN** no shared badge is shown, unchanged from before this
  capability existed

### Requirement: Tags can be created inline from the entry form

The entry form's tag input SHALL match against the visitor's existing,
non-disabled tags as they type, and SHALL NOT offer a disabled tag as a
suggestion. On submission, any entered tag value that does not match an
existing tag SHALL be created (`POST /api/tags`) before the entry is saved
with it attached; a value that matches an existing tag reuses it,
regardless of whether that tag is disabled. An entry being edited that
already carries a tag which has since been disabled SHALL continue to
display and resubmit that tag normally — the exclusion applies only to the
autocomplete suggestion list, not to a tag the entry already has.

#### Scenario: Typing a new tag name creates it

- **WHEN** an authenticated visitor types a tag name that does not match
  any of their existing tags and submits the entry form
- **THEN** a new tag with that name is created and attached to the entry

#### Scenario: Typing an existing tag name reuses it

- **WHEN** an authenticated visitor types a tag name matching one of their
  existing tags and submits the entry form
- **THEN** the existing tag is attached, and no duplicate tag is created

#### Scenario: Disabled tags are excluded from suggestions

- **WHEN** an authenticated visitor types into the tag input and one of
  their existing tags matching the typed text is disabled
- **THEN** that tag does not appear among the suggestions

#### Scenario: An entry keeps showing a tag disabled after it was attached

- **WHEN** an authenticated visitor opens the edit form for an entry that
  carries a tag which has since been disabled
- **THEN** that tag's name still appears among the entry's attached tags,
  and saving the form without removing it succeeds

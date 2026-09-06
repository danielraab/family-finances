## ADDED Requirements

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

`/entries` SHALL represent its current account, category, tag, kind, date
range, and free-text search filters, and its sort field and direction, as
typed URL search parameters, readable and writable through TanStack
Router's search-param APIs. Reloading a URL with search parameters SHALL
reproduce the same filtered/sorted view. Arriving at `/entries` with an
`account_id` parameter already set (for example, via the link from an
account's details page) SHALL apply that filter immediately on load.

#### Scenario: A filtered view survives a reload

- **WHEN** an authenticated visitor applies a category filter and a sort
  order, then reloads the page
- **THEN** the same category filter and sort order are applied after the
  reload

#### Scenario: Arriving with a preset account filter

- **WHEN** an authenticated visitor follows a link to
  `/entries?account_id={id}`
- **THEN** the entry list is immediately filtered to that account

#### Scenario: Changing a filter updates the URL

- **WHEN** an authenticated visitor changes the search text or a filter
  control
- **THEN** the corresponding URL search parameter changes to match

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

`/entries` SHALL offer controls for every backend filter (`account_id`,
`category_id`, `tag_id`, `kind`, a `from`/`to` date range), a free-text
search input, and a way to sort by booking timestamp or amount in either
direction. When no entries match the current filters, the page SHALL show
text distinguishing "no entries match these filters" from "no entries
exist yet."

#### Scenario: Combining filters narrows the results

- **WHEN** an authenticated visitor sets both an account filter and a kind
  filter
- **THEN** only entries matching both are shown

#### Scenario: No matches under the current filters

- **WHEN** an authenticated visitor's current filters match no entries but
  the visitor does have entries overall
- **THEN** the page shows text indicating no entries match the current
  filters, distinct from having none at all

### Requirement: Creating and editing an entry

`/entries/new` SHALL offer a form for account (pre-selected and, when
arrived at via `?account_id=`, not changeable in that flow; otherwise a
choice among the visitor's own non-deleted accounts), kind, amount (entered
and displayed at full stored precision, independent of the visitor's
display-rounding preference), booking timestamp, title, description, a
category (required unless kind is `balance_adjustment`), and tags.
Submitting calls `POST /api/entries`. `/entries/{id}/edit` SHALL offer the
same form pre-populated from the existing entry, with account and kind
rendered read-only (immutable per the backend), submitting an update on
save, and SHALL offer a delete action behind a confirmation step.

#### Scenario: Creating a transaction requires a category

- **WHEN** an authenticated visitor submits the create form with
  `kind: transaction` and no category selected
- **THEN** the form shows a validation error and does not submit

#### Scenario: Creating a balance adjustment allows an empty category

- **WHEN** an authenticated visitor submits the create form with
  `kind: balance_adjustment` and no category selected
- **THEN** the entry is created successfully

#### Scenario: Account and kind are not editable

- **WHEN** an authenticated visitor opens `/entries/{id}/edit`
- **THEN** the account and kind fields are shown but cannot be changed

#### Scenario: Deleting an entry requires confirmation

- **WHEN** an authenticated visitor activates "Delete" on the edit page
- **THEN** a confirmation step appears and the delete request is not sent
  until it is confirmed

### Requirement: Tags can be created inline from the entry form

The entry form's tag input SHALL match against the visitor's existing tags
as they type. On submission, any entered tag value that does not match an
existing tag SHALL be created (`POST /api/tags`) before the entry is saved
with it attached. No separate tag-management page is required to use a new
tag.

#### Scenario: Typing a new tag name creates it

- **WHEN** an authenticated visitor types a tag name that does not match
  any of their existing tags and submits the entry form
- **THEN** a new tag with that name is created and attached to the entry

#### Scenario: Typing an existing tag name reuses it

- **WHEN** an authenticated visitor types a tag name matching one of their
  existing tags and submits the entry form
- **THEN** the existing tag is attached, and no duplicate tag is created

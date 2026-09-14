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
`category_id`, `tag_id`, `kind`, a date range via the shared date-range
filter — see `web-client-date-range-filter`), a free-text search input, and
a way to sort by booking timestamp or amount in either direction. When no
entries match the current filters, the page SHALL show text distinguishing
"no entries match these filters" from "no entries exist yet."

#### Scenario: Combining filters narrows the results

- **WHEN** an authenticated visitor sets an account filter, a category
  filter, and a date range together
- **THEN** only entries matching all three narrow the results

#### Scenario: No matches under the current filters

- **WHEN** the current filters match no entries but entries exist on the
  account
- **THEN** the page shows text indicating no entries match the current
  filters, not that none exist at all

### Requirement: The entry ledger defaults to the last 2 weeks

When `/entries` is opened with no date-range parameter (`range`, `from`, or
`to`) present in the URL, the entry ledger SHALL apply the "Last 2 weeks"
preset as its effective date filter, without writing that preset into the
URL. The date-range filter's dropdown SHALL show "Last 2 weeks" selected in
this state.

#### Scenario: Opening the ledger with no filters applies a two-week default

- **WHEN** an authenticated visitor opens `/entries` with no `range`,
  `from`, or `to` parameter
- **THEN** the list shows only entries within the last 2 weeks, the
  date-range filter shows "Last 2 weeks" selected, and the URL is not
  modified

#### Scenario: An explicit date filter overrides the default

- **WHEN** an authenticated visitor opens `/entries?range=this_month`
- **THEN** the list shows entries within the current month instead of the
  last-2-weeks default

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
`balance_adjustment`), and tags. When kind is `transaction`, the form
SHALL additionally offer a `counterparty` text input (suggested values
from `GET /api/entries/counterparties` via a `<datalist>`) and a
`location` field (a plain text input, editable directly, alongside a "Use
GPS" button and a "Pick on map" button — see the location requirements
below); both are hidden when kind is `balance_adjustment`, mirroring how
category is treated. Submitting calls `POST /api/entries`.
`/entries/{id}/edit` SHALL offer the same form pre-populated from the
existing entry, with kind rendered read-only (immutable per the backend)
and account rendered locked by default, changeable only through the
unlock interaction described below, submitting an update on save. The edit
page SHALL offer its full form, including the delete action behind a
confirmation step, only to a visitor with `entry_admin`+ permission on the
entry's account, or with `append` permission and `created_by` matching
them; a visitor with `view` permission, or `append` permission on an entry
they did not create, SHALL instead see the entry's details as read-only,
with no edit or delete action.

#### Scenario: Creating a transaction requires a category

- **WHEN** an authenticated visitor submits the create form with
  `kind: transaction` and no category selected
- **THEN** the form shows a validation error and does not submit

#### Scenario: Creating a balance adjustment allows an empty category

- **WHEN** an authenticated visitor submits the create form with
  `kind: balance_adjustment` and no category selected
- **THEN** the entry is created successfully

#### Scenario: Counterparty and location are hidden for a balance adjustment

- **WHEN** an authenticated visitor selects `kind: balance_adjustment` on
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

### Requirement: A linked entry shows a badge to its recurring transaction in the ledger

`/entries` SHALL show a small icon/badge on any entry whose
`recurring_transaction_id` is set, distinct from the row's other content,
that navigates to that recurring transaction's edit page when activated.
An entry with no `recurring_transaction_id` SHALL show no such badge.

#### Scenario: Linked entry shows the badge

- **WHEN** the ledger lists an entry with a non-null
  `recurring_transaction_id`
- **THEN** its row shows the recurring-transaction badge

#### Scenario: Badge links to the recurring transaction

- **WHEN** the visitor activates a linked entry's badge
- **THEN** the browser navigates to that recurring transaction's edit page

#### Scenario: Unlinked entry shows no badge

- **WHEN** the ledger lists an entry with a null `recurring_transaction_id`
- **THEN** no recurring-transaction badge is shown on its row

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

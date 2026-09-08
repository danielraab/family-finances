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
choice among the visitor's own non-deleted accounts), kind, amount (entered
and displayed at full stored precision, independent of the visitor's
display-rounding preference), booking timestamp, title, description, a
category (required unless kind is `balance_adjustment`), and tags.
Submitting calls `POST /api/entries`. `/entries/{id}/edit` SHALL offer the
same form pre-populated from the existing entry, with kind rendered
read-only (immutable per the backend) and account rendered locked by
default, changeable only through the unlock interaction described below,
submitting an update on save, and SHALL offer a delete action behind a
confirmation step.

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
- **THEN** the account and kind fields are shown but cannot be changed —
  the account field's unlock interaction (see below) is a separate,
  deliberate action, not a default-editable state

#### Scenario: Deleting an entry requires confirmation

- **WHEN** an authenticated visitor activates "Delete" on the edit page
- **THEN** a confirmation step appears and the delete request is not sent
  until it is confirmed

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
exclude disabled categories from the choices offered when picking a
category. If the entry being edited currently holds a category that has
since been disabled, that category SHALL still render as the field's
current, selected value (labeled distinctly, e.g. as disabled) rather than
disappearing — but it SHALL NOT appear as a choosable option, so it cannot
be re-selected once cleared. Leaving the field untouched and saving the
rest of the form SHALL succeed normally; the entry is not forced to change
its category just because that category was disabled.

#### Scenario: Creating an entry only offers live categories

- **WHEN** an authenticated visitor opens the new-entry form
- **THEN** the category picker lists only non-disabled categories

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

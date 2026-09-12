## MODIFIED Requirements

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

## ADDED Requirements

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

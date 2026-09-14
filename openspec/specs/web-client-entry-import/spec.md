# web-client-entry-import Specification

## Purpose

The authenticated `/entries/import` wizard: a client-side CSV/JSON entry
importer that walks the visitor through choosing an account, selecting a
file, mapping its columns/fields to entry fields, reviewing a client-side
dry run with per-row remapping, and running the import with live progress
and a results summary. See `web-client-accounts` and `web-client-entries`
for the entry points that link into this wizard, and `account-entries` for
the backend capability entries are created against.

## Requirements

### Requirement: Import routes require authentication

`/entries/import` SHALL be accessible only to an authenticated visitor. An
anonymous visitor navigating to it SHALL be redirected to `/login`.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/entries/import`
- **THEN** the client redirects them to `/login`

### Requirement: The account is chosen first, optionally preset

`/entries/import` SHALL first ask the visitor to choose the target account
among their own non-deleted accounts and every non-deleted shared account
they hold at least `append` permission on — the same set
`/entries/new` offers. Arriving via `/entries/import?account_id={id}` SHALL
preset and lock that choice, the same convention `/entries/new`'s
`?account_id=` already uses. No later step SHALL allow changing the
account without starting over.

#### Scenario: Choosing an account manually

- **WHEN** an authenticated visitor opens `/entries/import` with no
  `account_id` and selects one of their `append`+ accounts
- **THEN** the wizard proceeds to the file-selection step for that account

#### Scenario: Arriving with a preset account

- **WHEN** an authenticated visitor opens
  `/entries/import?account_id={id}` for an account they hold `append`+
  permission on
- **THEN** the account step is skipped, that account is shown as fixed,
  and the wizard proceeds directly to file selection

#### Scenario: A view-only account is not offered

- **WHEN** an authenticated visitor with only `view` permission on an
  account opens `/entries/import`
- **THEN** that account does not appear among the choices

### Requirement: A CSV or JSON file is selected and parsed client-side

The file-selection step SHALL let the visitor pick a single file, parsed
entirely in the browser — no file content is uploaded until the import
step creates entries from it. The file picker SHALL NOT restrict which
files are selectable by type (no `accept` filter): a picked file whose
name does not end in `.csv` or `.json` (case-insensitive) SHALL be
rejected immediately with an on-screen error, before any parse is
attempted, and SHALL NOT advance the wizard. The step SHALL display, as
persistent on-screen guidance (not only in an error state), that a CSV
file's first row must be column headings. A `.csv`-named file SHALL be
parsed with its first row consumed as headings. A `.json`-named file
SHALL be parsed only when its top-level value is an array of objects;
any other top-level shape (a bare object, or an array of non-objects)
SHALL be rejected with an on-screen error and SHALL NOT advance the
wizard.

#### Scenario: Selecting a valid CSV file

- **WHEN** an authenticated visitor selects a `.csv` file whose first row
  is column headings
- **THEN** the wizard advances to the mapping step, offering each header
  as a mappable column

#### Scenario: Selecting a valid JSON file

- **WHEN** an authenticated visitor selects a `.json` file whose content
  is a top-level array of objects
- **THEN** the wizard advances to the mapping step, offering the union of
  the objects' mappable keys as mappable fields (see the next requirement
  for which keys are mappable)

#### Scenario: Selecting a JSON file with the wrong top-level shape

- **WHEN** an authenticated visitor selects a `.json` file whose top-level
  value is not an array of objects (for example, an object wrapping the
  array, or an array of plain strings)
- **THEN** the wizard shows an error explaining the expected shape and
  does not advance past file selection

#### Scenario: Selecting a file with an unsupported extension

- **WHEN** an authenticated visitor picks a file whose name does not end
  in `.csv` or `.json`
- **THEN** the wizard shows an error asking for a `.csv` or `.json` file,
  without attempting to parse it, and does not advance past file
  selection

#### Scenario: The file picker is not filtered by type

- **WHEN** an authenticated visitor opens the file-selection step's native
  file chooser
- **THEN** every file is selectable, regardless of extension or reported
  type — filtering happens after selection, not in the chooser itself
  (needed for reliable `.csv` selection on Android, where some file
  providers hide files that don't match the chooser's requested type)

#### Scenario: The CSV header note is always visible

- **WHEN** an authenticated visitor reaches the file-selection step
- **THEN** a note stating that the first row of a CSV file must contain
  column headings is shown, regardless of whether a file has been
  selected yet

### Requirement: A structured amount field is recognized and converted; other complex fields are skipped, not fatal

Within a JSON file's objects, a top-level field whose value is a nested
object shaped `{ value: number, precision: number, currency?: string }`
(recognized by shape, under any key name) SHALL be treated as a
structured amount: converted to a plain decimal value (`value` divided by
`10^precision`) and offered as a mappable field under its original key,
exactly as if the source had held that decimal directly. When the object
also carries `currency`, it SHALL additionally be offered as a mappable
field named `<key>.currency`. A top-level field whose value is any other
nested object, or an array, SHALL NOT be offered as a mappable field and
SHALL NOT cause the file to be rejected; every such field name (across
all of the file's objects) SHALL instead be collected and shown as a
single, non-blocking hint on the mapping step, indicating that some
fields could not be mapped.

#### Scenario: A structured amount field is usable like a plain one

- **WHEN** a `.json` file's objects carry a field shaped
  `{"value": -3595, "precision": 2, "currency": "EUR"}`
- **THEN** that field is offered as a mappable field, and mapping it as
  the amount column produces the decimal amount `-35.95`

#### Scenario: An unrecognized nested field is skipped with a hint, not a rejection

- **WHEN** a `.json` file's objects carry a field whose value is a nested
  object that is not a `{value, precision}`-shaped amount, or a field
  whose value is an array
- **THEN** the file still parses and advances to the mapping step; that
  field is absent from the mappable fields, and its name appears in a
  hint on the mapping step noting that some fields could not be mapped

#### Scenario: The ignored-fields hint lists every skipped field once

- **WHEN** a dry run's file has multiple unmappable fields across its
  objects
- **THEN** the mapping step's hint lists each distinct field name once,
  not once per row

### Requirement: Source columns/fields are mapped to entry fields

The mapping step SHALL let the visitor map, for each importable entry
field, a source column (CSV) or field (JSON) detected from the selected
file, or leave it unmapped where the entry field is optional. Title,
amount, and booking timestamp SHALL be required mappings; description,
counterparty, and location SHALL be optional mappings. Each source
column/field offered in a mapping control SHALL be shown together with an
example value for that column/field drawn from the selected file (the
first row that has a non-empty value for it), so the visitor can tell
what a column contains without leaving the mapping step to inspect the
raw file.

#### Scenario: Required fields must be mapped before proceeding

- **WHEN** an authenticated visitor has not mapped a source column/field
  to title, amount, or booking timestamp
- **THEN** the mapping step's "Continue" action is disabled and the
  unmapped required field(s) are indicated

#### Scenario: Optional fields may be left unmapped

- **WHEN** an authenticated visitor leaves description, counterparty, and
  location unmapped
- **THEN** the dry run and subsequent import proceed, creating entries
  with those fields empty

#### Scenario: A column's example value is shown alongside its name

- **WHEN** an authenticated visitor opens a mapping control (for example,
  the title mapping) after selecting a file
- **THEN** each offered source column/field is shown with an example
  value taken from the file, not just its bare name

### Requirement: Category and tags are chosen once for the whole batch

The mapping step SHALL offer exactly one category picker (from the
visitor's own usable categories and any `append`-tier shared category,
excluding disabled ones — the same set `/entries/new` offers) and one tag
input (existing-tag suggestions plus free-text entry, matching
`/entries/new`'s inline tag creation), each applying to every entry the
import creates. There SHALL be no way to map a source column/field to
category or to tags on a per-row basis.

#### Scenario: Every imported entry gets the same category

- **WHEN** an authenticated visitor selects a category in the mapping step
  and completes an import of multiple rows
- **THEN** every created entry has that same `category_id`

#### Scenario: Every imported entry gets the same tag set

- **WHEN** an authenticated visitor adds two tags (one new, one existing)
  in the mapping step and completes an import
- **THEN** every created entry carries both tags, and the new tag is
  created exactly once, not once per row

#### Scenario: A category is required before continuing to the dry run

- **WHEN** an authenticated visitor has not selected a category
- **THEN** the mapping step's "Continue" action is disabled

### Requirement: Amount mapping supports a single signed column or a split debit/credit pair

The mapping step's amount section SHALL offer two mutually exclusive
modes: a single source column holding a signed amount, or two source
columns (debit, credit) combined as `credit − debit`. The single-column
mode SHALL offer a "flip sign" option. Both modes SHALL offer a decimal
separator setting (auto-detect, `.`, or `,`) and a thousands separator
setting (auto-detect, none, `,`, `.`, space, or `'`), applied when parsing
the mapped column(s).

#### Scenario: Single signed column with a comma decimal separator

- **WHEN** an authenticated visitor maps a single amount column containing
  values like `"1.234,56"` and sets decimal separator to `,` and thousands
  separator to `.`
- **THEN** the dry run parses that value as `1234.56` in the account's
  currency

#### Scenario: Split debit/credit columns

- **WHEN** an authenticated visitor maps separate debit and credit
  columns instead of a single amount column
- **THEN** each row's amount is computed as its credit value minus its
  debit value, treating a blank cell in either column as zero

#### Scenario: Flipping the sign

- **WHEN** an authenticated visitor enables "flip sign" in single-column
  mode
- **THEN** every row's parsed amount is negated relative to what it would
  otherwise be

### Requirement: Date mapping auto-detects, with an explicit format override

The mapping step's booking-timestamp section SHALL default to
auto-detecting the mapped column's date format. The visitor SHALL be able
to instead supply an explicit format string (using tokens `YYYY`, `MM`,
`DD`, `HH`, `mm`, `ss`), which SHALL then be used instead of
auto-detection for every row.

#### Scenario: Auto-detection parses an ISO-shaped date

- **WHEN** an authenticated visitor maps a column of `YYYY-MM-DD`-shaped
  values with date format left at "auto"
- **THEN** the dry run parses every row's date correctly

#### Scenario: An explicit format overrides auto-detection

- **WHEN** an authenticated visitor's mapped date column holds values like
  `03.04.2026` and they set an explicit format of `DD.MM.YYYY`
- **THEN** the dry run parses every row using that format instead of
  guessing

### Requirement: The dry run is a distinct wizard step, computed on entry

The dry run SHALL be its own step, reached from the mapping step's
"Continue" action (enabled only once every required field is mapped and a
category is selected) and entered before the import step. On entering
this step, the wizard SHALL parse and validate every row in the file
against the mapping produced by the mapping step, entirely client-side,
with no request sent to the backend, with no separate manual trigger
needed. Each row SHALL be classified as one of: **failed** (a required
mapped field is missing or does not parse — bad amount, bad date, blank
title), **suspicious** (parses successfully but looks likely to be wrong —
an amount of exactly zero, or a date ambiguous between day-first and
month-first orderings while date format is "auto"), or **ok** (parses
successfully and is not suspicious). The dry-run results SHALL show the
total row count and counts per classification, and a list of every row in
the file — **ok** included, not only failed/suspicious ones — each
showing its row number and classification. There is no separate
single-row mapping preview outside this step; its own per-row list is how
the visitor checks mapped results (see the following requirements).

A visitor SHALL be able to hide **ok**-classified rows from the list via a
toggle, to focus on the rows that need attention in a large file; toggling
it back SHALL restore them. Hiding **ok** rows SHALL NOT change the
summary counts (still reflecting every row) or which rows are submitted
if the visitor proceeds — it only affects which rows are listed.

#### Scenario: Continuing from mapping enters the dry-run step with results already shown

- **WHEN** an authenticated visitor completes the mapping step and selects
  "Continue"
- **THEN** the wizard advances to the dry-run step and the classified
  results are already present, with no separate action needed to compute
  them

#### Scenario: A file with no problems still lists every row

- **WHEN** an authenticated visitor reaches the dry-run step for a file
  where every row maps and parses cleanly
- **THEN** the results list every row, each classified ok

#### Scenario: A failed row is listed with its reason

- **WHEN** a row's mapped amount column contains a value that does not
  parse as a number under the current separator settings
- **THEN** the dry run lists that row as failed, with the reason, and it
  is excluded from the count of rows that will be imported

#### Scenario: A suspicious row is listed but still counted for import

- **WHEN** a row's mapped amount evaluates to exactly zero
- **THEN** the dry run lists that row as suspicious, with the reason, and
  it is still counted among the rows that will be imported if the visitor
  proceeds

#### Scenario: Hiding successful rows narrows the list to what needs attention

- **WHEN** an authenticated visitor on the dry-run step enables "Hide
  successful rows"
- **THEN** every row classified ok is removed from the list while
  suspicious and failed rows remain, and the summary counts stay
  unchanged

#### Scenario: Un-hiding restores the full row list

- **WHEN** an authenticated visitor disables "Hide successful rows" after
  enabling it
- **THEN** every ok row reappears in the list in its original position

### Requirement: Clicking a dry-run row reveals its source data and mapped result

Each row in the dry-run results SHALL be clickable (independently of every
other row) to reveal, inline, that row's raw source values for each
mapped column/field and — for a row classified ok or suspicious — the
entry it would create (title, amount, booking timestamp, and any mapped
optional fields), rendered the same way a manually-created entry's fields
would read. A failed row's expanded detail SHALL show its source values
but has no mapped entry to show, since none was produced. Expanding one
row SHALL NOT affect whether any other row is expanded. Re-running the
dry run SHALL collapse every row back to its unexpanded state.

#### Scenario: Expanding an ok row shows its mapped result

- **WHEN** an authenticated visitor clicks a row classified ok in the
  dry-run results
- **THEN** that row expands to show its source values and the entry it
  would create

#### Scenario: Expanding a failed row shows only its source data

- **WHEN** an authenticated visitor clicks a row classified failed
- **THEN** that row expands to show its source values, with no mapped
  entry shown

#### Scenario: Multiple rows can be expanded independently

- **WHEN** an authenticated visitor expands two different rows
- **THEN** both remain expanded at once, and collapsing one leaves the
  other expanded

#### Scenario: Returning to the dry-run step fresh collapses every row

- **WHEN** an authenticated visitor has a row expanded, goes back to the
  mapping step, and continues forward to the dry-run step again
- **THEN** the new results start with every row collapsed

### Requirement: A failed row's individual fields can be remapped to a different column, scoped to that row alone

A row classified **failed** SHALL, in its expanded detail, offer a remap
control for each field that caused the failure (title, amount — one
control in single-column mode or two, debit and credit, in split mode —
or booking date), each defaulting to the column the mapping step currently
uses for that field and offering every column/field detected in the file.
Changing a remap control SHALL immediately re-classify that row alone —
against the same amount-separator settings resolved for the rest of the
file — updating its classification, reason, and (once it succeeds) mapped
entry in place; no other row's classification, and no file-wide mapping
setting, SHALL be affected. The row-level counts and totals SHALL reflect
the current classification of every row, remaps included. A row that is
not classified failed SHALL NOT offer a remap control — an ok or
suspicious row already produced a valid entry. Each remap control's
options SHALL show an example value next to the column name, same as the
mapping step's own column pickers, but sourced from *that row's own*
value for the column rather than a file-wide example — a different row's
value would misrepresent what this specific failing row actually
contains.

#### Scenario: A remap control's example values come from the row being remapped

- **WHEN** an authenticated visitor expands a row classified failed and
  opens one of its remap controls
- **THEN** each option shows that row's own value for the corresponding
  column, not any other row's value for that column

#### Scenario: Remapping a failed row's date fixes only that row

- **WHEN** an authenticated visitor expands a row classified failed for an
  unparseable date and selects a different column in its booking-date
  remap control, whose value parses successfully
- **THEN** that row's classification changes to ok (or suspicious, if the
  parsed value is itself flagged), its mapped entry is shown, and the
  dry-run counts update to reflect the change

#### Scenario: A remap does not affect other rows

- **WHEN** an authenticated visitor remaps a failed row's amount column
- **THEN** every other row's classification and mapped entry stay exactly
  as they were before the remap

#### Scenario: A remapped row is submitted using its remapped values

- **WHEN** an authenticated visitor remaps a failed row into a successful
  classification and proceeds to import
- **THEN** the entry created for that row reflects the remapped column's
  value, not the original mapping's

#### Scenario: Split amount mode offers both debit and credit remap controls

- **WHEN** an authenticated visitor expands a row classified failed for an
  invalid amount while the mapping uses split debit/credit columns
- **THEN** the expanded detail offers a remap control for both the debit
  column and the credit column

#### Scenario: A row with no failure offers no remap control

- **WHEN** an authenticated visitor expands a row classified ok or
  suspicious
- **THEN** no remap control is shown for that row

### Requirement: The mapping used for the whole file can be revised by returning to the mapping step

The visitor SHALL be able to go back from the dry-run step to the mapping
step, change any mapping or parsing setting there (including date format
and amount separators), and continue forward again to get a fresh dry run
reflecting the new settings, at no point requiring a network request or
losing the selected file. This is separate from, and does not require,
the per-row remap capability above — a change here re-validates the whole
file, not one row.

#### Scenario: Adjusting the date format after a failed dry run

- **WHEN** a dry run reports date-parsing failures and the visitor goes
  back to the mapping step, sets an explicit date format, and continues
  forward again
- **THEN** the new dry-run results reflect the updated format without
  re-selecting the file

### Requirement: The browser's Back button steps back one wizard step without losing progress

The account, file, mapping, and dry-run steps SHALL each correspond to a
distinct browser history entry, so that pressing the browser's own Back
button steps back exactly one wizard step — matching the in-page "Back"
link's behavior exactly — rather than navigating away from the import
wizard. Every setting already entered (the selected account, the parsed
file, the column mapping and parsing settings, any per-row remaps applied
in the dry-run step) SHALL remain intact after stepping back and then
forward again. The run and result steps SHALL NOT be reachable via
back/forward navigation: once an import run has started or finished,
pressing Back SHALL return to the dry-run step's ordinary view (not a
stale run/result screen) and SHALL NOT resubmit any entry.

#### Scenario: The browser Back button steps back one wizard step

- **WHEN** an authenticated visitor has advanced through the account,
  file, mapping, and dry-run steps and presses the browser's Back button
- **THEN** the wizard shows the mapping step, with the column mapping and
  parsing settings exactly as they were left

#### Scenario: Stepping back and forward again preserves every setting

- **WHEN** an authenticated visitor presses Back from the dry-run step to
  the mapping step, then presses Forward again
- **THEN** the dry-run step is shown with the same classified rows,
  including any per-row remaps applied before navigating back

#### Scenario: Back does not leave the import wizard

- **WHEN** an authenticated visitor on the file, mapping, or dry-run step
  (having arrived at `/entries/import` without a preset account) presses
  the browser's Back button
- **THEN** the wizard shows the previous wizard step, not a different page

#### Scenario: Back after a completed import does not resubmit entries

- **WHEN** an authenticated visitor completes an import run and then
  presses the browser's Back button
- **THEN** the wizard returns to the dry-run step's ordinary view and no
  additional `POST /api/entries` request is sent

### Requirement: Import creates entries sequentially with live progress and can be canceled

Once a dry run has completed, the visitor SHALL be able to start the
import. The import SHALL create one entry per non-failed row (both **ok**
and **suspicious** classifications), in file order, via one
`POST /api/entries` request per row; every entry created this way SHALL
have `kind: "transaction"`. Any tag names entered in the mapping step SHALL
be resolved to tag ids once, before the first row is submitted — reusing
an existing tag by name, or creating it, exactly once regardless of row
count. Progress (rows completed out of the total to be imported) SHALL be
shown live as the import proceeds. The visitor SHALL be able to cancel the
import; canceling stops further submissions but does not undo entries
already created.

#### Scenario: Only non-failed rows are submitted

- **WHEN** a dry run classified 3 of 100 rows as failed and the visitor
  starts the import
- **THEN** 97 `POST /api/entries` requests are made, one for each
  non-failed row

#### Scenario: A new tag is created once, not once per row

- **WHEN** the mapping step's tag input includes a tag name that does not
  match any existing tag, and the import creates 50 entries
- **THEN** exactly one `POST /api/tags` request is made for that name,
  and all 50 entries carry the resulting tag id

#### Scenario: Progress is shown during the import

- **WHEN** an import of many rows is in progress
- **THEN** the visitor sees how many rows have been processed out of the
  total

#### Scenario: Canceling stops further submissions

- **WHEN** an authenticated visitor cancels an import partway through
- **THEN** no further `POST /api/entries` requests are made, and the
  entries already created remain

### Requirement: A row rejected during import is skipped, not fatal

A row whose `POST /api/entries` call fails during the import step SHALL
be recorded as failed with the backend's reason and SHALL NOT stop the
import from continuing with the remaining rows. "The backend's reason"
means the actual error message the rejection response carried, not only a
generic "rejected" label — the results view (see below) SHALL show it
alongside the generic reason so the cause of a rejection that client-side
validation could never have caught (a category disabled moments earlier;
an account no longer accessible) is visible without inspecting network
traffic.

#### Scenario: One row's backend rejection does not stop the import

- **WHEN** one row is rejected by `POST /api/entries` during an import of
  many rows (for example, a race where the selected category was disabled
  moments earlier)
- **THEN** that row is recorded as failed and every subsequent row is
  still submitted

#### Scenario: A rejected row's result shows the backend's actual error message

- **WHEN** a row is rejected by `POST /api/entries` with an error response
  carrying a message (for example, `"invalid value"` for a category that
  was disabled after the mapping step)
- **THEN** the results view's reason for that row includes that message,
  not only the generic "rejected by the server" label

### Requirement: The results view reports created and failed counts, without listing every success

When the import finishes (completes or is canceled), the visitor SHALL see
the number of entries successfully created and a list of every failed row
(from the dry run's failed classification and any rows rejected during
the import step) with its reason. Successfully created rows SHALL NOT be
listed individually.

#### Scenario: A completed import's results

- **WHEN** an import finishes having created 297 entries and failed on 3
  rows (combining dry-run failures and any rejected during submission)
- **THEN** the results show "297 created" and list the 3 failed rows with
  their reasons, without listing the 297 successes

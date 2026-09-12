## ADDED Requirements

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

The file-selection step SHALL accept a single `.csv` or `.json` file,
parsed entirely in the browser — no file content is uploaded until the
import step creates entries from it. The step SHALL display, as
persistent on-screen guidance (not only in an error state), that a CSV
file's first row must be column headings. A CSV file SHALL be parsed with
its first row consumed as headings. A JSON file SHALL be parsed only when
its top-level value is an array of flat objects (string/number/boolean/
null values); any other top-level shape SHALL be rejected with an
on-screen error and SHALL NOT advance the wizard.

#### Scenario: Selecting a valid CSV file

- **WHEN** an authenticated visitor selects a `.csv` file whose first row
  is column headings
- **THEN** the wizard advances to the mapping step, offering each header
  as a mappable column

#### Scenario: Selecting a valid JSON file

- **WHEN** an authenticated visitor selects a `.json` file whose content
  is a top-level array of flat objects
- **THEN** the wizard advances to the mapping step, offering the union of
  the objects' keys as mappable fields

#### Scenario: Selecting a JSON file with the wrong shape

- **WHEN** an authenticated visitor selects a `.json` file whose top-level
  value is not an array of flat objects (for example, an object wrapping
  the array, or an array containing nested objects)
- **THEN** the wizard shows an error explaining the expected shape and
  does not advance past file selection

#### Scenario: The CSV header note is always visible

- **WHEN** an authenticated visitor reaches the file-selection step
- **THEN** a note stating that the first row of a CSV file must contain
  column headings is shown, regardless of whether a file has been
  selected yet

### Requirement: Source columns/fields are mapped to entry fields

The mapping step SHALL let the visitor map, for each importable entry
field, a source column (CSV) or field (JSON) detected from the selected
file, or leave it unmapped where the entry field is optional. Title,
amount, and booking timestamp SHALL be required mappings; description,
counterparty, and location SHALL be optional mappings.

#### Scenario: Required fields must be mapped before proceeding

- **WHEN** an authenticated visitor has not mapped a source column/field
  to title, amount, or booking timestamp
- **THEN** the dry run cannot be run and the unmapped required field(s)
  are indicated

#### Scenario: Optional fields may be left unmapped

- **WHEN** an authenticated visitor leaves description, counterparty, and
  location unmapped
- **THEN** the dry run and subsequent import proceed, creating entries
  with those fields empty

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

#### Scenario: A category is required to run the dry run

- **WHEN** an authenticated visitor has not selected a category
- **THEN** the dry run cannot be run

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

### Requirement: A mapping preview shows one row's result

The mapping step SHALL show, alongside the mapping controls, the result of
applying the current mapping and settings to one row from the selected
file (rendered the same way a manually-created entry's fields would read),
updating live as the mapping changes.

#### Scenario: The preview updates as the mapping changes

- **WHEN** an authenticated visitor changes which source column is mapped
  to title
- **THEN** the preview immediately reflects the new value it would produce

### Requirement: A dry run validates every row locally before any entry is created

The visitor SHALL be able to trigger a dry run once every required field
is mapped and a category is selected. The dry run SHALL parse and validate
every row in the file against the current mapping and settings, entirely
client-side, with no request sent to the backend. Each row SHALL be
classified as one of: **failed** (a required mapped field is missing or
does not parse — bad amount, bad date, blank title), **suspicious**
(parses successfully but looks likely to be wrong — an amount of exactly
zero, or a date ambiguous between day-first and month-first orderings
while date format is "auto"), or **ok** (parses successfully and is not
suspicious). The dry-run results SHALL show, at minimum, the total row
count and counts per classification, and a list of every failed and
suspicious row with its reason; **ok** rows SHALL NOT be listed
individually.

#### Scenario: A file with no problems

- **WHEN** an authenticated visitor runs the dry run on a file where every
  row maps and parses cleanly
- **THEN** the results show every row as ok, with no failed/suspicious
  rows listed

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

#### Scenario: Successful rows are not listed one by one

- **WHEN** a dry run completes with 300 ok rows and 2 failed rows
- **THEN** the results show a count of 300 and the reasons for the 2
  failed rows, without listing the 300 ok rows

### Requirement: Mapping and parsing settings can be adjusted and re-validated without leaving the wizard

The visitor SHALL be able to change any mapping or parsing setting
(including date format and amount separators) and re-run the dry run as
many times as needed, with each re-run reflecting the current settings, at
no point requiring a network request or losing the selected file.

#### Scenario: Adjusting the date format after a failed dry run

- **WHEN** a dry run reports date-parsing failures and the visitor sets an
  explicit date format and re-runs the dry run
- **THEN** the new results reflect the updated format without re-selecting
  the file

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
import from continuing with the remaining rows.

#### Scenario: One row's backend rejection does not stop the import

- **WHEN** one row is rejected by `POST /api/entries` during an import of
  many rows (for example, a race where the selected category was disabled
  moments earlier)
- **THEN** that row is recorded as failed and every subsequent row is
  still submitted

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

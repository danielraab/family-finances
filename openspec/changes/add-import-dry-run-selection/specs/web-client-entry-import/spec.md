## ADDED Requirements

### Requirement: Dry-run rows can be selected with checkboxes

The dry-run results list SHALL carry a checkbox on every listed row, plus
a select-all checkbox in the list's header row, so that any subset of the
file's rows can be selected. Toggling a row's checkbox SHALL NOT expand or
collapse that row, and expanding a row SHALL NOT change whether it is
selected.

The header checkbox SHALL reflect the **currently listed** rows: checked
when every listed row is selected, indeterminate when only some are, and
unchecked otherwise. Pressing it SHALL select every listed row, except
when every listed row is already selected, in which case it SHALL clear
the entire selection.

The current number of selected rows SHALL be shown above the list, with an
action that clears the selection outright. Hiding **ok** rows via "Hide
successful rows" SHALL NOT deselect them — the count and everything acting
on the selection SHALL keep counting a selected row that is currently
hidden.

#### Scenario: Selecting rows individually

- **WHEN** an authenticated visitor on the dry-run step checks three rows'
  checkboxes
- **THEN** all three are shown as selected and the list reports three rows
  selected

#### Scenario: Selecting a row does not expand it

- **WHEN** an authenticated visitor checks a row's checkbox
- **THEN** that row's source data and mapped result stay collapsed

#### Scenario: Expanding a row does not select it

- **WHEN** an authenticated visitor clicks a row to expand it
- **THEN** the row's checkbox stays unchecked and the selected count is
  unchanged

#### Scenario: Select-all covers the listed rows

- **WHEN** an authenticated visitor presses the header checkbox with no
  rows selected
- **THEN** every row currently listed becomes selected

#### Scenario: Select-all respects the hide-successful toggle

- **WHEN** an authenticated visitor enables "Hide successful rows" and
  then presses the header checkbox
- **THEN** only the listed (suspicious and failed) rows become selected,
  and no ok row does

#### Scenario: A selected row stays selected while hidden

- **WHEN** an authenticated visitor selects an ok row and then enables
  "Hide successful rows"
- **THEN** the row leaves the list but the selected count still counts it,
  and disabling the toggle shows it still selected

#### Scenario: Clearing the selection

- **WHEN** an authenticated visitor with rows selected uses the clear
  action
- **THEN** no row is selected and the selected count is zero

### Requirement: A remap can be applied to every selected row at once

Whenever at least one row is selected, the dry-run step SHALL offer a bulk
remap panel above the list, carrying one remap control per remappable
field — title, amount (one control in single-column mode, or debit and
credit in split mode), and booking date — each offering every column/field
detected in the file, alongside a file-wide example value for it (the
first non-empty value anywhere in the file, as the mapping step's own
pickers show), and each defaulting to an explicit "keep current" option.

The panel SHALL apply on an explicit action naming how many rows it will
affect. Applying SHALL write every control the visitor changed — and only
those — as a per-row override on **every selected row**, whatever its
current classification, re-classifying each against the same file-resolved
amount separators the dry run used, exactly as the per-row remap control
does for a single row. A field left at "keep current" SHALL leave that
field's mapping untouched on every selected row, and an unselected row
SHALL NOT be affected. The file-wide mapping SHALL NOT be changed. After
applying, every control SHALL return to "keep current", since the selected
rows no longer share a single resting value to display.

The counts and totals SHALL reflect the new classification of every
affected row, and a row whose bulk remap makes it invalid SHALL become
failed in place, the same as any other remap.

#### Scenario: The panel appears only with a selection

- **WHEN** an authenticated visitor on the dry-run step has no row
  selected
- **THEN** no bulk remap panel is shown; selecting a row reveals it

#### Scenario: Remapping many rows in one action

- **WHEN** an authenticated visitor selects forty rows that failed for a
  missing title, picks a different title column in the bulk panel, and
  applies it
- **THEN** all forty rows are re-classified at once against that column,
  their classifications and mapped entries update in place, and the
  summary counts reflect the change

#### Scenario: Only the changed controls are applied

- **WHEN** an authenticated visitor changes only the booking-date control
  and applies it to the selection
- **THEN** every selected row's booking date is remapped and no selected
  row's title or amount mapping changes

#### Scenario: Unselected rows are untouched

- **WHEN** an authenticated visitor applies a bulk remap to a selection
  that excludes a given row
- **THEN** that row's classification, reason and mapped entry stay exactly
  as they were

#### Scenario: A bulk remap applies to an ok row too

- **WHEN** an authenticated visitor selects a row classified ok along with
  several failed rows and applies a bulk title remap
- **THEN** the ok row's title is remapped as well, and it is re-classified
  like the rest

#### Scenario: A bulk remap into invalid values fails those rows

- **WHEN** an authenticated visitor applies a bulk amount remap to a column
  whose values do not parse for the selected rows
- **THEN** those rows become failed with the amount reason, and the
  summary counts update

#### Scenario: The file-wide mapping is not changed by a bulk remap

- **WHEN** an authenticated visitor applies a bulk remap and then returns
  to the mapping step
- **THEN** the mapping step's controls show the mapping as it was, with no
  field changed by the bulk remap

#### Scenario: A bulk control's options show file-wide examples

- **WHEN** an authenticated visitor opens one of the bulk panel's remap
  controls
- **THEN** each offered column is shown with an example value drawn from
  the file, the same way the mapping step's own pickers show one

## MODIFIED Requirements

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
not classified failed SHALL NOT offer this in-row remap control — an ok or
suspicious row already produced a valid entry (a remap can still be
applied to it via the selection-driven bulk panel, which the visitor
reaches only by explicitly selecting the row). Each remap control's
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

#### Scenario: A row with no failure offers no in-row remap control

- **WHEN** an authenticated visitor expands a row classified ok or
  suspicious
- **THEN** no remap control is shown inside that row's expansion

### Requirement: Import creates entries sequentially with live progress and can be canceled

Once a dry run has completed, the visitor SHALL be able to start the
import, choosing between two actions: **Import all**, covering every row
in the file, and **Import selected**, covering only the rows currently
selected (see the selection requirement above). Each action SHALL show the
number of entries it would create — the non-failed rows within its scope —
and SHALL be unavailable when that number is zero, so a selection holding
only failed rows cannot start an empty import. The scope chosen this way
is the import's **chosen scope**; everything below applies within it.

The import SHALL create one entry per non-failed row (both **ok**
and **suspicious** classifications) in the chosen scope, in file order, via
one `POST /api/entries` request per row; every entry created this way SHALL
have `kind: "transaction"`. Any tag names entered in the mapping step SHALL
be resolved to tag ids once, before the first row is submitted — reusing
an existing tag by name, or creating it, exactly once regardless of row
count. Progress (rows completed out of the total to be imported) SHALL be
shown live as the import proceeds. The visitor SHALL be able to cancel the
import; canceling stops further submissions but does not undo entries
already created.

#### Scenario: Only non-failed rows are submitted

- **WHEN** a dry run classified 3 of 100 rows as failed and the visitor
  chooses "Import all"
- **THEN** 97 `POST /api/entries` requests are made, one for each
  non-failed row

#### Scenario: Importing only the selected rows

- **WHEN** a dry run classified 500 rows and the visitor selects 20 of
  them, of which 18 are non-failed, and chooses "Import selected"
- **THEN** 18 `POST /api/entries` requests are made — one per selected
  non-failed row — and no unselected row is submitted

#### Scenario: Each import action shows what it would create

- **WHEN** an authenticated visitor selects rows on the dry-run step
- **THEN** "Import all" shows the count of non-failed rows in the file and
  "Import selected" shows the count of non-failed rows among the selected
  ones

#### Scenario: Importing a selection of only failed rows is not offered

- **WHEN** an authenticated visitor selects rows that are all classified
  failed
- **THEN** "Import selected" is unavailable

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

### Requirement: The results view reports created and failed counts, without listing every success

When the import finishes (completes or is canceled), the visitor SHALL see
the number of entries successfully created and a list of every failed row
**within the import's chosen scope** (from the dry run's failed
classification and any rows rejected during the import step) with its
reason. A row outside the chosen scope SHALL NOT be reported as a failure —
it was never submitted because the visitor did not ask for it, not because
anything went wrong with it. Successfully created rows SHALL NOT be listed
individually.

#### Scenario: A completed import's results

- **WHEN** an import started with "Import all" finishes having created 297
  entries and failed on 3 rows (combining dry-run failures and any
  rejected during submission)
- **THEN** the results show "297 created" and list the 3 failed rows with
  their reasons, without listing the 297 successes

#### Scenario: A partial import reports only its own scope

- **WHEN** a visitor selects 20 rows out of a file whose dry run failed on
  30 rows — 2 of them within the selection — and chooses "Import selected"
- **THEN** the results list those 2 failed rows (plus any rejected during
  submission) and none of the 28 failed rows outside the selection

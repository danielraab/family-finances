# Design — dry-run selection, bulk remap, partial import

## One selection, two uses

The ask names two features — checkboxes for remapping several rows
together, and choosing part of the file to import — and they could have
been two independent selections (a "remap these" set and an "import these"
set). They are one set here, for two reasons:

- Two checkbox columns on the same row list is unreadable, and the
  alternative (a mode switch above the table) makes the visitor state an
  intent before they know which rows they care about.
- The workflow actually runs both ways round in sequence: check the forty
  rows that fail the same way, fix them together, then — still with them
  checked — import just those. A shared set makes that two clicks; two sets
  would make it two selections of the same forty rows.

The cost is that a selection made for a remap is still standing when the
visitor reaches the buttons. That is why **Import selected** carries its
count in its label and sits beside a full-strength **Import all**: the
button says how many rows it is about to submit before it is pressed.

## Selection is by row index, and survives hiding

Selection is a `Set<number>` of `ClassifiedRow.index` — the row's position
in the file, stable across re-classification, which is what the override
map is already keyed by. It is deliberately *not* filtered when "Hide
successful rows" is toggled: hiding is a viewing aid, and silently
dropping rows from a selection because they scrolled out of view would make
"Import selected (12)" mean something different from what the visitor
checked.

The header checkbox works on the **listed** rows, matching what the
visitor can see: it is checked when every listed row is selected,
indeterminate when some are, and selects every listed row when pressed.
Pressing it while every listed row is already selected clears the **whole**
selection, hidden rows included — the one case where acting on the listed
rows alone would leave a "0 of 12 listed rows selected, 3 rows selected"
contradiction on screen. The count line above the table always reports the
true total and offers an explicit clear, so the two can't drift.

## Bulk remap applies to whatever is selected, not to failed rows only

The per-row remap control is offered only on a **failed** row, and the
reasoning holds there: an ok row already produced a valid entry, and its
expansion has nothing to fix. The bulk panel does not inherit that rule. It
applies to every selected row whatever its classification, because:

- A selection can straddle classifications (the forty rows with the
  merchant in the wrong column may include some that happen to parse
  anyway), and silently skipping part of an explicit selection is worse
  than applying to all of it.
- "Ok" only means *parses*, not *correct*. A row whose title came from a
  column that happens to be non-empty is ok and wrong, and the bulk panel
  is the only place to say so without re-mapping the whole file.

A remap that makes a row worse re-classifies it as failed immediately, in
place, same as the per-row control — nothing is destructive and nothing is
submitted until an import button is pressed.

## Apply is explicit, unlike the per-row control

The per-row remap select applies on change, with no confirmation: it
affects one visible row, and the result is right there. The bulk panel
instead collects a draft across its controls and applies on an **Apply to
N rows** button, because:

- It can set several fields in one pass (title *and* date), which an
  apply-on-change control would turn into two separate re-classification
  sweeps across the selection.
- Each control needs a resting value, and with many rows selected there
  isn't one to show — the selected rows may currently use different
  columns for the same field. The draft solves that: every control starts
  at an explicit **Keep current** and returns there after Apply, so the
  panel never claims the selection shares a mapping it doesn't.

Only touched controls are written, so Apply with the date control set and
the title control left alone overrides the date and nothing else.

## Chosen scope, not chosen rows

`ImportDryRunStep`'s `onContinue` already handed the route a
`ClassifiedRow[]`, and the route already split it into "submit these" and
"report these as failed". So partial import needs no new plumbing: **Import
all** passes every row, **Import selected** passes the selected ones, and
the split downstream does the rest.

That also settles what the results view reports, and it is the honest
answer: a failed row the visitor didn't select never entered the run, so
listing it as a failure of that run would be wrong. Importing all keeps
today's behaviour exactly, because the chosen scope is then the whole file.

## Examples in a bulk column picker

The per-row remap picker shows *that row's* value next to each column name,
deliberately — a file-wide example would misrepresent the row being fixed.
A bulk picker has no such row, so it shows the same file-wide example the
mapping step's pickers show (first non-empty value anywhere in the file).
That lookup existed as an inline `useMemo` in `ImportMappingStep`; it moves
to `lib/import/formatExample.ts` beside `truncateExample`, so both callers
compute it one way rather than two.

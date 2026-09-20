## Why

The import wizard's dry-run step lists every row of the file and lets the
visitor fix a **failed** row by remapping one of its fields — but only one
row at a time, and only that row. Two things follow from that, and both bite
on a real bank export:

- **A remap that applies to a whole class of rows has to be redone row by
  row.** The typical CSV export mixes two record shapes: card transactions
  carry the merchant in `Description`, direct debits carry it in
  `Counterparty`, and the title mapping can only point at one of them. The
  forty rows of the other shape all fail for the same reason, and today
  fixing them means expanding forty rows and picking the same column forty
  times. Going back to the mapping step is no help — it re-maps the file,
  so it just moves the failure to the other forty rows.
- **It is all or nothing.** The dry run's only exit is "Start import",
  which submits every non-failed row. A visitor who wants only last
  quarter's rows out of a full-year export, or who wants to leave the
  suspicious rows out of a first run, has no way to say so from here: they
  have to go edit the file and start over.

Both are selection problems, and the step already renders one row per
line — it just has nothing to select with.

## What Changes

- The dry-run row list gains a **checkbox column**, with a select-all
  checkbox in the header and a "N rows selected" line with a clear action
  above the table. One selection serves both features below; clicking a
  checkbox does not expand the row, and expanding a row does not select it.
- Selecting one or more rows reveals a **bulk remap panel** above the
  table, offering the same remap controls the per-row expansion offers
  (title, amount — one control or debit/credit — and booking date), each
  defaulting to "keep current". **Apply** writes the touched fields as
  per-row overrides on **every selected row at once** and re-classifies
  them, exactly as if each had been remapped individually. Untouched
  fields and unselected rows are not affected, and the file-wide mapping is
  never changed.
- The step's single "Start import" action becomes two: **Import all**,
  which behaves exactly as today (every non-failed row in the file), and
  **Import selected**, which submits only the non-failed rows among the
  checked ones. Each button carries the count it would submit.
- The results view reports on the **chosen scope only**: a row the visitor
  did not choose to import is not listed as a failure, since it never
  failed — it wasn't asked for. Importing all keeps today's report exactly.

## Non-goals

- **No change to what a remap can do.** The bulk panel offers the same
  three (or four, in split mode) column overrides the per-row control
  already offers, against the same file-resolved amount separators. It is
  a way to apply them to many rows at once, not a new capability.
- **No filter-driven or range selection.** Selection is by checkbox, plus
  the header's select-all over the currently listed rows. No "select all
  failed", no shift-click range, no saved selection — the existing "Hide
  successful rows" toggle is the only narrowing tool, and select-all
  respects it.
- **No per-row category, tag or account override.** Those stay batch-wide,
  per the existing spec.
- **No backend change.** The dry run is entirely client-side and the import
  loop still posts one entry per row; only which rows reach it changes.

## Capabilities

### Modified Capabilities

- `web-client-entry-import`: dry-run rows are selectable via checkboxes; a
  remap can be applied to every selected row at once; the import step can
  be started for all rows or for the selection alone, and the results view
  reports on whichever scope was chosen.

## Impact

- `frontend/src/components/import/ImportDryRunStep.tsx` — the checkbox
  column, the selection state, the bulk remap panel, and the two import
  buttons; `onContinue` now hands up the chosen scope rather than always
  every row.
- `frontend/src/lib/import/formatExample.ts` — gains `collectFieldExamples`,
  the file-wide example lookup the mapping step computes inline today, so
  the bulk panel's column pickers can show the same examples (a bulk
  control has no single row to draw one from).
- `frontend/src/components/import/ImportMappingStep.tsx` — uses that shared
  helper instead of its own copy.
- `frontend/src/i18n/locales/{en,de}.json` — selection, bulk remap and the
  two import button labels; `startImport` is replaced.
- `frontend/src/components/import/ImportResultStep.tsx` — the failure
  table's two column headings pointed at `steps.mapping.rowColumn` /
  `.reasonColumn`, keys that only exist under `steps.dryRun`, so they
  rendered as raw key names; fixed while verifying the scoped results
  view.
- `frontend/src/routes/entries.import.tsx` — unchanged: it already splits
  whatever the dry-run step hands it into submittable and failed rows, so
  narrowing that list narrows both.

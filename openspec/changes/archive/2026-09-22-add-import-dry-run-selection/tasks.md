## 1. Shared example lookup

- [x] 1.1 In `frontend/src/lib/import/formatExample.ts`, add
      `collectFieldExamples(fields, rows)` — the first non-empty value
      found anywhere in the file per column — documenting that the
      per-row remap pickers deliberately do *not* use it (they show the
      row's own value).
- [x] 1.2 Replace `ImportMappingStep`'s inline `fieldExamples` `useMemo`
      body with a call to it, keeping the memo.

## 2. Selection in the dry-run step

- [x] 2.1 Add a `selected: Set<number>` state keyed by
      `ClassifiedRow.index`, with toggle/clear helpers.
- [x] 2.2 Add a checkbox `<th>` and `<td>` as the row list's first
      column. The row checkbox stops click propagation so it doesn't
      toggle the row's expansion; every checkbox carries an `aria-label`
      naming its row.
- [x] 2.3 Make the header checkbox reflect the listed rows: checked when
      all listed are selected, `indeterminate` (set via a ref) when some
      are; pressing it selects every listed row, or clears the whole
      selection when every listed row is already selected.
- [x] 2.4 Show the selected count above the list with a clear action,
      counting selected rows hidden by "Hide successful rows" too.

## 3. Bulk remap panel

- [x] 3.1 Render the panel above the row list whenever the selection is
      non-empty: one control per remappable field (title, amount or
      debit+credit per the mapping's amount mode, booking date), each
      offering every detected column with its file-wide example and an
      explicit "keep current" default, driven by a local draft state.
- [x] 3.2 Add the apply action, labelled with the number of rows it will
      affect and disabled while no control has been changed. It merges
      the touched fields into each selected row's `RowOverride` and
      re-classifies that row via `classifyRow` against
      `initial.separators`, then resets the draft to "keep current".
- [x] 3.3 Reuse the existing `applyOverride`/`handleRemap` machinery
      rather than a second override path, so a bulk remap and a per-row
      remap produce identical state.

## 4. Two import actions

- [x] 4.1 Replace the single "Start import" button with "Import all" and
      "Import selected", each labelled with the number of non-failed rows
      in its scope and disabled when that count is zero.
- [x] 4.2 Have both call `onContinue` with the rows of their chosen scope
      (every row, or the selected rows) — the route already splits that
      list into submittable and failed, so the results view follows the
      scope with no route change.

## 5. i18n

- [x] 5.1 Add the new `entries.import.steps.dryRun.*` keys to
      `en.json` (selection count/clear/aria labels, bulk panel heading,
      note, keep-current option, apply action, and the two import button
      labels) and remove the now-unused `startImport`.
- [x] 5.2 Mirror every key in `de.json` — both locales stay at 100%.

## 6. Verification

- [x] 6.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 6.2 Drive the wizard in Chromium against a stubbed `/api` with a
      CSV whose rows fail for one shared reason: select several, bulk
      remap them green, then import only the selection and confirm the
      `POST /api/entries` calls match the selected non-failed rows and
      the results list no out-of-scope failures.

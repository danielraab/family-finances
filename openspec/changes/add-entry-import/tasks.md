## 1. Dependency

- [x] 1.1 `cd frontend && pnpm add papaparse` (+ `pnpm add -D @types/papaparse`
  if papaparse ships no types of its own).

## 2. Parsing and mapping helpers (`frontend/src/lib/import/`)

- [x] 2.1 `parseFile.ts`: given a selected `File`, detect CSV vs. JSON by
  extension, and return `{ rows: Record<string, string>[], fields: string[] }`
  or a shape error.
  - CSV: `Papa.parse(text, { header: true, skipEmptyLines: true })`;
    `fields` from `meta.fields`.
  - JSON: `JSON.parse`; reject anything whose top-level value isn't an
    array of flat objects; `fields` is the union of keys across the first
    200 elements, in first-seen order; all values stringified for a
    uniform row shape.
- [x] 2.2 `parseAmount.ts`: single-column mode (decimal/thousands
  separator settings incl. "auto", flip-sign) and split debit/credit mode
  (`credit − debit`, blank treated as `0`); output the same fixed-4-
  decimal-place integer `lib/amount.ts`'s `inputToAmount` produces.
  Include an `autoDetectSeparators(samples: string[])` helper.
- [x] 2.3 `parseDate.ts`: `"auto"` (ISO/`Date.parse`) and an explicit
  `DD`/`MM`/`YYYY`/`HH`/`mm`/`ss`-token format parser; also exports an
  `isAmbiguousDate(raw: string)` check for the dry run's suspicious
  classification.
- [x] 2.4 `mapRow.ts`: `mapRow(rawRow, mapping) => { entry } | { errors }` —
  the single implementation of "is this row valid and what entry does it
  produce," used by both the dry run and the real import loop (see
  design.md). `entry` omits `account_id`/`tag_ids` (filled in by the
  caller) and is otherwise a ready `POST /api/entries` body for a
  `transaction`.
- [ ] 2.5 **Skipped**: `frontend/` has no test runner anywhere in the repo
  today (no vitest/jest dependency, no `*.test.ts` files, no test step in
  `frontend/AGENTS.md`'s "Before you're done" — verification there is
  lint + `tsc` + `build` + manual exercise only). Adding one is a real,
  deliberate toolchain decision outside this change's stated scope
  (proposal.md's Impact section doesn't mention it), so it wasn't added
  unilaterally. `lib/import/`'s correctness instead rests on `tsc`'s
  strict checks plus the manual golden-path pass in 7.2, which now
  exercises every case this task originally listed (quoted/embedded-comma
  CSV, a malformed JSON shape, both amount modes with each separator
  combination, auto vs. explicit date parsing, an ambiguous date, and the
  failed/suspicious/ok classification boundary). Revisit if/when the
  frontend adopts a test runner.

## 3. Wizard components (`frontend/src/components/import/`)

- [x] 3.1 `ImportAccountStep.tsx`: account picker mirroring
  `entries.new.tsx`'s `selectableAccounts` (append+ permission only),
  locked when arriving via `?account_id=`.
- [x] 3.2 `ImportFileStep.tsx`: file input restricted to `.csv`/`.json`,
  the persistent CSV-header-row note, calls `parseFile`, surfaces a shape
  error inline without advancing.
- [x] 3.3 `ImportMappingStep.tsx`: per-field mapping controls (title,
  amount mode + separators + flip-sign, booking timestamp + date format,
  description/counterparty/location as optional mappings), the single
  category picker (reusing `entries.new.tsx`'s `flattenCategoryTree`-based
  options, excluding disabled/view-tier-shared), the batch `TagInput`, the
  live one-row preview, the "Run dry run" action (disabled until title/
  amount/date are mapped and a category is selected), and the dry-run
  results (counts + failed/suspicious rows table, ok rows never listed).
  Running the dry run applies `mapRow` to every parsed row, then layers
  the suspicious checks (zero amount; ambiguous date when date format is
  "auto") on top of the ok rows.
- [x] 3.4 `ImportRunStep.tsx`: resolves the batch's tag names to ids once
  (lift `entries.new.tsx`'s `resolveTagIds` into a shared
  `frontend/src/lib/resolveTags.ts` used by both), then submits one
  `POST /api/entries` per non-failed row in file order, updating a live
  `created/failed/total` counter; a Cancel button that stops between rows.
- [x] 3.5 `ImportResultStep.tsx`: created count + unified failed-row list
  (dry-run failures plus any rows rejected during submission), no
  per-success listing; a way to start a new import (back to step 1).

## 4. Route

- [x] 4.1 `frontend/src/routes/entries.import.tsx` → `/entries/import`.
  Auth-gating: implemented by nesting under the existing `/entries`
  layout route (`entries.tsx`'s file-based-routing parent), which already
  redirects an anonymous visitor to `/login` for every route nested under
  it — the same mechanism `entries.new.tsx` already relies on, so no
  separate gate was added here. `validateSearch` for an optional
  `account_id`, same shape as `entries.new.tsx`'s `NewEntrySearch`; owns
  the step state machine (account → file → mapping/dry-run → run →
  result) and passes state down to the step components from section 3.

## 5. Entry points

- [x] 5.1 `entries.index.tsx`: add an "Import" link beside "New Entry" in
  the toolbar, carrying `search.account_id` the same way "New Entry"
  already does.
- [x] 5.2 `accounts.$accountId.index.tsx`: add an "Import" link beside the
  existing "New Entry" link (same `account.permission !== "view"` guard),
  to `/entries/import?account_id={accountId}`.

## 6. i18n

- [x] 6.1 Add `entries.import.*` keys to `frontend/src/i18n/locales/en.json`
  covering: step labels, the CSV-header note, mapping field labels, amount
  mode/separator/flip-sign controls, date format input, dry-run
  classification labels and counts, progress text, cancel/result copy, and
  the two new "Import" link labels (reusing `entries.create`'s neighbor
  slot). Add German equivalents to `de.json` where practical
  (non-blocking in CI if it lags, per `frontend/AGENTS.md`).

## 7. Verification

- [x] 7.1 `pnpm lint`, `pnpm exec tsc`, `pnpm build` from `frontend/` — all
  pass clean (lint's remaining warning/infos are pre-existing, in files
  this change didn't touch).
- [x] 7.2 Manually exercised end-to-end against a real backend: started
  PostgreSQL and the Go backend locally, seeded a test user/accounts/
  categories/tags via `server seed`, ran the frontend dev server, and
  drove the actual browser (Chromium via Playwright) through every
  scenario below, verifying both the UI and the entries actually created
  via the API:
  - Imported a small CSV (signed amount column, `,` decimal / `.`
    thousands, explicit `DD.MM.YYYY` date format) into one account: dry
    run correctly reported "3 ready, 1 suspicious, 1 failed" — a
    malformed-date row failed, a zero-amount row was flagged suspicious;
    only the 3 ok + 1 suspicious rows were submitted (4 created, 1
    failed), with amounts verified byte-exact via `GET /api/entries`
    (e.g. `"-45,90"` → stored `-459000`).
  - Imported a small JSON file (split debit/credit columns) into a
    different account: dry run reported "3 ready, 0 suspicious, 0
    failed"; all 3 created with `credit − debit` amounts verified exact,
    the same category on all three, and a newly-typed batch tag created
    exactly once and attached to all three (confirmed via
    `GET /api/entries` and `GET /api/tags`).
  - Confirmed the CSV-header note is visible on the file step before any
    file is chosen.
  - Confirmed changing the date format after a dry run immediately clears
    the dry-run result (re-run required), then re-ran the dry run against
    the same already-selected file with the new format and got the
    corrected classification — no file re-selection needed.
  - Confirmed a malformed JSON shape (`{"transactions": [...]}` instead of
    a top-level array) is rejected inline at the file step with the
    expected message, without advancing to mapping.
  - Imported a larger (300-row) file, clicked Cancel mid-run, and
    confirmed the run stopped (progress stopped advancing, "Import was
    canceled — entries created so far are kept" shown) with no further
    entries created afterward — rows already created stayed created.
  - Confirmed both new "Import" links (`/entries` toolbar, account detail
    page) navigate to `/entries/import`, the latter with the account
    preset and locked (file step shown directly, account step skipped).

## 8. Follow-up fixes (post-implementation feedback)

- [x] 8.1 Restyled the file-selection control: a hidden input behind a
  label styled like the app's other secondary buttons (was a raw,
  unstyled `<input type="file">`), with the picked filename shown beside
  it.
- [x] 8.2 Removed the file input's `accept` filter entirely and added
  `hasSupportedExtension()` (`lib/import/parseFile.ts`) plus an
  `unsupportedFileType` error, checked before parsing — fixes `.csv`
  files being unselectable on Android, where SAF-backed file providers
  filter inconsistently by MIME type regardless of what `accept` lists
  (see design.md's "File input has no accept filter" decision). Verified
  in a real browser that the chooser is now unfiltered and a wrong
  extension shows a clear error without attempting to parse.
- [x] 8.3 `parseFile.ts`'s JSON branch no longer rejects the whole file on
  any nested object/array field: a `{ value, precision, currency? }`
  -shaped field (any key name) is recognized and converted to a plain
  mappable decimal field (plus a `<key>.currency` field when currency is
  present); any other nested object or array is skipped for that field
  only and collected into `ParsedFile.ignoredFields`, shown as a
  non-blocking hint at the top of the mapping step
  (`ImportMappingStep.tsx`) rather than blocking the import. Verified
  end-to-end against a real backend: a structured `{"value": -350,
  "precision": 2, "currency": "EUR"}` field mapped and imported as the
  entry amount `-3.5000` (stored `-35000` at the fixed 4-decimal-place
  scale), and unrelated nested/array fields on the same file were
  correctly excluded from the mappable list and listed in the hint,
  without blocking the rows that did map.

## 9. Follow-up UX revision (post-implementation feedback)

- [x] 9.1 Each mapping `<select>` (`FieldSelect` in `ImportMappingStep.tsx`)
  now shows every offered column/field together with an example value
  from the file (first row with a non-empty value for it), via a
  `fieldExamples` map computed once per file with `useMemo`.
- [x] 9.2 Dropped the standalone "Preview" card. The dry-run results table
  now lists every row (not only failed/suspicious), scrollable
  (`max-h-[28rem] overflow-y-auto`) for large files, each row
  independently click-to-expand (local `expandedRows: Set<number>` state,
  reset on every dry-run re-run) to show its raw source values
  (`rawFieldsForRow`) and, for an ok/suspicious row, the entry it would
  create — the same fields the removed preview card used to show for one
  row, now available per row on demand.
- [x] 9.3 Updated design.md's dry-run decision and the
  `web-client-entry-import` delta spec (dropped the old single-row
  preview requirement; the dry-run requirement now lists every row;
  added requirements for per-row expansion and per-field examples).
- [x] 9.4 Verified end-to-end in a real browser: mapping `<select>`
  options show `"Amount — -45,90"`-style examples; after a dry run all 5
  rows of a small fixture are listed (3 ready, 1 suspicious, 1 failed,
  none hidden); clicking a row expands it showing matching source and
  mapped-entry values; a full import from the same mapping still creates
  the correct entries.

## 10. Dry run as its own step + per-row field remap (post-implementation feedback)

- [x] 10.1 Split the dry run out of `ImportMappingStep.tsx` into its own
  new component, `ImportDryRunStep.tsx`, and a new wizard step
  (`entries.import.tsx`'s `Step` union gains `"dryrun"` between
  `"mapping"` and `"run"`). `ImportMappingStep.tsx` now only maps fields
  and sets category/tags, with a "Continue" action (reusing the existing
  `entries.import.continue` key) gated on `toRowMapping(mapping) !==
  null`; it no longer holds any dry-run state or UI. The route no longer
  holds `dryRun`/`onDryRun` state; it holds `finalRows: ClassifiedRow[] |
  null`, set from `ImportDryRunStep`'s `onContinue` callback and used to
  derive both the run step's submittable rows and the result step's
  failed-row list.
- [x] 10.2 `ImportDryRunStep.tsx` computes the dry run once via a lazy
  `useState(() => runDryRun(sourceRows, rowMapping))` initializer (runs
  exactly once, on mount — no explicit "run" trigger, no
  `useExhaustiveDependencies` workaround needed) and keeps the classified
  rows in its own state, re-derived summary counts via `useMemo`. Carries
  over the full-row list, per-row click-to-expand, and source-data/
  mapped-entry display from the previous change unchanged, moved
  verbatim (`rawFieldsForRow`, the two-column expanded-detail layout).
- [x] 10.3 Added the per-row, per-field remap capability: a **failed**
  row's expanded detail now shows a `RemapSelect` per issue field (title;
  amount — one or two selects depending on single/split mode; booking
  date), defaulting to the mapping's current column for that field and
  offering every detected column. `applyOverride(rowMapping, override)`
  builds an effective `RowMapping` with just that row's override(s)
  applied; on change, `classifyRow` re-runs for that one row only (against
  the amount separators resolved once for the whole file, captured from
  the initial dry run), replacing its entry in local state — every other
  row, and the running counts, update accordingly with no full re-scan.
  Overrides live in a step-local `Record<rowIndex, RowOverride>`; "Continue"
  hands the current (remap-inclusive) classified rows up to the route.
- [x] 10.4 i18n: moved the dry-run-specific keys from
  `entries.import.steps.mapping.*` to a new `entries.import.steps.dryRun.*`
  namespace (`summary`, `rowColumn`, `statusColumn`, `reasonColumn`,
  `failed`, `suspicious`, `ready`, `sourceDataHeading`, renamed
  `previewHeading` → `mappedEntryHeading`, `startImport`), dropped the
  now-unused `runDryRun` key, added `heading` and the new
  `remapHeading`/`remapNote` strings, in both `en.json` and `de.json`.
  The mapping step's own action button reuses the existing
  `entries.import.continue` key instead of a dry-run-flavored label.
- [x] 10.5 Updated proposal.md, design.md, and the `web-client-entry-import`
  delta spec: the dry run is now documented as its own step (entered
  automatically, no manual "run" trigger), and new requirements/scenarios
  cover the per-row remap capability and the revised "change the whole
  file's mapping" flow (back to mapping, then continue forward again,
  rather than an in-step re-run button).
- [x] 10.6 Verified end-to-end in a real browser: mapping step shows no
  dry-run UI and its action button reads "Continue"; continuing lands on
  a distinct "Dry run" step whose results are already computed; expanding
  a row classified failed (an intentionally-bad date, with a second,
  correctly-formatted date column present in the fixture) shows a
  booking-date remap select; picking the alternate column immediately
  flips that row to ready with its mapped entry shown, updates the
  summary counts live, and leaves every other row unchanged; completing
  the import creates an entry for the remapped row using the remapped
  column's value, confirmed via `GET /api/entries`.

## 11. Browser back button + row-specific remap examples (post-implementation feedback)

- [x] 11.1 `entries.import.tsx`'s "account"/"file"/"mapping"/"dryrun"
  steps now live in the URL (`?step=`), not local `useState`. The route's
  `validateSearch` computes a default (`"file"` when `?account_id=` is
  preset, else `"account"`) and accepts only the four navigable step
  values, falling back to the default for anything else. Forward
  transitions call `navigate({ search: (prev) => ({ ...prev, step:
  next }) })` (a push, via `Route.useNavigate()` so the search updater is
  typed against this route's own search shape rather than the router's
  global union); every in-page "Back" link (except the account-locked
  file step's leave-the-wizard case, which still explicitly navigates to
  `/entries`) now calls `window.history.back()` instead of setting local
  step state, so the physical browser Back button and the in-page Back
  link are exactly symmetric. Because a search-param-only navigation
  re-renders the same route component instead of remounting it, every
  other piece of in-memory wizard state (parsed file, mapping, per-row
  overrides) survives a back/forward step change untouched.
- [x] 11.2 "run" and "result" are deliberately **not** part of the URL:
  they're a separate local `runPhase: "run" | "result" | null` state that
  takes rendering priority over the URL-derived step. A `useEffect` keyed
  on the URL `step` resets `runPhase` to `null` whenever it changes (i.e.
  on a genuine back/forward navigation), so a stale run/result view can't
  stick around after navigating away from it — but nothing in browser
  history can ever set `runPhase` back to `"run"`, so `ImportRunStep` can
  never be remounted, and its entries never resubmitted, via back/forward.
  `startOver()` resets `runPhase` and every other piece of local state and
  navigates with `replace: true` back to the wizard's first step, so the
  finished import isn't reachable again via forward-navigation.
- [x] 11.3 Added a defensive fallback: a `useEffect` redirects
  (`replace: true`) to the `"file"` step whenever the URL step is
  `"mapping"` or `"dryrun"` but there's no in-memory `parsed` file — the
  case a hard page reload lands in, since the parsed file/mapping state
  cannot survive a reload but the URL's `?step=` can.
- [x] 11.4 The dry-run step's per-row remap `<select>` (`RemapSelect` in
  `ImportDryRunStep.tsx`) now shows an example value next to each column
  name, same as the mapping step's pickers — but sourced from **that
  specific row's** own raw value for the column, not the file-wide
  first-non-empty-value example the mapping step uses (a different row's
  value would be misleading when remapping one failing row). The shared
  truncation logic (`EXAMPLE_MAX_LENGTH`/truncation to `"…"`) moved out of
  `ImportMappingStep.tsx` into a new `lib/import/formatExample.ts`
  (`truncateExample`), reused by both.
- [x] 11.5 Verified end-to-end in a real browser: stepping account → file
  → mapping → dry run and then pressing the browser Back button three
  times steps back exactly one wizard step each time (dry run → mapping →
  file → account) with the mapping selections still intact; pressing
  Forward replays the same steps with state preserved. Ran a full import
  to completion, then pressed Back — it returned to the mapping step
  without re-submitting either entry (`POST /api/entries` call count
  unchanged before/after), and pressing Forward again landed back on the
  dry-run view (not a stale result screen) rather than resubmitting.
  Confirmed a failed row's remap `<select>` shows that row's own value
  (e.g. `"AltDate — 05.03.2026"` for row 2) rather than a different row's
  file-wide example value (row 1's `"04.03.2026"`).

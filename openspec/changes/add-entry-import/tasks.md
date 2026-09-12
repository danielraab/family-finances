## 1. Dependency

- [ ] 1.1 `cd frontend && pnpm add papaparse` (+ `pnpm add -D @types/papaparse`
  if papaparse ships no types of its own).

## 2. Parsing and mapping helpers (`frontend/src/lib/import/`)

- [ ] 2.1 `parseFile.ts`: given a selected `File`, detect CSV vs. JSON by
  extension, and return `{ rows: Record<string, string>[], fields: string[] }`
  or a shape error.
  - CSV: `Papa.parse(text, { header: true, skipEmptyLines: true })`;
    `fields` from `meta.fields`.
  - JSON: `JSON.parse`; reject anything whose top-level value isn't an
    array of flat objects; `fields` is the union of keys across the first
    200 elements, in first-seen order; all values stringified for a
    uniform row shape.
- [ ] 2.2 `parseAmount.ts`: single-column mode (decimal/thousands
  separator settings incl. "auto", flip-sign) and split debit/credit mode
  (`credit − debit`, blank treated as `0`); output the same fixed-4-
  decimal-place integer `lib/amount.ts`'s `inputToAmount` produces.
  Include an `autoDetectSeparators(samples: string[])` helper.
- [ ] 2.3 `parseDate.ts`: `"auto"` (ISO/`Date.parse`) and an explicit
  `DD`/`MM`/`YYYY`/`HH`/`mm`/`ss`-token format parser; also exports an
  `isAmbiguousDate(raw: string)` check for the dry run's suspicious
  classification.
- [ ] 2.4 `mapRow.ts`: `mapRow(rawRow, mapping) => { entry } | { errors }` —
  the single implementation of "is this row valid and what entry does it
  produce," used by both the dry run and the real import loop (see
  design.md). `entry` omits `account_id`/`tag_ids` (filled in by the
  caller) and is otherwise a ready `POST /api/entries` body for a
  `transaction`.
- [ ] 2.5 Unit tests for 2.1-2.4: quoted/embedded-comma CSV fields, a
  malformed-shape JSON file, each amount mode and separator combination,
  auto vs. explicit date parsing, ambiguous-date detection, and `mapRow`'s
  failed/ok classification boundary (suspicious classification itself
  lives with the dry-run orchestration in 3.3, since it needs the whole
  row set for the zero-amount/ambiguous-date checks).

## 3. Wizard components (`frontend/src/components/import/`)

- [ ] 3.1 `ImportAccountStep.tsx`: account picker mirroring
  `entries.new.tsx`'s `selectableAccounts` (append+ permission only),
  locked when arriving via `?account_id=`.
- [ ] 3.2 `ImportFileStep.tsx`: file input restricted to `.csv`/`.json`,
  the persistent CSV-header-row note, calls `parseFile`, surfaces a shape
  error inline without advancing.
- [ ] 3.3 `ImportMappingStep.tsx`: per-field mapping controls (title,
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
- [ ] 3.4 `ImportRunStep.tsx`: resolves the batch's tag names to ids once
  (lift `entries.new.tsx`'s `resolveTagIds` into a shared
  `frontend/src/lib/resolveTags.ts` used by both), then submits one
  `POST /api/entries` per non-failed row in file order, updating a live
  `created/failed/total` counter; a Cancel button that stops between rows.
- [ ] 3.5 `ImportResultStep.tsx`: created count + unified failed-row list
  (dry-run failures plus any rows rejected during submission), no
  per-success listing; a way to start a new import (back to step 1).

## 4. Route

- [ ] 4.1 `frontend/src/routes/entries.import.tsx` → `/entries/import`,
  auth-gated (redirect anonymous to `/login`, mirroring `/entries`'s
  gate); `validateSearch` for an optional `account_id`, same shape as
  `entries.new.tsx`'s `NewEntrySearch`; owns the step state machine
  (account → file → mapping/dry-run → run → result) and passes state down
  to the step components from section 3.

## 5. Entry points

- [ ] 5.1 `entries.index.tsx`: add an "Import" link beside "New Entry" in
  the toolbar, carrying `search.account_id` the same way "New Entry"
  already does.
- [ ] 5.2 `accounts.$accountId.index.tsx`: add an "Import" link beside the
  existing "New Entry" link (same `account.permission !== "view"` guard),
  to `/entries/import?account_id={accountId}`.

## 6. i18n

- [ ] 6.1 Add `entries.import.*` keys to `frontend/src/i18n/locales/en.json`
  covering: step labels, the CSV-header note, mapping field labels, amount
  mode/separator/flip-sign controls, date format input, dry-run
  classification labels and counts, progress text, cancel/result copy, and
  the two new "Import" link labels (reusing `entries.create`'s neighbor
  slot). Add German equivalents to `de.json` where practical
  (non-blocking in CI if it lags, per `frontend/AGENTS.md`).

## 7. Verification

- [ ] 7.1 `pnpm lint`, `pnpm exec tsc`, `pnpm build` from `frontend/`.
- [ ] 7.2 Manually exercise the golden path: import a small CSV with a
  signed amount column and a small JSON file with split debit/credit
  columns, into two different accounts; confirm the CSV-header note is
  visible before a file is chosen; confirm the dry run lists an
  intentionally-broken row (bad date) as failed and a zero-amount row as
  suspicious, without listing the good rows; adjust the date format and
  re-run the dry run without re-selecting the file; complete an import and
  confirm live progress, the correct created count, category/tags applied
  to every created entry, and that a new tag name is created exactly
  once; cancel an import partway through and confirm earlier rows persist
  and no further requests are made; confirm both new "Import" entry-point
  links navigate correctly, with and without a preset account.

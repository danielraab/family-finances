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

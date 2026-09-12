## Why

Every entry today is created one at a time through `/entries/new`. A
visitor onboarding an account with months of history, or reconciling a
bank statement, has no way to bring in more than one entry without typing
each one by hand. Bank/export files (CSV or JSON) are the natural source,
but they never line up with the API's field names or shapes, so a raw
"upload and create" wouldn't work without a way to tell the app which
column means what — and a way to catch a bad date or amount format before
it silently creates a year's worth of misdated transactions.

## What Changes

- A new authenticated route, `/entries/import`, walks a visitor through
  importing **transaction** entries (never `balance_adjustment`, which has
  no place in a file export — it's a point-in-time reading, not a batch of
  history) into one account, from a CSV or JSON file:
  1. **Account** — pick the target account first (or arrive with it preset
     via `?account_id=`, same convention as `/entries/new`).
  2. **File** — select a `.csv` or `.json` file. The page states plainly
     that a CSV file's first row must be column headings.
  3. **Mapping** — map source columns/fields to entry fields (title,
     amount, booking timestamp, description, counterparty, location), plus
     one **category** and one **tags** set applied to every imported entry
     (not mapped per row — there is no per-row category/tag value mapping
     in this change). Amount mapping supports either a single signed
     column or a split debit/credit column pair, with configurable
     decimal/thousands separators and a flip-sign option, to cover the
     different shapes bank exports use. A live preview shows one mapped
     row so the visitor can sanity-check the mapping before going further.
  4. **Dry run** — the whole file is parsed and validated locally (no
     network calls) against the current mapping. Rows that fail (bad date,
     bad amount, missing a required mapped field) or look suspicious (an
     ambiguous date, a zero amount) are listed with a reason; successfully
     parsed rows are only counted, never listed one by one. The visitor
     can adjust the date format or amount separator settings and re-run
     the dry run, as many times as needed, entirely offline, before
     committing to anything.
  5. **Import** — creates each non-failed row as a real entry
     (`POST /api/entries`, one call per row, in file order), with a live
     progress indicator and a cancel action. A row that fails at this
     stage (rejected by the backend) is skipped and the rest continue.
  6. **Result** — a final count of created entries plus a list of every
     failed row and why, again with no per-success listing.
- **No backend or API changes.** Every entry is still created through the
  existing `POST /api/entries` (and, for any newly-typed tag names,
  `POST /api/tags`, reusing `entries.new.tsx`'s existing inline-tag-
  creation pattern) — parsing, mapping, and validation all happen in the
  browser.
- **New frontend dependency**: `papaparse` for CSV parsing. It also
  supports generating CSV (`Papa.unparse`), which this change does not use
  but which positions it to back a future CSV *export* feature without a
  second library.

## Capabilities

### Added Capabilities

- `web-client-entry-import`: the `/entries/import` wizard end to end —
  account/file selection, column/field mapping (including the amount and
  date parsing settings), the offline dry-run validation loop, the live
  import run, and the results view.

### Modified Capabilities

- `web-client-entries`: the entry ledger toolbar gains an "Import" action
  next to "New Entry", navigating to `/entries/import` (carrying the
  ledger's current account filter, when one is set, the same way "New
  Entry" already does).
- `web-client-accounts`: the account details page gains an "Import" action
  next to its existing "New Entry" action, navigating to
  `/entries/import?account_id={id}`.

## Impact

- **Backend**: none. No migration, no new/changed endpoint, no OpenAPI
  change.
- **Frontend**:
  - New dependency: `papaparse` (+ its types) via `pnpm add`.
  - New route `frontend/src/routes/entries.import.tsx`.
  - New `frontend/src/components/import/` components for the wizard steps
    (file/parse, mapping + dry-run, run + results) and
    `frontend/src/lib/import/` helpers (file parsing, amount parsing, date
    parsing, and the single row-mapping function shared by the dry run and
    the real import loop, so the two can never disagree about whether a
    row is valid).
  - Small additions to `entries.index.tsx` (Import link) and
    `accounts.$accountId.index.tsx` (Import link).
  - New i18n keys in `en.json` (and `de.json`, non-blocking if it lags).
- No changes to any existing table, endpoint, or response shape — entries
  created via import are ordinary entries, indistinguishable afterward
  from one created through `/entries/new`.

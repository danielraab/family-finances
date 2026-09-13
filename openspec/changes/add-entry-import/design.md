## Context

`POST /api/entries` already accepts exactly the shape one imported row
needs to become: `account_id`, `kind: "transaction"`, `amount`,
`booking_timestamp`, `title`, optional `description`/`counterparty`/
`location`, a required `category_id`, and `tag_ids`. `entries.new.tsx`
already establishes the pattern this change leans on hardest: `tag_ids` is
resolved client-side from free-text tag *names* by matching against
`GET /api/tags` and calling `POST /api/tags` for anything unmatched,
before the entry itself is created. Import does the same thing once, for
one batch-level tag set, rather than once per entry.

Everything else — file parsing, column mapping, amount/date parsing,
dry-run validation — is new, and entirely client-side. This mirrors the
project's general posture (`frontend/AGENTS.md`: "never open a database
connection from the frontend," same-origin `/api/...` calls only) taken
one step further: there's nothing here a backend endpoint would do better
enough to justify one. See "Client-side only, no batch endpoint" below for
the tradeoffs that were weighed.

## Goals / Non-Goals

**Goals:**
- Import transaction entries from a CSV or JSON file into one account, via
  a visitor-defined column/field mapping.
- Support the amount and date variance real-world exports have (signed vs.
  split debit/credit, different decimal/thousands separators, non-ISO
  dates) without guessing wrong silently.
- Let a visitor catch a bad mapping *before* creating anything, via a
  fully offline dry run over the whole file, not just a sample row.
- Apply one category and one set of tags to every entry in the batch.
- Never block the whole import on one bad row — skip and report instead.

**Non-Goals:**
- No `balance_adjustment` import — only `transaction` entries.
- No per-row category or tag mapping — both are chosen once for the whole
  batch, by explicit product decision.
- No backend batch-create endpoint — see "Client-side only" below.
- No duplicate detection against existing entries.
- No CSV *export* (though the chosen library supports it) and no resuming
  a partially-completed or canceled import — starting over means picking
  the file again from step 2 (mapping settings are not persisted between
  attempts).
- No streaming/worker-based parsing for very large files — the whole file
  is parsed and dry-run-validated synchronously in the main thread. Real
  bank/export files (thousands of rows) parse in well under a second;
  genuinely huge files are explicitly out of scope for this change.

## Decisions

### Client-side only, no batch endpoint

Two shapes were weighed: (A) parse/map/validate in the browser and call
the existing `POST /api/entries` once per row, or (B) send the mapped rows
to a new `POST /api/entries/import` that creates them in one backend
transaction.

(A) wins for this change: it needs no OpenAPI change, no new backend
package surface, and reuses the exact validation `POST /api/entries`
already does — an imported row is authorized, validated, and created
exactly like a manually-typed one, with zero risk of the two paths
drifting apart. Its downsides — no atomicity (a failure partway through
leaves the earlier rows created), and one HTTP round-trip per row — are
both accepted: atomicity was explicitly not requested (skip-and-continue
*is* the requested behavior), and `internal/entry`'s balance-adjustment
recompute is already a per-entry cost paid identically either way (see
`backend/AGENTS.md`'s "Recompute is synchronous" note) — a batch endpoint
would still be creating rows one at a time internally, just without the
network round-trip between them. If import volume or performance ever
becomes a real complaint, (B) remains available as a later, separate
change; nothing here forecloses it.

### `papaparse` for CSV, chosen with CSV *export* in mind too

A correct CSV parser has to handle quoted fields, embedded commas/
newlines, and escaped quotes — not worth hand-rolling. `papaparse` parses
(`Papa.parse`, with `header: true` to require/consume the mandatory first
header row) and can also generate CSV (`Papa.unparse`), so a future export
feature reuses the same dependency instead of adding a second one. JSON
needs no library: `JSON.parse` plus a shape check.

### JSON input shape: a top-level array of objects; unmappable fields are skipped, not fatal

Only `[{...}, {...}, ...]` is accepted at the top level — a bare object or
a nested/wrapped shape like `{ "transactions": [...] }` is still rejected
at the file-select step with a clear error, rather than guessing which
array inside an arbitrary structure was meant. Within each object,
however, a field is evaluated independently rather than as an all-or-
nothing gate on the whole file (revised after real export samples showed
this was too strict):

- A string/number/boolean/null value is mappable directly, as before.
- A nested object matching the shape `{ value: number, precision: number,
  currency?: string }` (e.g. `{"value": -3595, "precision": 2, "currency":
  "EUR"}`, a common structured-export encoding for `-35.95`) is recognized
  by shape, not by key name, and exposed as a normal mappable field under
  its original key — pre-converted to the plain decimal string
  `(value / 10**precision).toFixed(precision)`, so it slots into the
  existing single-column amount mapping with no special UI. When the
  object also carries `currency`, it's additionally exposed as a sibling
  `<key>.currency` field.
- Any other nested object, or an array, is skipped for that field only —
  never rejects the file. Every top-level key skipped this way (across
  any row) is collected and shown as a one-line, non-blocking hint at the
  top of the mapping step ("Some fields could not be mapped and were
  ignored (complex structure): meta, tags_from_bank"), so the visitor
  knows some source data wasn't offered, without losing the rows that
  *are* mappable.

The mappable field list is the union of keys across a bounded sample of
the array (first 200 objects), in first-seen order — bounded so one
pathological file with thousands of distinct keys can't make the mapping
UI unusable; large real-world exports are overwhelmingly uniform-shaped
anyway.

### File input has no `accept` filter, deliberately

The `<input type="file">` on the file-select step sets no `accept`
attribute at all (revised after real-device testing surfaced a problem
the original build didn't catch). `accept=".csv,.json"` looked correct
and worked in every desktop/Playwright check, but on Android, Chrome's
document picker filters by MIME type via a static, inconsistently
populated extension→MIME table; several SAF-backed providers (Google
Drive, various vendor "Files" apps) then hide real `.csv` files whose
reported MIME doesn't match what that table expects, regardless of what
`accept` lists. There is no `accept` value that reliably fixes this
across providers — the only dependable fix is to stop filtering and show
every file. `hasSupportedExtension()` (`lib/import/parseFile.ts`) then
checks the picked file's name before attempting to parse it, surfacing a
plain "Please choose a .csv or .json file." message instead of a
confusing parse failure when the visitor picks something else. The file
button itself is a styled `<label>` wrapping a visually hidden
(`sr-only`) input, matching the app's other secondary buttons, with the
picked filename shown beside it — the native, unstyled `<input
type="file">` button was reported as visually out of place.

### One shared `mapRow` function, used by both the dry run and the real import

`mapRow(rawRow, mapping) → { entry } | { errors }` is the single place a
raw row (whatever `Record<string, string>` shape the CSV/JSON parse step
produced) becomes either a valid entry-create payload or a list of
per-field problems. The dry run calls it over every row with no side
effects; the import step calls the *same* function over the same rows
immediately before each `POST /api/entries`. There is exactly one
implementation of "is this row valid," so the dry run's verdict can never
disagree with what actually gets submitted — the same reasoning
`add-entry-counterparty-location`'s design.md used for sharing one
`parseLocation` between the ledger's globe icon and the entry form's live
preview.

### Amount mapping: single signed column, or split debit/credit

```
 ○ Single column (signed)          ○ Split debit / credit
   [source column ▾]                 Debit  [source column ▾]  → negative
   decimal sep:  auto / . / ,        Credit [source column ▾]  → positive
   thousands sep: auto / none /
                  , / . / space / '
   [ ] flip sign
```

Both modes end at the same place: a signed integer at the fixed
`AmountScale = 4` the backend uses instance-wide (see
`backend/internal/entry`), via the same rounding `lib/amount.ts`'s
`inputToAmount` already does — just fed a normalized decimal string first.
Normalization strips the chosen thousands separator and rewrites the
chosen decimal separator to `.`; "auto" inspects the sample rows (majority
vote across the parsed values: whichever of `.`/`,` appears in the
rightmost position followed by 1-2 digits most often) and is always
overridable once the dry run flags a misparse. Split mode computes
`amount = credit - debit` (treating an unmapped/blank cell in either
column as `0`), so a row need not populate both columns.

### Date mapping: auto-detect first, explicit format on demand

Default `"auto"` tries `Date.parse`/ISO-8601-style parsing. When the dry
run flags a row as failed (unparseable) or suspicious (an ambiguous
`D?D/M?M/YYYY`-shaped value where both orderings are plausible), the
visitor can set an explicit format string using a small token vocabulary
(`YYYY`, `MM`, `DD`, `HH`, `mm`, `ss` — e.g. `DD.MM.YYYY`), parsed by a
small dedicated function, not a date library — the token set is
deliberately minimal, covering the formats real exports actually use
rather than general date parsing.

### Dry run classifies every row as ok, suspicious, or failed — every row is listed, click to expand

```
             mapRow() result           additional checks (ok rows only)
row ──▶ ┌─────────────────────┐   ok   ┌───────────────────────────┐
        │ required field      │───────▶│ amount == 0 ?              │──▶ suspicious
        │ present + parses?   │        │ ambiguous date (auto) ?    │
        └─────────────────────┘        └───────────────────────────┘
              │ no                            │ neither
              ▼                               ▼
           failed                             ok
```

- **failed**: excluded from the import step entirely — never submitted.
- **suspicious**: still included in the import step (it did parse) but
  called out with a reason so the visitor can look before proceeding.
- **ok**: no issue found.

Revised from an earlier version of this design that only listed
failed/suspicious rows and showed one row's mapped result in a separate
"preview" card: real use surfaced a need to actually see the mapped data
across a file, not just be told a count, so the dry-run table now lists
**every** row (in a scrolling container, since a large file can be
hundreds of rows) and the standalone preview card is gone entirely. Each
row is independently click-to-expand, revealing that row's raw source
values for every mapped column plus — for ok/suspicious rows — the entry
it would create, rendered the same way the old preview card rendered its
one row (title/amount/date/description/counterparty/location). A failed
row's expansion shows only its source values, since `mapRow` produced no
entry for it. Multiple rows can be expanded independently; re-running the
dry run collapses all of them, since the underlying rows are new.

Listing every row (not just problem ones) made a large, mostly-clean file
tedious to scan for the handful of rows that need attention — so a "Hide
successful rows" checkbox filters the *displayed* list down to
suspicious/failed rows only (`rows.filter((r) => r.classification !==
"ok")`, computed at render time, not stored back into `rows`). It's
deliberately display-only: the summary counts and what gets submitted on
"Continue" both still come from the full, unfiltered `rows` state — hiding
a row from view never removes it from the import.

The mapping controls themselves also gained a lighter-weight version of
"see the data before committing": every source column/field offered in a
`<select>` shows an example value from the file next to its name (the
first row with a non-empty value for that column), so a visitor picking,
say, the title column doesn't have to guess which raw header holds a
recognizable value.

### The dry run is its own wizard step, not embedded in mapping

Revised again after the dry run grew a per-row remap control (next
section) — cramming file-wide mapping, a possibly-long results table, and
per-row remap controls into one screen was too much at once, and
conceptually the dry run *reacts to* a completed mapping rather than being
part of building it. `ImportDryRunStep.tsx` is now its own step
(`ImportMappingStep.tsx` → `ImportDryRunStep.tsx` → `ImportRunStep.tsx`),
entered only once the mapping step's required fields and category are set
(its "Continue" is gated exactly the way the old "Run dry run" button
was). The dry run computes once, from a lazy `useState` initializer, when
the step mounts; there is no in-step "re-run" action any more — changing
the file-wide mapping means going Back to the mapping step and forward
again, which remounts this step and computes a fresh dry run (and drops
any per-row remaps from the previous attempt, along with any that were
never applied). This is a deliberate simplification: the two triggers for
"the dry run needs to change" (a file-wide mapping edit, or a per-row
remap) now have clearly different scopes — one restarts the step, the
other patches one row in place — rather than one "re-run" affordance
having to mean both.

### A failed row can be remapped, per field, for that row alone

Real files sometimes have a single field on a single row that doesn't fit
the file-wide mapping — a date typo, a row exported in a different format
than the rest. Re-mapping the *whole file's* date column to fix one row
would just break every other row that was already parsing correctly, and
manually editing raw file content before re-uploading is a worse
experience than the tool should ask for. So a **failed** row's expansion
(only failed — a suspicious or ok row already produced a valid entry, so
there's nothing to fix) additionally shows one `RemapSelect` per issue the
row actually has:

- `title` issue → a select for which column feeds this row's title.
- `amount` issue → one select (single-amount mode) or two (split debit/
  credit mode) for which column(s) feed this row's amount.
- `booking_timestamp` issue → a select for which column feeds this row's
  date.

(A `category_id` issue is never reachable here — the mapping step already
requires a category before "Continue" enables, so `mapRow` can't produce
that error by the time a row is on screen; no remap control exists for
it.) Each select defaults to the file-wide column already used for that
field, offers every column the file has, and on change calls
`applyOverride(rowMapping, override)` — a small helper that copies the
step's base `RowMapping` and substitutes just the overridden column(s) —
then re-runs `classifyRow` for *that one row* against the file's
already-resolved amount separators (captured once, from the initial dry
run, so a remap can't accidentally pick a different separator reading
than the rest of the file used). The result replaces that row's entry in
local state; every other row, and the summary counts, update from that
one row's new classification with no re-scan of the file. Overrides are
kept in a small `Record<rowIndex, RowOverride>` alongside the rows
themselves, both step-local — "Continue" hands the *current* classified
rows (remaps included) up to the wizard, which is what decides what
actually gets submitted in the run step.

This intentionally doesn't handle every possible failure — a value that's
wrong in every column (not just mis-mapped) still needs a fixed file — but
it directly targets the failure mode a large file most commonly has:
right data, wrong column, on just one or two rows.

Each `RemapSelect` option also shows an example value next to the column
name — but sourced from *this row's own* value for that column, not the
file-wide first-non-empty-value example the mapping step's pickers use.
The two examples answer different questions: the mapping step's asks "what
kind of data is generally in this column," which any row's value answers
equally well; a remap on one specific failing row asks "what does *this*
row actually have in this column," where a different row's value would be
actively misleading (e.g. suggesting a well-formed date when this row's own
value in that column is the very thing that needs checking). The shared
truncation behavior (`lib/import/formatExample.ts`'s `truncateExample`)
is factored out for both call sites to share, rather than duplicated.

### Wizard steps live in the URL, so the browser Back button works

Originally the wizard's step was pure local `useState`, with no
corresponding browser-history entry — so the physical Back button
navigated away from `/entries/import` entirely rather than stepping back
one wizard step, discarding all progress. Fixed by moving the
"account"/"file"/"mapping"/"dryrun" step into a `?step=` search param
(`entries.import.tsx`'s `validateSearch`, defaulting to `"file"` when
`?account_id=` is preset, else `"account"`, and rejecting anything else
back to that default). Forward transitions `navigate({ search: (prev) =>
({ ...prev, step: next }) })` (a push); every in-page "Back" link calls
`window.history.back()` instead of setting local state, so the physical
button and the in-page link are exactly symmetric — both step back
through the same history.

This works only because a search-param-only navigation re-renders
`ImportEntries` in place rather than remounting it: every other piece of
wizard state (the parsed file, the mapping, per-row overrides) is
ordinary local `useState` on that same component, so it survives a step
change completely untouched — the URL carries only *which step to show*,
never the step's data.

"run" and "result" are deliberately excluded from this and kept as a
separate local `runPhase: "run" | "result" | null`, which takes rendering
priority over the URL step. The reason is `ImportRunStep`'s side effects:
it submits `POST /api/entries` from a mount effect, so if a URL-driven
step *ever* caused it to remount, a back/forward navigation could
resubmit entries. Keeping `runPhase` purely local closes that off
entirely — nothing in browser history can set it, so it can only ever be
entered by the dry-run step's own "Continue" handler. The remaining loose
end this creates — a stale run/result view left on screen after a
back/forward navigates the URL step out from under it — is closed by a
`useEffect` that resets `runPhase` to `null` whenever the URL step
changes, dropping back to the ordinary step-driven view instead.

One more edge case this design introduces: a hard page reload lands on
whatever `?step=` the URL last had, but the in-memory `parsed` file (and
everything downstream of it) cannot survive a reload — so a reload on
`step=mapping` or `step=dryrun` would otherwise render blank. A
`useEffect` redirects (`replace: true`) to `"file"` whenever the URL step
requires `parsed` data that isn't there.

### Import step: resolve tags once, then submit rows sequentially with live progress and a cancel action

Before the first row, the batch's tag names are resolved to ids exactly
once (matching existing tags, creating any unmatched ones via
`POST /api/tags` — the same `resolveTagIds` logic `entries.new.tsx`
already has, lifted so import can share it rather than re-typing new tags
per row). Rows are then submitted in file order, one `POST /api/entries`
at a time, updating a `created / failed / total` counter after each. A
Cancel action stops issuing further requests between rows (already-created
entries are not rolled back — this is the same accepted partial-completion
posture as any other skip-and-continue failure). A row rejected by the
backend at this stage (a race like the chosen category having just been
disabled) is added to the same failed-rows list the dry run's failures
appear in, so the results view has one unified list regardless of which
stage caught the problem.

## Risks / Trade-offs

- **[Risk]** Sequential one-request-per-row import is slow for a large
  file (hundreds/thousands of rows), each also paying `internal/entry`'s
  per-create balance-adjustment recompute lookup → **Mitigation**: the
  live progress bar keeps this visible rather than looking hung, and
  Cancel lets a visitor bail out; a backend batch endpoint remains
  available as a future change if this proves to matter in practice (see
  "Client-side only" above).
- **[Risk]** A visitor could set the wrong decimal separator or date
  format and only notice after several bad rows are already created (the
  dry run reduces this risk but can't eliminate it — a value can be
  syntactically valid and still semantically wrong, e.g. day/month
  swapped on a non-ambiguous-looking date) → **Mitigation**: entries
  created via import are ordinary entries, editable/deletable exactly like
  any other, through the existing entry form — no special cleanup path is
  needed or built.
- **[Risk]** A malformed or unexpected file (wrong delimiter, nested JSON,
  no header row despite the on-screen note) produces a confusing mapping
  step → **Mitigation**: the file-select step validates shape up front
  (CSV: at least a header row; JSON: top-level array of flat objects) and
  rejects with a clear message before the visitor ever reaches mapping.

## Migration Plan

Purely additive, frontend-only — no data migration. Rollout is a normal
frontend deploy:
1. Add the `papaparse` dependency.
2. Build the `lib/import/` helpers (`parseFile`, `parseAmount`,
   `parseDate`, `mapRow`) with unit coverage, since they're the
   correctness-critical, framework-free part of this change.
3. Build the wizard components and `entries.import.tsx` route.
4. Wire the two "Import" entry-point links.
5. i18n strings.

Rollback is deleting the route/components/links and the dependency — no
backend or data implications either way.

## Open Questions

None outstanding for this change's scope — client-vs-backend execution,
the CSV library choice (and its export-readiness), amount/date parsing
flexibility, dry-run semantics (ok/suspicious/failed, no per-success
listing), and the two entry points were all resolved during exploration.
Two implementation-level details are deliberately left to task-time rather
than pinned here, since neither changes the shape of anything above: the
exact ambiguous-date heuristic thresholds, and the precise wording of
failure-reason messages.

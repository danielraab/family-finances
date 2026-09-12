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

### JSON input shape: a top-level array of flat objects, nothing else

Only `[{...}, {...}, ...]` is accepted — each element a flat object
(string/number/boolean/null values only, no nested objects/arrays as
mappable fields). Anything else (a bare object, a nested/wrapped shape
like `{ "transactions": [...] }`) is rejected at the file-select step with
a clear error, rather than guessing which array inside an arbitrary
structure was meant. The mappable field list is the union of keys across
a bounded sample of the array (first 200 objects), in first-seen order —
bounded so one pathological file with thousands of distinct keys can't
make the mapping UI unusable; large real-world exports are overwhelmingly
uniform-shaped anyway.

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

### Dry run classifies every row as ok, suspicious, or failed — only non-ok rows are ever listed

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
  called out in the dry-run table so the visitor can look before
  proceeding.
- **ok**: counted only ("312 rows ready"), never listed row by row — the
  point of a dry run over a real bank export is to see what's *wrong*,
  not to re-read every correct row.

The visitor can re-run the dry run after changing any mapping/setting, as
many times as they like, with no network activity — it's a pure function
over already-parsed, in-memory rows.

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

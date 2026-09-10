## Why

`/reports` currently forces a choice between filtering by one category *or*
one tag, and refuses to generate anything until one of them is set. The
backend's `/api/entries` and `/api/entries/summary` already accept
`category_id` and `tag_id` together (AND semantics) and work with neither
set, so the restriction is a self-imposed UI limit. Users can't ask "what
did entries tagged X in category Y sum to?", nor "what's the total for this
account and date range?" without picking a category or tag they don't care
about.

## What Changes

- Selecting a category no longer clears a selected tag, and vice versa —
  both may be set at once and are applied together as an AND filter.
- **BREAKING** (behavioral): "Generate report" is always enabled. Activating
  it with no category and no tag — with or without account/date filters —
  sends the requests and shows all matching transaction entries plus the
  per-currency sum, instead of doing nothing and prompting for a selection.
- The "include subcategories" checkbox shows whenever a category is
  selected, regardless of whether a tag is also selected.
- The pre-generation prompt and the stale/hint copy stop telling the user a
  category or tag is required. The `reports.selectOneHint` and
  `reports.beforeGenerate` i18n keys are reworded in `en.json` and
  `de.json`.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `web-client-reports`:
  - Remove the "Category and tag are an exclusive selection" requirement
    (its clearing behavior and both scenarios).
  - "A report is only generated on explicit action": drop the "no category
    or tag selected → no request is sent" carve-out and reverse the
    "Neither category nor tag selected" scenario so generation proceeds.
  - "A selected category offers an 'include subcategories' checkbox":
    remove the "SHALL NOT be shown while a tag is selected" clause; the
    checkbox is governed solely by whether a category is selected.
  - "The report shows matching entries and a per-currency sum": the
    pre-generation empty-state text no longer references selecting a
    category or tag.

## Impact

- `frontend/src/routes/reports.tsx` — `selectCategory` / `selectTag`
  (stop mutually clearing), `canGenerate` / `generateReport` (no selection
  gate), the "Generate report" button (`disabled` / `title`), the
  `include_subcategories` checkbox visibility condition, and the
  pre-generation hint text.
- `frontend/src/i18n/locales/en.json` and `de.json` — `reports.selectOneHint`,
  `reports.beforeGenerate`.
- `openspec/specs/web-client-reports/spec.md` — via the delta.
- No backend, API contract, or `schema.d.ts` changes: the query params are
  already forwarded and already optional.

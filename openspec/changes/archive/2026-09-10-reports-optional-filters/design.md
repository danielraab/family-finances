## Context

`/reports` is a single client route (`frontend/src/routes/reports.tsx`).
Its category-or-tag exclusivity and its "must select one" generate gate are
enforced entirely in that file:

- `selectCategory(id)` sets `category_id` and clears `tag_id` +
  `include_subcategories`; `selectTag(id)` does the mirror.
- `canGenerate = Boolean(search.category_id || search.tag_id)` drives the
  button's `disabled` and `title`, and `generateReport()` early-returns
  when both are unset.
- The `include_subcategories` checkbox is rendered inside
  `{search.category_id && (...)}`.
- `buildEntriesQuery` / `buildSummaryQuery` already forward both
  `category_id` and `tag_id` unconditionally and already omit them when
  unset. `isStale()` already compares both independently.

The backend `/api/entries` and `/api/entries/summary` params
`category_id`, `tag_id`, `category_mode`, `account_id`, `from`, `to` are
all optional and independent (`openapi/openapi.yaml`), so no contract or
`schema.d.ts` change is needed.

## Goals / Non-Goals

**Goals:**
- Category and tag filters are independent; both, either, or neither may be
  set, combined as AND.
- "Generate report" is always actionable; a zero-filter report returns all
  matching transaction entries + per-currency sum.
- `include_subcategories` checkbox visibility depends only on a category
  being selected.
- Copy no longer claims a category or tag is required.

**Non-Goals:**
- No backend, API-contract, or generated-type changes.
- No change to the explicit-generate model, the stale-results hint
  mechanic, infinite scroll, or the multi-currency sum display.
- No new "all entries" safety confirmation or result cap — a zero-filter
  report is just the existing query with fewer `where` clauses.

## Decisions

### Drop mutual clearing rather than add an "AND/OR" toggle

`selectCategory` / `selectTag` become plain `patchSearch({ category_id })`
/ `patchSearch({ tag_id })` calls. `selectCategory` still clears
`include_subcategories` only when its own value is cleared (keep the
existing reset-on-deselect behavior for that sub-control).

_Alternative considered_: an explicit match-mode control. Rejected —
AND is the only sensible combination for "entries in this category *and*
carrying this tag", and it matches how `/entries` already composes the two
filters.

### `canGenerate` is removed, not hard-coded to `true`

Delete the `canGenerate` binding, the `disabled` prop, the conditional
`title`, and the guard clause in `generateReport()`. Keeps the button
markup honest rather than carrying a permanently-true flag.

### Pre-generation copy

- `reports.beforeGenerate` → a prompt to set filters (optional) and click
  "Generate report", no "a category or a tag" requirement.
- `reports.selectOneHint` is now unused. Remove the key from `en.json` and
  `de.json` and delete its last reference (the button `title`) rather than
  leaving a dead key.

Both `en.json` (source of truth) and `de.json` are updated in the same
change; `de.json` is not left lagging since the German copy already
exists and the reword is small.

## Risks / Trade-offs

- **A zero-filter report can be large** (every transaction entry the user
  can see). → Already bounded by the same cursor pagination / infinite
  scroll `/entries` uses; the summary endpoint returns aggregates only.
  This is the explicit intent of the change.
- **Users who relied on the button being disabled as a "pick something"
  cue.** → The reworded pre-generation text carries the guidance; the
  behavior (nothing happens until you click) is unchanged.
- **Spec drift** if `web-client-reports` isn't updated in lockstep. →
  Delta spec is part of this change; `openspec validate` gates it.

## Migration Plan

Pure frontend behavior change, shipped in the app bundle. No data
migration, no flag. Rollback = revert the commit. Existing `/reports` URLs
remain valid (single-filter URLs behave identically; the new capability is
purely additive).

## Open Questions

None.

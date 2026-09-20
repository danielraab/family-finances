## Why

The shared date-range filter has no way to say "show everything". The
unbounded range itself already exists — pick `Custom` and clear both date
inputs and nothing is filtered, and the collapsed trigger even reads
"All time" (`dateRangeFilter.placeholder`) when it happens. What is missing
is a way to *choose* it.

Reaching it today takes three interactions with two native date inputs:
open the panel, switch to `Custom` (which carries the outgoing preset's
resolved dates straight into the two fields — see
`DateRangeFilter.tsx`'s `selectPreset`), then clear both fields. On
`/entries` that sequence does not even work. The ledger falls back to
"Last 2 weeks" whenever no `range`/`from`/`to` parameter is present
(`entries.index.tsx:65`), and an empty `Custom` range writes no parameter
at all — so clearing both dates silently snaps the ledger back to the last
two weeks rather than showing the whole history. The one page whose
default hides older entries is the one page where "all time" is
unreachable.

So the unbounded range needs its own preset key. It cannot be a "clear
both dates" shortcut: the URL has to distinguish "the visitor explicitly
asked for everything" from "the visitor set nothing, use the page's
default", and an absent `from`/`to` pair cannot carry that difference.

## What Changes

- **Add an `all_time` preset**, last in the dropdown after "This year" and
  before "Custom", resolving to no `from` and no `to`. It encodes in the
  URL as `?range=all_time` exactly like every other preset, so it is
  bookmarkable and survives a reload on a page that defaults to something
  narrower.
- **`resolvePreset` starts returning optional bounds.** Every preset so
  far resolves to two concrete dates; `all_time` is the first that
  resolves to neither, so the return type widens to
  `{ from?: string; to?: string }`. Nothing outside `dateRangePresets.ts`
  calls it.
- **A page with no default shows "All time" selected** instead of
  "Custom" with two empty fields. That is what is actually in effect on
  `/reports`, and the trigger already says so.
- **Clearing the last remaining bound under Custom selects All time**
  rather than dropping back to the page's default. This is the same fix as
  the point above, applied to the other route into the unbounded state, and
  it removes the `/entries` snap-back described in Why.
- **Under All time both date inputs are empty and disabled**, the way they
  are disabled under every other preset.

## Non-goals

- No change to how the backend filters. Presets resolve client-side and
  All time simply omits `from`/`to` from the query, which
  `GET /api/entries` and the summary endpoints already accept.
- No change to the API contract. A dashboard card stores its preset as an
  opaque string (`DashboardCardConfig.range.preset` in
  `openapi/openapi.yaml`), so `all_time` needs no schema edit and no
  regeneration.
- No change to `/entries`' own "Last 2 weeks" default, or to `/reports`'
  no-default behaviour — only to which option the dropdown displays for
  the latter.
- No pagination or performance work. `/entries` already pages at 30 rows,
  and `/reports` already runs unbounded whenever its filter is left empty.

## Capabilities

### Modified Capabilities

- `web-client-date-range-filter`: the preset list gains `All time`, a
  preset resolves to optional rather than required bounds, a page with no
  default shows All time selected, and clearing the last bound under
  Custom selects All time.
- `web-client-reports`: opening `/reports` with no date-range parameter
  shows "All time" selected rather than "Custom" with both fields empty.
  The applied filter is unchanged.

## Impact

- `frontend/src/lib/dateRangePresets.ts` — the `all_time` key, its place
  in `DATE_RANGE_PRESET_KEYS` and `PRESET_I18N_KEYS`, `resolvePreset`'s
  widened return type, and `resolveEffectiveRange`'s no-default branch.
- `frontend/src/components/DateRangeFilter.tsx` — `changeFrom`/`changeTo`
  select `all_time` when the last bound is cleared.
- `frontend/src/i18n/locales/{en,de}.json` — one new key,
  `dateRangeFilter.presets.allTime`.
- No backend, OpenAPI, or generated-artifact changes.
- No breaking changes: every existing `?range=<key>` and `?from`/`?to`
  link resolves exactly as before.

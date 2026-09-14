## Why

`/entries` and `/reports` each reimplement the same `from`/`to` date-range
filter — two raw date inputs plus a byte-for-byte duplicated pair of
UTC-boundary helper functions — and offer no quick way to pick a common
range like "last week" or "this month" without knowing exact calendar dates.
Consolidating this into one shared, preset-aware filter removes the
duplication and makes the common case (recent activity, a specific month) a
single click instead of two manual date entries.

## What Changes

- Add a shared `DateRangeFilter` frontend component/hook offering named
  presets (Today, Last 7/14/30 days, This week, Last week, Last 2 weeks,
  This month, Last month, This year) plus a Custom mode with two
  independently-optional date inputs.
- The From/To inputs stay visible under every preset, populated with that
  preset's resolved dates and disabled; they become editable only under
  Custom.
- Filter state is encoded in the URL as either `range=<preset-key>` or
  `from`/`to`, mutually exclusive — reload/bookmark reproduces the same
  effective range either way. Existing bookmarked `from`/`to` links keep
  working unchanged.
- `/entries` defaults (when no filter params are present at all) to "Last 2
  weeks" (last week + this week); `/reports` keeps its current default of no
  date filter (show all). Neither default is written into the URL — it's a
  fallback used only when resolving the effective range and when deciding
  the dropdown's initial selection.
- Week-based presets (This week, Last week, Last 2 weeks) resolve relative to
  a new per-user **week start** setting (Monday or Sunday, default Monday),
  added to the existing user settings resource alongside language, timezone,
  currency, and decimals.
- Replace the duplicated `toRangeStart`/`toRangeEnd` + raw `<input
  type="date">` pair in `entries.index.tsx` and `reports.tsx` with the new
  shared component.

## Capabilities

### New Capabilities
- `web-client-date-range-filter`: the shared date-range filter UI —
  preset list and their date math, the URL encoding scheme
  (`range` vs `from`/`to`), the always-visible From/To inputs and their
  disabled-under-preset behavior, and how a route supplies its own default.

### Modified Capabilities
- `user-settings`: add a `week_start` field (`monday` | `sunday`, default
  `monday`) to the settings resource, following the same nullable-column /
  resolve-with-default / partial-update pattern as the existing fields.
- `web-client-settings`: the Profile tab gains a sixth control for week
  start.
- `web-client-entries`: the `/entries` date filter is now the shared
  `DateRangeFilter`, defaulting to "Last 2 weeks" when no filter is present.
- `web-client-reports`: the `/reports` date filter is now the shared
  `DateRangeFilter`, keeping its existing no-default (show all) behavior.

## Impact

- **Backend**: `backend/internal/settings/` (`settings.go`, `service.go`,
  `handler.go`), `backend/internal/storage/postgres/settings.go`,
  `backend/internal/storage/memory/settings.go`, a new migration adding the
  `week_start` column, and the `UserSettings`/`UserSettingsUpdate` schemas in
  `openapi/openapi.yaml` (regenerate `backend/openapi.yaml` and
  `frontend/src/api/schema.d.ts` per the contract workflow). The
  `GET /api/entries` and reports query endpoints themselves are unaffected —
  presets resolve to concrete `from`/`to` client-side before the request is
  made.
- **Frontend**: a new shared component/hook (e.g.
  `frontend/src/components/DateRangeFilter.tsx` and
  `frontend/src/lib/dateRangePresets.ts`, mirroring the existing
  `frontend/src/lib/recurrence.ts` preset pattern), edits to
  `frontend/src/routes/entries.index.tsx`, `frontend/src/routes/reports.tsx`,
  and `frontend/src/routes/settings.index.tsx`, plus new i18n keys in
  `frontend/src/i18n/locales/en.json` and `de.json`.
- No breaking changes: existing `from`/`to` bookmarked URLs continue to
  resolve exactly as before.

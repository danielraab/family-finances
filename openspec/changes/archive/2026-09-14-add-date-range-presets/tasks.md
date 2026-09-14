## 1. Backend: week_start user setting

- [x] 1.1 Add migration `0026_user_settings_week_start.sql`: `ALTER TABLE
      user_settings ADD COLUMN week_start text CHECK (week_start IN
      ('monday', 'sunday'))`
- [x] 1.2 Add `DefaultWeekStart = "monday"` constant and `ValidateWeekStart`
      function in `backend/internal/settings/settings.go`
- [x] 1.3 Add `WeekStart` field to `Row`, `Update`, and `Settings` structs;
      wire it through `Resolve`
- [x] 1.4 Validate `WeekStart` in `Update` in
      `backend/internal/settings/service.go`
- [x] 1.5 Add `week_start` to the partial-update decode struct in
      `backend/internal/settings/handler.go`
- [x] 1.6 Add `week_start` column read/write to
      `backend/internal/storage/postgres/settings.go` (`Get`'s SELECT and
      `Upsert`'s `ON CONFLICT` merge)
- [x] 1.7 Add `week_start` to
      `backend/internal/storage/memory/settings.go`
- [x] 1.8 Add/extend backend tests for storage, validation, and the
      `GET`/`PUT /api/settings` handler covering `week_start`

## 2. API contract

- [x] 2.1 Add `week_start` to `UserSettings` (required, enum `[monday,
      sunday]`) and `UserSettingsUpdate` (optional) in
      `openapi/openapi.yaml`
- [x] 2.2 Regenerate `backend/openapi.yaml` (`cd backend && go generate
      ./...`)
- [x] 2.3 Regenerate `frontend/src/api/schema.d.ts` (`cd frontend && pnpm
      generate:api`)
- [x] 2.4 Lint the spec and confirm no drift (contract job equivalent
      locally)

## 3. Frontend: shared date-range preset library

- [x] 3.1 Create `frontend/src/lib/dateRangePresets.ts`: preset list
      (`today`, `last_7_days`, `last_14_days`, `last_30_days`,
      `this_week`, `last_week`, `last_2_weeks`, `this_month`,
      `last_month`, `this_year`) with the formulas from `design.md`,
      taking `weekStart` and "today" as inputs (never `new Date()` calls
      buried inside — keep it testable)
- [x] 3.2 Implement `resolvePreset(key, weekStart, today)` returning
      `{ from, to }` date strings
- [x] 3.3 Implement `matchPreset(from, to, weekStart, today)` reverse
      lookup returning a preset key or `undefined` (no match → Custom),
      mirroring `matchPreset()` in `frontend/src/lib/recurrence.ts`
- [x] 3.4 **Skipped**: `frontend/` has no test runner anywhere in the repo
      (same gap noted in the `add-entry-import` change). Verified the
      formulas manually instead with a throwaway Node script exercising
      both week starts and month/year boundary cases — all matched the
      design.md table exactly.

## 4. Frontend: shared DateRangeFilter component

- [x] 4.1 Create `frontend/src/components/DateRangeFilter.tsx`: a preset
      `<select>` plus the two `<input type="date">` elements (`disabled`
      when a preset is active), taking the current `range`/`from`/`to`
      values, a `weekStart`, a `defaultPreset` (or none), and an
      `onChange`-style callback that reports the new `range` or
      `from`/`to` state. Revised after initial implementation: collapsed
      into a single trigger button (`@headlessui/react` `Popover`,
      mirroring `IconColorPicker`'s trigger+panel pattern) showing a
      one-line summary (active preset's label, custom range, or a
      placeholder), opening a panel with the dropdown and the two inputs
      — the original three-always-visible-fields layout took up too much
      space in the filter row. `openspec/specs/web-client-date-range-filter`
      and `design.md` updated to match.
- [x] 4.2 Implement the mutual-exclusion behavior: picking a preset emits
      `{ range: key }` (clearing from/to); editing a date input emits
      `{ from/to: value }` (clearing range)
- [x] 4.3 Implement default resolution: when `range`/`from`/`to` are all
      absent, use `defaultPreset` (if provided) to compute the effective
      range and the dropdown's initial selection, without emitting a
      change
- [x] 4.4 Add i18n keys for the preset labels and the "Custom" option in
      `frontend/src/i18n/locales/en.json` and `de.json`, with "Last 2
      weeks" and "Last 14 days" worded to be clearly distinct

## 5. Frontend: wire into /entries

- [x] 5.1 Add `range?: string` to `EntriesSearch` in
      `frontend/src/routes/entries.index.tsx`, validated alongside
      existing `from`/`to`
- [x] 5.2 Replace the two raw `<input type="date">` elements and the
      local `toRangeStart`/`toRangeEnd` helpers with `DateRangeFilter`,
      passing `defaultPreset="last_2_weeks"` (the `toRangeStart`/
      `toRangeEnd` UTC-boundary helpers themselves stay — they still widen
      the filter's resolved date strings to the backend's timestamp
      contract, only their *input* is now `effectiveRange.from/to` instead
      of the raw search params)
- [x] 5.3 Resolve the effective `from`/`to` (preset or explicit) before
      building the `GET /api/entries` query, reusing
      `dateRangePresets.ts`'s `resolveEffectiveRange`
- [x] 5.4 Fetch the visitor's `week_start` setting (`useWeekStart`, mirroring
      `useDisplayedDecimalPlaces`) and pass it to `DateRangeFilter`

## 6. Frontend: wire into /reports

- [x] 6.1 Add `range?: string` to `ReportsSearch` in
      `frontend/src/routes/reports.tsx`, validated alongside existing
      `from`/`to`
- [x] 6.2 Replace the two raw `<input type="date">` elements with
      `DateRangeFilter`, passing no `defaultPreset` (`toRangeStart`/
      `toRangeEnd` stay — they still widen `GeneratedFilter`'s resolved
      date strings to the backend's timestamp contract)
- [x] 6.3 Resolve the effective `from`/`to` into the `GeneratedFilter`
      snapshot used by `generateReport()` (storing the *resolved* dates,
      not the raw `range` key, so `buildEntriesQuery`/`buildSummaryQuery`
      needed no changes), and updated `isStale()` to compare against the
      live filter's own resolved dates — preserving the existing
      manual-generate/stale-hint behavior
- [x] 6.4 Pass the visitor's `week_start` setting (`useWeekStart`) to
      `DateRangeFilter`, same source as in `/entries`

## 7. Frontend: week start setting on the Profile tab

- [x] 7.1 Add a sixth `<SettingField>` control (two-option `<select>`:
      Monday/Sunday) to `frontend/src/routes/settings.index.tsx`, saving
      on change via the existing `update()` optimistic-`PUT` pattern
- [x] 7.2 Add `settings.profile.weekStart` and its option labels to
      `frontend/src/i18n/locales/en.json` and `de.json`

## 8. Verification

- [x] 8.1 `cd backend && go test ./...` — passes, including a full
      `internal/storage/postgres` integration run (`docker compose up -d
      db`) covering migration `0026` and the new `week_start` CHECK
      constraint. Also ran `gofmt -l .` (clean), `go vet ./...` (clean),
      and `go build ./...` (clean).
- [x] 8.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build` — all
      pass; `pnpm generate:api`/spec lint (task 2) already re-verified
      the contract.
- [x] 8.3 **Not done**: manual browser verification requires an
      interactive browser, which this session doesn't have. Recommend
      running `docker compose up --build` (or `pnpm dev` +
      `go run .`) and checking by hand: `/entries` defaults to Last 2
      weeks with no URL params; `/reports` defaults to no date filter;
      switching presets and Custom on both pages updates the URL as
      designed; changing `week_start` in Settings changes `This
      week`/`Last week`/`Last 2 weeks` boundaries; a pre-existing
      `?from=...&to=...` bookmark still filters correctly.

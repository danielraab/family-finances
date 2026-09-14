## Context

`frontend/src/routes/entries.index.tsx` and `frontend/src/routes/reports.tsx`
each hold an identical `from?: string` / `to?: string` URL search-param pair,
an identical pair of `toRangeStart`/`toRangeEnd` helpers that widen a plain
`YYYY-MM-DD` string to a UTC start-of-day/end-of-day ISO timestamp, and an
identical pair of raw `<input type="date">` elements. Both routes keep all
filter state in the URL (`Route.useSearch()`/`useNavigate()`), no form
library is used anywhere in the frontend, and no date library
(`date-fns`/`dayjs`/`luxon`) or date-picker library is a dependency — all
date handling today is native `Date`/`Intl`.

The closest existing "preset with a custom fallback" pattern is
`frontend/src/lib/recurrence.ts` (`RECURRENCE_PRESETS`, `CUSTOM_PRESET_KEY`,
`matchPreset()`), consumed by `RecurringTransactionForm.tsx`. This design
follows that shape closely rather than inventing a new one.

User settings (`backend/internal/settings/`) already store four resolved,
per-user preferences (language, timezone, default currency, displayed
decimal places) as nullable columns on a one-row-per-user `user_settings`
table, each with a Go constant default, a `Validate*` function, and identical
plumbing through `Row`/`Update`/`Settings`, the Postgres and in-memory
stores, the `GET`/`PUT /api/settings` handler, and the `UserSettings`/
`UserSettingsUpdate` OpenAPI schemas.

## Goals / Non-Goals

**Goals:**
- One shared frontend component/hook for a preset-aware date-range filter,
  used identically by `/entries` and `/reports`.
- Named presets resolve to concrete dates client-side; the backend's
  `from`/`to` query contract is untouched.
- A new `week_start` user setting drives the week-anchored presets, added
  via the established settings pattern.
- Existing bookmarked `from`/`to` URLs keep working with no migration step.

**Non-Goals:**
- No datetime precision bump — presets and Custom stay date-only, matching
  today's `type="date"` behavior. `booking_timestamp`'s own datetime-local
  input is unaffected.
- No calendar/range-picker UI library — Custom mode stays two independent
  `<input type="date">` elements.
- No live-updating "Today"/"This week" recomputation while a page stays open
  across a day/week boundary — the effective range is resolved once per
  navigation (route load / URL change), consistent with how the rest of the
  app already resolves URL-derived state.
- No change to `/recurring`'s filtering — it has no date-range filter today
  and none is added by this change.

## Decisions

### 1. URL encoding: `range=<key>` XOR `from`/`to`, never both

A preset selection is written as `range=<preset-key>` (e.g.
`range=last_2_weeks`); a Custom selection is written as plain `from`/`to`
(either or both may be absent, matching today's behavior exactly). The two
are mutually exclusive — selecting a preset clears any `from`/`to` from the
URL and vice versa. This keeps resolution unambiguous on load (no
precedence rule needed between `range` and `from`/`to`) and means every URL
that was valid before this change (`?from=...&to=...`) is still valid and
still means exactly what it meant before.

Alternative considered: always store concrete `from`/`to` in the URL, with
`range` as a separate hint for which preset produced them. Rejected because
a stored/shared "Last week" link would then show one fixed calendar week
forever rather than "whatever last week is" when reopened later — the
explicit goal of having presets at all.

### 2. Default range is a per-route fallback, not a URL default

When a route has neither `range` nor `from`/`to` in its URL, it supplies its
own fallback effective range: `/entries` resolves to the "Last 2 weeks"
preset's dates, `/reports` resolves to no filter (both `from`/`to` `undefined`).
Nothing is written into the URL for this — the fallback only affects (a) what
gets sent to the backend, and (b) which dropdown option is shown as selected
initially. The dropdown's selection is always computed by reverse-matching
the current *effective* `from`/`to` against every known preset's resolved
dates (the same role `matchPreset()` plays for recurrence presets); if
nothing matches, "Custom" is shown. This means:
- `/entries` loaded bare shows "Last 2 weeks" selected, dates populated,
  inputs disabled — indistinguishable from having explicitly picked it.
- `/reports` loaded bare shows "Custom" selected with both dates empty,
  which reads correctly as "no filter."
- No dedicated "All time" preset is needed; "Custom" with both fields empty
  already means that, and it's what `/reports` defaults to.

The first user interaction with the control always writes an explicit
`range` or `from`/`to` into the URL, even if the resolved value happens to
match the implicit default — the fallback is never itself round-tripped
through the URL.

### 3. Preset formulas

All "current period" math is relative to the visitor's local today (via the
timezone already resolved from user settings, same as existing date
formatting elsewhere in the app). `currentWeekMonday` below means "the most
recent occurrence of the configured week-start day, on or before today" —
substitute Sunday throughout when `week_start = sunday`.

| Preset key | Start | End |
|---|---|---|
| `today` | today | today |
| `last_7_days` | today − 6d | today |
| `last_14_days` | today − 13d | today |
| `last_30_days` | today − 29d | today |
| `this_week` | currentWeekStart | currentWeekStart + 6d |
| `last_week` | currentWeekStart − 7d | currentWeekStart − 1d |
| `last_2_weeks` | currentWeekStart − 7d | currentWeekStart + 6d |
| `this_month` | 1st of this month | last day of this month |
| `last_month` | 1st of last month | last day of last month |
| `this_year` | Jan 1 this year | Dec 31 this year |

`last_2_weeks` is literally `last_week` concatenated with `this_week` — not
a rolling 14-day window (that's `last_14_days`, a distinct option). Every
preset that includes "this week"/"this month"/"this year" routinely resolves
an `end` date in the future relative to today; this is intentional (future
already-recorded entries — e.g. from recurring transactions — are expected
to show up), not a bug to guard against.

Only `this_week`, `last_week`, and `last_2_weeks` depend on `week_start`;
every other preset is week-start-independent.

### 4. Shared component owns date math; routes own wiring

A new `frontend/src/lib/dateRangePresets.ts` exports the preset list, a
`resolvePreset(key, weekStart, today)` function, and a `matchPreset(from, to,
weekStart, today)` reverse-lookup — pure functions, no component state,
mirroring `recurrence.ts`'s split between pure preset math and the form
component that uses it. A `DateRangeFilter` component takes the route's current `range`/`from`/`to`
search values, its `patchSearch`-style setter, and a `defaultPreset` prop
(`"last_2_weeks"` for entries, `undefined` for reports). It renders as a
single compact trigger button (summarizing the active preset or custom
range, mirroring `IconColorPicker`'s trigger-plus-`@headlessui/react`-
`Popover` pattern) rather than three always-visible fields, so it doesn't
dominate a filter row already holding several other controls; activating it
opens a panel holding the preset dropdown and the two date inputs. Both routes keep their own `validateSearch` additions (`range?:
string`) and their own `patchSearch` wiring, matching how they already
independently own `account_id`/`category_id`/etc. — only the date-specific
logic and markup move into the shared piece.

### 5. Week start: new user setting, following the existing pattern exactly

`week_start`: nullable `text` column (`CHECK (week_start IN ('monday',
'sunday'))`) on `user_settings`, a `DefaultWeekStart = "monday"` constant, a
`ValidateWeekStart` function, wired through `Row`/`Update`/`Settings`/
`Resolve` and both stores exactly like `language`/`timezone`. `UserSettings`/
`UserSettingsUpdate` in `openapi/openapi.yaml` gain the field (required in
the former, optional in the latter); `backend/openapi.yaml` and
`frontend/src/api/schema.d.ts` are regenerated per the existing contract
workflow. The frontend Profile tab (`settings.index.tsx`) gains a sixth
`<SettingField>`, a two-option `<select>` (monday/sunday), saving on change
via the existing `update()` optimistic-PUT pattern — no special side effect
needed (unlike `language`, which also calls `i18n.changeLanguage`).

A literal fixed default (`monday`) is used rather than inferring one from
the browser's locale (e.g. `Intl.Locale().weekInfo`), matching the
established convention for every setting except language, where the
existing browser-language-detector is a deliberate, separate mechanism.

## Risks / Trade-offs

- **`last_2_weeks` vs. `last_14_days` reads as confusingly similar in a
  dropdown despite being different questions** → give them clearly
  distinguishing i18n labels (e.g. "Last 2 weeks (Mon–Sun)" vs. "Last 14
  days") rather than relying on the visitor to infer the difference from
  behavior alone; call this out explicitly during spec/i18n-key drafting.
- **Presets that reach into the future (`this_week`, `this_month`,
  `this_year`, `last_2_weeks`) could surprise a visitor expecting
  date-range filters to always end at "now"** → mitigated by the From/To
  inputs staying visible and populated even under a preset, so the visitor
  can always see exactly what range is actually applied.
- **Reverse-matching effective dates back to a preset is inherently
  fuzzy at day boundaries** (e.g. a custom `from`/`to` that happens to
  exactly equal today's `last_7_days` bounds) → treat an exact date-string
  match as suficient to show that preset selected; this mirrors
  `matchPreset()`'s existing exact-match behavior for recurrence and is a
  cosmetic dropdown-label concern only, never a filtering-correctness one
  (the effective `from`/`to` sent to the backend is identical either way).
- **`week_start` only affects three of ten presets** → acceptable; it's
  still a single well-scoped, previously-nonexistent per-user preference
  that other future week-oriented features (e.g. a weekly flow-summary
  bucket) could reuse rather than duplicate.

## Migration Plan

1. Backend: add the `week_start` migration, domain/store/handler/OpenAPI
   changes, regenerate generated artifacts. Deployable independently —
   purely additive, defaults to `monday` for every existing user.
2. Frontend: add `dateRangePresets.ts` and the `DateRangeFilter`
   component, wire into `/entries` and `/reports`, add the Profile-tab
   control. Deployable together with or after the backend change (the
   settings `GET` already resolves a default when the column is absent
   pre-migration, so ordering isn't strict, but shipping backend first
   avoids a brief window where the frontend requests a field the API
   doesn't yet return).

No rollback complexity: the new column is nullable and additive, and old
`from`/`to`-only URLs remain valid throughout.

## Open Questions

None outstanding — preset list, URL encoding, per-route defaults, and the
week-start setting were all settled during exploration prior to this
proposal.

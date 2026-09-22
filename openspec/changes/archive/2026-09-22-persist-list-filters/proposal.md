## Why

`/entries`, `/recurring`, and `/reports` all lose their applied filters the
moment a visitor navigates away and back (sidebar click, or any other route
change) — each page resets to its hardcoded default. `/entries` already has
a narrow, localStorage-backed exception (`?last=true`), but it only fires on
two specific "save and return" round trips, not on ordinary navigation, and
the other two pages have no persistence at all. Filters a visitor just set
up (an account, a date range, a category) shouldn't have to be re-entered
every time they step away to look at something else.

## What Changes

- `/entries`, `/recurring`, and `/reports` each persist their own current
  filter/search/sort state to `localStorage` (one storage key per page)
  whenever it changes.
- Arriving at any of the three pages with a completely bare URL (no filter
  parameters at all) restores that page's persisted state instead of
  falling back to its hardcoded default — including a plain sidebar click
  from anywhere else in the app.
- An incoming URL that already carries explicit parameters (a bookmark, a
  shared link, an `account_id` link from an account's detail page, the
  browser back button) is left alone and is not overridden by persisted
  state.
- **BREAKING** (internal only): `/entries`'s `?last=true` mechanism —
  the `last` search param, `isPendingLastResolution`, and the dedicated
  resolve-effect — is removed. Its two callers (`entries.$entryId.edit.tsx`,
  `entries.$entryId.self-transfer.tsx`) navigate to plain `/entries` instead;
  the new arrival-restores-last-filters behavior covers the same round trip.
  No externally observable behavior changes for these two flows.
- The persistence logic (write-through on change, restore-once on arrival)
  is implemented once as a shared hook and used by all three pages, rather
  than copied per page.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `web-client-entries`: filter/search/sort state now persists to
  `localStorage` and is restored on any bare arrival at `/entries`, not just
  the `?last=true` round trip, which is removed.
- `web-client-recurring-transactions`: filter state now persists to
  `localStorage` and is restored on any bare arrival at `/recurring`.
- `web-client-reports`: filter state now persists to `localStorage` and is
  restored on any bare arrival at `/reports`; report generation still
  requires the existing explicit "Generate report" action.

## Impact

- `frontend/src/routes/entries.index.tsx` — drop `last`, `PersistedFilters`,
  `isPendingLastResolution`, and the resolve-effect; adopt the shared hook.
- `frontend/src/routes/recurring.index.tsx`,
  `frontend/src/routes/reports.tsx` — adopt the shared hook (new
  persistence for both).
- `frontend/src/routes/entries.$entryId.edit.tsx`,
  `frontend/src/routes/entries.$entryId.self-transfer.tsx` — simplify their
  post-save navigation to plain `/entries`.
- New shared hook under `frontend/src/lib/` (exact name/shape in
  `design.md`).
- New `localStorage` keys `ff:recurring-last-filters` and
  `ff:reports-last-filters`; the existing `ff:entries-last-filters` is
  reused as-is (no migration needed).
- No backend, API, or database changes.

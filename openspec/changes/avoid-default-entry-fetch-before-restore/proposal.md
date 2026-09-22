## Why

When a visitor navigates to bare `/entries`, the page restores the last
persisted filter/search/sort state from `localStorage`. Today that restoration
happens in a React effect after the first render. The entries fetch effect also
runs after that same first render, so the page can issue one `GET
/api/entries` using the default date filter before the saved filter navigation
lands and issues the intended request.

That wastes backend work and briefly asks for data the visitor will not see.
If a last filter exists, the first entries query for that arrival should be the
restored one.

## What Changes

- Teach the shared persisted-list-filter hook to report that a bare-url arrival
  is currently being restored to a non-bare persisted search state.
- While that restore is pending, `/entries` skips entry-list and recurring
  preview requests that would otherwise use the default search state.
- Keep the existing restore semantics: only bare arrivals restore persisted
  state, explicit URLs are never overridden, and same-page "Clear all filters"
  still clears rather than immediately restoring.
- Apply the same fetch guard to `/recurring`, which uses the same persistence
  hook and has the same duplicate request shape on bare arrivals with persisted
  recurring filters.
- Leave `/reports` generation unchanged. Restoring report draft filters still
  must not auto-generate a report.

## Non-goals

- No backend, API contract, persistence, or OpenAPI changes.
- No change to the stored `localStorage` keys or stored search shapes.
- No cross-tab synchronization or validation of stale persisted entity ids.
- No change to the default `/entries` date range when no persisted state
  exists.

## Capabilities

### Modified Capabilities

- `web-client-entries`: a bare `/entries` arrival with persisted filters
  restores those filters before issuing the entries query, avoiding the extra
  default-filter `GET /api/entries`.
- `web-client-recurring-transactions`: the same restore-before-fetch behavior
  applies to `/recurring` for its persisted list filters.

## Impact

- `frontend/src/lib/usePersistedListFilters.ts` — return restore-pending state
  while a bare arrival is being replaced with persisted search params.
- `frontend/src/routes/entries.index.tsx` — skip entries and recurring-preview
  fetches until pending persisted-filter restoration completes.
- `frontend/src/routes/recurring.index.tsx` — skip list/summary/options fetches
  until pending persisted-filter restoration completes.
- Frontend verification only: lint, typecheck, build, and a focused browser or
  network-log check that bare `/entries` with saved filters issues no default
  entries query.

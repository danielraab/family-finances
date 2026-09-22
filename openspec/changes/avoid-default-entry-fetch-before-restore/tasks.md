## 1. Shared persisted-filter hook

- [x] 1.1 Update `usePersistedListFilters` to return `{ restoring: boolean }`
      while a bare initial search is being replaced with a non-bare persisted
      search.
- [x] 1.2 Keep `localStorage` reads and writes guarded with `try`/`catch`, and
      skip the write-through effect while the initial restore is pending.
- [x] 1.3 Preserve existing semantics for explicit URLs, empty persisted state,
      and same-page "Clear all filters".

## 2. Entries route

- [x] 2.1 Use the hook's `restoring` flag in `/entries`.
- [x] 2.2 Skip the main `GET /api/entries` effect while restoration is pending.
- [x] 2.3 Skip the recurring-preview fetch effect while restoration is pending.
- [x] 2.4 Confirm the first entries request after a restored bare arrival uses
      the persisted filter/search/sort state, not the default range.

## 3. Recurring route

- [x] 3.1 Use the hook's `restoring` flag in `/recurring`.
- [x] 3.2 Skip the recurring list/summary/options fetch effect while
      restoration is pending.
- [x] 3.3 Confirm ordinary bare `/recurring` with no persisted state still loads
      immediately.

## 4. Reports route

- [x] 4.1 Keep `/reports` behavior unchanged: restored draft filters do not
      trigger report generation.
- [x] 4.2 Adjust the call site for the hook API change without introducing new
      report fetch behavior.

## 5. Verification

- [x] 5.1 From `frontend/`, run `pnpm lint`.
- [x] 5.2 From `frontend/`, run `pnpm exec tsc --noEmit`.
- [x] 5.3 From `frontend/`, run `pnpm build`.
- [ ] 5.4 Browser-check or instrument network calls: with
      `ff:entries-last-filters` set to a non-bare value, navigating from another
      route to bare `/entries` issues no default-filter `GET /api/entries` and
      then issues one request with the restored query.

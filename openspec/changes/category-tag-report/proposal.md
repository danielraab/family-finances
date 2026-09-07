## Why

There is no way today to answer "how much did I spend on Groceries (or
everything tagged Vacation) in total?" without manually paging through
`/entries` and adding amounts by hand. `/entries` filters by a single
category (always including its whole subtree — there's no way to ask for
just that category) or a single tag, but never computes a total, and
totalling client-side across a cursor-paginated, 200-row-capped list would
silently under-count for anyone with more than a page of matching entries.

## What Changes

- Add a new `/reports` page: pick exactly one of a category (with an
  "include subcategories" checkbox, defaulting to checked) or a tag,
  optionally narrow by account and a date range, then click "Generate
  report" to see the matching transaction entries and a sum per currency.
  Balance adjustments are excluded — they're absolute readings, not
  categorized deltas, so a "total for this category" reading a mix of the
  two would be meaningless.
- Nothing on `/reports` auto-fetches: changing any control only updates
  its draft/URL state. The list and sums only (re)populate when "Generate
  report" is explicitly clicked, including on first load with filters
  already present in the URL.
- Backend: add an explicit exact-category-vs-subtree mode to `GET
  /api/entries`'s `category_id` filter, additive and defaulting to today's
  subtree-always behavior so existing callers (including today's
  `/entries` page) are unaffected.
- Backend: add `GET /api/entries/summary`, a new aggregate endpoint
  accepting the same filter vocabulary as `GET /api/entries` (minus
  sort/cursor) and returning the matching `kind: transaction` entries'
  amounts summed per account currency, computed in SQL rather than by
  paging through rows.

## Capabilities

### New Capabilities
- `web-client-reports`: the `/reports` page — sidebar link, auth gate,
  category-or-tag selector with the subcategories checkbox, account/date
  filters, explicit "Generate report" action, the resulting entry list,
  and the per-currency sum display.

### Modified Capabilities
- `account-entries`: `GET /api/entries`'s `category_id` filter gains an
  explicit exact-vs-subtree mode (additive, default unchanged); a new
  `GET /api/entries/summary` endpoint is added, sharing `List`'s
  category/tag/account filter resolution.

## Impact

- Backend: `internal/entry` (`Filter`, `Service.List`, new
  `Service.Sum`/`Store.Sum`), `internal/storage/memory` and
  `internal/storage/postgres` (new `Sum` implementations), `internal/httpapi`
  routing for the new endpoint.
- API contract: `openapi/openapi.yaml` gains the `category_id` mode
  parameter and the `GET /api/entries/summary` operation; regenerate
  `backend/openapi.yaml` and `frontend/src/api/schema.d.ts` in the same
  change.
- Frontend: new `src/routes/reports.tsx` (+ nested route(s) as needed),
  a `Sidebar` entry, reuse of `flattenCategoryTree`, the entries table
  styling, and the filter-bar patterns from `entries.index.tsx`.

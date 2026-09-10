## Why

The `/categories` page lists a user's whole category tree but gives no sense
of which categories are actually in use. The Tags settings tab already shows
a per-tag `entry_count` for exactly this reason; categories deserve the same
signal, so a user can see at a glance which categories carry entries before
renaming, disabling, or trying to delete one.

## What Changes

- Every `Category` returned by the API (`GET /api/categories` and every other
  category response) gains an `entry_count`: the number of the caller's own
  non-deleted entries whose `category_id` is that category — direct
  references only, not rolled up from descendant categories.
- The Postgres category store computes `entry_count` as a correlated
  subquery folded into the column list every category query already selects
  (mirroring `internal/storage/postgres/tag.go`). The in-memory store
  reports `0`, the same accepted gap it already has for the delete in-use
  check.
- `openapi/openapi.yaml` adds `entry_count` to the `Category` schema
  (required), and both generated artifacts (`backend/openapi.yaml`,
  `frontend/src/api/schema.d.ts`) are regenerated in the same change.
- The `/categories` tree renders each node's entry count as a small badge,
  with new i18n keys in `en.json` and `de.json`.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `entry-categories`: category API responses now include a required
  `entry_count` field reflecting the caller's own non-deleted entries that
  directly reference the category.
- `web-client-categories`: the category tree now displays each node's entry
  count.

## Impact

- **Spec:** `openapi/openapi.yaml` `Category` schema (+ regenerated
  `backend/openapi.yaml` and `frontend/src/api/schema.d.ts`).
- **Backend:** `internal/category/category.go` (domain type gains
  `EntryCount`), `internal/storage/postgres/category.go` (`categoryCols` +
  `scanCategory`), `internal/storage/memory` category store (reports `0`),
  handler/store tests.
- **Frontend:** `frontend/src/routes/categories.tsx` (`renderNode` badge),
  `frontend/src/i18n/locales/{en,de}.json`.
- No database migration — `entries.category_id` already exists; this is a
  read-only derived field.

## Why

`/recurring/{id}/edit` today only shows a bare `linked_entry_count`
number — a visitor who wants to see (or jump to) the actual entries a
recurring transaction generated has no way to do it from here; they'd
have to guess and search the ledger manually, or rely on each entry's own
"linked to a recurring transaction" badge to work backward one at a time.

## What Changes

- Add a `recurring_transaction_id` filter to `GET /api/entries` (backend +
  API contract): scoped on top of the existing caller-visible-accounts
  resolution, so it can't leak entries outside what the caller could
  already see.
- `/recurring/{id}/edit` gains a "Linked transactions" section listing the
  entries linked to that template, newest-first, cursor-paginated via a
  "Load more" action (not scroll-triggered infinite scroll — this is a
  secondary section on an edit page, not the primary ledger) — skipped
  entirely (no fetch) when `linked_entry_count` is `0`.
- Each row links to that entry's own edit page, mirroring how
  `RecurringTransactionBadge` already links the other direction.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `account-entries`: `GET /api/entries` gains the `recurring_transaction_id`
  filter parameter.
- `web-client-recurring-transactions`: `/recurring/{id}/edit` gains the
  paginated linked-transactions list.

## Impact

- `openapi/openapi.yaml` (+ synced `backend/openapi.yaml`,
  `frontend/src/api/schema.d.ts`) — new `recurring_transaction_id` query
  parameter on `GET /api/entries`.
- `backend/internal/entry/entry.go` — `Filter.RecurringTransactionID`.
- `backend/internal/entry/handler.go` — parse the new query param.
- `backend/internal/storage/postgres/entry.go` — `buildWhere` clause.
- `frontend/src/routes/recurring.$id.edit.tsx` — the new list section and
  its cursor-pagination state.

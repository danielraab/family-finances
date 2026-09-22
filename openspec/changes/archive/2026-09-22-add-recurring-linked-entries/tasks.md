## 1. API contract

- [x] 1.1 In `openapi/openapi.yaml`, add the `recurring_transaction_id`
      query parameter to `GET /api/entries`, per `account-entries`'s
      modified requirement.
- [x] 1.2 Sync `backend/openapi.yaml` (`cd backend && go generate ./...`)
      and regenerate `frontend/src/api/schema.d.ts`
      (`cd frontend && pnpm generate:api`); confirm no other diff.

## 2. Backend: recurring_transaction_id filter

- [x] 2.1 In `backend/internal/entry/entry.go`, add
      `RecurringTransactionID *string` to `Filter`.
- [x] 2.2 In `backend/internal/entry/handler.go`'s `parseCommonFilter`,
      parse `recurring_transaction_id` the same way `category_id`/`tag_id`
      are parsed.
- [x] 2.3 In `backend/internal/storage/postgres/entry.go`'s `buildWhere`,
      add the `entries.recurring_transaction_id = ...::uuid` clause when
      `f.RecurringTransactionID != nil`.
- [x] 2.4 Unit tests: filtering by `recurring_transaction_id` returns only
      matching entries; combined with another filter (e.g. `from`/`to`)
      both apply; still scoped to the caller's visible accounts when
      `recurring_transaction_id` names a template on an account the
      caller can't see (empty result, not an error).

## 3. Frontend: linked-transactions list

- [x] 3.1 `recurring.$id.edit.tsx`: add a "Linked transactions" section
      below the existing danger-zone section (or wherever reads best next
      to `linkedEntryCount`), presentational rows (date, title,
      `CategoryLabel`, sign-colored amount via `amountColorClass`/
      `formatAmount`, mirroring `UpcomingBlock.tsx`'s row shape) each
      wrapped in a `Link` to `/entries/{entryId}/edit`.
- [x] 3.2 Fetch the first page (`GET /api/entries?recurring_transaction_id=
      {id}&limit=...`) only when `linked_entry_count > 0`; track `items`
      and `next_cursor` state.
- [x] 3.3 "Load more" button, shown only when `next_cursor` is non-null,
      appending the next page's items on click (no `IntersectionObserver`).
- [x] 3.4 Empty state copy when `linked_entry_count` is `0`.
- [x] 3.5 i18n: add the new `recurring.edit.linkedTransactions.*` keys
      (heading, empty state, "Load more" label) to `en.json` first, then
      `de.json`.

## 4. Verification

- [x] 4.1 Backend: `gofmt -l .`, `go vet ./...`, `go test ./...` (from
      `backend/`), including the postgres integration suite against the
      local compose DB.
- [x] 4.2 Frontend: `pnpm lint`, `pnpm exec tsc`, `pnpm build` (from
      `frontend/`).
- [x] 4.3 Manually exercise, in a real browser against the dev server: a
      template with 0 linked entries shows the empty state and issues no
      list request; a template with linked entries shows them newest
      first with working "Load more" pagination across a page boundary; a
      row's link opens the right entry's edit page.
- [x] 4.4 Confirm no drift: `openapi/openapi.yaml` and
      `backend/openapi.yaml` are byte-identical, and re-running
      `pnpm generate:api` against the current spec produces no further
      diff in `schema.d.ts`.

## 1. API contract

- [x] 1.1 Add `entry_count` (integer, with description) to the `Category`
  schema in `openapi/openapi.yaml` and add it to that schema's `required`
  list.
- [x] 1.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [x] 1.3 `cd frontend && pnpm generate:api` to regenerate
  `frontend/src/api/schema.d.ts`.
- [x] 1.4 Confirm no other drift: `git diff --stat` shows only the three
  contract files changed by this step.

## 2. Backend domain + stores

- [x] 2.1 Add `EntryCount int` with `json:"entry_count"` to
  `category.Category` in `backend/internal/category/category.go`, placed
  last, mirroring `tag.Tag.EntryCount`.
- [x] 2.2 In `backend/internal/storage/postgres/category.go`, extend
  `categoryCols` with the correlated subquery
  `(SELECT count(*) FROM entries e WHERE e.category_id = categories.id AND
  e.deleted_at IS NULL)` and add a trailing `&c.EntryCount` to
  `scanCategory`.
- [x] 2.3 In `backend/internal/storage/memory`'s category store, leave
  `EntryCount` at its zero value and add a short comment noting the parity
  with the memory tag store's `entry_count` gap.

## 3. Backend tests

- [x] 3.1 In `backend/internal/category/handler_test.go`, assert every
  category response body includes `"entry_count":0` on the memory store
  (mirror `tag/handler_test.go`'s `TestHandlerCreateReportsZeroEntryCount`).
- [x] 3.2 Ensure the handler tests' OpenAPI response conformance assertions
  (`internal/openapicheck.AssertResponse`) still pass with the new required
  field.
- [x] 3.3 In `backend/internal/storage/postgres/category_test.go`, add an
  integration test: create a category, categorize N entries under it (and
  some under a child), soft-delete one, assert `List`/`Get` report the
  correct direct, non-deleted `EntryCount` and that the child's entries do
  not count toward the parent.

## 4. Frontend

- [x] 4.1 Add `categories.entryCount` plural keys
  (`categories.entryCount_one` / `categories.entryCount_other`, e.g.
  `"{{count}} entry"` / `"{{count}} entries"`) to
  `frontend/src/i18n/locales/en.json`, then the German equivalents to
  `de.json`.
- [x] 4.2 In `frontend/src/routes/categories.tsx` `renderNode`, render a
  non-interactive neutral chip showing `node.entry_count` after the
  active/disabled status pill, using the count key as its accessible label
  (`title` / visually-hidden text).
- [x] 4.3 Verify the `Category` type from the regenerated `schema.d.ts`
  exposes `entry_count` (no local type cast needed).

## 5. Verification

- [x] 5.1 Backend: `cd backend && gofmt -l . && go vet ./... && go test ./...`.
- [x] 5.2 Frontend: `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 5.3 Manual check: run the app, open `/categories`, confirm each node
  shows its entry count and that recategorizing/deleting an entry updates
  the number on reload.
- [x] 5.4 Update `backend/AGENTS.md` (Categories section) and
  `frontend/AGENTS.md` (Categories section) to mention `entry_count`,
  cross-referencing the tag precedent.

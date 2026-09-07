## 1. Backend: schema

- [x] 1.1 Add migration
  `backend/internal/storage/postgres/migrations/0014_category_ownership.sql`:
  drop the `categories_parent_id_name_key` unique constraint; add
  `owner_id uuid NOT NULL REFERENCES users(id)`,
  `sort_order integer NOT NULL DEFAULT 0`,
  `disabled boolean NOT NULL DEFAULT false`, `deleted_at timestamptz`; add
  an index on `(owner_id, parent_id)`. Greenfield — no backfill.

## 2. Backend: `internal/category` domain + store

- [x] 2.1 `category.go`: add `OwnerID string` (`json:"-"`),
  `SortOrder int` (`json:"sort_order"`), `Disabled bool`
  (`json:"disabled"`), `DeletedAt *time.Time` (`json:"-"`) to `Category`.
  `New`/`Update` stay `ParentID`/`Name` only — `owner_id`, `sort_order`,
  and `disabled` are never client-settable directly.
- [x] 2.2 `store.go`: add `ownerID` as the first parameter to every
  existing `Store` method (`List`, `Get`, `Create`, `Update`, `Delete`,
  `Exists`, `Subtree`); add `SetDisabled(ctx, ownerID, id string, disabled
  bool) (Category, error)`, `MoveUp(ctx, ownerID, id string) (Category,
  error)`, `MoveDown(ctx, ownerID, id string) (Category, error)`. Update
  the doc comment on `ErrInUse`/`Delete` to describe the soft-delete
  behavior.
- [x] 2.3 `service.go`: thread `ownerID` through every method (from
  `auth.UserFromContext` in `handler.go`); `Create` computes the
  append-to-end `sort_order` for the target `(owner_id, parent_id)` group;
  `Update`'s reparent path recomputes `sort_order` for the new parent
  group; add `Disable`/`Enable` wrapping `SetDisabled`, and
  `MoveUp`/`MoveDown` wrapping the store methods.
- [x] 2.4 `handler.go`: remove `requireAdmin` from every category route;
  every handler resolves `ownerID` via `auth.UserFromContext`; extend the
  category response body with `disabled`, `sort_order`; add
  `POST /api/categories/{id}/disable`, `/enable`, `/move-up`, `/move-down`.

## 3. Backend: store implementations

- [x] 3.1 `internal/storage/memory`: update every `CategoryStore` method
  for the new `ownerID` scoping; `Delete` becomes a soft delete (set
  `DeletedAt`, still guarded by the children/entry-reference checks, now
  scoped to non-deleted rows); implement `SetDisabled`, `MoveUp`,
  `MoveDown`; `Create` and the reparent path in `Update` compute
  append-to-end `sort_order` within `(ownerID, parentID)`.
- [x] 3.2 `internal/storage/postgres/category.go`: same scoping —
  `categoryCols` gains `owner_id::text` (internal use only, not returned to
  callers directly beyond the scoping `WHERE`), `sort_order`, `disabled`;
  every query adds `owner_id = $ownerID AND deleted_at IS NULL` (dropped
  for `Subtree`'s recursive CTE only insofar as it should also stay scoped
  to the caller's own tree); `Delete` becomes `UPDATE ... SET deleted_at =
  now() WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL AND NOT
  EXISTS (... non-deleted child ...) AND NOT EXISTS (... non-deleted entry
  ...)`; `Create`'s and the reparent path's `sort_order` computed via a
  `COALESCE(MAX(sort_order) + 1, 0)` subquery scoped to `(owner_id,
  parent_id)` (using `parent_id IS NOT DISTINCT FROM $parent`); `MoveUp`/
  `MoveDown` find the adjacent sibling by `sort_order` (tiebreak `id`) and
  swap both rows' `sort_order` in one statement or a short transaction —
  a no-op (no adjacent sibling) returns the category unchanged rather than
  an error.

## 4. Backend: `internal/entry` category ownership

- [x] 4.1 Extend the existing per-entry ownership check (the one already
  enforcing "a tag on an entry must belong to the entry's owner") to also
  require `category_id`, when present, to belong to the entry's owner —
  same `422` treatment, on both create and update.
- [x] 4.2 Unit + handler tests: creating/updating an entry with a
  `category_id` owned by a different user is rejected (`422`).

## 5. Backend: tests

- [x] 5.1 `internal/category` unit + handler tests: cross-owner access
  reads as `404` on every verb (`GET` single, `PATCH`, `DELETE`,
  `/disable`, `/enable`, `/move-up`, `/move-down`); any authenticated user
  (no `is_admin` check) can fully manage their own categories; disabling a
  category in use leaves referencing entries/children untouched and it
  still appears in `GET /api/categories`; a disabled, unreferenced category
  can still be deleted directly; deleting a category with a live child or a
  live entry reference is rejected (`409`) regardless of `disabled`;
  deleting an eligible category succeeds (`204`) and it disappears from
  `GET /api/categories`; self/descendant reparent cycle rejected (`422`),
  scoped to the caller's own tree; move-up/move-down swap order correctly
  and no-op (still `200`, unchanged order) at either end of the sibling
  list; a new category and a reparented category both land at the end of
  their (new) sibling group.
- [x] 5.2 `internal/storage/postgres` integration tests for the migration
  and every new/changed store method. Written, but **not run** in this
  environment (no Docker/Postgres available here) — they self-skip locally
  without `DATABASE_URL` per `backend/AGENTS.md`; run them for real via
  `docker compose up -d db` + `go test ./internal/storage/postgres/...`, or
  let the `backend-integration` CI job do it on the PR.

## 6. API contract

- [x] 6.1 `openapi/openapi.yaml`: `Category` — add `disabled` (boolean,
  required) and `sort_order` (integer, required). Update the
  `GET`/`POST`/`PATCH`/`DELETE /api/categories...` summaries and
  descriptions to drop every "(admin only)" note and drop their `403`
  responses (no longer reachable). Add
  `POST /api/categories/{id}/disable`, `/enable`, `/move-up`, `/move-down`
  (`200` → `Category`, `401`, `404`), documented with every status code the
  handler can return, matching `/api/accounts/{id}/disable`'s shape.
- [x] 6.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [x] 6.3 `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`.
- [x] 6.4 Add `internal/openapicheck.AssertResponse` assertions to the
  new/changed handler tests; lint the spec with spectral.

## 7. Frontend: sidebar + routing

- [x] 7.1 `frontend/src/components/Sidebar.tsx`: add a "Categories" entry
  to `NAV` (after "Entries"), with its own inline-SVG glyph following the
  existing `HomeGlyph`/`AccountsGlyph`/`EntriesGlyph` pattern.
- [x] 7.2 New route(s) under `frontend/src/routes/categories*.tsx`
  (`/categories`), redirecting an anonymous visitor to `/login` the same
  way `entries.tsx`/`accounts.tsx` do.

## 8. Frontend: categories page

- [x] 8.1 Fetch `GET /api/categories` on mount; build the nested tree from
  the flat list (group by `parent_id`, sort each group's children by
  `sort_order`) — a new tree-building helper alongside (not replacing)
  `frontend/src/lib/categoryTree.ts`'s existing flatten-for-`<select>`
  helper, which the entry form still needs.
- [x] 8.2 Render each node as a mobile-friendly row/card (see `design.md`):
  name, disabled/enabled status, and action buttons — ▲/▼ (disabled at the
  ends of a sibling group), "Move to…", edit/rename, disable/enable
  toggle, delete (disabled client-side when the node has any child or any
  known entry reference — derive this from the fetched tree; a
  `used_by_entries`-style signal isn't added to the API, mirror
  `account-types-settings-tab`'s reactive-delete-error decision: attempt
  the delete and surface a `409` inline if the client-side guess was
  stale).
- [x] 8.3 Create form/button (name + parent picker, defaulting to root or
  the currently viewed node) calling `POST /api/categories`; rename calling
  `PATCH /api/categories/{id}` with `name`.
- [x] 8.4 ▲/▼ buttons call `POST /api/categories/{id}/move-up` /
  `/move-down` and refresh the affected siblings' order from the response
  (or a full re-fetch, kept simple).
- [x] 8.5 "Move to…" opens a picker (e.g. a `@headlessui/react` `Dialog` or
  `Listbox`, matching this codebase's existing component choices) listing
  the caller's tree plus a "make root" option; selecting calls
  `PATCH /api/categories/{id}` with `parent_id`; a `422` (cycle) surfaces
  as an inline error without mutating the displayed tree.
- [x] 8.6 Disable/enable toggle calls `POST /api/categories/{id}/disable`
  or `/enable`.
- [x] 8.7 Delete goes through a `@headlessui/react` `Dialog` confirmation,
  same pattern as the Users/Account Types settings tabs; a `409` surfaces
  as an inline error rather than removing the node from the list.

## 9. Frontend: entry form category picker

- [x] 9.1 `frontend/src/routes/entries.new.tsx` /
  `entries.$entryId.edit.tsx`: filter the flattened category list (from
  `frontend/src/lib/categoryTree.ts`) to non-disabled categories for a new
  selection; when editing an entry whose current `category_id` is
  disabled, keep it in the list as a visibly-disabled, non-reselectable
  option so the current value still renders — same visual treatment
  `AccountForm.tsx` uses for a disabled account type, but **not** its
  reselection-forcing validation: leaving the field untouched and saving
  other changes stays allowed (see the corrected `specs/web-client-entries`
  and `specs/account-entries` deltas — a disabled category, unlike a
  disabled account type, never blocks an otherwise-unrelated edit).

## 10. i18n

- [x] 10.1 Add new keys to `frontend/src/i18n/locales/en.json` first, then
  `de.json`: sidebar label, page title/empty state, create/rename form
  labels, move/reorder/disable/enable/delete action labels and confirm
  copy, the "category is disabled — choose another" hint on the entry
  form.

## 11. Verify

- [x] 11.1 `cd backend && gofmt -l . && go vet ./... && go test ./...` — all
  pass. The `internal/storage/postgres` integration tests did **not** run
  against a real Postgres in this environment (no Docker daemon available
  here); they self-skip without `DATABASE_URL`. Run them for real before
  merging (`docker compose up -d db` + `go test
  ./internal/storage/postgres/...`), or via the `backend-integration` CI job.
- [x] 11.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build` — all
  pass (`pnpm generate-routes` was also needed once, for the new route).
- [x] 11.3 Manual pass on `/categories` in a real browser against the real
  backend — **not done**: no Postgres available in this environment to run
  the backend against. What *was* verified here: the built frontend
  compiles/type-checks/lints clean, and a headless-browser check of
  `/categories` against the Vite dev server (no backend reachable) shows
  the auth-gate redirect to `/login` working with no console/page errors.
  Still needs a real pass against a running backend: create a small
  multi-level tree; reorder siblings; reparent via "Move to…" (including a
  rejected cycle attempt); disable a category and confirm it's excluded
  from the entry form's picker for a new selection while an entry already
  on it still shows it read-only and unrelated edits still save; confirm
  delete is disabled on any category with children and enabled once none
  apply, with a live `409` (entry-referenced) surfacing inline; try it on a
  narrow (mobile) viewport.
- [x] 11.4 Update `backend/AGENTS.md`'s (new "Categories" section) and
  `frontend/AGENTS.md`'s routing section + new "Categories" section.

## 12. Spec sync

- [x] 12.1 Apply this change's `specs/entry-categories`, `specs/
  account-entries`, and `specs/web-client-entries` deltas onto
  `openspec/specs/` by hand (the `openspec` CLI is unavailable in this
  environment, as for prior changes); add `specs/web-client-categories` as
  a new capability under `openspec/specs/`.

## 1. Backend: schema

- [x] 1.1 Add migration
  `backend/internal/storage/postgres/migrations/0017_tag_disabled.sql`:
  `ALTER TABLE tags ADD COLUMN disabled boolean NOT NULL DEFAULT false`.

## 2. Backend: `internal/tag` domain + store

- [x] 2.1 `tag.go`: add `Disabled bool` and `EntryCount int` to `Tag`
      (`json:"entry_count"`).
- [x] 2.2 `store.go`: add `SetDisabled(ctx, ownerID, id string, disabled
      bool) (Tag, error)` and `Usable(ctx, ownerID string, tagIDs []string)
      (bool, error)` — reports whether every id in `tagIDs` exists, belongs
      to `ownerID`, and is not disabled (empty slice → `true`, matching
      `OwnedBy`'s existing empty-slice behavior). `OwnedBy` itself is
      unchanged.
- [x] 2.3 Implement `SetDisabled` and `Usable` in `internal/storage/memory`
      and `internal/storage/postgres`. Postgres: fold `disabled` and a
      correlated subquery `entry_count` (`(SELECT count(*) FROM entry_tags
      et JOIN entries e ON e.id = et.entry_id AND e.deleted_at IS NULL
      WHERE et.tag_id = tags.id)`) into the shared `tagCols` expression used
      by `List`/`Get`/`Create`/`Update`/`SetDisabled`, so every one of them
      returns an accurate, live count with no separate query. In-memory:
      `EntryCount` always reads `0` (no entries visibility, same accepted
      gap as `memory.CategoryStore`'s in-use check — note it in a doc
      comment mirroring `category.go`'s).

## 3. Backend: `internal/tag` service + handler

- [x] 3.1 `service.go`: add `Disable`/`Enable` (wrapping `store.
      SetDisabled`) and `Usable` (delegating to `store.Usable`,
      structurally satisfying `internal/entry`'s extended `TagLookup`).
- [x] 3.2 `handler.go`: add `POST /api/tags/{id}/disable` and
      `POST /api/tags/{id}/enable`, no admin gate (every tag route is
      already caller-scoped).

## 4. Backend: `internal/entry` — disabled-tag rule

- [x] 4.1 `store.go`: add `Usable(ctx context.Context, owner string, tagIDs
      []string) (bool, error)` to `TagLookup`, alongside the existing
      `OwnedBy`.
- [x] 4.2 `service.go`: `Create` calls `s.tags.Usable(ctx, ownerID, in.
      TagIDs)` instead of `OwnedBy` (every tag id is new at creation).
      `Update` keeps its existing `s.tags.OwnedBy(ctx, ownerID, *upd.
      TagIDs)` call for the whole resubmitted array unchanged, then — when
      that array is non-empty — computes the ids present in it but not in
      `current.TagIDs` (a small local `diff` helper) and, only if that set
      is non-empty, calls `s.tags.Usable(ctx, ownerID, newlyAdded)`;
      `!ok` on either call returns `ErrInvalidValue` (no new sentinel,
      matching the disabled-category convention: `400`, not `422`).
- [x] 4.3 Update the hand-rolled `TagLookup` fakes (`newStubTags()` in
      `internal/entry/handler_test.go` and
      `internal/entry/service_scenarios_test.go`) to implement `Usable`.
- [x] 4.4 Service tests: creating an entry with a disabled `tag_ids` entry
      is rejected (`ErrInvalidValue`); an entry whose current tags include
      a since-disabled one, updated on an unrelated field only (no
      `tag_ids` in the request), succeeds and keeps every tag; the same
      entry's owner resubmitting the full, unchanged `tag_ids` array
      (including the disabled id) succeeds; adding a *different* disabled
      tag id alongside the untouched existing one is rejected and neither
      the tags nor the rest of the update are applied.

## 5. API contract

- [x] 5.1 `openapi/openapi.yaml`: `Tag` — add `disabled` (boolean,
      required) and `entry_count` (integer, required) to `properties` and
      `required`. Add paths `POST /api/tags/{id}/disable` and
      `POST /api/tags/{id}/enable` (`200` → `Tag`, `401`, `404`), matching
      `/api/categories/{id}/disable`'s shape. `TagWrite` is unchanged
      (still just `name`).
- [x] 5.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [x] 5.3 `cd frontend && pnpm generate:api` to regenerate
      `src/api/schema.d.ts`.
- [x] 5.4 Add `internal/openapicheck.AssertResponse` assertions to the
      new/changed handler tests; lint the spec with spectral.

## 6. Backend: handler + Postgres integration tests

- [x] 6.1 `internal/tag` handler tests: disable/enable happy path (`200`,
      `disabled` flips), disable/enable on another user's or a nonexistent
      tag id (`404`); `entry_count` present (`0`) on create.
- [x] 6.2 `internal/storage/postgres/tag_test.go`: `SetDisabled`, `Usable`
      (owned + not-disabled true; foreign id false; disabled id false;
      mixed set false), and `entry_count` reflecting real attached entries
      — including that a soft-deleted entry no longer counts.

## 7. Frontend: Tags settings tab

- [x] 7.1 `frontend/src/routes/settings.tags.tsx` (new route,
      `/settings/tags`, no admin gate — mirroring `settings.account-
      types.tsx` but with a single `name` field, no description). Fetch
      `GET /api/tags` on mount; render a table (name, entry count, status)
      with per-row Rename / Disable-or-Enable / Delete actions and a create
      form.
- [x] 7.2 Add "Tags" to the tab nav in `frontend/src/routes/settings.tsx`,
      alongside Common, My Invitations, and Account Types (open to every
      authenticated visitor).
- [x] 7.3 Create/rename go through `POST`/`PATCH /api/tags`. Disable and
      enable call `POST /api/tags/{id}/disable` / `/enable` directly on
      click, with **no** confirmation dialog (settled for this change,
      unlike the Account Types tab). Delete is confirmed via the same
      `@headlessui/react` `Dialog` pattern as elsewhere in `/settings`,
      with copy noting the tag will be removed from every entry that
      carries it; `DELETE /api/tags/{id}` always succeeds (`204`), so no
      inline-error-on-409 handling is needed (unlike Account Types).

## 8. Frontend: entry form

- [x] 8.1 `frontend/src/routes/entries.new.tsx` and `entries.$entryId.
      edit.tsx`: pass `existingTags={tags.filter((t) => !t.disabled)}` to
      `TagInput` instead of the raw `tags` state — suggestions exclude
      disabled tags. Leave `resolveTagIds()` (and every other use of the
      unfiltered `tags` state) untouched, so an edited entry's existing,
      since-disabled tag still resolves and resubmits correctly.

## 9. i18n

- [x] 9.1 Add new keys to `frontend/src/i18n/locales/en.json` first, then
      `de.json`: `settings.tabs.tags` label, `settings.tags.*` (heading,
      name field label, create/creating, table column headers — name/
      entries/status, edit/save/saving/cancel, disable/enable, delete,
      status labels, delete-confirmation title/body/confirm-action,
      create/edit error text, empty-state text).

## 10. Verify

- [x] 10.1 `cd backend && gofmt -l . && go vet ./... && go test ./...` —
      clean. `internal/storage/postgres` integration tests (`SetDisabled`,
      `Usable`, `entry_count`) are written in `tag_test.go` but were **not**
      run against a real Postgres in this environment (no reachable
      `DATABASE_URL`/Docker daemon here); they self-skip without one, per
      `backend/AGENTS.md`, and are exercised for real in CI's
      `backend-integration` job.
- [x] 10.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build` —
      clean (route tree regenerated via `pnpm generate-routes` first).
- [x] 10.3 Manual pass: create/rename/disable/enable/delete a tag on the
      new tab; confirm entry counts update as entries are tagged/untagged/
      deleted; confirm the entry form's tag suggestions exclude disabled
      tags while an entry that already carries one still displays and
      saves it; confirm disable/enable apply with no confirmation step and
      delete does. **Not done** — this environment has no reachable
      Postgres to run the full stack against for a real click-through;
      left for the user (or a CI/preview build) to verify.
- [x] 10.4 Update `backend/AGENTS.md` (new "Tags" section, mirroring
      "Categories"/"Account types") and `frontend/AGENTS.md` (the
      "Settings" section's tab list and route table) so their descriptions
      of `internal/tag` and the settings tabs don't go stale.

## 11. Spec sync

- [x] 11.1 Apply this change's `specs/entry-tags`, `specs/account-entries`,
      `specs/web-client-settings`, and `specs/web-client-entries` deltas
      onto `openspec/specs/` by hand (the `openspec` CLI is unavailable in
      this environment, as for prior changes).

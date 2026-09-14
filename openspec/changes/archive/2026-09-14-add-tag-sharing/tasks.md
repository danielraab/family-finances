## 1. Database

- [x] 1.1 Add migration `0023_tag_sharing.sql` creating `tag_shares`
  (`id`, `tag_id` references `tags(id)`, `user_id` references `users(id)`,
  `permission` text CHECK IN `('view','append')`, `granted_by` references
  `users(id)`, `created_at`, `updated_at`, UNIQUE `(tag_id, user_id)`),
  structurally identical to `category_shares`
  (`0022_category_sharing.sql`), plus an index on `user_id`
  (`tag_shares_user_id_idx`).

## 2. Backend domain model (`backend/internal/tag/`)

- [x] 2.1 `tag.go`: add `Permission` type (`view`/`append`/`owner`) with
  the same `permissionRank`/`AtLeast`/`valid` shape as
  `category.Permission` (only `view`/`append` shareable); add
  `Permission`, `Shared`, `OwnerID`, `OwnerName` fields to `Tag`; add
  `TagShare` and `ShareResult` structs mirroring `category.CategoryShare`/
  `ShareResult`.
- [x] 2.2 `store.go`: add `GetForCaller` (owner or shared-viewer,
  populating permission/shared/owner_name), share-management methods
  (`ListShares`, `CreateOrUpdateShare`, `UpdateSharePermission`,
  `DeleteShare`), and a way to check "has any active share" for the
  delete guard — mirror `category.Store`'s equivalents.
- [x] 2.3 `service.go`: add `UserLookup`/`Mailer` interfaces,
  `WithMailer`/`WithBaseURL` options, and `NewService(store, opts...)`
  signature change; add `InviteShare`, `ListShares`,
  `UpdateSharePermission`, `RevokeShare` (owner-manages-anyone /
  any-non-owner-self-leaves authorization, mirroring
  `category.Service.RevokeShare` exactly per design.md); add
  `SetUserLookup`.
- [x] 2.4 `service.go`: change `Delete` to reject (`ErrInUse` or
  equivalent) when the tag currently has any active share; unshared-tag
  delete stays unconditional/hard-delete-cascade, unchanged.
- [x] 2.5 `service.go`: change `entry_count` computation to be scoped to
  the viewing caller (owner or share recipient), not unconditionally the
  owner.
- [x] 2.6 `handler.go`: add `GET /api/tags/{id}` (single-fetch) and
  `GET`/`POST /api/tags/{id}/shares`,
  `PATCH`/`DELETE /api/tags/{id}/shares/{userId}`.

## 3. Backend storage (`backend/internal/storage/postgres/`)

- [x] 3.1 `tag.go`: port `categoryCols`/`categoryViewCols`'s
  caller-parameterized SQL pattern to tags (`tagCols`/`tagViewCols`);
  update `List`/`Get` to use caller-scoped views including shared tags;
  add `GetForCaller`.
- [x] 3.2 `tag.go`: implement `ListShares`, `CreateOrUpdateShare`,
  `UpdateSharePermission`, `DeleteShare`, and the has-active-share check
  backing the delete guard — mirror `category.go`'s equivalents.
- [x] 3.3 `storage/memory`: mirror the same additions for the in-memory
  test store used by domain-package unit tests.

## 4. Backend mailer and wiring

- [x] 4.1 `backend/internal/mailer/mailer.go`: add `SendTagShare(ctx,
  addr, tagName, granterName, permission, link)`, alongside
  `SendAccountShare`/`SendCategoryShare`.
- [x] 4.2 `backend/main.go`: change `buildTag` to accept
  `(pool, mail, baseURL)` like `buildCategory`; call
  `tagSvc.SetUserLookup(authSvc)` after `buildAuth`, same as
  `categorySvc.SetUserLookup(authSvc)`.

## 5. API contract

- [x] 5.1 Update `openapi/openapi.yaml`: `Tag` schema gains `permission`,
  `shared`, `owner_name`; add `TagPermission`, `TagShare`,
  `TagShareInvite`, `TagShareInviteResult` schemas (mirror the `Category`
  equivalents); add `GET /api/tags/{id}`; add
  `/api/tags/{id}/shares` (GET, POST) and
  `/api/tags/{id}/shares/{userId}` (PATCH, DELETE); document the `409`
  on `DELETE /api/tags/{id}` when shared.
- [x] 5.2 Run `cd backend && go generate ./...` to sync
  `backend/openapi.yaml`.
- [x] 5.3 Run `cd frontend && pnpm generate:api` to regenerate
  `frontend/src/api/schema.d.ts`.
- [x] 5.4 Add response-conformance assertions
  (`internal/openapicheck.AssertResponse`) to the new handler tests, per
  `backend/AGENTS.md`.

## 6. Backend tests

- [x] 6.1 Add `backend/internal/tag/sharing_test.go` and
  `sharing_handler_test.go`, mirroring `backend/internal/category/
  sharing_test.go` / `sharing_handler_test.go` (permission tiers, invite
  match/no-match/self-share-rejected, re-share updates existing share,
  revoke/self-leave authorization, owner-cannot-self-leave).
- [x] 6.2 Add `backend/internal/storage/postgres/tag_sharing_test.go`
  mirroring `category_sharing_test.go`.
- [x] 6.3 Add/extend tests for: delete rejected while shared, delete
  succeeds once unshared; caller-scoped `entry_count` for a share
  recipient; entry tagging with an `append`-shared tag succeeds, with a
  `view`-shared tag is rejected (`422`).

## 7. Frontend: shared components and API layer

- [x] 7.1 Create `frontend/src/components/TagLabel.tsx`: renders the
  tag name plus, when `shared: true`, an always-visible shared icon with
  `owner_name` in a hover/title tooltip (not standing text) — see
  design.md's TagLabel decision.
- [x] 7.2 Verify `frontend/src/api/schema.d.ts` (regenerated in 5.3)
  exposes the new endpoints/types; add any thin API-client wrapper
  functions consistent with how `categories.tsx`/`categories_.$categoryId
  .sharing.tsx` call the category endpoints.

## 8. Frontend: `/tags` page

- [x] 8.1 Create `frontend/src/routes/tags.tsx`: own auth gate (mirror
  `categories.tsx`'s, since it's no longer nested under `/settings`);
  fetch `GET /api/tags`; render the caller's own tags as a flat list
  (name, `entry_count`, Active/Disabled status) with create/rename
  (dialog)/disable-enable (immediate)/delete (confirmed, surfaces `409`
  inline when shared) actions; add a Share action per row linking to
  `/tags/{id}/sharing`.
- [x] 8.2 In `tags.tsx`, add the "Shared with me" section below the
  owned list: flat rows via `TagLabel`, permission badge, no
  `entry_count`, Leave action (confirmed) and a link to
  `/tags/{id}/sharing`; omit the section entirely when empty.
- [x] 8.3 Delete `frontend/src/routes/settings.tags.tsx`.
- [x] 8.4 Regenerate `routeTree.gen.ts` (via the dev server or
  `pnpm build`) after adding/removing routes — never hand-edit it.

## 9. Frontend: `/tags/{id}/sharing` page

- [x] 9.1 Create `frontend/src/routes/tags_.$tagId.sharing.tsx`
  (trailing-underscore from the start, per design.md's route-naming
  decision), mirroring `categories_.$categoryId.sharing.tsx`: fetch
  `GET /api/tags/{id}/shares`; owner row first/unremovable; non-owner
  read-only view with only their own Leave; owner sees invite form
  (email + view/append selector, unmatched-email nudge respecting
  `invite_allowed`), per-row permission editor (immediate) and Revoke
  (confirmed).

## 10. Navigation and Settings cleanup

- [x] 10.1 `frontend/src/components/Sidebar.tsx`: add a "Tags" entry to
  `NAV` (positioned alongside "Categories"), add a bespoke inline SVG
  glyph function and its `GLYPHS` map entry, following the existing
  per-item glyph pattern.
- [x] 10.2 `frontend/src/routes/settings.tsx`: remove the
  `/settings/tags` tab from the `tabs` list.

## 11. i18n

- [x] 11.1 Add `nav.tags` and any new `tags.*`/`tags.sharing.*` keys
  (mirroring the `categories.*`/`categories.sharing.*` namespace shape)
  to `frontend/src/i18n/locales/en.json`; add German equivalents to
  `de.json` where practical (non-blocking in CI if it lags).
- [x] 11.2 Remove now-unused `settings.tabs.tags`/`settings.tags.*` keys
  if nothing else references them (check before removing).

## 12. Verification

- [x] 12.1 Backend: `gofmt -l .`, `go vet ./...`, `go test ./...` from
  `backend/`.
- [x] 12.2 Frontend: `pnpm lint`, `pnpm exec tsc`, `pnpm build` from
  `frontend/`.
- [x] 12.3 Manually exercise the golden path in a browser: create a tag
  on `/tags`, share it with a second account at `view` then `append`,
  confirm entry-form selectability matches the tier, confirm delete is
  blocked while shared and succeeds after revoke, confirm the shared
  icon/hover-owner and omitted entry_count on the recipient's "Shared
  with me" row.

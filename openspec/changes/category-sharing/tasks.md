## 1. Backend: schema

- [ ] 1.1 Add migration `backend/internal/storage/postgres/migrations/
  00NN_category_sharing.sql` (next available number after the current
  latest): `CREATE TABLE category_shares (id, category_id FK, user_id FK,
  permission CHECK IN (view, append), granted_by FK, created_at,
  updated_at, UNIQUE (category_id, user_id))` + index on `user_id`.

## 2. Backend: `internal/category` — sharing model and store

- [ ] 2.1 Add `Permission` type (`view`, `append`) with `AtLeast`,
  `CategoryShare` type, and `ShareResult` type to `category.go`, mirroring
  `account.go`'s `Permission`/`AccountShare`/`ShareResult` shapes. Add
  `Permission`/`Shared`/`OwnerName` fields to `Category`.
- [ ] 2.2 `Store`: `GetForCaller(ctx, callerID, id) (Category, error)`
  (real ownership → `Permission: "owner"`; else a matching share's
  permission; else `Permission: ""`, not itself an error — mirrors
  `account.Store.Get`'s convention); widen `List(ctx, callerID)
  ([]Category, error)` to owned (full tree) ∪ shared-with (flat rows).
  Leave the existing owner-scoped `Get`/`Exists`/`Subtree` as-is — they
  keep backing parent validation and reparent-cycle detection, which stay
  strictly within the caller's own tree.
- [ ] 2.3 `Store`: `CreateOrUpdateShare(ctx, categoryID, userID,
  permission, grantedBy) (CategoryShare, error)`, `ListShares(ctx,
  categoryID) ([]CategoryShare, error)`, `ShareByUser(ctx, categoryID,
  userID) (CategoryShare, error)` (`ErrNotFound` if none),
  `UpdateSharePermission(ctx, categoryID, userID, permission)
  (CategoryShare, error)`, `DeleteShare(ctx, categoryID, userID) error`.
- [ ] 2.4 Implement all of the above in `internal/storage/memory` and
  `internal/storage/postgres`.

## 3. Backend: `internal/category` — sharing service + authorization

- [ ] 3.1 New `UserLookup` interface (`ByEmail`, `InviteEnabled`) and
  `Mailer` interface (`SendCategoryShare`), plus `Option`s
  (`WithUserLookup`, `WithMailer`, `WithBaseURL`) — mirroring
  `account.go`/`account/service.go`'s versions; `*auth.Service` and
  `internal/mailer` satisfy them structurally.
- [ ] 3.2 `Service.Usable(ctx, callerID, id) (bool, error)`: widen from
  "owned by callerID and not disabled" to "owned by callerID, or shared
  with callerID at `append`, and not disabled" — via `GetForCaller` +
  `Permission.AtLeast(PermissionAppend)` + `!Disabled`.
- [ ] 3.3 New `Service.FilterSubtree(ctx, callerID, id) ([]string,
  error)`: for an owned category, delegates to the existing `Subtree`;
  for a category visible to callerID only via a share, returns `[id]`
  (no cascade); for one invisible to callerID entirely, returns empty —
  used only by `internal/entry`'s category-filter resolution (wired in
  main.go, see 6.x), never for reparent-cycle detection.
- [ ] 3.4 `Service.List`/`Get` widen to owned-or-shared per the widened
  `Store` methods (`Get` becomes `GetForCaller` under the hood, or the
  existing `Get` is repointed — pick whichever keeps the owner-scoped
  parent-validation call sites, which must stay narrow, unambiguous).
- [ ] 3.5 `Service.ListShares(ctx, callerID, categoryID)
  ([]CategoryShare, error)`: requires `GetForCaller(...).Permission !=
  ""` (`ErrNotFound` otherwise).
- [ ] 3.6 `Service.InviteShare(ctx, callerID, callerName, categoryID,
  email, permission) (ShareResult, error)`: requires `category.OwnerID
  == callerID` (`ErrForbidden` otherwise — there is no shared-owner tier
  to also admit here, unlike accounts); rejects `email` matching the
  caller (`ErrInvalidValue`); on a `UserLookup.ByEmail` match,
  `store.CreateOrUpdateShare` + send the notification email (task 4.2);
  on no match, `ShareResult{Matched: false, InviteAllowed:
  users.InviteEnabled()}`, no store write.
- [ ] 3.7 `Service.UpdateSharePermission`/`RevokeShare(ctx, callerID,
  categoryID, targetUserID, ...)`: real-owner-only (`category.OwnerID ==
  callerID`) required unless `callerID == targetUserID` (self-leave, any
  tier, no owner check).
- [ ] 3.8 `handler.go`: `GET`/`POST /api/categories/{id}/shares`,
  `PATCH`/`DELETE /api/categories/{id}/shares/{userId}`, mirroring
  `account.Handler`'s routes and body shapes with two permission values
  instead of four.
- [ ] 3.9 `Category` JSON response gains `permission` (the caller's own
  effective tier: `owner`/`view`/`append`) and, only when the caller
  isn't the real owner, `shared: true` + `owner_name`.
- [ ] 3.10 Unit + handler tests: invite matched/unmatched (with
  `invite_allowed` both true/false), invite rejected for the caller's own
  email, re-sharing an already-shared email updates in place, a
  non-owner (including one with `append`) forbidden from invite/patch/
  revoke, self-leave works at either tier, list visible at `view`+ and
  `404` for no access, `Usable` true for an owned category and for an
  `append`-shared one, false for a `view`-only shared one and for a
  disabled shared one, `FilterSubtree` returns `[id]` for a shared
  category and the full recursive subtree for an owned one,
  `entry_count` on a shared category reflects only the viewer's own
  entries.

## 4. Backend: cross-package wiring

- [ ] 4.1 `internal/mailer`: `SendCategoryShare` (category name,
  granter's name, permission, application link) — mirrors
  `SendAccountShare`.
- [ ] 4.2 `main.go`: `category.WithUserLookup(authSvc)`,
  `category.WithMailer(...)`, `category.WithBaseURL(...)`; re-wire
  `entry.NewService`'s `CategoryLookup.Subtree` implementation to call
  `categorySvc.FilterSubtree` instead of `categorySvc.Subtree` (the
  `CategoryLookup` interface itself, and everything else in
  `internal/entry`, stays unchanged).

## 5. API contract

- [ ] 5.1 `openapi/openapi.yaml`: `CategoryPermission` (enum `view`,
  `append`), `CategoryShare`, `CategoryShareInvite`,
  `CategoryShareInviteResult`, `CategorySharePermissionUpdate` — mirror
  the `Account*` schemas of the same shapes; `Category` gains
  `permission`, `shared`, `owner_name` (nullable/optional, matching
  `Account`'s); paths `GET`/`POST /api/categories/{id}/shares`,
  `PATCH`/`DELETE /api/categories/{id}/shares/{userId}`; `POST .../
  shares` response covers both matched (`201` → `CategoryShareInviteResult`)
  and unmatched (`200` → same schema, `matched: false`) cases.
- [ ] 5.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [ ] 5.3 `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`.

## 6. Frontend: shared badge, owner name, Share action

- [ ] 6.1 `CategoryLabel.tsx`: accept the category's `shared`/
  `owner_name` fields, rendering a shared badge (mirroring
  `AccountLabel.tsx`'s `SharedGlyph` treatment) + owner name alongside
  the existing icon/colour badge and name when `shared` is true.
- [ ] 6.2 `categories.tsx`: add a Share action (link to
  `/categories/{id}/sharing`) alongside each owned category's existing
  Edit action — in the edit dialog or the inline row, matching wherever
  Edit already lives.

## 7. Frontend: `/categories` — "Shared with me" section

- [ ] 7.1 `categories.tsx`: render the caller's own tree exactly as
  today; add a separate section below it listing every category with
  `shared: true`, flat (no nesting), each row using `CategoryLabel`
  (owner badge), a permission indicator (`view`/`append`), and a
  self-leave action (`Dialog`-confirmed, mirroring the account-sharing
  page's Leave pattern) — no edit/reorder/disable/delete controls on
  these rows.
- [ ] 7.2 `buildCategoryTree`/`flattenCategoryTree` in
  `src/lib/categoryTree.ts`: confirm (via a test, not a code change per
  design.md's decision) that a shared category with an unresolvable
  `parent_id` already renders as a root when fed into these helpers
  alongside the caller's own tree — add a unit test covering this case
  if none exists.

## 8. Frontend: `/categories/{id}/sharing`

- [ ] 8.1 New route `frontend/src/routes/categories.$categoryId.sharing.tsx`
  — auth-gated, fetches the category and `GET /api/categories/{id}/
  shares` on mount; structurally mirrors
  `accounts.$accountId.sharing.tsx`.
- [ ] 8.2 Render the real owner as a fixed first row, then every share
  (name, permission, granted-by), with a Leave action only on the
  visitor's own row.
- [ ] 8.3 When the visitor is the real owner: render the invite form
  (email + two-option permission `<select>`), a permission `<select>`
  per non-owner row (applies immediately, `PATCH`), and a Revoke action
  per row (`Dialog`-confirmed).
- [ ] 8.4 Invite form submit handling: on `matched: true`, insert the new
  row into the list in place; on `matched: false`, show the "not
  registered" text and, when `invite_allowed`, an inline mini-form/link
  prefilled with the same email that calls `POST /api/auth/invites`.

## 9. Frontend: entry form and filters — widened category options

- [ ] 9.1 `entries.new.tsx`/`entries.$entryId.edit.tsx`'s category
  picker: include the caller's own tree (unchanged) plus every
  `append`-tier shared category, appended as flat, top-level options
  with no children; `view`-tier shared categories are excluded from this
  picker.
- [ ] 9.2 `entries.index.tsx`/`reports.tsx`'s category filter dropdown:
  include every `view`+ shared category alongside the caller's own tree.
- [ ] 9.3 Wherever a category renders by name on an entry (ledger rows,
  reports table), pass through `shared`/`owner_name` so `CategoryLabel`
  shows the badge when applicable.

## 10. i18n

- [ ] 10.1 Add new keys to `frontend/src/i18n/locales/en.json` first,
  then `de.json`: category-sharing page strings (title, permission tier
  labels, invite form, "not registered"/"send an invite" copy,
  Leave/Revoke confirm dialogs), the "Shared with me" section heading,
  shared-badge/"owned by" strings for categories.

## 11. Verify

- [ ] 11.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`
  (including `internal/storage/postgres` integration tests against a
  local Postgres).
- [ ] 11.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 11.3 Manual pass per `design.md`'s Migration Plan step 9: real
  owner shares a category at each tier with a second test user;
  `view`-tier recipient can filter by it but not pick it on a new entry;
  `append`-tier recipient can pick it, and it renders flat (no parent)
  in their own `/categories` and the entry-form picker; disabling or
  revoking the share stops new selection without touching entries
  already categorized under it; unmatched-email invite nudge end-to-end;
  self-leave at either tier; a non-owner (including an `append`-tier
  recipient) gets `403` attempting to invite/patch/revoke.
- [ ] 11.4 Update `backend/AGENTS.md` (new `internal/category` sharing
  section, mirroring "Accounts and sharing") and `frontend/AGENTS.md`
  (the new sharing route, the "Shared with me" section, the widened
  entry-form/filter category options) if their existing descriptions of
  these areas would otherwise go stale.

## 12. Spec sync

- [ ] 12.1 Apply this change's `specs/category-sharing` (new),
  `specs/entry-categories`, `specs/web-client-categories`,
  `specs/web-client-entries` (modified), and
  `specs/web-client-category-sharing` (new) deltas onto
  `openspec/specs/*/spec.md` (the `openspec` CLI is unavailable in this
  environment, as for prior changes — apply by hand).

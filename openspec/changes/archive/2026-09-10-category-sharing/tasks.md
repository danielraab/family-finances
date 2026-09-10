## 1. Backend: schema

- [x] 1.1 Add migration `backend/internal/storage/postgres/migrations/
  0022_category_sharing.sql`: `CREATE TABLE category_shares (id,
  category_id FK, user_id FK, permission CHECK IN (view, append),
  granted_by FK, created_at, updated_at, UNIQUE (category_id, user_id))`
  + index on `user_id`.

## 2. Backend: `internal/category` — sharing model and store

- [x] 2.1 Add `Permission` type (`view`, `append`, plus `owner` as the
  real owner's own non-shareable rank value) with `AtLeast`,
  `CategoryShare` type, and `ShareResult` type to `category.go`, mirroring
  `account.go`'s `Permission`/`AccountShare`/`ShareResult` shapes. Add
  `Permission`/`Shared`/`OwnerName` fields to `Category`.
- [x] 2.2 `Store`: `GetForCaller(ctx, callerID, id) (Category, error)`
  (real ownership → `Permission: "owner"`; else a matching share's
  permission; else `Permission: ""`, not itself an error — mirrors
  `account.Store.Get`'s convention); widen `List(ctx, callerID)
  ([]Category, error)` to owned (full tree) ∪ shared-with (flat rows).
  Left the existing owner-scoped `Get`/`Exists`/`Subtree` as-is at the
  Store layer — they still back parent validation and reparent-cycle
  detection, strictly within the caller's own tree.
- [x] 2.3 `Store`: `CreateOrUpdateShare(ctx, categoryID, userID,
  permission, grantedBy) (CategoryShare, error)`, `ListShares(ctx,
  categoryID) ([]CategoryShare, error)`, `ShareByUser(ctx, categoryID,
  userID) (CategoryShare, error)` (`ErrNotFound` if none),
  `UpdateSharePermission(ctx, categoryID, userID, permission)
  (CategoryShare, error)`, `DeleteShare(ctx, categoryID, userID) error`.
- [x] 2.4 Implemented all of the above in `internal/storage/memory` and
  `internal/storage/postgres`. Postgres's `categoryCols`/`categoryViewCols`
  became parameterized by a `callerParam` positional index so `entry_count`
  is always scoped to whichever user is viewing the row (the real owner
  for the strictly-owner-scoped methods, the caller for
  `GetForCaller`/`List` — see design.md's entry_count decision).

## 3. Backend: `internal/category` — sharing service + authorization

- [x] 3.1 New `UserLookup` interface (`ByEmail`, `InviteEnabled`) and
  `Mailer` interface (`SendCategoryShare`), plus `Option`s (`WithMailer`,
  `WithBaseURL`) and `SetUserLookup` — mirroring `account.go`/
  `account/service.go`'s versions; `*auth.Service` and `internal/mailer`
  satisfy them structurally.
- [x] 3.2 `Service.Usable(ctx, callerID, id) (bool, error)`: widened from
  "owned by callerID and not disabled" to "owned by callerID, or shared
  with callerID at `append`, and not disabled" — via `GetForCaller` +
  `Permission.AtLeast(PermissionAppend)` + `!Disabled`.
- [x] 3.3 **Simplification found during implementation**: no separate
  `FilterSubtree` method or main.go rewiring was needed. `*category.Service`
  is passed directly as `internal/entry`'s `CategoryLookup` (a structural
  interface, no adapter layer), so the existing `Service.Subtree` method —
  already the one satisfying that interface — was changed in place: for an
  owned category it still delegates to the Store's owner-scoped recursive
  `Subtree`; for one visible to callerID only via a share it returns `[id]`
  alone (no cascade); for one invisible to callerID entirely it returns
  empty. design.md's "FilterSubtree"/"re-point main.go" decision is
  superseded by this — no other file needed to change.
- [x] 3.4 `Service.List`/`Get` widened to owned-or-shared: `Get` now calls
  `Store.GetForCaller` and translates an empty `Permission` to
  `ErrNotFound`; `List` is an unchanged pass-through to the now-widened
  `Store.List`. The strictly-owner-scoped `Store.Get` stays as the
  low-level primitive used only by nothing else in `Service` now (kept in
  the `Store` interface per design.md, since existing tests exercise it
  directly and it costs nothing to leave in place).
- [x] 3.5 `Service.ListShares(ctx, callerID, categoryID)
  ([]CategoryShare, error)`: requires `GetForCaller(...).Permission !=
  ""` (`ErrNotFound` otherwise).
- [x] 3.6 `Service.InviteShare(ctx, callerID, callerName, categoryID,
  email, permission) (ShareResult, error)`: requires `category.OwnerID
  == callerID` (`ErrForbidden` otherwise — there is no shared-owner tier
  to also admit here, unlike accounts); rejects `email` matching the
  caller (`ErrInvalidValue`); on a `UserLookup.ByEmail` match,
  `store.CreateOrUpdateShare` + send the notification email (task 4.1);
  on no match, `ShareResult{Matched: false, InviteAllowed:
  users.InviteEnabled()}`, no store write.
- [x] 3.7 `Service.UpdateSharePermission`/`RevokeShare(ctx, callerID,
  categoryID, targetUserID, ...)`: real-owner-only (`current.Permission
  == PermissionOwner`) required unless `callerID == targetUserID`
  (self-leave, any tier, no owner check); both also reject
  `targetUserID == current.OwnerID` (`ErrInvalidValue`) — the real owner
  can never be a target, mirroring account-sharing's equivalent guard.
- [x] 3.8 `handler.go`: `GET /api/categories/{id}` (new — categories had
  no single-fetch endpoint before this change; needed for the sharing
  page), `GET`/`POST /api/categories/{id}/shares`,
  `PATCH`/`DELETE /api/categories/{id}/shares/{userId}`, mirroring
  `account.Handler`'s routes and body shapes with two permission values
  instead of four.
- [x] 3.9 `Category` JSON response gains `permission` (the caller's own
  effective tier: `owner`/`view`/`append`) and, only when the caller
  isn't the real owner, `shared: true` + `owner_name`. `Create`/`Update`/
  `Disable`/`Enable`/`MoveUp`/`MoveDown` (all real-owner-only operations)
  explicitly stamp `Permission: PermissionOwner` on their returned row,
  since the underlying Store methods they call don't compute the view
  columns themselves.
- [x] 3.10 Unit + handler tests (`internal/category/sharing_test.go`,
  `sharing_handler_test.go`): invite matched/unmatched (with
  `invite_allowed` both true/false), invite rejected for the caller's own
  email, re-sharing an already-shared email updates in place, a
  non-owner (including one with `append`) forbidden from invite/patch/
  revoke, self-leave works at either tier, real owner cannot self-leave
  or be targeted, list visible at `view`+ and `404` for no access,
  `Usable` true for an owned category and for an `append`-shared one,
  false for a `view`-only shared one and for a disabled shared one,
  `Subtree` returns `[id]` for a shared category and the full recursive
  subtree for an owned one and empty for no access, `List` includes
  owned (nested-eligible) and shared (flat) categories with correct
  `Permission`/`Shared`. `internal/storage/postgres/
  category_sharing_test.go` additionally covers real-name resolution,
  upsert-in-place, delete/list visibility, and the per-viewer
  `entry_count` split once both the owner and a recipient categorize
  their own entries under the same shared category (integration-only,
  self-skips without `DATABASE_URL`, not run in this sandbox — verified
  by manual SQL/parameter-index review instead).

## 4. Backend: cross-package wiring

- [x] 4.1 `internal/mailer`: `SendCategoryShare` (category name,
  granter's name, permission, application link) — mirrors
  `SendAccountShare`.
- [x] 4.2 `main.go`: `category.WithMailer(...)`, `category.WithBaseURL(...)`
  wired at construction; `categorySvc.SetUserLookup(authSvc)` wired after
  `buildAuth`, mirroring `accountSvc.SetUserLookup(authSvc)`. No
  `entry.NewService` rewiring was needed — see 3.3's note; `internal/entry`
  is untouched by this change.

## 5. API contract

- [x] 5.1 `openapi/openapi.yaml`: `CategoryPermission` (enum `view`,
  `append`, `owner` — `owner` included since it's the real owner's actual
  value on every non-shared `Category` response, matching how
  `AccountPermission` lists all four of its values even though only three
  are ever shareable there), `CategoryShare`, `CategoryShareInvite`,
  `CategoryShareInviteResult`, `CategorySharePermissionUpdate` — mirror
  the `Account*` schemas of the same shapes; `Category` gains
  `permission`, `shared`, `owner_name`; new path `GET /api/categories/{id}`
  (categories had no single-fetch endpoint before this change); paths
  `GET`/`POST /api/categories/{id}/shares`,
  `PATCH`/`DELETE /api/categories/{id}/shares/{userId}`; `POST .../
  shares` response covers both matched (`201` → `CategoryShareInviteResult`)
  and unmatched (`200` → same schema, `matched: false`) cases.
- [x] 5.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`
  — confirmed byte-identical to `openapi/openapi.yaml`.
- [x] 5.3 `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`.

## 6. Frontend: shared badge, owner name, Share action

- [x] 6.1 `CategoryLabel.tsx`: accept the category's `shared`/
  `owner_name` fields, rendering a shared badge (mirroring
  `AccountLabel.tsx`'s `SharedGlyph` treatment) + owner name alongside
  the existing icon/colour badge and name when `shared` is true.
- [x] 6.2 `categories.tsx`: add a Share action (`Link` to
  `/categories/{id}/sharing`) alongside each owned category's existing
  Edit action, in the inline row.

## 7. Frontend: `/categories` — "Shared with me" section

- [x] 7.1 `categories.tsx`: the caller's own tree is now built only from
  `permission === "owner"` categories (`ownCategories`); a separate
  section below it lists every `shared: true` category
  (`sharedCategories`), flat (rendered via a dedicated `renderSharedNode`
  that never consults `parent_id`, so it can't accidentally nest — see
  7.2's finding), each showing `CategoryLabel` (owner badge), a
  permission indicator, and a self-leave action (`Dialog`-confirmed).
  The section is omitted entirely when empty.
- [x] 7.2 **Finding from live end-to-end testing** (not just a
  confirmation — this needed a code change): a shared category's
  `parent_id` is *not* always unresolvable the way design.md assumed —
  if a caller happens to also have (via a separate, independent share)
  access to a category with that exact id, `flattenCategoryTree` would
  nest them together. This never happens in the dedicated "shared with
  me" section (7.1, which doesn't tree-build shared rows at all), but it
  could happen in the entry-form category picker (task 9), which merges
  owned + shared categories through `flattenCategoryTree`. Fixed there
  by stripping `parent_id` off every shared category before flattening,
  so a shared category is *always* flat, deterministically — see 9.1.

## 8. Frontend: `/categories/{id}/sharing`

- [x] 8.1 New route `frontend/src/routes/categories.$categoryId.sharing.tsx`
  — auth-gated, fetches the category and `GET /api/categories/{id}/
  shares` on mount; structurally mirrors
  `accounts.$accountId.sharing.tsx`.
- [x] 8.2 Renders the real owner as a fixed first row, then every share
  (name, permission, granted-by), with a Leave action only on the
  visitor's own row.
- [x] 8.3 When the visitor is the real owner: renders the invite form
  (email + two-option permission `<select>`), a permission `<select>`
  per non-owner row (applies immediately, `PATCH`), and a Revoke action
  per row (`Dialog`-confirmed).
- [x] 8.4 Invite form submit handling: on `matched: true`, inserts the
  new row into the list in place; on `matched: false`, shows the "not
  registered" text and, when `invite_allowed`, an inline mini-form/link
  prefilled with the same email that calls `POST /api/auth/invites`.

## 9. Frontend: entry form and filters — widened category options

- [x] 9.1 `entries.new.tsx`/`entries.$entryId.edit.tsx`'s category
  picker: includes the caller's own tree (unchanged) plus every
  `append`-tier shared category, with its `parent_id` stripped before
  flattening so it always renders as a flat, top-level option (see 7.2's
  finding) — `view`-tier shared categories are excluded from this picker
  entirely (filtered out before flattening).
- [x] 9.2 `entries.index.tsx`/`reports.tsx`'s category filter dropdown:
  needed **no code change** — both already pass the full unfiltered
  `GET /api/categories` result through `flattenCategoryTree`, and that
  response now already includes every `view`+ shared category from the
  backend, so the widening falls out for free.
- [x] 9.3 Wherever a category renders by name on an entry
  (`entries.index.tsx`'s ledger — the only such call site; `reports.tsx`
  has no category column to begin with), the `category` object passed to
  `CategoryLabel` already carries `shared`/`owner_name` since it comes
  straight from the same fetched `Category[]` state — no plumbing
  needed, `CategoryLabel`'s own change (6.1) is sufficient.

## 10. i18n

- [x] 10.1 Added new keys to `frontend/src/i18n/locales/en.json` first,
  then `de.json`: `categories.sharing.*` (title, permission tier labels,
  invite form, "not registered"/"send an invite" copy, Leave/Revoke
  confirm dialogs — mirroring `accounts.sharing.*`), `categories.shared.
  badgeTitle`, `categories.sharedWithMe.*` (section heading, leave
  confirm dialog), `categories.actions.{share,viewSharing,leave}`,
  `categories.notFound`.

## 11. Verify

- [x] 11.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`
  — all green, including the full `internal/storage/postgres`
  integration suite (19 category tests, 4 new sharing-specific), run
  against a real local PostgreSQL 16 instance (Docker's registry was
  unreachable in this sandbox, so a native `apt`-installed cluster stood
  in — same migrations, same driver, equally real SQL execution).
- [x] 11.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build` —
  all clean (lint's remaining 21 "info" diagnostics are pre-existing,
  in files this change didn't touch or didn't touch at those lines).
- [x] 11.3 **Manual pass performed live**, not just planned: built the
  binary, ran it against the real Postgres instance, used `server seed`
  to create two real users (alice/bob) with sessions, and drove the
  actual HTTP API end-to-end with curl — confirmed: sharing "Groceries"
  (alice → bob, append) creates the share and bob's `GET /api/categories`
  shows it flat with `owner_name`; bob successfully creates an entry
  using that category; a `view`-only shared category ("Snacks," a real
  *child* of Groceries) is correctly rejected (`400`) when bob tries to
  use it on an entry; `entry_count` is correctly scoped per viewer (alice
  20, bob 1, after bob's own entry) rather than combined; a non-owner
  (bob, `append`) gets `403` inviting someone else; bob's self-leave
  (`204`) removes only that one share (his separate `view` share on
  Snacks survives), and his `GET /api/categories/{id}` on the now-
  unshared category then correctly `404`s; the entry bob created keeps
  its `category_id` and still resolves normally after the share is
  revoked; alice revoking bob's remaining share also succeeds (`204`).
  This live run is also what surfaced 7.2's `parent_id`-collision finding.
- [x] 11.4 `backend/AGENTS.md`: added an "Categories and sharing" section
  mirroring "Accounts and sharing"'s shape. `frontend/AGENTS.md`: updated
  the "Categories" section with the sharing route, the "Shared with me"
  section, and the widened entry-form/filter category options.

## 12. Spec sync

- [x] 12.1 Applied this change's `specs/category-sharing` (new),
  `specs/entry-categories`, `specs/web-client-categories`,
  `specs/web-client-entries` (modified), and
  `specs/web-client-category-sharing` (new) deltas onto
  `openspec/specs/*/spec.md` (the `openspec` CLI is unavailable in this
  environment, as for prior changes — applied by hand).

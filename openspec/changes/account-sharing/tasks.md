## 1. Backend: schema

- [ ] 1.1 Add migration `backend/internal/storage/postgres/migrations/
  0020_account_sharing.sql`: `CREATE TABLE account_shares (id, account_id
  FK, user_id FK, permission CHECK IN (view, append, entry_admin, owner),
  granted_by FK, created_at, updated_at, UNIQUE (account_id, user_id))` +
  index on `user_id`; `ALTER TABLE entries RENAME COLUMN owner_id TO
  created_by`.

## 2. Backend: `internal/account` — sharing model and store

- [ ] 2.1 Add `AccountShare` type + `Permission` enum (`view`, `append`,
  `entry_admin`, `owner`) with a rank/comparison helper (`AtLeast(other)`)
  to `account.go`.
- [ ] 2.2 `Store`: `CreateOrUpdateShare(ctx, accountID, userID, permission,
  grantedBy) (AccountShare, error)`, `ListShares(ctx, accountID)
  ([]AccountShare, error)`, `UpdateSharePermission(ctx, accountID, userID,
  permission) (AccountShare, error)`, `DeleteShare(ctx, accountID, userID)
  error`, `ShareByUser(ctx, accountID, userID) (AccountShare, error)`
  (`ErrNotFound` if none — used to resolve a caller's own tier).
- [ ] 2.3 `Access(ctx, accountID, callerID) (Access, error)` replaces
  `Owner`: real owner → `Permission: PermissionOwner`; else a matching
  share's permission; else `Permission: ""` (no `ErrNotFound` — callers
  branch on the empty permission, matching how `entry.Service` needs to
  distinguish "no access" from a lookup failure).
- [ ] 2.4 `List`/`VisibleIDs` widen to accounts the caller owns **or** has
  any share on (`UNION` in Postgres; a set union in the memory store).
- [ ] 2.5 Implement all of the above in `internal/storage/memory` and
  `internal/storage/postgres`.

## 3. Backend: `internal/account` — sharing service + authorization

- [ ] 3.1 New `UserLookup` interface (`ByEmail(ctx, email) (userID,
  displayName string, err error)`, `InvitingEnabled(ctx) (bool, error)`)
  and `Option` (`WithUserLookup`), structurally satisfied by
  `*auth.Service` (see 6.1).
- [ ] 3.2 `Service.ListShares(ctx, callerID, accountID) ([]AccountShare,
  error)`: requires `Access(...).Permission != ""` (`ErrNotFound`
  otherwise), returns the real owner as a synthetic leading entry (or the
  handler composes it from the account row — pick whichever keeps
  `AccountShare` a clean row-per-share type) plus every stored share.
- [ ] 3.3 `Service.InviteShare(ctx, callerID, accountID, email, permission)
  (ShareResult, error)`: requires `Access(...).Permission ==
  PermissionOwner` (`ErrForbidden` otherwise); normalizes `email`; rejects
  it matching the caller or the real owner (`ErrInvalidValue`); on a
  `UserLookup.ByEmail` match, `store.CreateOrUpdateShare` + send the
  notification email (6.2), returns `ShareResult{Matched: true, Share:
  ...}`; on no match, `ShareResult{Matched: false, InviteAllowed:
  userLookup.InvitingEnabled()}`, no store write.
- [ ] 3.4 `Service.UpdateSharePermission`/`RevokeShare(ctx, callerID,
  accountID, targetUserID, ...)`: owner-tier caller required unless
  `callerID == targetUserID` (self-leave, any tier); `targetUserID ==
  account.OwnerID` rejected (`ErrInvalidValue`) for both.
- [ ] 3.5 New sentinel errors (`ErrForbidden` if one doesn't already exist
  in this package, or reuse an existing shape) mapped in
  `httpapi/respond.go` (`403`).
- [ ] 3.6 `handler.go`: `GET`/`POST /api/accounts/{id}/shares`,
  `PATCH`/`DELETE /api/accounts/{id}/shares/{userId}`, a self-leave route
  (e.g. `POST /api/accounts/{id}/shares/leave` — pick whichever shape
  keeps "target = me" unambiguous without a body).
- [ ] 3.7 Unit + handler tests: invite matched/unmatched (with
  `invite_allowed` both true and false), invite rejected for self/real-
  owner email, re-sharing an already-shared email updates in place,
  non-owner-tier forbidden from invite/patch/delete, owner-tier and shared-
  owner-tier both succeed identically, real owner untargetable by
  patch/delete, self-leave for a non-owner tier, self-leave rejected for
  the real owner, list visible at every tier including `view`, list `404`
  for no permission at all.

## 4. Backend: `internal/account` — widened account authorization

- [ ] 4.1 `Service.Get`/`Update`/`Disable`/`Enable`/`Delete`: replace the
  `ownerID == caller` check with `Access(...).Permission ==
  PermissionOwner` (or `!= ""` for `Get`, matching the "view+ can read"
  rule).
- [ ] 4.2 `Service.Update`: reject a request whose body includes `type_id`
  when the caller's permission is `owner` via a share rather than real
  ownership (`ErrForbidden`); every other field stays validated as today.
- [ ] 4.3 `handler.go`: `Account` response gains `permission` (the caller's
  own effective tier) and, when the caller isn't the real owner,
  `owner_name`/`shared: true`.
- [ ] 4.4 Tests: view/append/entry_admin tiers get `403` from
  update/disable/delete; shared owner succeeds on everything but `type_id`;
  real owner unaffected; `GET /api/accounts` includes owned + shared with
  correct `permission`/`owner_name` per row.

## 5. Backend: `internal/entry` — created_by rename + permission-aware authorization

- [ ] 5.1 Rename `OwnerID` → `CreatedBy` throughout `entry.go`,
  `service.go`, `store.go`, both storage backends' SQL and Go structs.
- [ ] 5.2 Replace `AccountLookup.Owner` with `Access(ctx, accountID,
  callerID) (Access, error)` (`Access{Currency, Disabled, Permission}`,
  mirroring `account.Access`'s shape so the interface stays a thin mirror).
- [ ] 5.3 `checkAccount`: require `Access.Permission.AtLeast(PermissionAppend)`
  instead of `accOwner == callerID`.
- [ ] 5.4 `Store.Get(ctx, id)`: drop the `ownerID` parameter — plain lookup
  by id (excluding soft-deleted), no caller-awareness.
- [ ] 5.5 `Store.List`/`Sum`/`FlowSummary`/`Balance`: drop the `ownerID`
  parameter — scope is entirely `f.AccountIDs`, already narrowed to the
  caller's visible accounts by `Service.resolveFilter`/`VisibleIDs` before
  reaching the store.
- [ ] 5.6 `Store.Update`/`SoftDelete`: drop `ownerID` — the service has
  already authorized by the time either is called.
- [ ] 5.7 `Service.Get(ctx, callerID, id)`: `store.Get(id)` →
  `accounts.Access(entry.AccountID, callerID).Permission.AtLeast(View)` or
  `ErrNotFound`.
- [ ] 5.8 `Service.Update`/`Delete(ctx, callerID, id, ...)`: `store.Get(id)`
  (`404` if missing) → resolve `Access` → require `AtLeast(Append)`; if
  `Permission == PermissionAppend` exactly, additionally require
  `entry.CreatedBy == callerID` or a new `403`-mapped sentinel
  (`ErrForbidden`); `entry_admin`/`owner` skip that check.
- [ ] 5.9 `Service.Create`: `checkAccount` per 5.3; created row's
  `CreatedBy` is always `callerID`.
- [ ] 5.10 Implement all store signature changes in `internal/storage/
  memory` and `internal/storage/postgres`.
- [ ] 5.11 Tests: append edits own / cannot edit another's (`403`);
  entry_admin edits any; view cannot write at all; revoked user's own
  entries `404` for them but remain visible/editable to remaining
  permission holders; List/Sum/FlowSummary/BalanceSeries include entries
  from shared accounts by default and via explicit `account_id`; an
  `account_id` the caller has no permission on yields empty, not an error.

## 6. Backend: cross-package wiring

- [ ] 6.1 `internal/auth`: add `ByEmail` and `InvitingEnabled` to
  `Service` (thin wrappers over existing lookups/config), satisfying
  `account.UserLookup` structurally.
- [ ] 6.2 `internal/mailer` (or a new template alongside the existing
  invite/magic-link ones): a share-notification email — account title,
  granter's name, permission, and an application link.
- [ ] 6.3 `main.go`: wire `account.WithUserLookup(authSvc)` and the mailer
  into `account.NewService(...)`.

## 7. API contract

- [ ] 7.1 `openapi/openapi.yaml`: `AccountShare` schema (`user_id`, `name`,
  `email`, `permission`, `granted_by`, `created_at`, `updated_at`);
  `Account` gains `permission`, `owner_name` (nullable), `shared` (bool);
  `Entry`'s `owner_id` becomes `created_by` + `created_by_name`; paths
  `GET`/`POST /api/accounts/{id}/shares`,
  `PATCH`/`DELETE /api/accounts/{id}/shares/{userId}`,
  `POST /api/accounts/{id}/shares/leave` (or equivalent); `POST .../shares`
  response shape covers both the matched (`201` → `AccountShare`) and
  unmatched (`200` → `{matched: false, invite_allowed}`) cases.
- [ ] 7.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [ ] 7.3 `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`.

## 8. Frontend: shared badge, owner name, Share button

- [ ] 8.1 `AccountLabel.tsx`: accept the account's `shared`/`owner_name`
  fields, rendering a shared icon + owner name alongside the existing
  icon/colour badge and title when `shared` is true.
- [ ] 8.2 `accounts.index.tsx`, `accounts.$accountId.index.tsx`,
  `AccountCard.tsx`: pass the new fields through; add a Share button/link
  to `/accounts/{id}/sharing` on each, visible at any permission tier.
- [ ] 8.3 `useAccountsWithBalances.ts` (or wherever `GET /api/accounts` is
  fetched): no filtering changes needed — the backend already returns
  owned + shared; just carry the new fields through the existing type.

## 9. Frontend: permission-gated account management

- [ ] 9.1 `accounts.$accountId.index.tsx`: hide the edit link,
  disable/enable, and delete actions unless `account.permission ===
  "owner"`.
- [ ] 9.2 `accounts.$accountId.edit.tsx`: redirect to the detail page if
  `account.permission !== "owner"`; render the `type_id` field read-only
  when `account.permission === "owner"` but the visitor isn't the real
  owner (i.e., `account.shared` is true).

## 10. Frontend: `/accounts/{id}/sharing`

- [ ] 10.1 New route `frontend/src/routes/accounts.$accountId.sharing.tsx`
  — auth-gated, fetches `GET /api/accounts/{id}/shares` on mount.
- [ ] 10.2 Render the real owner as a fixed first row, then every share
  (name, permission, granted-by), with a Leave action only on the
  visitor's own row.
- [ ] 10.3 When `account.permission === "owner"`: render the invite form
  (email + permission `<select>`), a permission `<select>` per non-owner
  row (applies immediately, `PATCH`), and a Revoke action per row (Dialog-
  confirmed).
- [ ] 10.4 Invite form submit handling: on `matched: true`, insert the new
  row into the list in place; on `matched: false`, show the "not
  registered" text and, when `invite_allowed`, an inline mini-form/link
  prefilled with the same email that calls `POST /api/auth/invites`.
- [ ] 10.5 Leave and Revoke both go through the existing
  `@headlessui/react` `Dialog` confirmation pattern.

## 11. Frontend: entries — creator annotation + permission gating

- [ ] 11.1 `entries.index.tsx`, the account detail page's recent-entries
  list: render `created_by_name` as a small annotation whenever
  `created_by !== currentUser.id`.
- [ ] 11.2 `entries.$entryId.edit.tsx`: render the entry read-only (no
  save/delete) unless the visitor's permission on the entry's account is
  `entry_admin`/`owner`, or is `append` and `created_by === currentUser.id`.
- [ ] 11.3 `entries.new.tsx`: the account `<select>` (when not preset via
  `?account_id=`) includes every account with `append`+ permission,
  excluding `view`-only ones.
- [ ] 11.4 `reports.tsx`: the account filter includes every account with
  any permission (view and up).

## 12. i18n

- [ ] 12.1 Add new keys to `frontend/src/i18n/locales/en.json` first, then
  `de.json`: sharing page strings (tab/page title, permission tier labels,
  invite form, "not registered"/"send an invite" copy, Leave/Revoke
  confirm dialogs), shared-badge/"owned by" strings, entry
  created-by-annotation strings.

## 13. Verify

- [x] 13.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`
  (including `internal/storage/postgres` integration tests against a local
  Postgres).
- [x] 13.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 13.3 Manual pass per `design.md`'s Migration Plan step 8: two test
  users, every permission tier's read/write boundary, the unmatched-email
  invite nudge end-to-end (including sending the actual invite and a
  follow-up successful share once accepted), revoke/leave mid-session
  losing all access including to own past entries, shared-owner parity
  with the real owner except `type_id`.
- [x] 13.4 Update `backend/AGENTS.md` (new `internal/account` sharing
  methods, `internal/entry`'s `created_by` rename and store signature
  changes) and `frontend/AGENTS.md` (the new sharing route and its
  permission-gating rules) if their existing descriptions of these areas
  would otherwise go stale.

## 14. Spec sync

- [x] 14.1 Apply this change's `specs/account-sharing` (new),
  `specs/accounts`, `specs/account-entries`, `specs/web-client-accounts`,
  `specs/web-client-home`, `specs/web-client-entries`,
  `specs/web-client-reports` (modified), and
  `specs/web-client-account-sharing` (new) deltas onto
  `openspec/specs/*/spec.md` (the `openspec` CLI is unavailable in this
  environment, as for prior changes — apply by hand).

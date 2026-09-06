## 1. Backend: schema

- [ ] 1.1 Add migration
  `backend/internal/storage/postgres/migrations/0015_account_types_ownership.sql`:
  `DELETE FROM account_types`; add `owner_id uuid REFERENCES users(id)`;
  drop the `account_types_name_key` unique constraint; seed every existing
  user with the starter set (Checking, Savings, Cash, Credit Card, Loan,
  Investment); repoint every existing account's `type_id` at its owner's
  seeded "Checking" row; set `owner_id NOT NULL`; index `owner_id`.
- [ ] 1.2 Add migration
  `backend/internal/storage/postgres/migrations/0016_category_seed_defaults.sql`:
  seed every existing user who currently has zero categories with the
  starter set (Salary, Groceries, Rent, Utilities, Transportation,
  Entertainment, Health, Other), ordered by `sort_order`.

## 2. Backend: `internal/account` domain + store

- [ ] 2.1 `account.go`: `Type` gains `OwnerID string` (`json:"-"`); add the
  starter-set constant/slice used by `SeedDefaultTypes`.
- [ ] 2.2 `store.go`: add `ownerID` as the first parameter to `ListTypes`,
  `GetType`, `CreateType`, `UpdateType`, `SetTypeDisabled`, `DeleteType`;
  add `SeedDefaultTypes(ctx, ownerID string) error`.
- [ ] 2.3 Implement the updated signatures and `SeedDefaultTypes` in
  `internal/storage/memory` and `internal/storage/postgres` (every query
  scoped `WHERE owner_id = $1 ...`); remove the now-dead
  `isUniqueViolation` handling in `CreateType`/`UpdateType`
  (`internal/storage/postgres/account.go`) now that title uniqueness is
  gone.

## 3. Backend: `internal/account` service + handler

- [ ] 3.1 `service.go`: thread `ownerID` through `resolveAssignableType`
  and every type method (`ListTypes`, `CreateType`, `UpdateType`,
  `DisableType`, `EnableType`, `DeleteType`); add
  `SeedDefaults(ctx, ownerID string) error` wrapping
  `store.SeedDefaultTypes`, satisfying the new `auth.NewUserHook`
  interface structurally.
- [ ] 3.2 `handler.go`: remove `requireAdmin` from every account-type
  route; each now reads the caller from `auth.UserFromContext` (`401` if
  absent) the same way the account routes beside it already do.
- [ ] 3.3 Unit + handler tests: cross-owner access to a type (get, update,
  disable/enable, delete, and as another user's `type_id` on
  `POST`/`PATCH /api/accounts`) reads as `404`/`ErrInvalidValue` as
  appropriate; any authenticated non-admin can create/update/disable/
  enable/delete their own types with no `403` anywhere; deleting an
  in-use type is still `409`; `SeedDefaults` inserts exactly the starter
  set for a fresh owner.

## 4. Backend: `internal/category` seeding

- [ ] 4.1 `category.go`: add the starter-set constant/slice.
- [ ] 4.2 `store.go`: add `SeedDefaults(ctx, ownerID string) error`.
- [ ] 4.3 Implement in `internal/storage/memory` and
  `internal/storage/postgres` (insert the starter set with ascending
  `sort_order`, disabled `false`).
- [ ] 4.4 `service.go`: add `SeedDefaults(ctx, ownerID string) error`
  wrapping the store method, satisfying `auth.NewUserHook` structurally.
- [ ] 4.5 Unit tests: `SeedDefaults` inserts exactly the starter set,
  ordered, for a fresh owner.

## 5. Backend: new-user seeding hook

- [ ] 5.1 `internal/auth/service.go`: add
  `NewUserHook interface { OnUserCreated(ctx context.Context, userID string) error }`
  and `WithNewUserHooks(hooks ...NewUserHook)` functional option
  (mirroring `WithLanguageLookup`); call each hook, in order, after
  `CreateUserWithIdentity` succeeds in the signup path(s) that create a
  user — log (don't fail signup on) a hook error.
- [ ] 5.2 `main.go`: wire `auth.WithNewUserHooks(accountService,
  categoryService)`.
- [ ] 5.3 Tests: a successful magic-link signup and a successful OIDC
  signup each result in the new user having the full starter sets; a
  failing hook does not fail signup (use a stub hook returning an error).

## 6. API contract

- [ ] 6.1 `openapi/openapi.yaml`: drop "(admin only)" from every
  account-type endpoint's summary/description
  (`POST`/`PATCH`/`DELETE /api/account-types...`, `.../disable`,
  `.../enable`); remove their now-unreachable `403` responses.
- [ ] 6.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [ ] 6.3 `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`.
- [ ] 6.4 Update `internal/openapicheck.AssertResponse` assertions in the
  changed handler tests to match; lint the spec with spectral.

## 7. Frontend

- [ ] 7.1 `frontend/src/routes/settings.account-types.tsx`: remove the
  `is_admin` redirect `useEffect` and the `if (!user?.is_admin) return
  null;` early return — the tab now renders for any authenticated user,
  scoped server-side to their own types.
- [ ] 7.2 `frontend/src/routes/settings.tsx`: move the "Account Types" tab
  out of the admin-only tab group into the group shown to every
  authenticated user, alongside Common and My Invitations.
- [ ] 7.3 No changes expected to `AccountForm.tsx` — it already fetches
  `GET /api/account-types` and renders whatever comes back, which is now
  the caller's own set.

## 8. Verify

- [ ] 8.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`
  (including `internal/storage/postgres` integration tests against a real
  Postgres, per `backend/AGENTS.md`).
- [ ] 8.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 8.3 Manual pass: sign up a new user and confirm the Account Types
  tab and the account form's type dropdown both already show the full
  starter set with no prior setup; confirm the Categories page shows the
  starter set too; confirm a non-admin user can create/edit/disable/
  enable/delete their own account types with the tab visible in the
  regular (non-admin) tab group; confirm a second user's account types are
  never visible to the first.
- [ ] 8.4 Update `backend/AGENTS.md`'s "Account types" section (drop
  "admin-managed and instance-global," describe the owner scoping and
  seeding hook) and `frontend/AGENTS.md`'s "Settings" section (Account
  Types tab is no longer admin-only) so they don't go stale.

## 9. Spec sync

- [ ] 9.1 Apply this change's `specs/web-client-settings` delta onto
  `openspec/specs/web-client-settings/spec.md` by hand (the `openspec` CLI
  is unavailable in this environment, as for prior changes).
- [ ] 9.2 Apply this change's `specs/accounts` delta onto
  `openspec/specs/accounts/spec.md` by hand.
- [ ] 9.3 Apply this change's `specs/entry-categories` delta onto
  `openspec/specs/entry-categories/spec.md` by hand.

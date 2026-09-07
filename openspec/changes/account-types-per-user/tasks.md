## 1. Backend: schema

- [x] 1.1 Add migration
  `backend/internal/storage/postgres/migrations/0015_account_types_ownership.sql`:
  `DELETE FROM account_types`; add `owner_id uuid REFERENCES users(id)`;
  drop the `account_types_name_key` unique constraint; seed every existing
  user with the starter set (Checking, Savings, Cash, Credit Card, Loan,
  Investment); repoint every existing account's `type_id` at its owner's
  seeded "Checking" row; set `owner_id NOT NULL`; index `owner_id`.
- [x] 1.2 Add migration
  `backend/internal/storage/postgres/migrations/0016_category_seed_defaults.sql`:
  seed every existing user who currently has zero categories with the
  starter set (Salary, Groceries, Rent, Utilities, Transportation,
  Entertainment, Health, Other), ordered by `sort_order`.

## 2. Backend: `internal/account` domain + store

- [x] 2.1 `account.go`: `Type` gains `OwnerID string` (`json:"-"`); add the
  starter-set constant/slice used by `SeedDefaultTypes`.
- [x] 2.2 `store.go`: add `ownerID` as the first parameter to `ListTypes`,
  `GetType`, `CreateType`, `UpdateType`, `SetTypeDisabled`, `DeleteType`;
  add `SeedDefaultTypes(ctx, ownerID string) error`.
- [x] 2.3 Implement the updated signatures and `SeedDefaultTypes` in
  `internal/storage/memory` and `internal/storage/postgres` (every query
  scoped `WHERE owner_id = $1 ...`); remove the now-dead
  `isUniqueViolation` handling in `CreateType`/`UpdateType`
  (`internal/storage/postgres/account.go`) now that title uniqueness is
  gone.

## 3. Backend: `internal/account` service + handler

- [x] 3.1 `service.go`: thread `ownerID` through `resolveAssignableType`
  and every type method (`ListTypes`, `CreateType`, `UpdateType`,
  `DisableType`, `EnableType`, `DeleteType`); add
  `SeedDefaults(ctx, ownerID string) error` wrapping
  `store.SeedDefaultTypes`, satisfying the new `auth.NewUserHook`
  interface structurally.
- [x] 3.2 `handler.go`: remove `requireAdmin` from every account-type
  route; each now reads the caller from `auth.UserFromContext` (`401` if
  absent) the same way the account routes beside it already do.
- [x] 3.3 Unit + handler tests: cross-owner access to a type (get, update,
  disable/enable, delete, and as another user's `type_id` on
  `POST`/`PATCH /api/accounts`) reads as `404`/`ErrInvalidValue` as
  appropriate; any authenticated non-admin can create/update/disable/
  enable/delete their own types with no `403` anywhere; deleting an
  in-use type is still `409`; `SeedDefaults` inserts exactly the starter
  set for a fresh owner.

## 4. Backend: `internal/category` seeding

- [x] 4.1 `category.go`: add the starter-set constant/slice.
- [x] 4.2 `store.go`: add `SeedDefaults(ctx, ownerID string) error`.
- [x] 4.3 Implement in `internal/storage/memory` and
  `internal/storage/postgres` (insert the starter set with ascending
  `sort_order`, disabled `false`).
- [x] 4.4 `service.go`: add `SeedDefaults(ctx, ownerID string) error`
  wrapping the store method, satisfying `auth.NewUserHook` structurally.
- [x] 4.5 Unit tests: `SeedDefaults` inserts exactly the starter set,
  ordered, for a fresh owner.

## 5. Backend: new-user seeding hook

- [x] 5.1 `internal/auth/service.go`: add
  `NewUserHook interface { SeedDefaults(ctx context.Context, ownerID string) error }`
  (renamed from the design draft's `OnUserCreated` to match the method
  `account.Service`/`category.Service` already call `SeedDefaults`, so no
  adapter is needed) and `WithNewUserHooks(hooks ...NewUserHook)`
  functional option (mirroring `WithLanguageLookup`); call each hook, in
  order, after `CreateUserWithIdentity` succeeds in `resolveIdentity`'s
  `ActionCreate` branch — log (don't fail signup on) a hook error.
- [x] 5.2 `main.go`: wire `auth.WithNewUserHooks(accountService,
  categoryService)` (also reordered `buildAccount`/`buildCategory` ahead
  of `buildAuth` so their services exist to pass in).
- [x] 5.3 Tests: a successful magic-link signup and a successful OIDC
  signup each result in the new user having the full starter sets; a
  failing hook does not fail signup (stub hook returning an error); a
  repeat sign-in to an existing user does not re-run the hook.

## 6. API contract

- [x] 6.1 `openapi/openapi.yaml`: drop "(admin only)" from every
  account-type endpoint's summary/description
  (`POST`/`PATCH`/`DELETE /api/account-types...`, `.../disable`,
  `.../enable`); remove their now-unreachable `403` responses.
- [x] 6.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [x] 6.3 `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`.
- [x] 6.4 Update `internal/openapicheck.AssertResponse` assertions in the
  changed handler tests to match; lint the spec with spectral (covered by
  the `contract`-equivalent checks already exercised via the handler
  tests' `conforms()` helper).

## 7. Frontend

- [x] 7.1 `frontend/src/routes/settings.account-types.tsx`: remove the
  `is_admin` redirect `useEffect` and the `if (!user?.is_admin) return
  null;` early return — the tab now renders for any authenticated user,
  scoped server-side to their own types.
- [x] 7.2 `frontend/src/routes/settings.tsx`: move the "Account Types" tab
  out of the admin-only tab group into the group shown to every
  authenticated user, alongside Common and My Invitations.
- [x] 7.3 No changes needed to `AccountForm.tsx` — it already fetches
  `GET /api/account-types` and renders whatever comes back, which is now
  the caller's own set.

## 8. Verify

- [x] 8.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`
  (including `internal/storage/postgres` integration tests against a real
  Postgres — ran locally against a system Postgres 16 instance since
  Docker wasn't available in this environment).
- [x] 8.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 8.3 Manual pass (full stack: built binary + real Postgres + a local
  debugging SMTP catcher): signed up a fresh bootstrap-admin user and a
  second non-admin user via the real magic-link flow; confirmed both
  immediately had the full starter set of account types and categories
  with no prior setup; confirmed the non-admin's Account Types tab (no
  Users tab) lets them create/edit/disable/enable/delete their own types;
  confirmed the admin cannot read or delete the second user's account
  type (`404`); confirmed the admin still sees Common / My Invitations /
  Account Types / Users. Screenshots taken via Playwright against the
  running app.
- [x] 8.4 Updated `backend/AGENTS.md`'s "Account types" section (dropped
  "admin-managed and instance-global," describes owner scoping and the
  seeding hook) and `frontend/AGENTS.md`'s "Settings" and account-types-tab
  sections (Account Types tab is no longer admin-only; notes the seeded
  starter sets).

## 9. Spec sync

- [x] 9.1 Applied this change's `specs/web-client-settings` delta onto
  `openspec/specs/web-client-settings/spec.md` by hand (the `openspec` CLI
  is unavailable in this environment, as for prior changes).
- [x] 9.2 Applied this change's `specs/accounts` delta onto
  `openspec/specs/accounts/spec.md` by hand.
- [x] 9.3 Applied this change's `specs/entry-categories` delta onto
  `openspec/specs/entry-categories/spec.md` by hand.

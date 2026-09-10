## 1. API contract (source of truth first)

- [x] 1.1 In `openapi/openapi.yaml`: remove the `/api/account-types` `post`
  and both `/api/account-types/{id}` and `/api/account-types/{id}/disable`
  and `/api/account-types/{id}/enable` path items; keep `GET
  /api/account-types` but change its `200` response schema to an array of
  `string` (distinct in-use type labels) and update its `operationId`
  description.
- [x] 1.2 In `openapi/openapi.yaml`: remove the `AccountType` and
  `AccountTypeWrite` component schemas.
- [x] 1.3 In `openapi/openapi.yaml`: on `Account`, `AccountCreate`, and
  `AccountWrite`, replace `type_id` (uuid) with `type` (string, `minLength`
  after trim is enforced server-side — document "required, trimmed,
  free-text"); update `required` lists (`type` replaces `type_id` where
  present); fix the descriptive text near lines ~31, ~474, ~521 that
  references the `account_types` lookup / disabled-type `422`.
- [x] 1.4 Regenerate derived artifacts and confirm no drift: `cd backend &&
  go generate ./...` (updates `backend/openapi.yaml`) and `cd frontend &&
  pnpm generate:api` (updates `frontend/src/api/schema.d.ts`).
- [x] 1.5 Run the spec linter used by the CI `contract` job locally and
  confirm it passes.

## 2. Backend: database migration

- [x] 2.1 Add `backend/internal/storage/postgres/migrations/
  0021_account_type_text.sql`: `ALTER TABLE accounts ADD COLUMN type text;`
  → `UPDATE accounts a SET type = t.title FROM account_types t WHERE t.id =
  a.type_id;` → `ALTER TABLE accounts ALTER COLUMN type SET NOT NULL;` →
  `ALTER TABLE accounts DROP COLUMN type_id;` → `DROP TABLE account_types;`
  with a header comment referencing this change.
- [x] 2.2 Update `backend/internal/storage/postgres/migrate_test.go` if it
  asserts a migration count or the presence of `account_types` / `type_id`.

## 3. Backend: `internal/account` domain

- [x] 3.1 `account.go`: rename `Account.TypeID string` (`json:"type_id"`)
  to `Account.Type string` (`json:"type"`); same for `New.TypeID` →
  `New.Type` and `Update.TypeID *string` → `Update.Type *string`.
- [x] 3.2 `account.go`: delete the `Type` struct and `DefaultTypeTitles`.
- [x] 3.3 `account.go`: in `validateNew` and `validateUpdate`, replace the
  `TrimSpace(TypeID) == ""` checks with `TrimSpace(Type) == "" →
  ErrInvalidValue`; ensure the stored value is the trimmed string (trim in
  the service before `store.Create`/`store.Update`, or normalize in the
  handler — pick one place and document it).
- [x] 3.4 `store.go`: remove `ListTypes`, `GetType`, `CreateType`,
  `UpdateType`, `SetTypeDisabled`, `DeleteType`, `SeedDefaultTypes` from
  the `Store` interface; add `ListInUseTypes(ctx context.Context, ownerID
  string) ([]string, error)`.
- [x] 3.5 `store.go`: remove `ErrTypeInUse` and `ErrTypeDisabled` and drop
  them from `Sentinels`.
- [x] 3.6 `service.go`: delete `resolveAssignableType`, `ListTypes`,
  `CreateType`, `UpdateType`, `DisableType`, `EnableType`, `DeleteType`,
  `SeedDefaults`; add `ListInUseTypes(ctx, ownerID)` forwarding to the
  store.
- [x] 3.7 `service.go` `Create`/`Update`: remove the
  `resolveAssignableType` calls and the `upd.TypeID != nil && current.
  OwnerID != callerID → ErrForbidden` branch (the shared-owner carve-out);
  `type` is now covered by the existing `AtLeast(PermissionOwner)` check.
- [x] 3.8 `handler.go`: remove routes `POST /api/account-types`, `PATCH
  /api/account-types/{id}`, `DELETE /api/account-types/{id}`, `POST
  /api/account-types/{id}/disable`, `POST /api/account-types/{id}/enable`
  and their handler funcs and `accountTypeBody`; change `GET
  /api/account-types` to call `svc.ListInUseTypes` and write a `[]string`
  (never `null` — default to `[]string{}`).
- [x] 3.9 `handler.go`: in `accountBody` rename `TypeID *string`
  (`json:"type_id"`) to `Type *string` (`json:"type"`) and update `create`
  / `update` mapping.
- [x] 3.10 `internal/httpapi/account.go`: remove the
  `registerErrStatus(account.ErrTypeInUse, ...)` and
  `registerErrStatus(account.ErrTypeDisabled, ...)` lines.

## 4. Backend: storage implementations

- [x] 4.1 `internal/storage/memory/account.go`: remove the `types` map,
  its seeding, and the seven type methods; implement `ListInUseTypes` by
  scanning the owner's non-deleted accounts for distinct trimmed `type`
  values, sorted case-insensitively; update the account struct/field and
  all account read/write paths from `type_id` to `type`.
- [x] 4.2 `internal/storage/postgres/account.go`: remove the
  `account_types` SQL and the seven type methods; add `ListInUseTypes` =
  `SELECT DISTINCT type FROM accounts WHERE owner_id = $1 AND deleted_at IS
  NULL ORDER BY lower(type)`; change every account `SELECT`/`INSERT`/
  `UPDATE` from `type_id` to `type` (column list, `RETURNING`, scan
  targets, `Update` set-clause builder).
- [x] 4.3 `internal/storage/postgres/reset.go`: drop `account_types` from
  the truncate/reset list; keep `accounts`.

## 5. Backend: wiring, CLI, remaining call sites

- [x] 5.1 `main.go`: drop `accountSvc` from
  `auth.WithNewUserHooks(accountSvc, categorySvc)` so only `categorySvc`
  remains; remove any now-unused local.
- [x] 5.2 `internal/cli/fixtures.go`: remove the `accountSvc.SeedDefaults`
  call and set each fixture account's `Type` to a literal label (reuse the
  former default titles inline); update the surrounding comment.
- [x] 5.3 `internal/cli/seed.go`: adjust the `account.NewService` usage /
  fixture account construction for the `Type` field if it references
  `type_id` or account types.
- [x] 5.4 `grep -rn "type_id\|TypeID\|ListTypes\|CreateType\|SeedDefault\|
  ErrTypeInUse\|ErrTypeDisabled\|DefaultTypeTitles\|account_types" backend`
  and confirm only intended references remain; `cd backend && go build
  ./...`.

## 6. Backend: tests

- [x] 6.1 `internal/account/service_test.go` and `handler_test.go`: delete
  account-type CRUD / seeding / disabled-type / `ErrTypeInUse` /
  `ErrTypeDisabled` cases; update account create/update fixtures to `type`;
  add cases for blank-`type` rejection (`422`) and `type` trimming.
- [x] 6.2 `internal/account/service_test.go` (or `sharing_test.go`): add a
  case that a shared `owner`-tier caller can `PATCH` `type` successfully;
  remove the case asserting `403` on `type_id` from a shared owner.
- [x] 6.3 Add a test for `GET /api/account-types` returning the caller's
  distinct in-use `type` values sorted, excluding another user's, and
  `[]` when the caller has no accounts.
- [x] 6.4 `internal/account/icon_color_test.go`: update any account
  fixtures from `type_id` to `type`.
- [x] 6.5 `internal/storage/postgres/account_test.go`,
  `account_sharing_test.go`, `icon_color_test.go`, `reset_test.go`,
  `helper_test.go`: replace `type_id`/type-lookup setup with a `type`
  string; drop `account_types` insertion helpers.
- [x] 6.6 `internal/cli/fixtures_test.go` and
  `internal/auth/service_test.go`: drop assertions about seeded account
  types; keep category-seeding assertions.
- [x] 6.7 `cd backend && go test ./...` green (with the postgres-backed
  suite against a local `postgres:17`).

## 7. Frontend

- [x] 7.1 Delete `frontend/src/routes/settings.account-types.tsx`; remove
  its entry from the tab list in `frontend/src/routes/settings.tsx`;
  regenerate the route tree (`routeTree.gen.ts`).
- [x] 7.2 Add a `DEFAULT_ACCOUNT_TYPES` constant (`["Checking","Savings",
  "Cash","Credit Card","Loan","Investment"]`) in a small `lib/` module or
  in `AccountForm.tsx`.
- [x] 7.3 `components/AccountForm.tsx`: replace the `AccountType` fetch and
  the `<select>` (with `currentType`/disabled-option/`typeLocked` logic)
  with a required text `<input>` plus suggestions = union of
  `DEFAULT_ACCOUNT_TYPES` and `GET /api/account-types` (`string[]`),
  deduped by exact match, using the same focus-chip pattern as
  `financial_institute` (or a `<datalist>`); update `AccountFormValues`
  (`type_id` → `type`), `emptyAccountForm`, `validate` (non-empty trimmed
  `type`), and the submit body (`type` always sent).
- [x] 7.4 `components/AccountForm.tsx`: remove the `typeLocked` prop and
  the conditional/omitted-field body construction it drove.
- [x] 7.5 `routes/accounts.$accountId.edit.tsx`: map `type: data.type`
  instead of `type_id: data.type_id`; stop passing `typeLocked`; remove
  the related comment.
- [x] 7.6 `routes/accounts.index.tsx`: replace `typeName(account.type_id)`
  / the `types` lookup with `account.type` directly; drop the
  `GET /api/account-types` fetch there.
- [x] 7.7 `lib/useAccountsWithBalances.ts`: remove the `AccountType` type,
  the `types` state, and the `GET /api/account-types` fetch; drop `types`
  from its return shape and fix consumers.
- [x] 7.8 `grep -rn "account-types\|AccountType\|type_id\|typeLocked"
  frontend/src` and confirm only the repurposed `string[]` endpoint use
  remains.

## 8. Frontend: i18n

- [x] 8.1 Remove the `settings.accountTypes.*` key subtree from
  `frontend/src/i18n/locales/en.json` and `de.json`.
- [x] 8.2 Adjust `accounts.form.type*` keys: keep `type` label and a
  `typeRequired` message; remove `typeLockedValue`, `typeLockedHint`,
  `typePlaceholder` (if unused), `typeDisabledOption`, `typeDisabledHint`
  in both locales.
- [x] 8.3 Remove the Account Types tab label key from `settings.*` in both
  locales.
- [x] 8.4 Run the `i18n-coverage` check locally (`cd frontend && pnpm
  <i18n script>`) and confirm en/de parity.

## 9. Verification

- [x] 9.1 `cd backend && go build ./... && go vet ./... && go test ./...`.
- [x] 9.2 `cd frontend && pnpm lint && pnpm build && pnpm test` (if a test
  script exists).
- [x] 9.3 Manually: fresh DB → migrations to `0021` apply cleanly; create
  an account typing a new `type`; edit it; a shared `owner`-tier user can
  change its `type`; the account form suggests defaults + in-use values;
  `/settings` shows no Account Types tab.
- [x] 9.4 Confirm the CI `contract` job would pass: `openapi/openapi.yaml`,
  `backend/openapi.yaml`, and `frontend/src/api/schema.d.ts` are in sync.
- [x] 9.5 Update `openspec/specs` via the change's spec deltas at archive
  time (`/opsx:archive`), not before.

## 1. Backend: schema

- [x] 1.1 Add migration
  `backend/internal/storage/postgres/migrations/0013_account_type_fields.sql`:
  `ALTER TABLE account_types RENAME COLUMN name TO title`, `ADD COLUMN
  description text`, `ADD COLUMN disabled boolean NOT NULL DEFAULT false`.

## 2. Backend: `internal/account` domain + store

- [x] 2.1 `account.go`: rename `Type.Name` to `Type.Title`, add
  `Type.Description string` (`omitempty`) and `Type.Disabled bool`.
- [x] 2.2 `store.go`: replace `TypeExists(ctx, id) (bool, error)` with
  `GetType(ctx, id) (Type, error)` (`ErrNotFound` if missing); add
  `SetTypeDisabled(ctx, id string, disabled bool) (Type, error)`; update
  `CreateType`/`UpdateType` signatures to take title + description; add
  sentinel `ErrTypeDisabled` to the sentinel list and `Sentinels`.
- [x] 2.3 Implement `GetType`, `SetTypeDisabled`, and the updated
  `CreateType`/`UpdateType` in `internal/storage/memory` and
  `internal/storage/postgres`.

## 3. Backend: `internal/account` service + handler

- [x] 3.1 `service.go`: add a `resolveType(ctx, id) (Type, error)` helper
  (via `store.GetType`) used by `Create`, `Update`, and the new
  `DisableType`/`EnableType`; `Create` and `Update` return `ErrTypeDisabled`
  when the *effective* `type_id` (submitted value, or the account's current
  one if `Update` doesn't touch `type_id`) resolves to a disabled type.
- [x] 3.2 `service.go`: `CreateType`/`UpdateType` take title + description
  (title required, non-empty); add `DisableType`/`EnableType` wrapping
  `store.SetTypeDisabled`.
- [x] 3.3 `handler.go`: extend the account-type request/response bodies
  with `description`; add `POST /api/account-types/{id}/disable` and
  `POST /api/account-types/{id}/enable`, admin-gated via the existing
  `requireAdmin` helper.
- [x] 3.4 `internal/httpapi/account.go`: register
  `account.ErrTypeDisabled` → `422` (matching `entry.ErrAccountDisabled`'s
  convention for a well-formed request rejected by a business rule).
- [x] 3.5 Unit + handler tests: create/update a type with title+description;
  non-admin blocked from create/update/delete/disable/enable (`403`);
  `POST /api/accounts` with a disabled `type_id` rejected (`422`);
  `PATCH /api/accounts/{id}` on an account with a disabled current type,
  changing an unrelated field only, rejected (`422`); the same `PATCH` with
  a live `type_id` included succeeds and updates both fields; a follow-up
  edit after that succeeds without resupplying `type_id`; disabling a type
  in use leaves referencing accounts untouched (still readable, still
  accept new entries); `DeleteType` unchanged — still `409` when
  referenced (disabled or not), `204` otherwise.

## 4. API contract

- [x] 4.1 `openapi/openapi.yaml`: `AccountType` — rename `name` to `title`,
  add `description` (optional) and `disabled` (boolean, required); update
  `AccountTypeWrite` — rename `name` to `title`, add optional
  `description`. Add paths `POST /api/account-types/{id}/disable` and
  `POST /api/account-types/{id}/enable` (`200` → `AccountType`, `401`,
  `403`, `404`), each documented with every status code the handler can
  return, matching `/api/accounts/{id}/disable`'s shape.
- [x] 4.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [x] 4.3 `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`.
- [x] 4.4 Add `internal/openapicheck.AssertResponse` assertions to the new/
  changed handler tests; lint the spec with spectral.

## 5. Frontend: Account Types settings tab

- [x] 5.1 `frontend/src/routes/settings.account-types.tsx` (new route,
  `/settings/account-types`) — redirect to `/settings` if
  `!user.is_admin` (mirroring `settings.users.tsx`); fetch
  `GET /api/account-types` on mount; render a table (title, description,
  status) with per-row Edit / Disable-Enable / Delete actions and a create
  form/button.
- [x] 5.2 Add "Account Types" to the tab nav in
  `frontend/src/routes/settings.tsx`, admin-only alongside Users.
- [x] 5.3 Create/edit go through `POST`/`PATCH /api/account-types`; each
  confirmable action (disable, enable, delete) uses the same
  `@headlessui/react` `Dialog` confirmation pattern as the Users tab; a
  `409` on delete surfaces an inline error instead of updating the list.

## 6. Frontend: account form

- [x] 6.1 `frontend/src/components/AccountForm.tsx`: filter the type
  dropdown to non-disabled types; when editing an account whose current
  `type_id` is disabled, include it as an extra, visibly-disabled option so
  the current value still renders, and treat the field as invalid (reusing
  the existing `validate()`/`invalidField` mechanism) until a different,
  non-disabled type is chosen.

## 7. i18n

- [x] 7.1 Add new keys to `frontend/src/i18n/locales/en.json` first, then
  `de.json`: tab label, table headers, create/edit form labels, status
  labels (Active/Disabled), confirm-dialog copy for disable/enable/delete,
  the "type is disabled — choose another" hint on the account form.

## 8. Verify

- [x] 8.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`
  (including `internal/storage/postgres` integration tests against a real
  Postgres, per `backend/AGENTS.md`).
- [x] 8.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 8.3 Manual pass: as an admin, create/edit/disable/enable/delete an
  account type on the new tab; confirm a non-admin never sees the tab
  (including a direct `/settings/account-types` link); confirm the account
  form excludes disabled types when creating; confirm editing an account
  whose type has since been disabled forces a reselection before saving,
  and that leaving the account untouched still works fine for viewing and
  posting entries.
- [x] 8.4 Update `backend/AGENTS.md` and `frontend/AGENTS.md` if their
  existing descriptions of `internal/account`'s account-type slice or the
  settings tabs would otherwise go stale.

## 9. Spec sync

- [x] 9.1 Apply this change's `specs/web-client-settings` delta onto
  `openspec/specs/web-client-settings/spec.md` by hand (the `openspec` CLI
  is unavailable in this environment, as for prior changes).
- [x] 9.2 `specs/accounts` is intentionally **not** synced onto
  `openspec/specs/` yet: that capability doesn't exist there at all — it's
  still pending from the un-archived `add-accounts-entries` change,
  alongside whose own account-type requirements this change's delta
  (`specs/accounts/spec.md` in this change's own directory) will need
  reconciling once both changes are archived.

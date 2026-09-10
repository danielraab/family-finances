## 1. API contract

- [x] 1.1 Add a `patch` operation to `/api/auth/me` in `openapi/openapi.yaml`: `operationId: patchAuthMe`, tag `auth`, request body referencing a new `ProfileUpdate` schema, `200` → `#/components/schemas/User`, `400` → `#/components/schemas/Error`, `401` → the shared Unauthorized response.
- [x] 1.2 Add the `ProfileUpdate` schema to `openapi/openapi.yaml`: object, `required: [display_name]`, `display_name` string with `maxLength: 150` and a description of the trim + `^[\p{L} .'-]{0,150}$` rule and that empty clears the name.
- [x] 1.3 Regenerate the synced copy: `cd backend && go generate ./...`; confirm `backend/openapi.yaml` updated.
- [x] 1.4 Regenerate the client types: `cd frontend && pnpm generate:api`; confirm `frontend/src/api/schema.d.ts` gained `patchAuthMe` / `ProfileUpdate`.
- [x] 1.5 Lint the spec (`cd openapi` per `openapi/README.md`) and confirm the `contract` checks pass locally.

## 2. Backend — auth domain helpers and errors

- [x] 2.1 In `backend/internal/auth/auth.go` add `NormalizeDisplayName(s string) string` (trim surrounding whitespace).
- [x] 2.2 In `auth.go` add `ValidateDisplayName(s string) error` — accepts the trimmed value when it matches `^[\p{L} .'-]{0,150}$` (empty allowed), else returns `ErrInvalidDisplayName`. Use a package-level compiled `regexp`.
- [x] 2.3 In `auth.go` add `SanitizeDisplayName(s string) string` — trim, delete every rune outside `[\p{L} .'-]`, collapse runs of spaces left behind, truncate to 150 runes. Used only for provider-sourced values.
- [x] 2.4 In `backend/internal/auth/store.go` add `ErrInvalidDisplayName` and include it in `Sentinels`.
- [x] 2.5 In `backend/internal/httpapi/auth.go` add `registerErrStatus(auth.ErrInvalidDisplayName, http.StatusBadRequest)`.
- [x] 2.6 Add table-driven tests for `NormalizeDisplayName`, `ValidateDisplayName`, `SanitizeDisplayName` (spaces, apostrophes, accented letters, `.`/`-`, markup, 150-char boundary, comma/paren stripping, empty).

## 3. Backend — store method

- [x] 3.1 In `backend/internal/auth/store.go` add `SetUserDisplayName(ctx context.Context, userID string, name *string) (User, error)` to the `Store` interface (nil ⇒ clear).
- [x] 3.2 Implement it in `backend/internal/storage/postgres/auth.go`: `UPDATE users SET display_name = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING <userCols>`, binding `nil` for the clear case; return `ErrNotFound` when no row.
- [x] 3.3 Implement it in `backend/internal/storage/memory/auth.go` with matching semantics.
- [x] 3.4 Add a postgres store test (set, clear, unknown id ⇒ `ErrNotFound`, soft-deleted user ⇒ `ErrNotFound`) and a memory store test mirroring it.

## 4. Backend — service and handler

- [x] 4.1 In `backend/internal/auth/service.go` add `SetDisplayName(ctx context.Context, userID, raw string) (User, error)`: normalize, validate, then `SetUserDisplayName` with `nil` when the normalized value is empty else `&value`.
- [x] 4.2 In `backend/internal/auth/handler.go` register `PATCH /api/auth/me` → `h.patchMe`; decode `{ display_name string }` (reuse the strict `decodeJSON`), require `UserFromContext` (else `writeUnauthorized`), call `svc.SetDisplayName`, and respond with the same `meResponse{User, Language}` shape `me` uses.
- [x] 4.3 In `backend/internal/oidcauth/oidcauth.go` add `Name string` to the claims struct and populate it from the ID token's `name` claim.
- [x] 4.4 In `service.go` `CompleteOIDC`, after the user is resolved and before the session is issued: compute `SanitizeDisplayName(claims.Name)`; if non-empty and different from the current `display_name`, call `SetUserDisplayName(ctx, user.ID, &sanitized)` and use the returned user. Empty ⇒ no-op.
- [x] 4.5 Handler tests: `PATCH /api/auth/me` happy path (returns updated user with `language`), trim, empty clears, invalid char ⇒ `400`, no session ⇒ `401`.
- [x] 4.6 Service/OIDC tests: `name` claim populates `display_name`; sanitised claim strips disallowed chars; claim overrides a previously self-set name; absent/empty claim leaves `display_name` unchanged.

## 5. Frontend — auth context and settings tab

- [x] 5.1 In `frontend/src/components/AuthProvider.tsx` add `setUser(user: User): void` to `AuthContextValue` and the provider value, wired to the existing `setUser` state setter.
- [x] 5.2 In `frontend/src/i18n/locales/en.json` and `de.json` rename `settings.tabs.common` → `settings.tabs.profile` and the `settings.common.*` field namespace → `settings.profile.*`; add `settings.profile.name` (label) and `settings.profile.nameError` (revert message) in both locales.
- [x] 5.3 In `frontend/src/routes/settings.tsx` update the tab entry to use `t("settings.tabs.profile")`.
- [x] 5.4 In `frontend/src/routes/settings.index.tsx` update all `settings.common.*` translation keys to `settings.profile.*`.
- [x] 5.5 In `settings.index.tsx` add a `Name` field at the top of the list: local draft seeded from `useAuth().user?.display_name`, save on blur only when the draft differs, `api.PATCH("/api/auth/me", { body: { display_name } })`; on `!response.ok` revert the draft and show `settings.profile.nameError`; on success call the context `setUser` with the response body.
- [x] 5.6 Confirm `SidebarUser` reflects the new name and monogram immediately after a save (no reload), via the updated context.

## 6. Verification

- [x] 6.1 `cd backend && go test ./... && go vet ./...` (and `gofmt`/lint per `backend/AGENTS.md`).
- [x] 6.2 `cd frontend && pnpm lint && pnpm test && pnpm build`.
- [x] 6.3 `cd frontend && pnpm i18n:coverage` (or the CI equivalent) passes with the renamed keys.
- [x] 6.4 Manual smoke: set a name with letters/spaces/`.`/`-`/`'` → sidebar updates; submit markup → field reverts with error; clear the field → sidebar falls back to email.
- [x] 6.5 `openspec validate profile-display-name` still passes; update `openspec/changes/profile-display-name/tasks.md` checkboxes as work completes.

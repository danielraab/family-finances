## Why

`display_name` is already carried end-to-end — the `users.display_name` column,
the `auth.User` / `auth.AdminUser` / `auth.InviteInviter` structs, the OpenAPI
`User` schema, the generated frontend types, and the UI that renders it (the
sidebar name line, the initials avatar) all exist today. Nothing ever writes
it, so every account's `display_name` is `NULL` and users are shown only their
email address. This change lights up the dormant field: it lets a signed-in
user set their own full name, and it seeds the name from the OIDC provider for
users who sign in that way.

## What Changes

- New endpoint `PATCH /api/auth/me` with body `{ "display_name": "..." }`,
  authenticated as the current user, returning the updated `User` (same shape
  as `GET /api/auth/me`, with the raw `language` folded in).
- The submitted value is trimmed of surrounding whitespace, then must match
  `^[\p{L} .'-]{0,150}$` (Unicode letters, spaces, `.`, `'`, `-`; at most 150
  characters). An empty string is valid and clears the name (stored as `NULL`).
  A value that fails validation is rejected with `400` and a new
  `auth.ErrInvalidDisplayName` sentinel.
- OIDC sign-in now re-syncs `display_name` from the provider's `name` claim on
  every sign-in: the claim value is trimmed, characters outside `[\p{L} .'-]`
  are stripped, the result is capped at 150 characters and stored. When the
  provider sends no `name` claim (or an empty one), the existing
  `display_name` is left untouched. A consequence, made explicit in the spec:
  a user with a linked OIDC identity cannot durably keep a self-edited name —
  their next OIDC sign-in overwrites it.
- The `openapi/openapi.yaml` source of truth gains the `PATCH /api/auth/me`
  operation and a `ProfileUpdate` request schema; both generated artifacts
  (`backend/openapi.yaml`, `frontend/src/api/schema.d.ts`) are regenerated in
  the same change.
- Frontend: the settings "Common" tab is renamed to "Profile" (visible label
  and i18n keys: `settings.tabs.common` → `settings.tabs.profile`,
  `settings.common.*` → `settings.profile.*`, in `en` and `de`; the route
  stays the `/settings` index). A "Name" field is added at the top of that
  tab, saving on blur with revert-on-error, mirroring the currency field's
  interaction. On a successful save the updated user is pushed into
  `AuthProvider` so the sidebar avatar and name update without a reload.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `authentication`: adds a requirement that an authenticated user can set
  their own display name (validation, trimming, 150-character cap, empty
  clears, unauthenticated rejected); modifies the OIDC sign-in requirement so
  a sign-in overrides `display_name` from the sanitized `name` claim when
  present and leaves it untouched when absent.
- `web-client-settings`: the "Common" tab becomes the "Profile" tab and gains
  a name field that saves immediately (blur), reverts on error, and updates
  the sidebar user control live.

## Impact

- **API contract**: `openapi/openapi.yaml` (+ regenerated `backend/openapi.yaml`
  and `frontend/src/api/schema.d.ts`); the CI `contract` job covers the new
  path automatically. Not breaking — a new optional endpoint plus a field that
  was already documented.
- **Backend** (`backend/internal/`):
  - `auth/auth.go` — `NormalizeDisplayName`, `ValidateDisplayName`,
    `SanitizeDisplayName`; new `ErrInvalidDisplayName` in `Sentinels`.
  - `auth/store.go` — new `Store.SetUserDisplayName(ctx, userID string, name *string) (User, error)` (`nil` clears).
  - `auth/service.go` — `SetDisplayName`; OIDC completion paths apply the
    sanitized claim.
  - `auth/handler.go` — `PATCH /api/auth/me` route + handler.
  - `oidcauth/oidcauth.go` — parse and expose the `name` ID-token claim.
  - `httpapi/auth.go` — map `ErrInvalidDisplayName` to `400`.
  - `storage/memory/auth.go`, `storage/postgres/auth.go` — implement
    `SetUserDisplayName`.
- **Frontend** (`frontend/src/`):
  - `components/AuthProvider.tsx` — expose a way to update the context user.
  - `routes/settings.index.tsx` — the name field and its save logic.
  - `routes/settings.tsx` — tab label key.
  - `i18n/locales/en.json`, `i18n/locales/de.json` — key rename + new strings.
- **Database**: no migration — `users.display_name` already exists (nullable
  `text`, migration `0002_auth.sql`).

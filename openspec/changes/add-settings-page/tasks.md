## 1. Backend: `user_settings` schema and `internal/settings`

- [ ] 1.1 Add migration `backend/internal/storage/postgres/migrations/0003_user_settings.sql`:
  `user_settings(user_id PK/FK, language, timezone, default_currency` all
  nullable`, updated_at)`, with a `CHECK (language IN ('en','de'))`.
- [ ] 1.2 Create `backend/internal/settings/` in the four-file shape
  (`settings.go` — domain type + hardcoded defaults `en`/`UTC`/`EUR` +
  validation; `service.go`; `store.go` — `Store` interface + sentinel errors;
  `handler.go` — `GET`/`PUT /api/settings`).
- [ ] 1.3 `timezone` validation via `time.LoadLocation`; `default_currency`
  validation via a three-uppercase-letter regexp; `language` validation
  against `{"en","de"}`.
- [ ] 1.4 Implement `Store` in `internal/storage/memory` (tests) and
  `internal/storage/postgres` (`INSERT ... ON CONFLICT (user_id) DO UPDATE`,
  touching only the fields present in the update).
- [ ] 1.5 Wire into `main.go`: build the store + service + handler, mount at
  `/api/settings`.
- [ ] 1.6 Unit tests (service, memory store) + handler tests
  (`httptest` + `internal/openapicheck.AssertResponse`).

## 2. Backend: user administration in `internal/auth`

- [ ] 2.1 Add migration `0004_user_administration.sql`:
  `ALTER TABLE users ADD COLUMN disabled boolean NOT NULL DEFAULT false, ADD
  COLUMN deleted_at timestamptz`.
- [ ] 2.2 Extend `auth.Store` with: list users (excluding soft-deleted), list
  invites (with inviter info), set `disabled`, set `deleted_at`, delete all
  sessions for a user. Implement in `storage/memory` and `storage/postgres`.
- [ ] 2.3 Auth middleware (`internal/httpapi/auth.go` / wherever session
  resolution lives): reject (`401`) when the resolved user is `disabled` or
  `deleted_at IS NOT NULL`, in addition to the existing session-validity
  check.
- [ ] 2.4 Magic-link callback and OIDC callback: refuse to establish a session
  for a disabled or soft-deleted account (new sentinel error, e.g.
  `ErrAccountDisabled`, mapped to a `400`-class response); magic-link
  `POST /api/auth/email/start` treats such an address like "no account" (no
  email sent, still `200`).
- [ ] 2.5 New handlers on `auth.Handler`, each gated on `user.IsAdmin`
  (`403` otherwise): `GET /api/auth/users`, `GET /api/auth/invites`,
  `POST /api/auth/users/{id}/disable`, `POST /api/auth/users/{id}/enable`,
  `DELETE /api/auth/users/{id}`. Disable/delete call the immediate
  session-revocation store method.
- [ ] 2.6 Unit tests: middleware rejects disabled/deleted; both sign-in flows
  reject disabled/deleted; each new endpoint (happy path, non-admin `403`,
  unauthenticated `401`, self-targeting including as the last admin).

## 3. API contract

- [ ] 3.1 `openapi/openapi.yaml`: add `UserSettings`, `UserSettingsUpdate`
  schemas; add `AdminUser` schema; extend `Invite` with `invited_by` (object:
  id/email/display_name) and `accepted_at`; add nullable `language` to
  `User`. Add paths `GET/PUT /api/settings`, `GET /api/auth/users`,
  `GET /api/auth/invites`, `POST /api/auth/users/{id}/disable`,
  `POST /api/auth/users/{id}/enable`, `DELETE /api/auth/users/{id}`, each with
  `401`/`403` responses where applicable.
- [ ] 3.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [ ] 3.3 `cd frontend && pnpm generate:api` to regenerate `src/api/schema.d.ts`.
- [ ] 3.4 Confirm the `contract` CI job's drift check would pass (lint the
  spec, diff the two generated copies).

## 4. Frontend: settings route and auth gate

- [ ] 4.1 `frontend/src/routes/settings.tsx` — layout route: redirect
  anonymous visitors to `/login`; render nothing while `useAuth` is
  `loading`; render the tab nav (Common always, Users only when
  `user.is_admin`) + `<Outlet/>`.
- [ ] 4.2 `frontend/src/routes/settings.index.tsx` — Common tab.
- [ ] 4.3 `frontend/src/routes/settings.users.tsx` — Users tab; redirect to
  `/settings` if `user.is_admin` is false.
- [ ] 4.4 `frontend/src/components/SidebarUser.tsx` — add the "Settings" menu
  item above "Log out", linking to `/settings`.

## 5. Frontend: Common tab

- [ ] 5.1 Fetch `GET /api/settings` on mount; render language `<select>`
  (English/German), timezone `<select>` (options from
  `Intl.supportedValuesOf("timeZone")`), default-currency text input
  (client-side validated to three letters).
- [ ] 5.2 Each control calls `PUT /api/settings` with only its own field on
  change; on a successful language change, call `i18n.changeLanguage(...)`
  immediately.
- [ ] 5.3 Surface a non-blocking error state per field on a failed update
  (value reverts to the last-known-good).

## 6. Frontend: Users tab

- [ ] 6.1 Fetch `GET /api/auth/users` and `GET /api/auth/invites` on mount;
  render a user table (avatar via existing `Avatar`, name/email, admin
  badge, disabled state, created date) and an invitations list
  (email, inviter, created/expires/accepted).
- [ ] 6.2 Invite form: email input, `POST /api/auth/invites`, append/refresh
  the invitations list on success.
- [ ] 6.3 Per-row Disable/Enable and Delete controls, each behind a
  confirmation step; self-targeting confirmation copy calls out that the
  admin will be signed out immediately.
- [ ] 6.4 On a successful self-disable/self-delete, transition `useAuth` to
  `anonymous` locally (mirroring `logout()`'s local state update) and
  navigate away from `/settings/users`.

## 7. i18n

- [ ] 7.1 Add new keys to `frontend/src/i18n/locales/en.json` first (per
  `web-client-i18n`), then `de.json`: sidebar menu ("Settings"), settings
  page chrome (tab labels), Common tab (field labels, save-error text),
  Users tab (table headers, invite form, confirmation copy, disable/delete
  action labels).
- [ ] 7.2 `frontend/src/i18n/index.ts` / `AuthProvider.tsx`: once
  `GET /api/auth/me` resolves with a non-null `language`, call
  `i18n.changeLanguage(user.language)`.

## 8. Verify

- [ ] 8.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`.
- [ ] 8.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 8.3 Manual pass: non-admin sees no Users tab and is redirected off
  `/settings/users`; admin sees and can use the Users tab; disabling a user in
  one session makes their other session's next request `401`; setting a
  language preference and reloading shows it beating browser detection;
  self-disable signs the acting admin out immediately in the UI.
- [ ] 8.4 Update `backend/AGENTS.md` (new `internal/settings` package, the
  `users` schema additions, the admin endpoints) and `frontend/AGENTS.md`
  (the settings route, the now-implemented account-level language override —
  remove the "planned follow-up" note it currently carries).

## 9. Spec sync

- [ ] 9.1 After implementation, fold these deltas into
  `openspec/specs/user-settings/spec.md`,
  `openspec/specs/user-administration/spec.md`,
  `openspec/specs/web-client-settings/spec.md` (new files), and apply the
  `authentication`, `web-client-auth`, `web-client-i18n` deltas to their
  existing specs (by hand if the `openspec` CLI remains unavailable in this
  environment, as for `2026-09-03-frontend-i18n`).

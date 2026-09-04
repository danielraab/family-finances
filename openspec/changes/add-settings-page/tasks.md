## 1. Backend: `user_settings` schema and `internal/settings`

- [x] 1.1 Add migration `backend/internal/storage/postgres/migrations/0003_user_settings.sql`:
  `user_settings(user_id PK/FK, language, timezone, default_currency` all
  nullable`, updated_at)`, with a `CHECK (language IN ('en','de'))`.
- [x] 1.2 Create `backend/internal/settings/` in the four-file shape
  (`settings.go` — domain type + hardcoded defaults `en`/`UTC`/`EUR` +
  validation; `service.go`; `store.go` — `Store` interface + sentinel errors;
  `handler.go` — `GET`/`PUT /api/settings`).
- [x] 1.3 `timezone` validation via `time.LoadLocation`; `default_currency`
  validation via a three-uppercase-letter regexp; `language` validation
  against `{"en","de"}`.
- [x] 1.4 Implement `Store` in `internal/storage/memory` (tests) and
  `internal/storage/postgres` (`INSERT ... ON CONFLICT (user_id) DO UPDATE`,
  touching only the fields present in the update).
- [x] 1.5 Wire into `main.go`: build the store + service + handler, mount at
  `/api/settings`.
- [x] 1.6 Unit tests (service, memory store) + handler tests
  (`httptest` + `internal/openapicheck.AssertResponse`).
  → Also added `internal/storage/postgres` integration tests (real Postgres,
  started locally in this environment since Docker wasn't available) for the
  upsert-merge SQL, the `CHECK` constraint, and `ON DELETE CASCADE`.

## 2. Backend: user administration in `internal/auth`

- [x] 2.1 Add migration `0004_user_administration.sql`:
  `ALTER TABLE users ADD COLUMN disabled boolean NOT NULL DEFAULT false, ADD
  COLUMN deleted_at timestamptz`.
- [x] 2.2 Extend `auth.Store` with: list users (excluding soft-deleted), list
  invites (with inviter info), set `disabled`, set `deleted_at`, delete all
  sessions for a user. Implement in `storage/memory` and `storage/postgres`.
- [x] 2.3 Auth middleware (`internal/httpapi/auth.go` / wherever session
  resolution lives): reject (`401`) when the resolved user is `disabled` or
  `deleted_at IS NOT NULL`, in addition to the existing session-validity
  check.
  → Implemented in `Service.Authenticate` (the single choke point
  `httpapi.authResolve` calls), not `httpapi/auth.go` itself — keeps the
  disabled/deleted check next to the rest of session validation rather than
  splitting it across packages.
- [x] 2.4 Magic-link callback and OIDC callback: refuse to establish a session
  for a disabled or soft-deleted account (new sentinel error
  `ErrAccountDisabled`, mapped to `403`); magic-link
  `POST /api/auth/email/start` treats such an address like "no account" (no
  email sent, still `200`).
- [x] 2.5 New handlers on `auth.Handler`, each gated on `user.IsAdmin`
  (`403` otherwise): `GET /api/auth/users`, `GET /api/auth/invites`,
  `POST /api/auth/users/{id}/disable`, `POST /api/auth/users/{id}/enable`,
  `DELETE /api/auth/users/{id}`. Disable/delete call the immediate
  session-revocation store method.
- [x] 2.6 Unit tests: middleware rejects disabled/deleted (including a
  surviving-session-row race case); both sign-in flows reject
  disabled/deleted; each new endpoint (happy path, non-admin `403`,
  unauthenticated `401`, self-targeting including as the last admin). Plus
  `internal/storage/postgres` integration tests for the new admin store
  methods against real Postgres.

## 3. API contract

- [x] 3.1 `openapi/openapi.yaml`: add `UserSettings`, `UserSettingsUpdate`
  schemas; add `AdminUser` schema; extend `Invite` with `invited_by` (object:
  id/email/display_name) and `accepted_at`; add nullable `language` to
  `User`. Add paths `GET/PUT /api/settings`, `GET /api/auth/users`,
  `GET /api/auth/invites`, `POST /api/auth/users/{id}/disable`,
  `POST /api/auth/users/{id}/enable`, `DELETE /api/auth/users/{id}`, each with
  `401`/`403` responses where applicable.
- [x] 3.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [ ] 3.3 `cd frontend && pnpm generate:api` to regenerate `src/api/schema.d.ts`.
- [ ] 3.4 Confirm the `contract` CI job's drift check would pass (lint the
  spec, diff the two generated copies).

## 4. Frontend: settings route and auth gate

- [x] 4.1 `frontend/src/routes/settings.tsx` — layout route: redirect
  anonymous visitors to `/login`; render nothing while `useAuth` is
  `loading`; render the tab nav (Common always, Users only when
  `user.is_admin`) + `<Outlet/>`.
- [x] 4.2 `frontend/src/routes/settings.index.tsx` — Common tab.
- [x] 4.3 `frontend/src/routes/settings.users.tsx` — Users tab; redirect to
  `/settings` if `user.is_admin` is false.
- [x] 4.4 `frontend/src/components/SidebarUser.tsx` — add the "Settings" menu
  item above "Log out", linking to `/settings`.

## 5. Frontend: Common tab

- [x] 5.1 Fetch `GET /api/settings` on mount; render language `<select>`
  (English/German), timezone `<select>` (options from
  `Intl.supportedValuesOf("timeZone")`), default-currency text input
  (client-side validated to three letters).
- [x] 5.2 Each control calls `PUT /api/settings` with only its own field on
  change; on a successful language change, call `i18n.changeLanguage(...)`
  immediately.
- [x] 5.3 Surface a non-blocking error state per field on a failed update
  (value reverts to the last-known-good).

## 6. Frontend: Users tab

- [x] 6.1 Fetch `GET /api/auth/users` and `GET /api/auth/invites` on mount;
  render a user table (avatar via existing `Avatar`, name/email, admin
  badge, disabled state, created date) and an invitations list
  (email, inviter, created/expires/accepted).
- [x] 6.2 Invite form: email input, `POST /api/auth/invites`, append/refresh
  the invitations list on success.
- [x] 6.3 Per-row Disable/Enable and Delete controls, each behind a
  confirmation step (a `@headlessui/react` `Dialog`, already a project
  dependency); self-targeting confirmation copy calls out that the admin
  will be signed out immediately.
- [x] 6.4 On a successful self-disable/self-delete, transition `useAuth` to
  `anonymous` locally (mirroring `logout()`'s local state update) and
  navigate away from `/settings/users`.

## 7. i18n

- [x] 7.1 Add new keys to `frontend/src/i18n/locales/en.json` first (per
  `web-client-i18n`), then `de.json`: sidebar menu ("Settings"), settings
  page chrome (tab labels), Common tab (field labels, save-error text),
  Users tab (table headers, invite form, confirmation copy, disable/delete
  action labels).
- [x] 7.2 `AuthProvider.tsx`: once `GET /api/auth/me` resolves with a
  non-null `language`, call `i18n.changeLanguage(user.language)`.

## 8. Verify

- [x] 8.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`.
  → Also ran the full suite against a real local PostgreSQL 16 (Docker's
  daemon wasn't reachable in this environment, so the OS package was
  installed and started directly) — all `internal/storage/postgres`
  integration tests pass, including new ones for the `user_settings` upsert
  SQL, its `CHECK` constraint, `ON DELETE CASCADE`, and the new admin store
  methods.
- [x] 8.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 8.3 Manual pass — done as a full integration smoke test, not just
  reasoning about the code: built the frontend, embedded it into a real
  `go build` binary, ran it against the local Postgres with a local
  debug SMTP catcher (`python3 -m smtpd`) standing in for `SMTP_HOST`, and
  drove it two ways:
  - `curl`, cookie-jar-based: bootstrap admin sign-in via a real magic-link
    email, `GET`/`PUT /api/settings` (language precedence confirmed —
    `/api/auth/me`'s `language` flips to `"de"` right after the `PUT`),
    a second user signs in, admin lists users (`GET /api/auth/users`),
    the second user is forbidden from the same endpoint (`403`), admin
    disables the second user and their session immediately 401s on the
    very next request, admin creates and lists an invite (inviter identity
    included), admin soft-deletes the second user and they disappear from
    the listing.
  - Playwright against the real Chromium, using the harvested session
    cookie: anonymous visit to `/settings` redirects to `/login`; signed-in
    admin sees both tabs; switching the language `<select>` to German
    re-renders the whole page (including the tab labels and page title) in
    German immediately, no reload; the Users tab renders the table and
    invitations list correctly; the sidebar user menu shows "Einstellungen"
    above "Abmelden"; clicking "Löschen" on the admin's own row shows the
    confirmation dialog with the self-targeting warning
    ("Dies ist dein eigenes Konto — du wirst sofort abgemeldet."); no
    browser console errors throughout.
- [x] 8.4 Updated `backend/AGENTS.md` (new `internal/settings` package, the
  `users` schema additions, the admin endpoints) and `frontend/AGENTS.md`
  (the settings route, the now-implemented account-level language override —
  removed the "planned follow-up" note it used to carry).

## 9. Spec sync

- [x] 9.1 Folded these deltas into `openspec/specs/user-settings/spec.md`,
  `openspec/specs/user-administration/spec.md`,
  `openspec/specs/web-client-settings/spec.md` (new files), and applied the
  `authentication`, `web-client-auth`, `web-client-i18n` deltas to their
  existing specs by hand (the `openspec` CLI remains unavailable in this
  environment, as for `2026-09-03-frontend-i18n`).

## Why

There is no account surface at all today: no way for a signed-in visitor to
pick a language/timezone/currency, and no way for an admin to see or manage
the instance's users and invitations except the `admin` CLI. `frontend/AGENTS.md`
already flags a per-user language setting as "a planned follow-up once a
settings surface exists" — this change builds that surface and uses it.

## What Changes

- Add a settings page at `/settings`, reachable only when signed in (redirects
  to `/login` otherwise), linked from a new "Settings" item in the sidebar
  user menu (`SidebarUser`, above "Log out").
- The page has tabs: **Common** (every signed-in user) and **Users** (admins
  only — not rendered, and not routable, for non-admins).
  - **Common**: display language (English/German), timezone, default
    currency. Each field saves immediately on change (no separate Save
    button), matching the existing `ThemeSwitch` interaction pattern.
  - **Users**: lists all users and all invitations (with who invited whom),
    can invite a new user (reusing the existing invite flow), and can disable
    or (soft) delete a user.
- New backend domain package `internal/settings`: a `user_settings` table
  (one nullable-columns row per user; a missing row means "all defaults"),
  resolved server-side against hardcoded application defaults
  (`en` / `UTC` / `EUR`). `GET /api/settings` / `PUT /api/settings`
  (authenticated, partial update).
- Extend `internal/auth` with admin-only user/invite administration:
  `GET /api/auth/users`, `GET /api/auth/invites`,
  `POST /api/auth/users/{id}/disable`, `POST /api/auth/users/{id}/enable`,
  `DELETE /api/auth/users/{id}` (soft delete). Disabling or deleting a user
  revokes their active sessions immediately. Admins may act on their own
  account, including disabling or deleting themselves even as the last
  remaining admin — no server-side guardrail against self-lockout.
- `users` gains `disabled boolean` and `deleted_at timestamptz`; the
  authentication middleware and both sign-in flows (magic link, OIDC) now
  refuse a disabled or soft-deleted account.
- The account-level `language` preference (when set) takes priority over
  browser detection for an authenticated visitor; browser detection remains
  the mechanism for anonymous visitors and for an authenticated visitor with
  no preference set yet. This amends `web-client-i18n`'s current "no account
  setting" language, which was written before this surface existed.

## Capabilities

### New Capabilities

- `user-settings`: the `user_settings` table, hardcoded-default resolution,
  and the `GET`/`PUT /api/settings` endpoints.
- `user-administration`: admin-only listing and lifecycle management
  (disable/enable/soft-delete) of users, and admin-only listing of
  invitations.
- `web-client-settings`: the `/settings` route, its auth gate, its Common and
  Users tabs, and the sidebar link to it.

### Modified Capabilities

- `authentication`: a disabled or soft-deleted user can no longer establish
  or use a session; `is_admin` now gates the new administration endpoints
  (previously documented as gating nothing).
- `web-client-auth`: the authenticated sidebar user menu gains a "Settings"
  item; the client-side `User` type gains a nullable `language` field.
- `web-client-i18n`: an authenticated visitor's account-level language
  preference, when set, takes priority over browser detection.

## Impact

- **Dependencies**: none new (no timezone-list or currency-list package —
  timezone options come from the browser's `Intl.supportedValuesOf`, currency
  is validated by shape, not a canonical list).
- **Code**:
  - New `backend/internal/settings/` (four-file shape), migrations
    `0003_user_settings.sql` and `0004_user_administration.sql`.
  - `backend/internal/auth/` gains the admin listing/lifecycle endpoints and
    the disabled/deleted checks in the middleware and sign-in flows.
  - `frontend/src/routes/settings.tsx` (+ `settings.index.tsx`,
    `settings.users.tsx`), `frontend/src/components/SidebarUser.tsx` (new
    menu item), `frontend/src/components/AuthProvider.tsx` and
    `frontend/src/i18n/index.ts` (account-language override wiring).
  - New i18n keys in `frontend/src/i18n/locales/{en,de}.json`.
- **API contract**: `openapi/openapi.yaml` gains `UserSettings`,
  `UserSettingsUpdate`, `AdminUser` schemas, an `invited_by`/`accepted_at`
  extension to `Invite`, a nullable `language` on `User`, and the new paths
  above — regenerate `backend/openapi.yaml` and
  `frontend/src/api/schema.d.ts` in the same change.
- **Spec**: new `user-settings`, `user-administration`, `web-client-settings`;
  deltas on `authentication`, `web-client-auth`, `web-client-i18n`.

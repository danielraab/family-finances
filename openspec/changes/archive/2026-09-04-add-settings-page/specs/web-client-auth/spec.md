## MODIFIED Requirements

### Requirement: Browser auth state resolved from the backend

The web client SHALL determine the visitor's authentication state at runtime by
calling `GET /api/auth/me` exactly once when the app mounts, and SHALL expose
the result through a single shared context (`useAuth`). The context state SHALL
be one of: `loading` (the request is in flight or has not started), `anonymous`
(the request returned `401` or failed), or `authenticated` (the request
returned `200`, carrying the user record — `id`, `email`, optional
`display_name`, `is_admin`, and a nullable `language` reflecting the user's
raw, unresolved language preference — `null` when none is set). The
TypeScript type of that user record SHALL be derived from the generated
OpenAPI schema (`frontend/src/api/schema.d.ts`), not a separately hand-declared
interface, and the `GET /api/auth/me` call SHALL go through the typed API
client.

Because the client is a static bundle and the session cookie is `HttpOnly`, the
client MUST NOT attempt to read the cookie or infer the state any other way.
The context MUST NOT poll or re-fetch on window focus.

#### Scenario: Signed-in visitor is recognised on load

- **WHEN** a visitor with a valid session cookie loads the app
- **THEN** the client calls `GET /api/auth/me` once, receives `200`, and
  `useAuth` reports `authenticated` with that user

#### Scenario: Signed-out visitor

- **WHEN** a visitor with no valid session loads the app
- **THEN** `GET /api/auth/me` returns `401` and `useAuth` reports `anonymous`

#### Scenario: Backend unreachable

- **WHEN** the `GET /api/auth/me` request fails (network error, `5xx`)
- **THEN** `useAuth` reports `anonymous` and the app still renders

#### Scenario: State is resolved once

- **WHEN** the app has mounted and `useAuth` has left the `loading` state
- **THEN** no further `GET /api/auth/me` request is made on tab focus or route
  change

#### Scenario: User record type comes from the generated schema

- **WHEN** `frontend/src/components/AuthProvider.tsx` is inspected
- **THEN** the user record type is taken from the generated OpenAPI schema and
  the `/api/auth/me` request is issued via the typed API client

#### Scenario: No language preference is null, not omitted

- **WHEN** an authenticated user has never set a language preference
- **THEN** the `GET /api/auth/me` response includes `language: null` rather
  than omitting the field

### Requirement: Authenticated user menu with sign-out

Activating the authenticated user control SHALL open an accessible menu — it
SHALL be dismissable with `Escape` and by clicking outside, SHALL move focus
into the menu on open and restore it on close, and SHALL be navigable by
keyboard. The menu SHALL contain two items, in order: "Settings" (navigates
to `/settings`) and "Log out".

Choosing "Log out" SHALL call `POST /api/auth/logout` and, on completion, set
`useAuth` to `anonymous` without a full page reload, so the sidebar control
immediately shows the "Log in" affordance again.

#### Scenario: Opening and dismissing the menu

- **WHEN** the visitor activates the authenticated user control
- **THEN** a menu opens containing "Settings" and "Log out" items, in that
  order
- **AND** pressing `Escape` or clicking outside closes it and returns focus to
  the control

#### Scenario: Settings item navigates to the settings page

- **WHEN** the visitor chooses "Settings" from the menu
- **THEN** the client navigates to `/settings`

#### Scenario: Signing out

- **WHEN** the visitor chooses "Log out"
- **THEN** the client sends `POST /api/auth/logout`
- **AND** after it resolves, `useAuth` is `anonymous` and the sidebar shows
  "Log in" without the page reloading

#### Scenario: The session is actually revoked

- **WHEN** the visitor has signed out and the app reloads
- **THEN** `GET /api/auth/me` returns `401` and the visitor is `anonymous`

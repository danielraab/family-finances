# web-client-auth Specification

## Purpose
TBD - created by archiving change migrate-frontend-to-vite-spa. Update Purpose after archive.
## Requirements
### Requirement: Browser auth state resolved from the backend

The web client SHALL determine the visitor's authentication state at runtime by
calling `GET /api/auth/me` exactly once when the app mounts, and SHALL expose
the result through a single shared context (`useAuth`). The context state SHALL
be one of: `loading` (the request is in flight or has not started), `anonymous`
(the request returned `401` or failed), or `authenticated` (the request
returned `200`, carrying the user record — `id`, `email`, optional
`display_name`, `is_admin`). The TypeScript type of that user record SHALL be
derived from the generated OpenAPI schema (`frontend/src/api/schema.d.ts`), not a
separately hand-declared interface, and the `GET /api/auth/me` call SHALL go
through the typed API client.

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

### Requirement: Sidebar user control

The sidebar footer SHALL contain a user control positioned above the
colour-theme control. Its appearance SHALL follow the `useAuth` state:

- **loading** — a neutral person glyph with no text label, so a signed-in
  visitor does not see a "Log in" affordance flash before their identity
  resolves;
- **anonymous** — a control labelled "Log in" that navigates to `/login`;
- **authenticated** — an avatar (see "Initials-monogram avatar") followed by
  the visitor's name and email; activating it opens a menu.

When the sidebar is collapsed the control SHALL render as the glyph or avatar
alone, with the label available as a tooltip, and any menu it opens SHALL
remain usable (anchored beside the collapsed sidebar rather than clipped).

#### Scenario: Anonymous visitor sees a login affordance

- **WHEN** `useAuth` is `anonymous`
- **THEN** the sidebar footer shows a "Log in" control
- **AND** activating it navigates the client router to `/login`

#### Scenario: Loading state shows no login affordance

- **WHEN** `useAuth` is `loading`
- **THEN** the sidebar footer shows a neutral user glyph and no "Log in" label

#### Scenario: Authenticated visitor sees their identity

- **WHEN** `useAuth` is `authenticated`
- **THEN** the sidebar footer shows the initials-monogram avatar and the
  visitor's name and email

#### Scenario: Control works in the collapsed sidebar

- **WHEN** the sidebar is collapsed and `useAuth` is `authenticated`
- **THEN** only the avatar is shown, with the name/email as a tooltip
- **AND** activating it still opens the menu, fully visible beside the sidebar

### Requirement: Initials-monogram avatar

When a display picture is unavailable, the authenticated user control SHALL
render an initials monogram: one or two letters derived from `display_name`
when it is set, otherwise from the local part of the email address, over a
background colour deterministically chosen from a small fixed palette keyed by
the user's `id` (so the same user always gets the same colour).

#### Scenario: Monogram from a display name

- **WHEN** the authenticated user has `display_name` "Jane Doe"
- **THEN** the avatar shows "JD"

#### Scenario: Monogram from the email

- **WHEN** the authenticated user has no `display_name` and email
  `jane@example.com`
- **THEN** the avatar shows initials derived from "jane"

#### Scenario: Colour is stable per user

- **WHEN** the same user's avatar is rendered on different loads
- **THEN** the background colour is the same each time

### Requirement: Authenticated user menu with sign-out

Activating the authenticated user control SHALL open an accessible menu — it
SHALL be dismissable with `Escape` and by clicking outside, SHALL move focus
into the menu on open and restore it on close, and SHALL be navigable by
keyboard. The menu SHALL contain a single item, "Log out".

Choosing "Log out" SHALL call `POST /api/auth/logout` and, on completion, set
`useAuth` to `anonymous` without a full page reload, so the sidebar control
immediately shows the "Log in" affordance again.

#### Scenario: Opening and dismissing the menu

- **WHEN** the visitor activates the authenticated user control
- **THEN** a menu opens containing a "Log out" item
- **AND** pressing `Escape` or clicking outside closes it and returns focus to
  the control

#### Scenario: Signing out

- **WHEN** the visitor chooses "Log out"
- **THEN** the client sends `POST /api/auth/logout`
- **AND** after it resolves, `useAuth` is `anonymous` and the sidebar shows
  "Log in" without the page reloading

#### Scenario: The session is actually revoked

- **WHEN** the visitor has signed out and the app reloads
- **THEN** `GET /api/auth/me` returns `401` and the visitor is `anonymous`

### Requirement: Login route starts the magic-link flow

The route `/login` SHALL render a view with a single email input and a submit
control. On submit the client SHALL validate that the value looks like an email
address and then call `POST /api/auth/email/start` with `{ "email": <value> }`.
Regardless of the response body (the endpoint always returns `200`), the view
SHALL then replace the form with a confirmation panel stating that a sign-in
link has been sent to that address, and offering a way to enter a different
address. A network or `5xx` failure on the request SHALL show a non-blocking
"please try again" message with the form intact.

The view SHALL NOT reveal whether an account exists for the address. It SHALL
NOT offer a password field.

The view SHALL also offer OIDC sign-in when the backend reports it is available.
On mount the client SHALL call `GET /api/auth/config` through the typed API
client. When the response's `oidc` field is a non-null object, the view SHALL
render, **above** the email field, a control labelled with `oidc.label` that is a
link to `oidc.start_path` (a full-page navigation, not a `fetch`), followed by a
visual "or" divider separating it from the email form. When `oidc` is `null`, or
the request fails, the view SHALL render exactly the email form described above
with no OIDC control and no divider. While the request is in flight the email
form SHALL already be usable.

#### Scenario: Submitting an address

- **WHEN** the visitor enters `person@example.com` on `/login` and submits
- **THEN** the client sends `POST /api/auth/email/start` with that email
- **AND** the view shows a "check your inbox — link sent to person@example.com"
  confirmation with an option to use a different address

#### Scenario: Invalid address is caught client-side

- **WHEN** the visitor submits a value that is not a valid email address
- **THEN** no request is sent and the field shows a validation message

#### Scenario: Request failure is recoverable

- **WHEN** the `POST /api/auth/email/start` request fails or returns `5xx`
- **THEN** the form remains visible with a "please try again" message

#### Scenario: No account enumeration

- **WHEN** the visitor submits an address with no account
- **THEN** the confirmation panel is identical to the one shown for an address
  that does have an account

#### Scenario: OIDC provider button shown above the form when available

- **WHEN** `/login` mounts and `GET /api/auth/config` returns
  `{ "oidc": { "label": "Continue with Google", "start_path": "/api/auth/oidc/start" } }`
- **THEN** the view renders a "Continue with Google" link to
  `/api/auth/oidc/start` above the email field, with an "or" divider between it
  and the email form

#### Scenario: No OIDC button when unavailable

- **WHEN** `/login` mounts and `GET /api/auth/config` returns `{ "oidc": null }`
  or the request fails
- **THEN** the view shows only the email form, with no OIDC control and no
  divider

#### Scenario: OIDC control is a navigation, not a fetch

- **WHEN** the visitor activates the OIDC sign-in control
- **THEN** the browser performs a full-page navigation to
  `/api/auth/oidc/start` (the client does not `fetch` it)

### Requirement: Login route redirects an already-authenticated visitor

When `/login` is opened while `useAuth` is `authenticated`, the client SHALL
navigate to `/` (replacing history) rather than showing the sign-in form.

#### Scenario: Signed-in visitor navigates to /login

- **WHEN** an authenticated visitor opens `/login`
- **THEN** the client navigates them to `/` without showing the email form

#### Scenario: Anonymous visitor stays on /login

- **WHEN** an anonymous visitor opens `/login`
- **THEN** the sign-in form is shown and no redirect occurs

### Requirement: Auth calls are same-origin and credentialed

Every backend call from the client SHALL use a relative `/api/...` path with
credentials included, so the session cookie is sent same-origin. The client
SHALL ship no backend base URL. In development, requests to `/api/*` SHALL be
proxied to the Go backend by the Vite dev server.

#### Scenario: Calls are relative and credentialed

- **WHEN** the client calls any `/api/auth/*` endpoint
- **THEN** the request uses a relative path and includes credentials so the
  session cookie is sent

#### Scenario: Dev server proxies the API

- **WHEN** the app runs under `pnpm dev` and calls `/api/auth/me`
- **THEN** the Vite dev server forwards the request to the Go backend and the
  response reaches the client without CORS configuration


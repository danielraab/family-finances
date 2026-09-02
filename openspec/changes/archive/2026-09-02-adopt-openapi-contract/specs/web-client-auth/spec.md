## MODIFIED Requirements

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

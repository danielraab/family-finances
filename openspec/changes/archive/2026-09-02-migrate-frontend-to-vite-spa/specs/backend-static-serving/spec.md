## MODIFIED Requirements

### Requirement: Backend serves the frontend static export

The backend SHALL serve the frontend's built bundle at the root path and at
every frontend route, self-contained within the compiled binary — no external
file path or separate static host SHALL be required at runtime.

Because the frontend is a single-page app, the backend SHALL fall back to the
bundle's `index.html` for client routes: a `GET` or `HEAD` request for a
non-`/api/` path that matches no bundled file and whose path has no file
extension SHALL be answered with `index.html` and status `200`, so the
in-browser router renders that view on a direct load, refresh, or bookmark
(including runtime-parameterised paths such as `/account/234/edit`). A
non-`/api/` request whose path has a file extension (a real asset request) and
matches no file SHALL still receive `404`.

#### Scenario: Home page loads from the backend

- **WHEN** a client requests `/` from a running backend built with the
  frontend embedded
- **THEN** the response is the frontend's home page HTML with a `200` status

#### Scenario: Static assets load

- **WHEN** a client requests a hashed frontend asset path (e.g. under
  `/assets/`)
- **THEN** the backend returns that asset's content with a `200` status

#### Scenario: Client route falls back to the SPA shell

- **WHEN** a client requests a non-`/api/` path that matches no bundled file
  and has no file extension (e.g. `/login` or `/account/234/edit`)
- **THEN** the backend returns `index.html` with a `200` status and the
  in-browser router renders the matching view

#### Scenario: Unknown path returns a not-found page

- **WHEN** a client requests a non-`/api/` path that has a file extension and
  matches no bundled file (e.g. `/assets/gone.js`)
- **THEN** the backend returns a `404` status, not the SPA shell — using the
  bundle's `404.html` body if one is present

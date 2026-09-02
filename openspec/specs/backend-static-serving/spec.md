# backend-static-serving Specification

## Purpose
TBD - created by archiving change go-serves-frontend-docker-ci. Update Purpose after archive.
## Requirements
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

### Requirement: Backend routes reserved under /api/

The backend SHALL reserve the `/api/` path prefix for its own routes,
distinct from static frontend paths, so future API endpoints cannot collide
with frontend routes.

#### Scenario: Health check under the reserved namespace

- **WHEN** a client requests `GET /api/healthz`
- **THEN** the backend returns a `200` status

#### Scenario: API namespace does not serve frontend files

- **WHEN** a client requests a path under `/api/` that is not a defined
  backend route
- **THEN** the backend does not return frontend HTML/JS content for it

### Requirement: Backend builds and tests without a built frontend present

Local development and testing of the backend (`go build`, `go test`,
`go run`) SHALL succeed without requiring a frontend build to have been
produced first.

#### Scenario: Backend builds and tests pass from a fresh clone

- **WHEN** a developer runs `go build .` or `go test ./...` in `backend/`
  immediately after cloning the repository, without having built the
  frontend
- **THEN** the build and tests succeed


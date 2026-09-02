## Purpose

Serving the built frontend directly from the Go backend so the deployed
application is a single self-contained process with no separate static host,
and reserving a namespace for the backend's own API routes.

## ADDED Requirements

### Requirement: Backend serves the frontend static export

The backend SHALL serve the frontend's built static site at the root path and
at all frontend routes, self-contained within the compiled binary — no
external file path or separate static host SHALL be required at runtime.

#### Scenario: Home page loads from the backend

- **WHEN** a client requests `/` from a running backend built with the
  frontend embedded
- **THEN** the response is the frontend's home page HTML with a `200` status

#### Scenario: Static assets load

- **WHEN** a client requests a hashed frontend asset path (e.g. under
  `/_next/`)
- **THEN** the backend returns that asset's content with a `200` status

#### Scenario: Unknown path returns a not-found page

- **WHEN** a client requests a path that matches no static file and no
  backend route
- **THEN** the backend returns a `404` status with a not-found page body

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

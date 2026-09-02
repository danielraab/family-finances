# backend-package-architecture Specification

## Purpose

The layering and dependency-direction rules for the Go backend: `internal/`
for all application code, one-way imports (`main` → `httpapi`/`config`/`storage`
→ domain packages), environment access confined to `internal/config`, no
`net/http` or HTTP status codes in domain code, routing owned by
`internal/httpapi` under `/api/`, and the `//go:embed` directive pinned to
`package main` at the module root. These rules keep the package layout
documented in `backend/AGENTS.md` enforceable as the backend grows.

## Requirements

### Requirement: Application code lives under internal/

All backend application code SHALL live under `backend/internal/` so that no
module outside `at.draab/familyfinances` can import it. `package main` at the
module root SHALL contain only process wiring, the Docker healthcheck probe,
and the `//go:embed` directive.

#### Scenario: Root package contains only wiring files

- **WHEN** the `backend/` module root is listed
- **THEN** the only `.go` files in `package main` are `main.go`,
  `healthcheck.go`, and `embed.go`
- **AND** every other package is under `backend/internal/`

#### Scenario: main.go contains no route strings or business logic

- **WHEN** `backend/main.go` is read
- **THEN** it constructs config, storage, services, and the HTTP server and
  starts/stops it, and contains no HTTP route pattern strings and no
  request-handling logic

### Requirement: Dependencies flow one way

Imports SHALL flow in a single direction: `main` MAY import `internal/httpapi`,
`internal/config`, and `internal/storage/*`; those MAY import domain packages;
domain packages SHALL import none of `httpapi`, `config`, `storage`, or any
database driver. `storage/*` packages SHALL implement interfaces that the
domain package declares, and `main` SHALL choose and inject the
implementation.

#### Scenario: Domain package does not import infrastructure

- **WHEN** the import graph of any `internal/<noun>/` domain package is
  inspected
- **THEN** it imports neither `internal/httpapi`, `internal/config`,
  `internal/storage/...`, nor any DB driver

#### Scenario: No import cycles

- **WHEN** `go build ./...` runs in `backend/`
- **THEN** it succeeds with no import-cycle error

### Requirement: Environment access is isolated to internal/config

`os.Getenv` (and any other environment read) SHALL be called only within
`internal/config`. Every configurable value SHALL be a field on
`config.Config`, populated by `config.Load()`, documented in `.env.example`,
and passed down as a value.

#### Scenario: PORT is read through config

- **WHEN** the server needs the listen port
- **THEN** it reads `Config.Port` populated by `config.Load()`, and no code
  outside `internal/config` calls `os.Getenv("PORT")`

#### Scenario: Config load falls back to documented defaults

- **WHEN** `config.Load()` runs with no environment variables set
- **THEN** it returns a `Config` with `Port` equal to `8080`

### Requirement: Domain code is free of HTTP concerns

Domain and service code SHALL NOT import `net/http` and SHALL NOT reference
HTTP status codes. It SHALL communicate failure with sentinel errors;
`internal/httpapi` SHALL translate those errors to HTTP status codes in one
place.

#### Scenario: Service returns a sentinel error

- **WHEN** a service operation cannot find a record
- **THEN** it returns a package-level sentinel error (e.g. `ErrNotFound`), not
  an `http`-package value

#### Scenario: Status mapping lives in one file

- **WHEN** an HTTP handler needs to turn a domain error into a response
- **THEN** it calls the shared `writeError` helper in
  `internal/httpapi/respond.go`, which owns the error → status mapping

### Requirement: Routing is owned by internal/httpapi under /api/

`internal/httpapi` SHALL build the `*http.ServeMux`, mount each domain
package's `http.Handler` under its own `/api/<noun>/` prefix, and route every
non-`/api/` path to the static handler. Backend routes SHALL always live under
`/api/`. An unmatched path under `/api/` SHALL receive a JSON `404` rather than
falling through to the static site.

#### Scenario: Health route served under /api/

- **WHEN** a client requests `GET /api/healthz`
- **THEN** `internal/httpapi` serves it and returns `200` with body `ok`

#### Scenario: Non-API path falls through to static

- **WHEN** a client requests a path that does not start with `/api/`
- **THEN** the request is handled by the static handler, not a backend route

#### Scenario: Unknown API path does not serve frontend content

- **WHEN** a client requests a path under `/api/` that is not a defined
  backend route
- **THEN** the response is a JSON `404` and contains no frontend HTML/JS

### Requirement: The go:embed directive is pinned to package main at the module root

The `//go:embed all:static/out` directive SHALL remain in a `package main`
file at the `backend/` module root. The serving logic that consumes it
(`staticHandler`, `notFoundInterceptor`) SHALL live in
`internal/httpapi/static.go` and operate on an `fs.FS` value passed in by
`main.go` via `fs.Sub`.

#### Scenario: Embed directive location

- **WHEN** `backend/embed.go` is read
- **THEN** it is in `package main`, contains `//go:embed all:static/out`, and
  exports the resulting `embed.FS`

#### Scenario: Static handler takes an fs.FS

- **WHEN** `internal/httpapi` serves static files
- **THEN** its `staticHandler` accepts an `fs.FS` parameter and does not
  itself reference `embed` or `static/out`

### Requirement: In-memory storage is the default store home

`internal/storage/memory/` SHALL be the location for in-memory `Store`
implementations, used as the default for local development and for tests. A
`sqlite` sibling MAY be added later through its own change.

#### Scenario: Memory storage package exists and builds

- **WHEN** `go build ./...` runs in `backend/`
- **THEN** `internal/storage/memory` compiles

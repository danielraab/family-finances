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
module root SHALL contain only process wiring: configuration and dependency
construction, HTTP server start/stop, and dispatch of CLI subcommands to their
implementations. The Docker healthcheck probe SHALL remain in
`healthcheck.go`; every other CLI subcommand (for example `admin …`) SHALL be
implemented in `internal/cli`. The `//go:embed` directive SHALL remain in
`package main` at the module root.

#### Scenario: Root package contains only wiring files

- **WHEN** the `backend/` module root is listed
- **THEN** the only `.go` files in `package main` are `main.go`,
  `healthcheck.go`, and `embed.go`
- **AND** every other package is under `backend/internal/`

#### Scenario: main.go contains no route strings or business logic

- **WHEN** `backend/main.go` is read
- **THEN** it constructs config, storage, services, and the HTTP server and
  starts/stops it, dispatches recognised subcommands, and contains no HTTP
  route pattern strings and no request-handling logic

#### Scenario: CLI subcommands live in internal/cli

- **WHEN** `main.go` handles an `admin` subcommand
- **THEN** it parses which subcommand was requested and delegates to
  `internal/cli`, which contains the command logic

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
falling through to the static site. `internal/httpapi` SHALL also serve the
embedded OpenAPI document at `GET /api/openapi.yaml`. `internal/httpapi` MAY
provide authentication middleware that resolves a request-scoped user from a
session token (bearer header or cookie); that middleware SHALL NOT import a
storage package or database driver, depending instead on an interface satisfied
by the auth service.

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

#### Scenario: OpenAPI document route is served by httpapi

- **WHEN** a client requests `GET /api/openapi.yaml`
- **THEN** `internal/httpapi` returns `200` with `Content-Type: application/yaml`
  and the embedded document

#### Scenario: Auth middleware stays free of storage imports

- **WHEN** the import graph of `internal/httpapi` is inspected
- **THEN** it imports neither `internal/storage/...` nor a database driver,
  even though it can resolve an authenticated user

### Requirement: The go:embed directive is pinned to package main at the module root

The `//go:embed` directive for the frontend bundle SHALL remain in a
`package main` file at the `backend/` module root, embedding `static/out` (the
directory the Docker build populates from `frontend/out/`). The serving logic
that consumes it (`staticHandler` and its SPA-fallback behaviour) SHALL live in
`internal/httpapi/static.go` and operate on an `fs.FS` value passed in by
`main.go` via `fs.Sub`.

A second `//go:embed` directive in `package main` at the module root SHALL embed
`backend/openapi.yaml` — a committed copy of `openapi/openapi.yaml`, kept
byte-identical by a sync mechanism and a CI check. `main.go` SHALL pass the
embedded bytes into `internal/httpapi`, which serves them; `internal/httpapi`
SHALL NOT itself reference `embed` or the file path.

#### Scenario: Embed directive location

- **WHEN** `backend/embed.go` is read
- **THEN** it is in `package main`, contains a `//go:embed` directive for
  `static/out`, and exports the resulting `embed.FS`

#### Scenario: Static handler takes an fs.FS

- **WHEN** `internal/httpapi` serves static files
- **THEN** its `staticHandler` accepts an `fs.FS` parameter and does not
  itself reference `embed` or `static/out`

#### Scenario: OpenAPI document is embedded by package main

- **WHEN** the `package main` files at the `backend/` module root are read
- **THEN** one of them contains a `//go:embed` directive for `openapi.yaml` and
  the bytes are handed to `internal/httpapi` for serving

#### Scenario: httpapi serves the document without embedding it itself

- **WHEN** the source of `internal/httpapi` is inspected
- **THEN** it receives the OpenAPI bytes as a passed-in value and contains no
  `//go:embed` directive and no reference to `openapi.yaml` on disk

### Requirement: In-memory storage is the default store home

`internal/storage/memory/` SHALL be the location for in-memory `Store`
implementations, used as the default for local development and for tests.
`internal/storage/postgres/` SHALL be the location for the real,
PostgreSQL-backed `Store` implementations, injected by `main.go` for
production. No third store engine SHALL be added and no cross-engine
abstraction SHALL be built.

Third-party dependencies are permitted where they earn their place: the backend
SHALL use **no web framework, no router library, and no ORM**, but vetted
libraries (for example the PostgreSQL driver `github.com/jackc/pgx/v5`) MAY be
added, each through its own OpenSpec proposal that justifies it. Domain
packages SHALL still import no storage package and no database driver.

#### Scenario: Memory storage package exists and builds

- **WHEN** `go build ./...` runs in `backend/`
- **THEN** `internal/storage/memory` compiles

#### Scenario: Postgres storage package exists and builds

- **WHEN** `go build ./...` runs in `backend/`
- **THEN** `internal/storage/postgres` compiles

#### Scenario: Domain package still imports no storage or driver

- **WHEN** the import graph of any `internal/<noun>/` domain package is
  inspected
- **THEN** it imports neither `internal/storage/...` nor `github.com/jackc/pgx/...`


## MODIFIED Requirements

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

### Requirement: Routing is owned by internal/httpapi under /api/

`internal/httpapi` SHALL build the `*http.ServeMux`, mount each domain
package's `http.Handler` under its own `/api/<noun>/` prefix, and route every
non-`/api/` path to the static handler. Backend routes SHALL always live under
`/api/`. An unmatched path under `/api/` SHALL receive a JSON `404` rather than
falling through to the static site. `internal/httpapi` MAY provide
authentication middleware that resolves a request-scoped user from a session
token (bearer header or cookie); that middleware SHALL NOT import a storage
package or database driver, depending instead on an interface satisfied by the
auth service.

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

#### Scenario: Auth middleware stays free of storage imports

- **WHEN** the import graph of `internal/httpapi` is inspected
- **THEN** it imports neither `internal/storage/...` nor a database driver,
  even though it can resolve an authenticated user

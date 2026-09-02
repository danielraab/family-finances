## MODIFIED Requirements

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

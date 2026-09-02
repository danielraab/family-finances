# AGENTS.md — backend

Go HTTP API for family-finances. Module `at.draab/familyfinances`.

## Stack

- Go 1.26, **standard library only** (`net/http`, `log/slog`, `os`, `database/sql`
  when persistence lands). No web framework, no router library, no ORM.
- Do not add a dependency without an OpenSpec proposal. A database *driver*
  (e.g. `modernc.org/sqlite`) is the one anticipated exception and still needs
  the proposal.

## Package layout

The backend is a single Go module. Application code lives under `internal/` so
nothing outside this module can import it and the packages stay free to change.
`package main` at the module root is wiring only.

```
backend/
├── main.go              # package main — compose the app, start/stop the server
├── healthcheck.go       # package main — `server healthcheck` self-probe (Docker HEALTHCHECK)
├── embed.go             # package main — //go:embed all:static/out  (must stay here; see below)
├── static/out/          # embed target — placeholder .gitkeep in git; Docker overwrites with the real build
├── go.mod
├── .env.example
└── internal/
    ├── config/          # Config struct + Load() — the only place os.Getenv is called
    ├── httpapi/         # HTTP wiring shared by every endpoint
    │   ├── server.go    #   New(cfg, deps) *http.Server; Routes(deps) *http.ServeMux
    │   ├── middleware.go #   request logging, panic recovery, request id
    │   ├── respond.go   #   writeJSON / writeError helpers; sentinel-error → status mapping
    │   ├── static.go    #   staticHandler(fs.FS) + notFoundInterceptor
    │   └── health.go    #   GET /api/healthz
    ├── account/         # one package per product noun (account, transaction, budget, …)
    │   ├── account.go   #   domain type + validation + invariants — no HTTP, no SQL
    │   ├── service.go   #   use-case logic; depends on the Store interface below
    │   ├── store.go     #   Store interface + sentinel errors (ErrNotFound, ErrConflict…)
    │   ├── handler.go   #   http.Handler for /api/accounts…; maps HTTP ⇄ service
    │   └── *_test.go
    └── storage/
        ├── memory/      #   in-memory Store implementations — default for dev and tests
        └── sqlite/      #   real persistence, added via its own proposal
```

### Rules that keep this layout honest

- **Dependencies flow one way.** `main` → `httpapi` + `config` + `storage/*` →
  domain packages (`account`, …). Domain packages import none of the others —
  no `httpapi`, no `storage`, no DB driver. `storage/*` packages implement the
  `Store` interface that the domain package *declares*; `main` picks an
  implementation and injects it. If you find yourself needing a domain package
  to import `httpapi`, the type or helper you want belongs in the domain
  package instead.
- **Package per noun, not per layer.** New feature = new `internal/<noun>/`
  with the four-file shape (`<noun>.go`, `service.go`, `store.go`,
  `handler.go`). Do not create `handlers/`, `services/`, `models/` buckets —
  they force every feature to touch every package and invite import cycles.
- **No HTTP status codes or `net/http` imports in domain code.** Domain and
  service code returns sentinel errors; `httpapi/respond.go` translates them to
  status codes in one place.
- **No `os.Getenv` outside `internal/config`.** Everything configurable is a
  field on `config.Config`, documented in `.env.example`, passed down as a
  value.
- **`main.go` is wiring only** — no route strings, no business logic:
  1. `os.Args[1] == "healthcheck"` → run the probe, exit.
  2. `config.Load()`.
  3. Build the storage implementation from config.
  4. Build each domain service + handler; hand them to `httpapi.New`.
  5. `fs.Sub` the embedded FS, pass it in.
  6. `http.Server` with SIGINT/SIGTERM → `srv.Shutdown(ctx)`.
- **Routing lives in `internal/httpapi`.** `Routes()` builds the `*http.ServeMux`
  and mounts each domain package's `http.Handler` under its `/api/<noun>/`
  prefix. Backend routes are always under `/api/`; everything else falls
  through to the static handler.

### Testing

- Domain and service tests use `storage/memory` — no HTTP, no network.
- Handler tests use `net/http/httptest` plus the in-memory store; assert status
  code and JSON body. Prefer black-box `package <noun>_test`.
- `httpapi` middleware is tested on its own with a stub handler.
- Keep `_test.go` beside the code it exercises.

## Conventions

- Routing uses Go 1.22+ pattern syntax: `mux.HandleFunc("GET /api/accounts/{id}", h)`.
- Configuration comes from environment variables, loaded once in
  `internal/config`. Current vars are in `.env.example` (`PORT`, default
  `8080`). Go does not auto-load `.env` — export the vars or use direnv.
- Logging via `log/slog` to stderr; one structured line per request from the
  logging middleware.

## Serving the frontend

The compiled binary embeds and serves the frontend's static export — there is
no separate static host in production. The mechanics are load-bearing:

- `embed.go` stays in `package main` at the module root with
  `//go:embed all:static/out`. `//go:embed` cannot reference parent
  directories, and the Docker build does `COPY … static/out/` relative to
  `backend/` — moving the directive into `internal/` would break both. The
  *serving logic* (`notFoundInterceptor` etc.) is ordinary code and lives in
  `internal/httpapi/static.go`, taking an `fs.FS`; `main.go` does
  `fs.Sub(embedded, "static/out")` and passes the result in.
- **The `all:` prefix is required.** Next's static export writes hashed assets
  under `_next/`, and Go's embed patterns silently skip any file or directory
  starting with `_` or `.` unless the pattern carries `all:`. Dropping `all:`
  compiles fine and only breaks at runtime (missing JS/CSS), so don't remove
  it.
- `backend/static/out/` holds only a committed `.gitkeep` in this repo —
  `//go:embed` requires the path to exist at compile time, so the placeholder
  keeps `go build` / `go test` / `go run .` working from a fresh clone with no
  frontend build present. **It is not what ships**: the Docker build (root
  `Dockerfile`) overwrites this directory with the real `frontend/out/` output
  before compiling. A local `go build` produces a binary that serves an empty
  site — build the Docker image (or copy a `pnpm build` output into
  `static/out/` first) to get the real thing.
- Routes split at `/api/`: backend routes live under that prefix
  (`GET /api/healthz` today). An unmatched path under `/api/` gets a JSON
  `404` — the reserved namespace never falls through to the static site.
  Every non-`/api/` path goes to the embedded static file server, which
  serves the embedded `404.html` for unmatched paths. Add new backend
  endpoints under `/api/` so they never collide with a frontend route.

## Layout status

The `internal/` layout above is in place (`internal/config`,
`internal/httpapi`, `internal/storage/memory`). `package main` is
`main.go` (wiring + graceful shutdown), `healthcheck.go`, and `embed.go`.
No `internal/<noun>/` domain package exists yet — the first resource
endpoint adds one (with its `storage/memory` implementation) through
OpenSpec, following the four-file shape and dependency rules above.

## Before you're done

```bash
gofmt -l .        # must print nothing
go vet ./...
go test ./...
```

## Commands

```bash
go run .          # start on :8080 (or $PORT)
go build .        # production binary (serves an empty site without a frontend build)
```

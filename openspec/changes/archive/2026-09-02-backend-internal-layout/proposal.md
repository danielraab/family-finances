## Why

`backend/AGENTS.md` now documents a target package layout — `internal/config`,
`internal/httpapi`, one package per product noun, `storage/*` behind a
domain-declared `Store` interface — but the code is still a flat `package main`
(`main.go` + `static.go`) with routing, config reads, HTTP wiring, and the
static handler all mixed together. Every rule in AGENTS.md that "keeps the
layout honest" is currently unenforceable because the layout does not exist.

The first resource endpoint would otherwise have to carry this whole
restructure alongside its own feature work. Doing the structural move on its
own keeps that change reviewable and lets the endpoint land against a settled
skeleton.

## What Changes

- Split `package main` into three files, all at the module root:
  - `main.go` — wiring only: healthcheck short-circuit, `config.Load()`, build
    storage, build services/handlers, hand them to `httpapi.New`, run the
    server with signal-driven graceful shutdown.
  - `healthcheck.go` — the `server healthcheck` self-probe for the Docker
    `HEALTHCHECK` (unchanged behavior, moved out of `main.go`).
  - `embed.go` — only the `//go:embed all:static/out` directive and the
    exported `embed.FS`. Stays in `package main` at the root because
    `//go:embed` cannot cross into a parent directory and the Docker build
    copies `static/out/` relative to `backend/`.
- Introduce `internal/config`: a `Config` struct and `Load()`. This becomes
  the **only** place `os.Getenv` is called. `PORT` moves here.
- Introduce `internal/httpapi`:
  - `server.go` — `New(cfg, deps) *http.Server` and `Routes() *http.ServeMux`;
    mounts each domain handler under its `/api/<noun>/` prefix and everything
    else under the static handler.
  - `middleware.go` — request logging (`log/slog`, one structured line per
    request), panic recovery, request id.
  - `respond.go` — `writeJSON` / `writeError` helpers and the single
    sentinel-error → HTTP-status mapping.
  - `static.go` — `staticHandler(fs.FS)` and `notFoundInterceptor`, moved
    verbatim from the root `static.go`, now taking an `fs.FS` so `main.go`
    passes `fs.Sub(embedded, "static/out")` in.
  - `health.go` — `GET /api/healthz`.
- Introduce `internal/storage/memory/` as the default in-memory store home.
  No `Store` interface or noun package is added yet — there is no domain type
  to declare one — but the directory and its role are established so the first
  endpoint drops into place.
- Establish and document the dependency direction as an enforced rule:
  `main` → `httpapi` + `config` + `storage/*` → domain packages; domain
  packages import none of the others; `storage/*` implements interfaces the
  domain package declares.
- Update `backend/AGENTS.md` "Migrating from the current flat layout" section
  (the migration it describes is now done) and `backend/README.md`.

No product endpoint, no persistence, and no HTTP behavior change:
`GET /api/healthz` still returns `200 ok`, static serving and the 404 page are
byte-for-byte the same, and `go build` / `go test` still work from a fresh
clone with no frontend build present.

## Capabilities

### New Capabilities

- `backend-package-architecture`: the layering and dependency-direction rules
  for the Go backend — `internal/` for all application code, one-way imports
  (`main` → `httpapi`/`config`/`storage` → domain), config reads isolated to
  `internal/config`, no `net/http` or HTTP status codes in domain code,
  routing owned by `internal/httpapi` under `/api/`, and the `//go:embed`
  directive pinned to `package main` at the module root.

### Modified Capabilities

_None._ `backend-static-serving` behavior is unchanged — the static handler
and reserved `/api/` namespace move to `internal/httpapi` without any change
to what clients observe.

## Impact

- `backend/main.go` — reduced to wiring; healthcheck and embed split out.
- `backend/static.go` — removed; contents move to
  `internal/httpapi/static.go`.
- New: `backend/healthcheck.go`, `backend/embed.go`,
  `backend/internal/config/`, `backend/internal/httpapi/`,
  `backend/internal/storage/memory/`.
- `backend/main_test.go`, `backend/static_test.go` — move/retarget to the
  packages they now exercise (`internal/httpapi` handler + middleware tests via
  `net/http/httptest`; `internal/config` load test).
- Docs: `backend/AGENTS.md` (migration section), `backend/README.md`.
- No dependency changes — standard library only. No change to the root
  `Dockerfile`, `.github/workflows/`, or `openspec/config.yaml`.

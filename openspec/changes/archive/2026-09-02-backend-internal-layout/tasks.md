## 1. internal/config

- [x] 1.1 Create `backend/internal/config/config.go` with a `Config` struct
  (`Port string`) and `Load() Config` that reads `os.Getenv("PORT")` and
  defaults to `"8080"`.
- [x] 1.2 Create `backend/internal/config/config_test.go`: default when unset,
  override when `PORT` is set.
- [x] 1.3 Confirm `os.Getenv` now appears only in `internal/config`
  (`grep -rn "os.Getenv" backend/` — expect one file).

## 2. Split package main

- [x] 2.1 Create `backend/embed.go` in `package main` with only
  `//go:embed all:static/out` and `var staticFiles embed.FS`, plus a comment
  pointing at `AGENTS.md` §"Serving the frontend".
- [x] 2.2 Create `backend/healthcheck.go` in `package main`: move
  `healthcheck`, `healthcheckURL` from `main.go`; resolve the port via
  `config.Load()` instead of `os.Getenv`.
- [x] 2.3 Create `backend/healthcheck_test.go` from the healthcheck assertions
  currently in `main_test.go`.

## 3. internal/httpapi

- [x] 3.1 Create `internal/httpapi/static.go`: move `staticHandler` and
  `notFoundInterceptor` verbatim from `backend/static.go`; `staticHandler`
  takes an `fs.FS`; remove any `embed` / `static/out` reference.
- [x] 3.2 Create `internal/httpapi/static_test.go` (white-box `package httpapi`
  — `staticHandler` is unexported —, `testing/fstest`): serves a file, serves
  `404.html` body on unmatched path, falls back to plain 404 when no
  `404.html`, asserts `Content-Type`.
- [x] 3.3 Create `internal/httpapi/health.go`: `handleHealthz` returning `200`
  `text/plain` `ok`.
- [x] 3.4 Create `internal/httpapi/respond.go`: `writeJSON`, `writeError`, and
  a sentinel-error → status map via `registerErrStatus` (empty until the first
  domain package registers its sentinels; unknown → 500 + log).
- [x] 3.5 Create `internal/httpapi/middleware.go`: slog request logging (one
  structured line per request), panic recovery (→ 500), request-id generation
  put on the request-scoped `*slog.Logger` / context.
- [x] 3.6 Create `internal/httpapi/middleware_test.go`: panic in a stub
  handler yields `500` and a logged line; normal request logs once.
- [x] 3.7 Create `internal/httpapi/server.go`: `type Deps struct { Static
  fs.FS }`, `Routes(deps) *http.ServeMux` (mount `GET /api/healthz`, a JSON
  `404` for other `/api/` paths, then `mux.Handle("/",
  staticHandler(deps.Static))`), `New(cfg config.Config, deps Deps)
  *http.Server` applying middleware and setting `Addr`.
- [x] 3.8 Create `internal/httpapi/server_test.go`: `GET /api/healthz` → `200`
  `ok`; a non-`/api/` path is handled by the static handler; an undefined
  `/api/` path does not return static HTML.

## 4. Reduce main.go to wiring

- [x] 4.1 Rewrite `backend/main.go`: (1) `os.Args[1] == "healthcheck"` →
  `os.Exit(healthcheck())`; (2) `cfg := config.Load()`; (3) storage (none yet
  — comment placeholder); (4) build `httpapi.Deps` with
  `fs.Sub(staticFiles, "static/out")`; (5) `srv := httpapi.New(cfg, deps)`;
  (6) `run(srv)` — `srv.ListenAndServe` in a goroutine, `signal.NotifyContext`
  (SIGINT/SIGTERM) → `srv.Shutdown(ctx)` with a 10s timeout.
- [x] 4.2 Delete `backend/static.go`, `backend/static_test.go`, and
  `backend/main_test.go` (all assertions migrated to
  `internal/httpapi/*_test.go` and `healthcheck_test.go`).
- [x] 4.3 Verify `main.go` contains no route pattern strings and no
  request-handling logic.

## 5. internal/storage

- [x] 5.1 Create `backend/internal/storage/memory/doc.go` with a package
  comment stating its role (default in-memory `Store` implementations for dev
  and tests; `sqlite` sibling added via its own change).

## 6. Verify behavior is unchanged

- [x] 6.1 `cd backend && gofmt -l .` prints nothing.
- [x] 6.2 `cd backend && go vet ./...` clean.
- [x] 6.3 `cd backend && go test ./...` passes.
- [x] 6.4 `cd backend && go build .` from a state with only
  `static/out/.gitkeep` succeeds.
- [x] 6.5 Manual: ran on `PORT=8087`. `/api/healthz` → `200 ok`; `/nope` →
  `404`; `/api/nope` → `404 {"error":"not found"}` (not the static site).
- [x] 6.6 `server healthcheck` against the running server exits `0`. Stopped-
  server case: the sandbox kills any process that dials a refused local port,
  so it is covered by the unit test `TestHealthcheckURL` instead (the
  `127.0.0.1:0` case returns exit `1`).

## 7. Docs

- [x] 7.1 Updated `backend/AGENTS.md`: replaced "Migrating from the current
  flat layout" with a "Layout status" note; corrected the `server.go` diagram
  line to `Routes(deps)`; noted the JSON `404` for unmatched `/api/` paths.
- [x] 7.2 `backend/README.md` needed no change — the endpoints table and
  commands are still accurate.
- [x] 7.3 `openspec validate backend-internal-layout` — valid.

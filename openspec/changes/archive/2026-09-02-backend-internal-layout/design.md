## Context

The backend is ~2 files of `package main`: `main.go` (server bootstrap +
healthcheck + `PORT` read + route registration) and `static.go` (embed
directive + static handler + 404 interceptor). `backend/AGENTS.md` already
describes the intended destination — `internal/config`, `internal/httpapi`,
package-per-noun domain packages, `storage/*` behind a domain-declared `Store`
interface — and a "Migrating from the current flat layout" note saying the
first resource endpoint should establish it via OpenSpec.

This change does that structural move on its own, with no endpoint attached,
so the diff is a pure reorganization that reviewers can check against
"behavior is identical" rather than reading it intertwined with feature logic.

Constraints: standard library only (Go 1.26); `//go:embed` cannot reference a
parent directory; the Docker build does `COPY ... static/out/` relative to
`backend/`; `go build`/`go test`/`go run .` must keep working from a fresh
clone with an empty `static/out/` (just `.gitkeep`).

## Goals / Non-Goals

**Goals:**

- Create `backend/internal/config` and `backend/internal/httpapi` with the file
  shape AGENTS.md specifies.
- Split `package main` into `main.go` (wiring), `healthcheck.go` (probe),
  `embed.go` (embed directive only).
- Move `staticHandler` / `notFoundInterceptor` verbatim into
  `internal/httpapi/static.go`, parameterized on `fs.FS`.
- Add `middleware.go` (slog request log, panic recovery, request id) and
  `respond.go` (`writeJSON` / `writeError` + sentinel→status map) as the
  shared HTTP kit the first endpoint will use.
- Create `internal/storage/memory/` as an established (possibly near-empty)
  package.
- Keep every client-observable behavior identical; retarget the existing tests
  to their new packages.
- Update `backend/AGENTS.md` (migration section) and `backend/README.md`.

**Non-Goals:**

- No product/domain package (`account`, `transaction`, …) — there is no domain
  type yet, so no `Store` interface is declared in this change.
- No `internal/storage/sqlite`, no database driver, no persistence.
- No new HTTP endpoint or behavior. No change to `Dockerfile`,
  `.github/workflows/`, `.env.example` keys, or `openspec/config.yaml`.
- No graceful-shutdown behavior change beyond moving where it is wired (a
  `signal.NotifyContext` + `srv.Shutdown` may be added since AGENTS.md lists it
  in the `main.go` steps — see Open Questions).

## Decisions

### D1: `internal/httpapi` owns the mux; `main` passes dependencies in

`httpapi.New(cfg config.Config, deps Deps) *http.Server` where `Deps` carries
the static `fs.FS` and (later) domain handlers. `Routes()` builds the
`*http.ServeMux`: mount `GET /api/healthz` from `health.go`, mount future
`/api/<noun>/` handlers, and `mux.Handle("/", staticHandler(deps.Static))`.

*Why not keep route registration in `main`:* AGENTS.md explicitly forbids route
strings in `main.go`; centralizing them in `httpapi` is what makes the "backend
routes always under `/api/`" rule enforceable in one place.

*Alternative considered:* a `Router` interface per domain package that
self-registers. Rejected as premature — with zero domain packages it is
indirection with nothing behind it. Revisit when the second noun lands.

### D2: `embed.go` holds only the directive; `main` does `fs.Sub`

`embed.go`:

```go
package main

import "embed"

//go:embed all:static/out
var staticFiles embed.FS
```

`main.go` does `sub, err := fs.Sub(staticFiles, "static/out")` and puts `sub`
into `httpapi.Deps`. `internal/httpapi/static.go` receives an `fs.FS` and never
mentions `embed` or the `static/out` path.

*Why:* keeps the compile-time filesystem coupling (`//go:embed`, the Docker
`COPY` path) at the module root where it is forced to live, while the serving
logic becomes ordinary, unit-testable code that a test can drive with
`fstest.MapFS`.

### D3: `healthcheck.go` stays in `package main`

The probe (`server healthcheck` → HTTP GET `/api/healthz` on localhost → exit
code) is invoked as `os.Args[1] == "healthcheck"` before any server setup. It
stays in `package main` next to `main.go`. It may import `internal/config` to
resolve the port so the `os.Getenv` rule holds.

*Alternative:* move it to `internal/httpapi`. Rejected — it is a CLI entry
concern, not HTTP wiring, and keeping it in `main` matches AGENTS.md's file
list.

### D4: `middleware.go` and `respond.go` land now, even unused

The first endpoint needs request logging, panic recovery, and a
sentinel-error→status map. Adding them here (with their own tests against a
stub handler) means the endpoint change is purely the endpoint. `writeError`
starts with a minimal map (`ErrNotFound`→404, `ErrConflict`→409, default→500)
that the first `store.go` sentinel set will extend.

*Risk of speculative code:* kept small and directly tested; each helper is
something AGENTS.md already names, not invented here.

### D5: Test placement

- `static_test.go` → `internal/httpapi/static_test.go`, black-box
  `package httpapi_test`, driven by `fstest.MapFS` (no dependency on a built
  frontend).
- `main_test.go`'s healthcheck assertions → `healthcheck_test.go` in
  `package main`.
- New `internal/config/config_test.go` for the default/override of `PORT`.
- New `internal/httpapi/middleware_test.go` for recovery + logging via a stub
  handler.

### D6: `internal/storage/memory/` with no `Store` yet

Create the directory with a `doc.go` (package comment stating its role) so the
package exists, compiles, and is the obvious drop point for the first
in-memory store. No interface is defined here — interfaces belong to the domain
package that consumes them (D1 of AGENTS.md's layout rules).

*Alternative:* don't create it until the first endpoint. Rejected — the user
asked for the full skeleton, and an empty package is a cheap, unambiguous
signal of where storage goes.

## Risks / Trade-offs

- **[Speculative packages with thin content]** `middleware.go`, `respond.go`,
  and `storage/memory` carry little logic until the first endpoint. →
  Mitigation: each is something AGENTS.md already prescribes; keep them minimal
  and unit-tested; no public API beyond what the first endpoint will call.
- **[Behavior drift during the move]** Moving `staticHandler` /
  `notFoundInterceptor` could subtly change 404 handling or headers. →
  Mitigation: move the code verbatim; port `static_test.go` unchanged in intent
  (same assertions on status, `Content-Type`, and 404 body); add a test that
  serves a tree with and without `404.html`.
- **[`go:embed` breakage]** Splitting files risks dropping the `all:` prefix or
  moving the directive. → Mitigation: `embed.go` is the only file with the
  directive; a build from a fresh clone (empty `static/out/`) in CI already
  guards compilation; keep the `all:` prefix and a comment pointing at
  AGENTS.md.
- **[Bigger review surface than a rename]** New packages + moved tests is more
  than `git mv`. → Mitigation: no behavior change claim is verifiable by
  running the ported tests plus `go vet ./...` and `gofmt -l .`; the proposal
  scopes out anything behavioral.
- **[Import-direction rules are convention, not compiler-enforced]** Nothing
  fails the build if a future domain package imports `httpapi`. → Mitigation:
  document in AGENTS.md (already there); optionally add a lightweight
  `go test` that inspects imports with `go/packages` — deferred to Open
  Questions.

## Migration Plan

1. Add `internal/config` (`Config`, `Load`), move the `PORT` read; wire
   `config.Load()` into `main.go`.
2. Add `embed.go` (directive only); delete the directive from `static.go`.
3. Add `internal/httpapi/static.go` (moved handler, `fs.FS` param) +
   `health.go` + `server.go` (`New` / `Routes`) + `middleware.go` +
   `respond.go`.
4. Reduce `main.go` to wiring: healthcheck short-circuit → `config.Load` →
   build `httpapi.Deps` (with `fs.Sub`) → `httpapi.New` → serve with
   SIGINT/SIGTERM shutdown.
5. Move `healthcheck` funcs to `healthcheck.go`; delete root `static.go`.
6. Add `internal/storage/memory/doc.go`.
7. Port/retarget tests (D5); run `gofmt -l .`, `go vet ./...`,
   `go test ./...`.
8. Update `backend/AGENTS.md` migration section and `backend/README.md`.

**Rollback:** the change is self-contained to `backend/` and docs; revert the
commit. No data, no deployed contract, no infra touched.

## Open Questions

- **Graceful shutdown:** AGENTS.md's `main.go` step 6 calls for
  `http.Server` + `signal` → `srv.Shutdown(ctx)`. Add it in this change
  (small, fits "wiring only") or leave `ListenAndServe` as-is and let a later
  change add it? Leaning: add it now since `httpapi.New` returns `*http.Server`
  specifically to enable it.
- **Import-direction test:** worth adding a `go/packages`-based guard test now,
  or rely on AGENTS.md + review until there is more than one internal package
  to protect? Leaning: defer.
- **`request id` propagation:** generate per request and put on the slog
  logger and a response header (`X-Request-Id`), or logger only for now?
  Leaning: logger + context value; header can come with the first real
  endpoint.

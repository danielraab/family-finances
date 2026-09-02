# AGENTS.md — backend

Go HTTP API for family-finances. Module `at.draab/familyfinances`.

## Stack

- Go 1.26, **standard library only** (`net/http`, `log`, `os`). No web framework,
  no ORM, no third-party dependencies.
- Do not add a dependency without an OpenSpec proposal.

## Conventions

- Routing uses Go 1.22+ pattern syntax: `mux.HandleFunc("GET /{$}", handler)`.
- Configuration comes from environment variables. Current vars are documented in
  `.env.example` (`PORT`, default `8080`). Go does not auto-load `.env` — export
  the vars or use direnv.
- No persistence layer yet. When one is added it lives here — the frontend never
  talks to a database.

## Serving the frontend

The compiled binary embeds and serves the frontend's static export — there is
no separate static host in production.

- `static.go` declares `//go:embed all:static/out` over `backend/static/out/`.
  **The `all:` prefix is required**: Next's static export writes hashed assets
  under `_next/`, and Go's embed patterns silently skip any file or directory
  starting with `_` or `.` unless the pattern carries `all:`. Dropping `all:`
  compiles fine and only breaks at runtime (missing JS/CSS), so don't remove it.
- `backend/static/out/` holds only a committed `.gitkeep` in this repo —
  `//go:embed` requires the path to exist at compile time, so the placeholder
  keeps `go build`/`go test`/`go run .` working from a fresh clone with no
  frontend build present. **It is not what ships**: the Docker build (see the
  root `Dockerfile`) always overwrites this directory with the real
  `frontend/out/` output before compiling. A local `go build` produces a
  binary serving an empty site — build the Docker image (or manually copy a
  `pnpm build` output into `static/out/` first) to get the real thing.
- Routes split at `/api/`: backend routes live under that prefix
  (`GET /api/healthz` today); everything else falls through to the embedded
  static file server, which serves the embedded `404.html` for unmatched
  paths. Add new backend endpoints under `/api/` so they never collide with a
  frontend route.

## Before you're done

```bash
gofmt -l .        # must print nothing
go vet ./...
go test ./...
```

## Commands

```bash
go run .          # start on :8080 (or $PORT)
go build .        # production binary
```

# family-finances

A family finances application: a Go HTTP API and a Next.js web client.

## Layout

| Path        | What                                                                      |
| ----------- | ------------------------------------------------------------------------- |
| `backend/`  | Go HTTP API (standard library `net/http`). Owns all persistence.         |
| `frontend/` | Next.js 16 web client (App Router, React 19, Tailwind 4, Biome).         |
| `openspec/` | Change proposals and specs — how non-trivial work is planned.            |

The two packages have separate toolchains for local development. There is no
root-level task runner; `cd` into a package before running its tools. In
production they ship as a single Docker image — see Architecture below.

## Prerequisites

- Go 1.26+
- Node 20+
- pnpm 11+ (`corepack enable`)

## Getting started

Run the backend:

```bash
cd backend
go run .                 # http://localhost:8080
```

Then the frontend, in a second terminal:

```bash
cd frontend
pnpm install
pnpm dev                 # http://localhost:3000
```

## Architecture

The frontend never connects to a database. The Go backend owns all persistence
and is the frontend's only backend.

The frontend builds to a static bundle (`pnpm build` → `frontend/out/`). In
production, the Go backend embeds that bundle and serves it directly — there
is no frontend server, no separate static host, and no nginx. The frontend
ships no backend URL: it's served same-origin by the backend, so it calls
`/api/...` directly with no CORS or base-URL configuration.

## Build & run the container

```bash
docker build -t family-finances .
docker run -p 8080:8080 family-finances   # http://localhost:8080
```

The image is a multi-stage build: it builds the frontend, embeds the result
into the Go binary, and ships only the compiled binary in a minimal
non-root runtime image. See the root `Dockerfile`.

CI (`.github/workflows/ci.yml`) lints and tests both packages on every push
and pull request, and publishes this image to the GitHub Container Registry
(tagged `YYYYMMDD-N`) on successful pushes to the default branch.

## Working on the code

Each package has its own `README.md` and `AGENTS.md`. Non-trivial changes go
through OpenSpec — see `openspec/`.

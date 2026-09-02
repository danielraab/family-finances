# family-finances

A family finances application: a Go HTTP API and a client-only web app
(Vite + React + TanStack Router).

## Layout

| Path        | What                                                                      |
| ----------- | ------------------------------------------------------------------------- |
| `backend/`  | Go HTTP API (`net/http`, no framework). Owns all persistence (PostgreSQL). |
| `frontend/` | Static SPA — Vite, React 19, TanStack Router, Tailwind 4, Biome.          |
| `openspec/` | Change proposals and specs — how non-trivial work is planned.            |

The two packages have separate toolchains for local development. There is no
root-level task runner; `cd` into a package before running its tools. In
production they ship as a single Docker image — see Architecture below.

## Prerequisites

- Go 1.26+
- Node 20+
- pnpm 11+ (`corepack enable`)
- Docker (for the PostgreSQL database)

## Getting started

Start PostgreSQL, then the backend:

```bash
docker compose up -d db  # PostgreSQL on localhost:5432
cd backend
export DATABASE_URL=postgres://familyfinances:familyfinances@localhost:5432/familyfinances?sslmode=disable
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
in **PostgreSQL** (`DATABASE_URL`, required) and is the frontend's only
backend. The root `compose.yaml` (app + `postgres:17` + a named volume) is the
reference topology for local development, CI, and production; the single app
image runs alongside a Postgres it does not contain.

The frontend is a client-only SPA that builds to a static bundle (`pnpm build`
→ `frontend/out/`). In production, the Go backend embeds that bundle and serves
it directly — there is no frontend server, no separate static host, and no
nginx. Unmatched non-`/api/` routes are served `index.html` so client-side
routing handles deep links and refreshes. The frontend ships no backend URL:
it's served same-origin by the backend, so it calls `/api/...` directly with no
CORS or base-URL configuration.

## Build & run the container

```bash
docker compose up --build     # app + PostgreSQL; http://localhost:8080
```

Or build and run the image alone against your own database:

```bash
docker build -t family-finances .
docker run -p 8080:8080 -e DATABASE_URL=... family-finances
```

The image is a multi-stage build: it builds the frontend, embeds the result
into the Go binary, and ships only the compiled binary in a minimal
non-root runtime image. See the root `Dockerfile`. It needs a reachable
`DATABASE_URL` at startup — it applies migrations and then serves.

CI (`.github/workflows/ci.yml`) lints and tests both packages on every push
and pull request, and publishes this image to the GitHub Container Registry
(tagged `YYYYMMDD-N`) on successful pushes to the default branch.

## Working on the code

Each package has its own `README.md` and `AGENTS.md`. Non-trivial changes go
through OpenSpec — see `openspec/`.

# AGENTS.md

Monorepo for **family-finances**: a Go HTTP API (`backend/`) and a client-only
web app (`frontend/` — Vite + React + TanStack Router). The two have separate
toolchains for local development —
there is no root-level task runner — but ship as a **single Docker image** in
production: the Go backend embeds and serves the built frontend (see
Architecture below).

## Working in this repo

- **`cd` into `backend/` or `frontend/` before running any language tooling.**
  Commands run from the repo root will not work.
- Each package has its own `AGENTS.md` — read it before changing code there.
- `backend/` uses **Go modules**. `frontend/` uses **pnpm** exclusively — never
  `npm` or `yarn`, and never commit a `package-lock.json`.

## Architecture

- The frontend **never connects to a database**. The Go backend owns all
  persistence and is the frontend's only backend.
- The frontend builds to a **static bundle** (`pnpm build` → `frontend/out/`).
  In production, the Go backend embeds that bundle at compile time
  (`//go:embed`) and serves it directly — no Node server, no separate static
  host, no nginx. See `backend/AGENTS.md` for the embed mechanism.
- The frontend ships no backend URL: the backend serves it same-origin, so
  frontend code calls the backend via relative `/api/...` paths, with no CORS
  or base-URL configuration needed.
- Local development keeps the two toolchains separate (`cd backend` /
  `cd frontend`, no shared build). The **production release artifact** is a
  single Docker image (root `Dockerfile`) that fuses the two — that fusion is
  deliberate and confined to the release build; it does not change how you
  develop each package day to day.
- The backend requires a **PostgreSQL** database (`DATABASE_URL`). The single
  app image runs alongside a Postgres it does not contain: the root
  `compose.yaml` (app + `postgres:17` + a named volume) is the reference
  topology for local development, CI, and production. The frontend still never
  touches the database — the Go backend remains its only backend.
- CI (`.github/workflows/ci.yml`) lints and tests both packages on every push
  and pull request (the backend job runs against a `postgres` service
  container), runs a **contract** job (see API contract below), and publishes
  the image to GitHub Container Registry (tagged with the pushed git tag, plus
  `latest`) when a git tag is pushed.

## API contract

- `openapi/openapi.yaml` is the **hand-written source of truth** for the
  backend's JSON HTTP API (spec-first; no server code is generated from it). It
  is served at `GET /api/openapi.yaml`. See `openapi/README.md`.
- Two generated artifacts are derived from it and **committed**; edit the spec
  and regenerate both in the same change:
  - `backend/openapi.yaml` — a synced copy (`//go:embed` can't cross `..`);
    `cd backend && go generate ./...`.
  - `frontend/src/api/schema.d.ts` — `cd frontend && pnpm generate:api`.
- The CI `contract` job lints the spec and fails on any drift in those two
  files. The Docker build copies `openapi/openapi.yaml` into `backend/` before
  compiling, so the shipped document is always the source of truth.

## Changes

- Non-trivial work goes through **OpenSpec** (`openspec/`). Propose a change
  before implementing.

## Commits

- **Conventional Commits.** Type is one of `feat`, `fix`, `chore`, `docs`,
  `refactor`, `test`, `build`, `ci`, `perf`. Scope is optional and names the
  package, e.g. `feat(backend): add accounts endpoint`,
  `chore(frontend): bump biome`.
- Keep the subject imperative and under ~72 characters; explain *why* in the body
  when it isn't obvious.
- End every commit message with a `Co-Authored-By:` trailer for each author.
  Claude Code sessions also append a `Claude-Session:` trailer linking the
  session.

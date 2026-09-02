# AGENTS.md

Monorepo for **family-finances**: a Go HTTP API (`backend/`) and a Next.js web
client (`frontend/`). The two have separate toolchains for local development —
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
- CI (`.github/workflows/ci.yml`) lints and tests both packages on every push
  and pull request, and publishes the image to GitHub Container Registry
  (tagged `YYYYMMDD-N`) on successful pushes to the default branch.

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

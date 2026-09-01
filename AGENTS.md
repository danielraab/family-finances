# AGENTS.md

Monorepo for **family-finances**: a Go HTTP API (`backend/`) and a Next.js web
client (`frontend/`). The two are independent deployables with separate
toolchains — there is no root-level task runner.

## Working in this repo

- **`cd` into `backend/` or `frontend/` before running any language tooling.**
  Commands run from the repo root will not work.
- Each package has its own `AGENTS.md` — read it before changing code there.
- `backend/` uses **Go modules**. `frontend/` uses **pnpm** exclusively — never
  `npm` or `yarn`, and never commit a `package-lock.json`.

## Architecture

- The frontend **never connects to a database**. The Go backend owns all
  persistence and is the frontend's only backend.
- The frontend builds to a **static bundle** (`pnpm build` → `frontend/out/`)
  served by a plain static host — no Node server. It ships no backend URL; how
  the deployed client reaches the backend at runtime is not yet decided (see
  `openspec/changes/frontend-static-shell/design.md`, open question O1).
- Keep the two packages decoupled: no shared build, no shared code.

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

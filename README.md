# family-finances

[![CI](https://github.com/danielraab/family-finances/actions/workflows/ci.yml/badge.svg)](https://github.com/danielraab/family-finances/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](backend/go.mod)
[![React](https://img.shields.io/badge/React-19-149ECA?logo=react&logoColor=white)](frontend/package.json)
[![pnpm](https://img.shields.io/badge/pnpm-only-F69220?logo=pnpm&logoColor=white)](frontend/package.json)
[![Conventional Commits](https://img.shields.io/badge/commits-conventional-yellow.svg)](https://www.conventionalcommits.org)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A self-hosted family/household finance tracker: accounts, transactions,
recurring costs, categories, tags, reports, and a customizable dashboard —
with multi-user sharing so a household can track money together without
sharing one login.

It ships as a Go HTTP API (`backend/`) plus a client-only web app
(`frontend/` — Vite + React + TanStack Router), built together into a
**single Docker image** for production.

<img src="docs/images/home-dashboard.png" alt="family-finances home dashboard" width="800">

## Contents

- [Features](#features)
- [Tech stack](#tech-stack)
- [Getting started](#getting-started)
- [Architecture](#architecture)
- [API contract](#api-contract)
- [Build & run the container](#build--run-the-container)
- [Working on the code](#working-on-the-code)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Accounts** — balances, currencies, financial institutes, open/close
  lifecycle, per-account icon and color.
- **Entries** — a filterable, searchable, sortable transaction ledger with
  balance adjustments, bookmarkable filter state.
- **CSV/JSON import** — a guided wizard with column mapping and a dry-run
  preview before committing.
- **Recurring transactions** — templates for rent, subscriptions, salary,
  and the like, with per-year totals.
- **Categories & tags** — a per-user category tree plus flat tags, both
  shareable.
- **Reports** — on-demand summaries filtered by category, tag, account, and
  date range.
- **Home dashboard** — a per-user grid of configurable cards (balances,
  filtered sums, recent entries, bar charts).
- **Sharing** — invite other registered users to an account, category, or
  tag with tiered permissions.
- **Multi-user administration** — invites, admin roles, disable/soft-delete.
- **Magic-link + optional OIDC authentication**, English/German i18n, and a
  light/dark/system theme.

See **[docs/FEATURES.md](docs/FEATURES.md)** for a full walkthrough with
screenshots of every feature above.

## Tech stack

| Layer    | Stack                                                                 |
| -------- | ---------------------------------------------------------------------- |
| Backend  | Go, `net/http` (no framework), PostgreSQL                              |
| Frontend | Vite, React 19, TanStack Router, Tailwind 4, `i18next`, Biome          |
| API      | Hand-written OpenAPI contract (`openapi/openapi.yaml`), spec-first     |
| Release  | Single multi-stage Docker image; Go embeds the built frontend          |
| CI/CD    | GitHub Actions — lint, test, OpenAPI contract check, image publish     |
| Planning | [OpenSpec](openspec/) change proposals for non-trivial work            |

## Layout

| Path        | What                                                                      |
| ----------- | ------------------------------------------------------------------------- |
| `backend/`  | Go HTTP API. Owns all persistence (PostgreSQL).                           |
| `frontend/` | Static SPA — Vite, React 19, TanStack Router, Tailwind 4, Biome.          |
| `openapi/`  | Hand-written API contract, source of truth for both packages.            |
| `openspec/` | Change proposals and specs — how non-trivial work is planned.            |
| `docs/`     | Feature walkthrough and screenshots.                                     |

The two packages have separate toolchains for local development. There is no
root-level task runner; `cd` into a package before running its tools. In
production they ship as a single Docker image — see [Architecture](#architecture).

## Getting started

### Prerequisites

- Go 1.26+
- Node 20+
- pnpm 11+ (`corepack enable`)
- Docker (for the PostgreSQL database)

### Run it

Start PostgreSQL, then the backend:

```bash
docker compose --profile dev up -d # PostgreSQL on localhost:5432, adminer on 8081, mailpit on 8025
cd backend
set -a && source .env && set +a  # or: export DATABASE_URL=postgres://familyfinances:familyfinances@localhost:5432/familyfinances?sslmode=disable
go run .                 # http://localhost:8080
```

Then the frontend, in a second terminal:

```bash
cd frontend
pnpm install
pnpm dev                 # http://localhost:3000
```

Sign in at `http://localhost:3000` — magic-link emails land in mailpit at
`http://localhost:8025` in local dev (no real SMTP required).

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

## API contract

`openapi/openapi.yaml` is the hand-written source of truth for the backend's
JSON HTTP API (spec-first; no server code is generated from it) and is served
at `GET /api/openapi.yaml`. Two generated artifacts are derived from it and
committed — edit the spec and regenerate both in the same change:

- `backend/openapi.yaml` — synced copy for `//go:embed` (`cd backend && go generate ./...`)
- `frontend/src/api/schema.d.ts` — typed client types (`cd frontend && pnpm generate:api`)

CI's `contract` job lints the spec and fails on drift in either generated
file. See `openapi/README.md` for details.

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

CI (`.github/workflows/ci.yml`) lints and tests both packages on every pull
request and on every push to `master`, runs an API-contract check, and
publishes this image to the GitHub Container Registry (tagged with the
pushed git tag, plus `latest`) when a git tag is pushed.

## Working on the code

Each package has its own `README.md` and `AGENTS.md` — read the relevant one
before changing code there:

- [`backend/AGENTS.md`](backend/AGENTS.md)
- [`frontend/AGENTS.md`](frontend/AGENTS.md)

## Contributing

Non-trivial changes go through [OpenSpec](openspec/) — propose a change
before implementing (see `openspec/specs/` for the current, archived specs).

Commits follow [Conventional Commits](https://www.conventionalcommits.org/):
type one of `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `build`,
`ci`, `perf`, with an optional package scope, e.g.
`feat(backend): add accounts endpoint`. See [CHANGELOG.md](CHANGELOG.md) for
release history.

## License

Licensed under the [GNU General Public License v3.0](LICENSE) (GPL-3.0).

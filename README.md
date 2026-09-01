# family-finances

A family finances application: a Go HTTP API and a Next.js web client.

## Layout

| Path        | What                                                                      |
| ----------- | ------------------------------------------------------------------------- |
| `backend/`  | Go HTTP API (standard library `net/http`). Owns all persistence.         |
| `frontend/` | Next.js 16 web client (App Router, React 19, Tailwind 4, Biome).         |
| `openspec/` | Change proposals and specs — how non-trivial work is planned.            |

The two packages are independent deployables with separate toolchains. There is
no root-level task runner; `cd` into a package before running its tools.

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
cp .env.example .env.local
pnpm install
pnpm dev                 # http://localhost:3000
```

## Architecture

The frontend never connects to a database. The Go backend owns all persistence
and is the frontend's only backend; the frontend calls it over HTTP via
`BACKEND_URL` (server-side only).

## Working on the code

Each package has its own `README.md` and `AGENTS.md`. Non-trivial changes go
through OpenSpec — see `openspec/`.

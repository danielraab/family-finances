# frontend

Client-only web app for family-finances: **Vite + React 19 + TanStack Router**,
Tailwind 4, Biome, TypeScript. Builds to a **static bundle** in `out/` — no
Node server in production (the Go backend embeds and serves it).

## Prerequisites

- Node 22+
- pnpm 11+ (`corepack enable`)

## Getting started

```bash
pnpm install
pnpm dev            # Vite dev server on http://localhost:3000
```

The app calls the backend at relative `/api/...` paths; `pnpm dev` proxies
those to `http://localhost:8080`, so **run the Go backend too** (see
`../backend/README.md`). For the auth flows also start Mailpit
(`docker compose -f ../docker-compose.dev.yml up -d`) and set the backend's
`AUTH_BASE_URL=http://localhost:3000` so emailed sign-in links come back
through the dev origin.

## Scripts

| Command                | Description                                   |
| ---------------------- | --------------------------------------------- |
| `pnpm dev`             | Vite dev server (port 3000, `/api` proxied)   |
| `pnpm build`           | Static build to `out/`                        |
| `pnpm preview`         | Serve the built `out/` locally                |
| `pnpm lint`            | Biome check (lint + format + import order)    |
| `pnpm format`          | Biome format (writes changes)                 |
| `pnpm generate-routes` | Regenerate `src/routeTree.gen.ts`             |
| `pnpm exec tsc`        | Type-check (no emit)                          |

## Routing

File-based (`src/routes/`), via `@tanstack/router-plugin`.
`src/routeTree.gen.ts` is generated — don't edit it. Add a route by adding a
file; navigate with `<Link to="…">` from `@tanstack/react-router`.

## Backend

Ships no backend URL. In production the Go backend embeds this build and serves
it same-origin, returning `index.html` for any unmatched client route so deep
links and refreshes work. See [`../backend/AGENTS.md`](../backend/AGENTS.md).
The frontend never opens a database connection.

## Docker image

Built from the repo root, not here — the root `Dockerfile` builds this bundle
and embeds it into the Go binary. `docker build -t family-finances .` from the
repo root.

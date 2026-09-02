# frontend

Next.js 16 web client for family-finances (App Router, React 19, Tailwind 4,
Biome, TypeScript). Builds to a **static bundle** — no Node server in
production.

## Prerequisites

- Node 20+
- pnpm 11+ (`corepack enable`)

## Getting started

```bash
pnpm install
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000). Edit `app/page.tsx` — the
page hot-reloads.

## Building

```bash
pnpm build        # writes ./out (static HTML/CSS/JS)
npx serve out     # preview the static bundle, no Node server
```

`out/` is what gets deployed — in production, the Go backend embeds it at
compile time and serves it directly (see `../backend/AGENTS.md`); there is no
separate static host or nginx.

## Scripts

| Command       | Description                        |
| ------------- | --------------------------------- |
| `pnpm dev`    | Start the dev server              |
| `pnpm build`  | Static export to `out/`           |
| `pnpm lint`   | Biome check                       |
| `pnpm format` | Biome format (writes changes)     |

## Backend

The client ships no backend URL and currently fetches nothing. The Go backend
serves this static build directly (same origin), so when fetching starts it
should call relative `/api/...` paths — no base URL or CORS config needed. See
[`../backend/AGENTS.md`](../backend/AGENTS.md). The frontend never opens a
database connection; the Go backend owns all persistence.

# frontend

Next.js 16 web client for family-finances (App Router, React 19, Tailwind 4,
Biome, TypeScript).

## Prerequisites

- Node 20+
- pnpm 11+ (`corepack enable`)
- The [backend](../backend) running (default `http://localhost:8080`)

## Getting started

```bash
cp .env.example .env.local
pnpm install
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000). Edit `app/page.tsx` — the
page hot-reloads.

## Scripts

| Command        | Description                     |
| -------------- | ------------------------------- |
| `pnpm dev`     | Start the dev server           |
| `pnpm build`   | Production build               |
| `pnpm start`   | Serve the production build     |
| `pnpm lint`    | Biome check                    |
| `pnpm format`  | Biome format (writes changes)  |

## Configuration

| Variable      | Default                   | Description                                    |
| ------------- | ------------------------- | --------------------------------------------- |
| `BACKEND_URL` | `http://localhost:8080`   | Base URL of the Go backend. Server-side only. |

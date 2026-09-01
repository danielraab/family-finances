# AGENTS.md — frontend

Next.js web client for family-finances.

## Package manager

**pnpm only.** `pnpm install`, `pnpm dev`, `pnpm build`. Never use `npm` or
`yarn`; there is no `package-lock.json`.

## Tooling

- **Biome** for lint + format: `pnpm lint` (`biome check`), `pnpm format`
  (`biome format --write`). 2-space indent.
- **Tailwind 4**, CSS-first — configured via `@import "tailwindcss"` in
  `app/globals.css`. There is no `tailwind.config.js`.
- **TypeScript**, App Router, React 19. Components are Server Components by
  default; add `"use client"` only when a component needs it.

## Data

- The frontend talks to the Go backend over HTTP via `BACKEND_URL` (see
  `.env.example`), server-side only.
- **Never** open a database connection from the frontend. The Go backend owns
  all persistence.

<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

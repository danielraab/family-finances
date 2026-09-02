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
- **Theming** is class-based, not media-query-based. `app/globals.css` declares
  `@custom-variant dark (&:where(.dark, .dark *));`, so every `dark:` utility
  keys off a `.dark` class on `<html>`. `next-themes` (`app/components/Providers.tsx`)
  owns that class: three states (system / light / dark), persisted to
  `localStorage` under `ff:theme`, with an inline pre-paint script it injects to
  avoid a flash. `<html>` carries `suppressHydrationWarning` for that reason.
  The user-facing control is `app/components/ThemeToggle.tsx` in the sidebar
  footer.

## Build output

- `pnpm build` produces a **fully static** site in `out/` (`output: "export"` in
  `next.config.ts`). There is no Node server — in production the Go backend
  embeds and serves `out/` directly (no nginx, no separate static host).
  Preview it locally with `npx serve out`.
- This rules out server-only Next.js features: route handlers, middleware,
  request-time Server Component rendering, `next/image` optimization, ISR,
  rewrites/redirects/headers, server actions. Server Components still run, but at
  build time only.
- The app icon must stay a static asset (`app/icon.svg` + `app/favicon.ico`);
  regenerate the `.ico` from the SVG with `node scripts/generate-favicon.mjs`
  (pure Node, no deps).

## Data

- The frontend ships **no backend URL** and currently fetches nothing. At
  runtime, the Go backend embeds and serves this build directly (same
  origin) — see `../backend/AGENTS.md` §"Serving the frontend". When code
  here does start fetching, call relative `/api/...` paths; no base URL, env
  var, or CORS handling is needed since the backend serves both.
- **Never** open a database connection from the frontend. The Go backend owns
  all persistence.

<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

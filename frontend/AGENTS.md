# AGENTS.md — frontend

Client-only web app for family-finances. **Vite + React 19 + TanStack Router**,
built to a static bundle the Go backend embeds and serves. No Node server, no
SSR, no `BACKEND_URL`.

## Package manager

**pnpm only** (`packageManager` pins the version). `pnpm install`, `pnpm dev`,
`pnpm build`, `pnpm lint`. Never `npm` / `yarn`; `package-lock.json` is
git-ignored. Build-script allow-listing lives in `pnpm-workspace.yaml`
(`onlyBuiltDependencies`), not `package.json`.

## Tooling

- **Vite 8** (`vite.config.ts`): `@vitejs/plugin-react`, `@tailwindcss/vite`,
  and `@tanstack/router-plugin` (file-based routing + code-splitting).
  `build.outDir` is `out/` (the Go embed path); `server.proxy` forwards
  `/api/*` to `http://localhost:8080` in dev.
- **TanStack Router**, file-based. Routes are files under `src/routes/`;
  `src/routeTree.gen.ts` is **generated** — never edit it (it's excluded from
  Biome and marked read-only in `.vscode`). `pnpm dev` / `pnpm build`
  regenerate it; `pnpm generate-routes` does it on demand.
- **Biome** for lint + format: `pnpm lint` (`biome check`), `pnpm format`
  (`biome format --write`). 2-space indent. Scoped to `src/`, `index.html`,
  `vite.config.ts`.
- **Tailwind 4**, CSS-first via `@import "tailwindcss"` in `src/styles.css`.
  No `tailwind.config.js`.
- **TypeScript**, strict. `pnpm exec tsc` type-checks (no emit).

## Routing & layout

- `src/routes/__root.tsx` is the app shell: `ThemeProvider` + `AuthProvider`
  wrap `<Sidebar />` + `<main><Outlet /></main>`. Router devtools render only
  in dev.
- `src/routes/index.tsx` → `/`. `src/routes/login.tsx` → `/login`.
- Navigate with `@tanstack/react-router`'s `<Link to="…">` / `useNavigate()`;
  read the path with `useLocation()`. `to` is type-checked against the route
  tree.
- The Go backend serves `index.html` for any unmatched extensionless path, so
  every client route works on a hard load / refresh (see
  `../backend/AGENTS.md` §"Serving the frontend").

## Theming

Class-based (`.dark` on `<html>`), not media-query. `src/styles.css` declares
`@custom-variant dark (&:where(.dark, .dark *))`. `src/lib/theme.tsx` owns the
class: three states (system / light / dark), persisted to `localStorage` under
`ff:theme`, with an inline pre-paint script in `index.html` to avoid a flash.
The control is `src/components/ThemeToggle.tsx` in the sidebar footer. Fonts
(Geist / Geist Mono) are self-hosted via `@fontsource-variable/*`, imported in
`src/main.tsx`.

## Data & auth

- The app ships **no backend URL**. Call the Go backend at relative
  `/api/...` paths with `credentials: "same-origin"` so the `ff_session`
  cookie is sent. In dev the Vite proxy forwards `/api` to `:8080`; in
  production the Go binary serves both same-origin.
- `src/components/AuthProvider.tsx` resolves the session once on mount via
  `GET /api/auth/me` and exposes `useAuth() → { status, user, logout }`
  (`status ∈ loading | anonymous | authenticated`). No polling, no
  focus-refetch. `logout()` calls `POST /api/auth/logout` and flips to
  `anonymous` in place.
- `src/components/SidebarUser.tsx` renders the footer user control (initials
  monogram + sign-out menu via `@headlessui/react`).
- Data fetching is plain `fetch` in effects — no query library yet.
- **Never** open a database connection from the frontend.

## Build output

`pnpm build` → `frontend/out/` (`index.html` + hashed `assets/`), a fully
static bundle. Preview with `pnpm preview`. In production the Go backend
embeds `out/` at compile time (root `Dockerfile`); a plain local `go build`
serves an empty site until a real bundle is copied in.

## Before you're done

```bash
pnpm lint          # biome check — must pass
pnpm exec tsc      # type-check
pnpm build         # must write out/index.html
```

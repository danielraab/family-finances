## Why

The frontend is a Next.js App-Router project built with `output: "export"`,
but production runs no Node server — the Go binary embeds and serves the static
bundle. That combination is a poor fit: `output: "export"` prerenders a fixed
set of routes at build time, so a runtime URL like `/account/234/edit` has no
prerendered HTML and 404s on refresh, bookmark, or deep link. Working around it
means fighting the framework (catch-all hacks, `generateStaticParams`, RSC
payload 404s). The frontend is a client-only SPA in all but name; this change
makes it one.

It also absorbs the superseded `web-client-auth` change (sidebar user widget +
login page): its spec is framework-agnostic and its UI ports onto the new
stack, and keeping it as a separate delta against the same `web-client-shell`
requirement would create a spec-merge ordering hazard.

## What Changes

- **Replace Next.js with a Vite + React SPA.** New `frontend/` scaffolded with
  `create-tsrouter-app` (**TanStack Router**, file-based routing, Vite). The
  old app moves to `frontend-old/` as porting reference.
  - Build: `pnpm build` (Vite) → `frontend/out/` (`build.outDir: "out"`, so the
    backend embed path and Docker copy are unchanged); a single `index.html`
    plus hashed assets under `assets/`.
  - Dev: `vite.config.ts` `server.proxy` sends `/api/*` to
    `http://localhost:8080` (replacing the deleted `next.config.ts` rewrite).
  - Package manager: **pnpm only** — delete the scaffold's `package-lock.json`,
    set `.cta.json` accordingly, commit `pnpm-lock.yaml`.
  - Biome config realigned to the repo norm (2-space, `pnpm lint` runs
    `biome check`), excluding the generated `routeTree.gen.ts`.
- **Theme & fonts without Next.** Drop `next-themes` and `next/font`: a
  hand-rolled theme provider (three states, `localStorage` `ff:theme`, `.dark`
  on `<html>`) with a pre-paint inline script in `index.html`; Geist self-hosted
  via `@fontsource-variable/*`. `app/icon.svg` / `favicon.ico` become
  `public/` assets referenced from `index.html`.
- **Port every component** to the SPA: `Icon`, `Sidebar`, `ThemeToggle`,
  `Placeholder`, `HealthCheck`, and the `web-client-auth` pieces
  (`AuthProvider`/`useAuth`, `Avatar`, `monogram`, `SidebarUser`, the `/login`
  page). `next/link` → `@tanstack/react-router` `Link`; `usePathname` →
  `useLocation`/`useMatchRoute`; `next/navigation` `useRouter` → `useNavigate`;
  drop every `"use client"`. Route tree: `__root.tsx` = layout (providers +
  `Sidebar` + `<Outlet/>`), `routes/index.tsx` = home, `routes/login.tsx`.
- **Backend serves the SPA shell.** `internal/httpapi/static.go` gains an SPA
  fallback: an unmatched non-`/api/` path that is not an asset request is
  answered with `index.html` and `200` (so client routing handles any URL,
  dynamic params included). Missing assets and unmatched `/api/` paths still
  get a real `404`. The `//go:embed` rationale (the `all:` prefix existed for
  Next's `_next/`) is updated for Vite's `assets/`.
- **Docs, Docker, CI.** `frontend/AGENTS.md` / `README.md` / `CLAUDE.md`
  rewritten for the Vite/TanStack stack; `backend/AGENTS.md` "Serving the
  frontend" updated for the SPA fallback; `openspec/config.yaml` context
  updated. `Dockerfile` and `.github/workflows/ci.yml` need only minor
  adjustment (both already `pnpm`-based and both keep working against
  `frontend/out/` + `pnpm lint` / `pnpm build`).

Non-goals: route guarding / redirect-if-unauthenticated (still a later change);
an OIDC "sign in with a provider" button; TanStack Query or any data-fetching
library (hand-rolled `fetch` stays); TanStack **Start** / any server runtime;
migrating `frontend-old/` history beyond the `git mv`.

## Capabilities

### New Capabilities

- `web-client-auth`: the browser-side authentication experience — a `useAuth`
  context fed by one `GET /api/auth/me`, the sidebar user control (loading /
  anonymous / authenticated with an initials-monogram avatar and a sign-out
  menu), and the `/login` page that starts the magic-link flow and redirects
  away when already signed in. (Carried from the superseded `web-client-auth`
  change; rebased onto the Vite/TanStack stack.)

### Modified Capabilities

- `web-client-shell`: the shell is now a **Vite + React + TanStack Router
  SPA**, not a Next.js static export — no Server Components, no `next-themes`,
  no `next/font`, icon served as a `public/` asset. The build-output and
  no-`BACKEND_URL` contracts are preserved but restated for Vite (with the
  dev-only `/api` proxy noted). The collapsible-sidebar layout requirement also
  gains the user / sign-in control in its footer.
- `backend-static-serving`: the static handler serves the SPA `index.html`
  (200) for any unmatched non-`/api/` `GET`/`HEAD` path with no file extension,
  instead of a not-found page, so client-side routing owns every application
  URL (dynamic params included); missing assets (paths with an extension) still
  return `404`. Hashed-asset scenario updated from `_next/` to Vite's
  `assets/`.
- `backend-package-architecture`: only the `//go:embed` requirement changes —
  rationale updated for Vite's asset layout (`all:` no longer load-bearing) and
  the static handler's SPA-fallback behaviour is named. The "Routing is owned
  by `internal/httpapi`" requirement is unchanged by this change (the
  authentication change owns its current wording).

## Impact

- **`frontend/`**: full rebuild. New: `vite.config.ts`, `index.html`,
  `src/main.tsx`, `src/router.tsx`, `src/routes/*`, `src/styles.css`,
  `src/components/*` (ported), `src/lib/*`. Removed (moved to `frontend-old/`):
  `app/`, `next.config.ts`, `next-env.d.ts`, `postcss.config.mjs`.
  Dependencies: `+ @tanstack/react-router` (+ plugin/cli), `+ @headlessui/react`,
  `+ @fontsource-variable/geist(-mono)`, `+ vite`, `+ @vitejs/plugin-react`,
  `+ @tailwindcss/vite`; `- next`, `- next-themes`.
- **`backend/internal/httpapi/static.go`** + tests: SPA fallback.
  `backend/embed.go` comment, `backend/AGENTS.md` "Serving the frontend".
- **`Dockerfile`** frontend stage (paths already align if `outDir: "out"`),
  **`.github/workflows/ci.yml`** frontend job (verify `pnpm lint` / `pnpm build`).
- **Docs**: `frontend/AGENTS.md`, `frontend/README.md`, `frontend/CLAUDE.md`,
  `backend/AGENTS.md`, root `AGENTS.md` / `README.md` (frontend stack line),
  `openspec/config.yaml`.
- **No backend API change.** No database change. `/api/*` contract unchanged.
- Supersedes and removes the `web-client-auth` change (never applied).

## 1. Toolchain reconciliation

- [x] 1.1 Delete `frontend/package-lock.json`; set `.cta.json` `packageManager`
  to `pnpm`; run `pnpm install` so `frontend/pnpm-lock.yaml` is written; confirm
  no `package-lock.json` reappears.
- [x] 1.2 `frontend/vite.config.ts`: add `build.outDir: "out"` and
  `server.proxy` mapping `/api` → `http://localhost:8080` (keep `server.port`
  3000 via the `dev` script). Keep the TanStack Router + React + Tailwind
  plugins.
- [x] 1.3 `frontend/.gitignore`: add `/out` (Vite build) and `package-lock.json`.
- [x] 1.4 Realign `frontend/biome.json` to the repo norm: 2-space indent, keep
  `routeTree.gen.ts` and `styles.css` excluded; set scripts so `pnpm lint` runs
  `biome check` (format + lint + organize-imports), `pnpm format` runs
  `biome format --write`.
- [x] 1.5 Add `@headlessui/react` and `@fontsource-variable/geist` +
  `@fontsource-variable/geist-mono` with pnpm. Remove the CTA
  `@tanstack/react-devtools` panel wiring from `__root.tsx` if trimming; keep
  router devtools optional.

## 2. Styling, theme, fonts, shell chrome

- [x] 2.1 Merge `frontend-old/app/globals.css` into `frontend/src/styles.css`:
  `@import "tailwindcss"`, `@custom-variant dark (&:where(.dark, .dark *))`,
  `:root` / `.dark` `--background` / `--foreground`, `@theme inline` font vars,
  `body` background/color.
- [x] 2.2 `src/lib/theme.tsx` (or `src/components/ThemeProvider.tsx`): in-app
  provider — three states, `localStorage["ff:theme"]` (default `system`),
  resolve via `matchMedia`, toggle `.dark` on `<html>`, live-update while
  `system`. Expose a `useTheme()` returning `{ theme, setTheme }`.
- [x] 2.3 `frontend/index.html`: `<title>Family Finances</title>`, `lang="en"`,
  viewport, `<link rel="icon">` entries, and an inline pre-paint script that
  applies the resolved `.dark` class from `ff:theme` before the module loads.
- [x] 2.4 Fonts: `import "@fontsource-variable/geist"` +
  `"@fontsource-variable/geist-mono"` in `src/main.tsx`; set
  `--font-geist-sans` / `--font-geist-mono` in `styles.css`; apply the sans font
  on `body`.
- [x] 2.5 Copy `frontend-old/app/icon.svg` → `frontend/public/icon.svg` and
  `favicon.ico` → `frontend/public/favicon.ico`; port
  `frontend-old/scripts/generate-favicon.mjs` → `frontend/scripts/`.

## 3. Port existing components

- [x] 3.1 `src/components/Icon.tsx` — copy verbatim (no changes needed).
- [x] 3.2 `src/components/ThemeToggle.tsx` — port: drop `"use client"`, swap
  `next-themes` `useTheme` for the in-app `useTheme` from 2.2; keep the
  cycle/glyph logic and `collapsed` prop.
- [x] 3.3 `src/components/Sidebar.tsx` — port: drop `"use client"`;
  `next/link` → `@tanstack/react-router` `Link` (`to` not `href`);
  `usePathname()` → `useLocation({ select: l => l.pathname })` (or
  `useMatchRoute`) for the active nav item; keep collapse state +
  `localStorage["ff:sidebar-collapsed"]`.
- [x] 3.4 `src/components/Placeholder.tsx` and `src/components/HealthCheck.tsx` —
  copy; `HealthCheck` already uses plain `fetch`, keep it.

## 4. Route tree

- [x] 4.1 `src/routes/__root.tsx`: wrap the app in `ThemeProvider` +
  `AuthProvider`; render `<Sidebar />` + `<main class="flex-1 overflow-x-hidden"><Outlet /></main>`
  in a flex row matching the old `layout.tsx` body.
- [x] 4.2 `src/routes/index.tsx`: render `<Placeholder />` + `<HealthCheck />`
  (the old home page).
- [x] 4.3 Ensure `pnpm dev` regenerates `routeTree.gen.ts` and both routes
  resolve; delete the CTA demo copy from `index.tsx`.

## 5. Port web-client-auth

- [x] 5.1 `src/components/AuthProvider.tsx` — port: drop `"use client"`; keep
  the single `GET /api/auth/me` on mount (`credentials: "same-origin"`),
  `{ status, user, logout }`, `logout()` → `POST /api/auth/logout` then set
  `anonymous`. Export `useAuth()` and the `User` type.
- [x] 5.2 `src/lib/monogram.ts` — copy verbatim (`initials`, `colorFor`).
- [x] 5.3 `src/components/Avatar.tsx` — copy verbatim.
- [x] 5.4 `src/components/SidebarUser.tsx` — port: `next/link` → router `Link`;
  keep the Headless UI `Menu` / `MenuItems` (`anchor` top/right per
  `collapsed`), the loading / anonymous / authenticated branches, and the
  glyphs. Mount `<SidebarUser collapsed={collapsed} />` in the `Sidebar`
  footer, above `<ThemeToggle />`.
- [x] 5.5 `src/routes/login.tsx`: port the login view — email input + client
  validation → `POST /api/auth/email/start` → confirmation panel ("check your
  inbox", "use a different address"); `5xx`/network → retryable error; read
  `useAuth()` and `useNavigate()({ to: "/", replace: true })` when
  `authenticated`; no password field, no provider button.

## 6. Backend SPA fallback

- [x] 6.1 `backend/internal/httpapi/static.go`: on a would-be `404` for a
  `GET`/`HEAD` non-`/api/` request whose path has no file extension, serve the
  bundle's `index.html` with `200`; a path with an extension keeps the `404`.
  Preserve the existing behaviour for `/` and for real files.
- [x] 6.2 `backend/internal/httpapi/static_test.go`: known file → `200`;
  extensionless unknown (`/login`, `/account/234/edit`) → `index.html` `200`;
  `*.js` unknown → `404`; confirm `Routes` still returns JSON `404` for
  `/api/nope` (unchanged).
- [x] 6.3 `backend/embed.go` comment: note the bundle is now a Vite build under
  `assets/`; the `all:` prefix is retained but no longer load-bearing.
- [x] 6.4 `backend/AGENTS.md` "Serving the frontend": rewrite for the Vite SPA +
  `index.html` fallback (drop the Next `_next/` / `all:` explanation, drop
  `404.html`-for-every-unmatched-path wording).

## 7. Docs, Docker, CI

- [x] 7.1 `frontend/AGENTS.md`: replace the CTA boilerplate — Vite + React +
  TanStack Router (file-based routing, `routeTree.gen.ts` generated), pnpm,
  Biome, Tailwind 4 CSS-first, static build to `out/`, no Node server, no
  `BACKEND_URL`, `/api` dev proxy, in-app theme provider, `useAuth` from
  `AuthProvider`, `@headlessui/react` for overlays.
- [x] 7.2 `frontend/README.md`: prerequisites, `pnpm install` / `pnpm dev`
  (needs the Go backend on `:8080` for `/api`), `pnpm build` → `out/`,
  `pnpm lint`. `frontend/CLAUDE.md`: `@AGENTS.md`.
- [x] 7.3 `Dockerfile` frontend stage: confirm it still works — `pnpm install
  --frozen-lockfile`, `pnpm build`, `COPY --from=frontend /src/frontend/out/.
  static/out/`. Adjust only if paths moved.
- [x] 7.4 `.github/workflows/ci.yml` frontend job: confirm `pnpm lint` +
  `pnpm build` pass on the new stack; adjust node version / cache path only if
  needed.
- [x] 7.5 Root `AGENTS.md` / `README.md`: update the frontend stack line
  (Next.js → Vite + React + TanStack Router). `openspec/config.yaml` context:
  same.

## 8. Verify

- [x] 8.1 `frontend`: `pnpm lint` clean; `pnpm build` succeeds and writes
  `frontend/out/index.html` + `frontend/out/assets/`.
- [x] 8.2 `backend`: `gofmt -l .` clean, `go vet ./...`, `go test ./...` pass
  (with and without `DATABASE_URL`).
- [x] 8.3 `docker build -t family-finances .` from the repo root succeeds.
- [x] 8.4 Manual (backend + Mailpit up, `AUTH_BASE_URL=http://localhost:3000`,
  `pnpm dev`): home renders with sidebar; theme cycle works with no flash on
  reload; sidebar footer shows "Log in" → `/login` → submit → Mailpit link →
  back to `/` signed in → avatar + email → menu → "Log out" → "Log in" again
  without reload; `/login` while signed in redirects to `/`.
- [x] 8.5 Manual: hard-refresh on `/login` and on a made-up deep path
  (`/account/234/edit`) served by the built bundle behind the Go binary — both
  load the SPA shell, not a 404; a missing `*.js` still 404s.
- [x] 8.6 `openspec validate migrate-frontend-to-vite-spa` passes. Archive the
  completed `authentication`, `add-postgres-persistence`, and
  `go-serves-frontend-docker-ci` changes before archiving this one so the
  `backend-package-architecture` deltas compose cleanly.

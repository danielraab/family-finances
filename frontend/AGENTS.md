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
- **TypeScript**, strict — plus `noUncheckedIndexedAccess`,
  `exactOptionalPropertyTypes`, `noPropertyAccessFromIndexSignature`.
  `pnpm exec tsc` type-checks (no emit).
- **API types** — `pnpm generate:api` runs `scripts/generate-api.mjs`, which
  reads the repo-root `openapi/openapi.yaml` (the hand-written contract), drops
  `x-internal` browser-redirect operations, and writes
  `src/api/schema.d.ts` via `openapi-typescript`. That file is **generated and
  committed** — never edit it (excluded from Biome, like `routeTree.gen.ts`);
  CI regenerates it and fails on a diff. `src/api/client.ts` wraps it in an
  `openapi-fetch` client (`api.GET("/api/...")`, typed paths + responses).
  See root `AGENTS.md` and `openapi/README.md`.

## Routing & layout

- `src/routes/__root.tsx` is the app shell: `ThemeProvider` + `AuthProvider`
  wrap `<Sidebar />` beside a column of `<TopBar />` + `<main><Outlet /></main>`.
  `__root.tsx` owns both the sidebar's persistent `collapsed` state
  (`localStorage` under `ff:sidebar-collapsed`) and its mobile-only
  `mobileOpen` drawer state (not persisted); `TopBar`'s single button drives
  whichever applies for the current viewport (collapse toggle at `md` and up,
  open/close the off-canvas drawer below it). `Sidebar` is presentational.
  Router devtools render only in dev.
- `src/routes/index.tsx` → `/`. `src/routes/login.tsx` → `/login`.
  `src/routes/settings.tsx` (+ `settings.index.tsx`, `settings.users.tsx`) →
  `/settings` and `/settings/users` — see "Settings" below.
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
The control is `src/components/ThemeSwitch.tsx`, rendered in `Sidebar`'s
footer above `SidebarUser` — a device preference, not an account setting, so
it's visible to anonymous and authenticated visitors alike, unlike
`SidebarUser`'s own dropdown. Expanded it's a single-row segmented pill
(System/Light/Dark); collapsed to icon-only sidebar width it falls back to
one button that cycles the three. Fonts (Geist / Geist Mono) are self-hosted
via `@fontsource-variable/*`, imported in `src/main.tsx`.

## i18n

`i18next` + `react-i18next`, initialized in `src/i18n/index.ts` (imported
once, at the top of `src/main.tsx`). Two languages ship today: **English**
(default/fallback) and **German**; resources live in
`src/i18n/locales/{en,de}.json` as one flat `translation` namespace. The
active language is resolved from the browser
(`i18next-browser-languagedetector`, `order: ["navigator"]`, `caches: []`) —
every load re-detects fresh from `navigator.language`, no `localStorage`; a
`de*` browser language renders German, everything else falls back to
English. `src/routes/__root.tsx` keeps `<html lang>` (statically `"en"` in
`index.html`) in sync with the resolved language at runtime.

An authenticated visitor's account-level language preference — set on the
Common tab of `/settings`, per `user-settings` — takes priority over browser
detection: `AuthProvider` calls `i18n.changeLanguage(user.language)` once
`GET /api/auth/me` resolves with a non-null `language` (the *raw* preference,
distinct from `GET /api/settings`'s resolved value — see
`backend/internal/settings`'s design note on why). No preference set means
browser detection still applies, same as for an anonymous visitor. Because
this depends on an async request, a visitor whose browser and account
language disagree can see a brief flash of the browser-detected language
before the account preference applies — accepted, not solved the way the
colour theme's pre-paint script avoids its flash (that value lives in
`localStorage`, this one doesn't).

Every hardcoded user-facing string (labels, titles, `aria-label`/`title`
text, etc.) MUST go through `useTranslation()` / `t()` (or `<Trans>` for
strings with embedded markup) instead of a literal — the "Family Finances"
brand name is the one exception. `src/i18n/locales/en.json` is the source of
truth: a key MUST exist there before or at the same time it's added to any
other locale file. Other locale files are allowed to temporarily lag
`en.json` — CI's `i18n-coverage` job (`scripts/i18n-coverage.mjs`) reports
each locale's coverage against it as a job summary and a PR comment, but
that check is informational only and never blocks merging.

## Data & auth

- The app ships **no backend URL**. Call the Go backend through the typed
  client `api` from `src/api/client.ts` (`openapi-fetch`, `baseUrl: "/"`,
  `credentials: "same-origin"` so the `ff_session` cookie is sent). In dev the
  Vite proxy forwards `/api` to `:8080`; in production the Go binary serves both
  same-origin. Types come from the committed, generated `src/api/schema.d.ts`
  (see Tooling).
- `src/components/AuthProvider.tsx` resolves the session once on mount via
  `api.GET("/api/auth/me")` and exposes `useAuth() → { status, user, logout }`
  (`status ∈ loading | anonymous | authenticated`; `user` is the generated
  `components["schemas"]["User"]`). No polling, no focus-refetch. `logout()`
  calls `api.POST("/api/auth/logout")` and flips to `anonymous` in place.
- `src/components/SidebarUser.tsx` renders the footer user control (initials
  monogram + a `@headlessui/react` menu: "Settings" → `/settings`, then
  "Log out").
- `src/routes/login.tsx` is the magic-link form. On mount it calls
  `api.GET("/api/auth/config")`; when the response's `oidc` is non-null it
  renders a provider button (a plain `<a href={oidc.start_path}>` — a full-page
  navigation, not `fetch`) above the email field with an "or" divider. A null
  `oidc` or a failed request just shows the email form.
- Data fetching is plain `fetch`/`api` calls in effects — no query library yet.
- **Never** open a database connection from the frontend.

## Settings

`/settings` is the first route that requires authentication: `settings.tsx`
(the layout route) redirects an anonymous `useAuth` to `/login`, renders
nothing while `loading`, and otherwise renders the tab nav (Common always;
Users only when `user.is_admin`, per `AdminUser`) plus `<Outlet/>`.

- `settings.index.tsx` (`/settings`, Common tab) — language/timezone/default
  currency. Each field calls `PUT /api/settings` with only itself on change
  (no separate Save button, mirroring `ThemeSwitch`'s immediate-apply
  interaction); a successful language change also calls
  `i18n.changeLanguage()` so the running app switches immediately. A failed
  update reverts the field and shows an inline error.
- `settings.users.tsx` (`/settings/users`, admin-only) — redirects to
  `/settings` if `user.is_admin` is false (defense in depth beyond the tab
  simply not being listed). Lists users (`GET /api/auth/users`) and
  invitations (`GET /api/auth/invites`), can invite
  (`POST /api/auth/invites` — open to any authenticated user per the
  backend, this tab is just its current UI surface), and can
  disable/enable/(soft) delete a user, each behind a `@headlessui/react`
  `Dialog` confirmation whose copy calls out self-targeting. A successful
  self-disable/self-delete flips `useAuth` to `anonymous` locally (the same
  transition `logout()` performs) and navigates away, rather than waiting for
  a subsequent request to discover the `401`.

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

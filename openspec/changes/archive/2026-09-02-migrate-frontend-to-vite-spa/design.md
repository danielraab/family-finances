## Context

The frontend is Next.js 16 (App Router) built with `output: "export"`. Production
runs no Node server: the Go binary `//go:embed`s `frontend/out/` and serves it,
with `internal/httpapi/static.go` returning the bundle's `404.html` for any
unmatched path.

The app is about to grow real routes, including runtime-parameterised ones
(`/account/234/edit`). `output: "export"` prerenders a fixed, build-time set of
routes; an un-enumerated dynamic instance has no HTML, so a direct load /
refresh / bookmark of it hits `404.html`. Only `/` works today because
`http.FileServerFS` special-cases a directory to its `index.html`. Client-side
`<Link>` navigation always works; the gap is on hard navigations.

The paused `web-client-auth` change (sidebar user widget + `/login`) is absorbed
here: its UI ports onto the new stack, and a second un-archived delta against
the same `web-client-shell` requirement would be a spec-merge hazard.

Constraints carried forward: static bundle, Go serves it, no Node server, no
`BACKEND_URL`, pnpm only, Biome, Tailwind 4, class-based dark mode, deliberate
minimal dependencies, all data via the Go `/api/*`.

## Goals / Non-Goals

**Goals:**

- Replace Next.js with a Vite + React + TanStack Router SPA that renders any URL
  (dynamic params included) client-side.
- Preserve the build-output contract: `pnpm build` → `frontend/out/`, a static
  bundle the Go binary embeds and serves, no Node server.
- Add an SPA fallback to the Go static handler so hard navigations to app routes
  work.
- Port the existing shell (Sidebar, ThemeToggle, Icon, Placeholder, HealthCheck)
  and land the `web-client-auth` widget + `/login` on the new stack.
- Keep Docker/CI working with minimal edits.

**Non-Goals:**

- Route guarding / redirect-if-unauthenticated (still a later change; the
  `beforeLoad` + router-context seam is left open).
- TanStack **Start** or any server runtime; TanStack Query or another
  data-fetching library (hand-rolled `fetch` stays).
- An OIDC "sign in with a provider" button; display-name editing.
- Reworking `frontend-old/` — it stays as read-only porting reference until a
  later cleanup deletes it.

## Decisions

### D1: Vite + React SPA, not Next.js static export

The project already forbids a Node server, so every Next server feature was
already unused. Using Next purely as a static-SPA builder means fighting it
(catch-all routes, `generateStaticParams`, RSC-payload 404s on deep links). A
plain Vite React SPA is the honest expression: one `index.html`, the client
router owns every path, the Go server needs one rule (`index.html` for unmatched
app paths). Build is faster and the mental model is smaller.

*Alternatives considered:* keep Next + `trailingSlash: true` — fixes only
*static* routes, not `/account/:id/edit`; keep Next + a `[[...slug]]` catch-all
client shell — works but discards file-based routing for the app's real pages
and is fiddly; run Next with a server — breaks the single-binary embed model.

### D2: TanStack Router (chosen by the scaffold)

`create-tsrouter-app` (routerOnly — no Start) with file-based routing and the
Vite plugin. Rationale for this project: the codebase is TS-strict end to end,
so fully-typed params and links fit the house style; first-class typed **search
params** are a real future win for finance filter/range views and are painful to
retrofit onto React Router; `beforeLoad` + router `context` is the natural home
for the auth guard that is a known upcoming change and plugs straight into
`useAuth`; and the app is tiny now — the cheapest moment to adopt it.

*Alternative considered:* React Router v7 in data mode — more mature, larger
ecosystem, lower migration friction, but params/search-params are loosely typed
and guards/loaders are less ergonomic. Defensible; not chosen.

### D3: Keep `frontend/out/` as the build output

Set Vite `build.outDir: "out"`. This keeps `backend/embed.go`
(`//go:embed static/out`), `main.go`'s `fs.Sub(…, "static/out")`, the Dockerfile
`COPY … /frontend/out/. static/out/`, and CI unchanged. Add `/out` to
`frontend/.gitignore`. Cost: a non-default Vite `outDir` — one config line,
documented.

*Alternative:* adopt Vite's default `dist/` and change the embed path,
`fs.Sub`, Dockerfile, `.gitkeep` location, and docs — more surface, more risk,
no benefit.

### D4: SPA fallback in `internal/httpapi/static.go`

`staticHandler` currently swaps `http.FileServerFS`'s plain-text 404 for the
bundle's `404.html`. New behaviour: on a would-be 404 for a `GET`/`HEAD`,
non-`/api/` request, if the path has **no file extension**, serve `index.html`
with `200`; if it has an extension (a real asset request), keep the `404`. This
is the standard static-host SPA rule (`try_files $uri /index.html`). `/api/`
routing is untouched — unmatched `/api/` paths still return JSON `404` from the
mux, never the shell. Covered by handler tests: known file → 200, extensionless
unknown → `index.html` 200, `*.js` unknown → 404, `/api/nope` → JSON 404.

The `all:` prefix on `//go:embed` existed because Next wrote assets under
`_next/` (embed skips `_`/`.` names without it). Vite writes to `assets/`; the
prefix is no longer required but is harmless and left in place.

### D5: Theme without `next-themes`

A ~40-line in-app provider: read `localStorage["ff:theme"]` (`system` default),
resolve against `matchMedia("(prefers-color-scheme: dark)")`, toggle `.dark` on
`<html>`, listen for OS changes while in `system`. A small inline script in
`index.html` applies the resolved class **before** the app mounts (no flash).
`app/globals.css` tokens (`--background`/`--foreground`, `@custom-variant dark`,
`@theme inline` font vars) move into `src/styles.css` unchanged. `ThemeToggle`'s
cycle/glyph logic ports verbatim.

### D6: Fonts self-hosted

`next/font/google` (Geist, Geist Mono) → `@fontsource-variable/geist` +
`@fontsource-variable/geist-mono`, imported once in `main.tsx`; set
`--font-geist-sans` / `--font-geist-mono` in `styles.css`. No Google runtime
request — matches the "no external coupling" stance.

### D7: Route tree

```
src/routes/
  __root.tsx   ThemeProvider + AuthProvider + <Sidebar/> + <main><Outlet/></main>
  index.tsx    home  (Placeholder + HealthCheck)
  login.tsx    the /login view
```

`__root.tsx` owns the providers so `useAuth` / theme are available everywhere,
including `beforeLoad` context later. `routeTree.gen.ts` is generated by the
Vite plugin and excluded from Biome and VCS-noise (already in `.vscode`
settings).

### D8: Component port mechanics

- `"use client"` — deleted everywhere (no RSC).
- `next/link` `<Link href>` → `@tanstack/react-router` `<Link to>`.
- `usePathname()` → `useLocation({ select: l => l.pathname })` (or `useMatchRoute`
  for active-nav).
- `next/navigation` `useRouter().replace(p)` → `useNavigate()({ to: p, replace: true })`.
- `AuthProvider`, `useAuth`, `Avatar`, `monogram`, `SidebarUser`, the login form
  body — logic unchanged; only the three router touch-points above and import
  paths change. `@headlessui/react` re-added via pnpm.

### D9: pnpm, and Biome realignment

Delete `package-lock.json`; set `.cta.json` `packageManager` to `pnpm`;
`pnpm install` to write `pnpm-lock.yaml`. Biome: keep the scaffold's file but
match the repo norm — 2-space indent, `"lint": "biome check"` (so CI's
`pnpm lint` runs format + lint + organize-imports as before), `routeTree.gen.ts`
and `styles.css` excluded.

## Risks / Trade-offs

- **Overlapping `backend-package-architecture` deltas** → `authentication`
  (complete, not archived) also modifies the "Routing …" requirement. Archive
  the three completed changes (`authentication`, `add-postgres-persistence`,
  `go-serves-frontend-docker-ci`) before archiving this one; reconcile the
  merged requirement text at archive time (both the auth-middleware sentence and
  the SPA-fallback paragraph belong in the final version).
- **SPA fallback masks genuine 404s for extensionless paths** → `/typo` now
  serves the shell (`200`) and the client renders its not-found view. Standard
  SPA behaviour; the extension heuristic keeps asset 404s honest.
- **TanStack Router is younger than React Router** → v1 since Dec 2024, stable;
  smaller SO/AI corpus. Mitigation: the app is small, the docs are strong, and
  the typed-route payoff is real for this codebase.
- **Bleeding-edge scaffold pins** (Vite 8, TS 6, several `latest`) → pin exact
  versions after the first green `pnpm install` + `pnpm build`; treat a broken
  transitive bump as a lockfile fix.
- **`frontend-old/` lingers** → it is git-ignored for tooling (its own
  `biome.json`, `node_modules`) and not built by CI or Docker; a follow-up
  deletes it once the port is confirmed in the running app.
- **Two toolchains briefly disagree on Biome version** (`frontend` 2.4.5 vs
  `frontend-old` 2.4.2) → irrelevant; `frontend-old` is not linted by CI.

## Migration Plan

1. `frontend/`: pnpm + Biome realign; `vite.config.ts` (`outDir: out`,
   `/api` proxy); `.gitignore` `/out`.
2. `styles.css` (merge tokens); theme provider + pre-paint script; fonts;
   `public/` icon + favicon; `index.html` head.
3. Port `Icon`, `Sidebar`, `ThemeToggle`, `Placeholder`, `HealthCheck`.
4. Route tree: `__root.tsx`, `routes/index.tsx`.
5. Port `web-client-auth`: `AuthProvider`/`useAuth`, `monogram`, `Avatar`,
   `SidebarUser` (Headless UI menu), `routes/login.tsx`; mount widget in the
   Sidebar footer.
6. Backend: SPA fallback in `static.go` + tests; `embed.go` comment;
   `backend/AGENTS.md`.
7. Docs: `frontend/AGENTS.md` / `README.md` / `CLAUDE.md`; root docs frontend
   line; `openspec/config.yaml`.
8. Verify: `pnpm lint` + `pnpm build` (→ `frontend/out/index.html`);
   `go test ./...`; `docker build`; manual run of the auth round trip and a
   deep-link refresh; `openspec validate`.

**Rollback:** `git mv frontend frontend-broken && git mv frontend-old frontend`
and revert the backend `static.go` + doc commits. No data or API surface
changed.

## Open Questions

_None blocking._ Deferred by scope: route guarding (its own change), deleting
`frontend-old/` (follow-up), pinning exact dependency versions (done during
apply, not a design decision).

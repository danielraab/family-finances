## 1. Static export

- [x] 1.1 Set `output: "export"` in `frontend/next.config.ts` (no `next/image` remains, so `images.unoptimized` not needed).
- [x] 1.2 Remove the `start` script from `frontend/package.json`.
- [x] 1.3 Run `pnpm build` in `frontend/` and confirm it writes `frontend/out/` with `index.html` and hashed assets, no server needed.
- [x] 1.4 Confirm `frontend/.gitignore` already ignores `/out/` — it does (line 18).

## 2. Icon and favicon

- [x] 2.1 Author `frontend/app/icon.svg`: `#15803d` rounded tile, white house glyph, `€` subtracted via `<mask>`. Verified legible at 48px (rendered PNG frame).
- [x] 2.2 Generate `frontend/app/favicon.ico` (16/32/48 px) via `frontend/scripts/generate-favicon.mjs` — pure Node (zlib only), no build dependency, no route.
- [x] 2.3 Default `frontend/app/favicon.ico` deleted; the generated multi-res file lands at the same path.
- [x] 2.4 `favicon.ico` regeneration command documented in `frontend/AGENTS.md` ("Build output" section).
- [x] 2.5 Shared `Icon` component at `app/components/Icon.tsx` (inlined markup, kept in sync with `icon.svg`), used by the sidebar.

## 3. App shell and layout

- [x] 3.1 `frontend/app/layout.tsx` metadata: `title: "Family Finances"`, real `description`. Geist fonts kept.
- [x] 3.2 `layout.tsx` renders `<Sidebar />` + `<main>{children}</main>` in a flex row; stays a Server Component.
- [x] 3.3 `Sidebar` (`app/components/Sidebar.tsx`) is `"use client"`: `Icon` + "Family Finances" wordmark, nav list, collapse/expand toggle.
- [x] 3.4 Nav has a single "Home" item via `next/link`; active state from `usePathname()` (`aria-current="page"` + styling).
- [x] 3.5 Collapsed sidebar narrows to `w-16` (icon + glyphs only), main widens; width animates via Tailwind `transition-[width]`.
- [x] 3.6 Collapsed state persisted in `localStorage` (`ff:sidebar-collapsed`): read in `useEffect` on mount, written on toggle; first render uses the expanded default; `mounted` flag gates the transition. `localStorage` access wrapped in try/catch (static-export browser-API guidance).
- [x] 3.7 Removed the `body { font-family: Arial… }` override in `app/globals.css`; Tailwind import and `--background`/`--foreground` tokens kept.

## 4. Home page

- [x] 4.1 `Placeholder` Server Component at `app/components/Placeholder.tsx`: heading + dashed empty-state card ("Nothing here yet").
- [x] 4.2 `frontend/app/page.tsx` now just renders `<Placeholder />`. No `next/image`, no Next.js/Vercel branding, no template links (grep-verified in `out/index.html`).

## 5. Delete scaffold assets

- [x] 5.1 Deleted `frontend/public/{file,globe,next,vercel,window}.svg` (`public/` now empty).
- [x] 5.2 Deleted `frontend/.env.example`.
- [x] 5.3 Repo grep: no `BACKEND_URL` in code, config, or docs (only in this change's own artifacts, which document its removal).

## 6. Documentation

- [x] 6.1 `frontend/AGENTS.md`: replaced "Data" with a "Build output" section + mechanism-agnostic "Data" section; no `BACKEND_URL`; kept the no-database rule. `next dev` agent-rules block left intact.
- [x] 6.2 `frontend/README.md`: rewritten — no `.env.example` step, no "backend running" prerequisite, no `BACKEND_URL` table; adds a "Building" section (`out/` + `npx serve out`).
- [x] 6.3 Root `AGENTS.md`: Architecture bullet updated — static bundle, no backend URL, runtime access is open (points at `design.md` O1).
- [x] 6.4 Root `README.md`: Architecture + Getting started updated to match 6.3.
- [x] 6.5 `openspec/config.yaml`: `context` block updated — no `BACKEND_URL`, static bundle served by a static host.

## 7. Verification

- [x] 7.1 `pnpm lint` (`biome check`) and `pnpm format` clean.
- [x] 7.2 `pnpm build` succeeds; `out/` served by `python3 -m http.server` (no Node) returns 200 for `/`, `/favicon.ico` (`image/vnd.microsoft.icon`), `/icon.svg`, and JS chunks.
- [x] 7.3 Prerendered `out/index.html` contains the expanded sidebar (icon, wordmark, "Home", "Collapse sidebar" toggle) — expanded is the server default, so no hydration mismatch. Live click-through (collapse → reload persistence) not run: no GUI browser in this environment; logic reviewed and compiles.
- [x] 7.4 `out/index.html` links both `/favicon.ico` (48x48, `image/x-icon`) and `/icon.svg` (`image/svg+xml`); the sidebar renders the same `<Icon>` mark (mask `ff-euro-cut` present in HTML). Icon rendering confirmed via an extracted PNG frame.
- [x] 7.5 `out/index.html` grep for `next.svg|vercel|create next app|Deploy Now|nextjs.org/learn` → no matches.
- [x] 7.6 Repo grep: no `BACKEND_URL` (outside this change's artifacts); `frontend/.env.example` and the five scaffold SVGs are gone.

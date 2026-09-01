## Context

`frontend/` is an unmodified `create-next-app` scaffold (Next.js 16, React 19,
Tailwind 4, Biome, pnpm). Nothing in it has been customised: the home page is the
template, `public/` holds only Next.js sample SVGs, and `app/favicon.ico` is the
default. `BACKEND_URL` is documented in five files but read by no code — the
frontend currently fetches nothing, and the backend is a stub returning `"ok"`.

We want production to serve plain static files from nginx in a Docker container —
no Node runtime, no `next start`. The two packages stay independent deployables
(root `AGENTS.md`: "no shared build, no shared code").

Constraints:

- pnpm only; Biome for lint/format (2-space); Tailwind 4 CSS-first (no config
  file); App Router; Server Components by default.
- `frontend/AGENTS.md` carries a `next dev`-managed rules block warning that this
  Next.js differs from training data — consult `node_modules/next/dist/docs/`
  for the static-export specifics during implementation.

## Goals / Non-Goals

**Goals:**

- `pnpm build` emits a static `out/` that any static host can serve.
- A shared app layout: collapsible left sidebar (icon + wordmark, one "Home"
  nav item, collapse toggle with persisted state) plus a main content region.
- A home page showing only a small placeholder component.
- An owned icon — house with a Euro cut-out on a deep-green tile — as favicon
  and sidebar logo.
- Remove `BACKEND_URL` entirely and correct every doc that mentions it.

**Non-Goals:**

- Any real data fetching, API integration, or auth.
- Choosing the production runtime path from browser to backend (see Open
  Questions) — this change only removes the stale `BACKEND_URL` model.
- Multi-route navigation, theming/dark-mode toggle, i18n.
- Writing the production Dockerfile / nginx image (documented here, built with
  the deployment change).
- Backend changes of any kind.

## Decisions

### D1: `output: "export"` in `next.config.ts`

Set `output: "export"`; `pnpm build` then writes `out/`. Remove the `start`
script from `package.json`. Keep `pnpm dev` for local iteration.

- **Why:** it is the built-in, supported way to get a serverless static bundle;
  no extra dependency.
- **Implications:** no route handlers, middleware, `revalidate`, or request-time
  RSC. Server Components still render, but at build time only. `next/image`
  needs `images: { unoptimized: true }` — simplest is to not use `next/image`
  at all (the template's only use is deleted).
- **Alternative — keep the Next server, deploy Node:** rejected; the explicit
  goal is no frontend server.
- **Alternative — eject to a plain Vite SPA:** rejected as a larger, unrelated
  migration; the team knows Next and the App Router shell is fine at build time.

### D2: Sidebar is the sole Client Component; state in `localStorage`

`app/layout.tsx` stays a Server Component and renders `<Sidebar>` +
`<main>{children}</main>`. `<Sidebar>` is `"use client"` and owns the
collapsed boolean. On mount it reads `localStorage` in a `useEffect` and writes
back on every toggle; the first server-rendered markup uses the expanded default
so build-time HTML is deterministic and there is no hydration mismatch. A brief
expanded→collapsed flash on load for returning users is acceptable; if it
grates, gate the transition with a `mounted` flag.

- **Why:** keeps almost the whole tree as build-time Server Components (per
  `frontend/AGENTS.md`), confines interactivity to one file.
- **Alternative — cookie + Server Component read:** needs a request-time server;
  impossible under static export.
- **Alternative — CSS `:has()` / checkbox hack, no JS:** no persistence, and
  awkward for the toggle affordance.
- **Nav state:** `next/link` + `usePathname()` for the active class. Works
  under static export.

### D3: Icon shipped as static files, referenced by App Router conventions

Author `app/icon.svg` by hand: a rounded-square tile filled `#15803d`, a white
house glyph (square body + triangular roof) centred, and a `€` subtracted from
the house body via `fill-rule="evenodd"` (single path) or a `<mask>`, so the
tile green shows through the glyph. Generate `app/favicon.ico` (16/32/48 px)
from the SVG with a one-off tool (e.g. `sharp`/ImageMagick/an online converter);
commit the `.ico` as a static asset — do **not** add a build dependency or an
`icon.tsx` route (route handlers do not run under `output: "export"`). Delete
the existing `app/favicon.ico`. Reuse the SVG as the sidebar logo by importing
it as a React component or inlining the markup in a shared `Icon` component.

- **Why static, not `app/icon.tsx`:** `ImageResponse` / route handlers are
  server features, incompatible with static export.
- **Alternative — keep `.ico` only:** SVG favicons are crisp at all sizes and
  let us reuse one source; `.ico` retained purely for legacy tab support.

### D4: Delete `BACKEND_URL`, keep docs mechanism-agnostic

Remove `frontend/.env.example` (its only content). Edit `frontend/AGENTS.md`,
`frontend/README.md`, root `AGENTS.md`, root `README.md`, and
`openspec/config.yaml` so the frontend is described as "builds to a static
bundle served by a static host; no Node server; no `BACKEND_URL`". Keep the
"frontend never opens a database connection; backend owns persistence" rule.
Do **not** assert any particular browser→backend path — that is D-future.

- **Why:** the current wording ("`BACKEND_URL`, server-side only") describes a
  server that this change removes, and the var is unused. Leaving it would be
  actively misleading.

### D-future (Open Question O1): runtime path from browser to backend

Not decided in this change; nothing fetches yet. Options for the first data
feature:

| Option | CORS | Backend public | Env var | Keeps pkgs separate |
| --- | --- | --- | --- | --- |
| ① nginx `location /api` → `proxy_pass` backend; client fetches relative `/api/...` | no | no | no | yes |
| ② Go binary serves `out/` (embed) and `/api` handlers | no | no | no | no (fused build/deploy) |
| ③ Browser calls the backend directly | yes | yes | `NEXT_PUBLIC_API_URL` baked at build | yes |

Leaning ① — it matches the "nginx serves static files in a container" plan
(just add an `/api` location), needs no CORS, keeps the backend private, adds no
env var, and preserves the "browser never holds a backend URL" property. To be
confirmed when the first endpoint is consumed.

## Risks / Trade-offs

- **Hydration mismatch from `localStorage` sidebar state** → server markup
  always renders the expanded default; client reconciles in `useEffect`. Accept
  a one-frame flash for returning collapsed users, or gate with a `mounted`
  flag.
- **A future feature needs a server-only Next capability** (route handler, ISR,
  middleware) → revisit D1; `output: "export"` is one line to remove, but any
  code written against static assumptions would need rework. Low likelihood
  given the nginx-proxy direction.
- **`favicon.ico` drifts from `icon.svg`** (regenerated by hand) → document the
  regeneration command in `frontend/AGENTS.md` or a `scripts/` note; the pair
  changes rarely.
- **`next dev` rewrites the agent-rules block in `frontend/AGENTS.md`** while we
  also edit that file → commit the block together with our edits so the tree
  stays clean (its own note says so).
- **Docs go stale again once O1 is decided** → the follow-up deployment change
  updates the same five files; keep the "no BACKEND_URL" wording, add the chosen
  mechanism then.

## Migration Plan

1. Land this change: static export config, shell, icon, cleanup, doc edits.
2. Verify locally: `pnpm build` → `npx serve out` → sidebar collapse/persist,
   Home nav active state, favicon in tab, no template content, `pnpm lint`.
3. Deployment change (separate): Dockerfile building `out/` into an nginx image,
   compose wiring, and — resolving O1 — the `/api` proxy (option ①) or the
   chosen alternative. Update the five docs with the concrete mechanism.

Rollback: revert the change commit; `git restore` the deleted assets. No data,
no infra, no backend touched.

## Open Questions

- **O1:** Runtime path from the deployed browser to the backend — option ①/②/③
  above. Deferred to the first data feature; current lean is ①.
- **O2:** Keep the Geist / Geist_Mono fonts from the scaffold, or pick a
  typeface? Default: keep them (valid build-time `next/font/google` use).
- **O3:** Regeneration tooling for `favicon.ico` — commit a `scripts/` helper,
  or just document the one-liner? Default: document it.

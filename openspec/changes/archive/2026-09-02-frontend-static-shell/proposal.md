## Why

The frontend is still an unmodified `create-next-app` scaffold: the landing page
is the Next.js template, the favicon is the default, and `public/` holds only
Next.js sample art. It also assumes a running Next.js server (`next start`,
`BACKEND_URL` read server-side) that we do not want — production should serve
plain static files from nginx in a container, with no Node runtime. And
`BACKEND_URL` is documented in five places but referenced by zero lines of code.

## What Changes

- **Static output.** `next.config.ts` gets `output: "export"`; `pnpm build`
  produces `out/` with no server. `next start` is removed.
  **BREAKING** for anyone relying on SSR / route handlers / middleware — none
  exist yet, so no runtime impact.
- **App shell.** New layout with a collapsible left sidebar: the app icon and a
  wordmark at the top, a single "Home" navigation item, and a collapse toggle
  whose state persists in `localStorage`. The sidebar is the only Client
  Component; everything else stays a Server Component rendered at build time.
- **Home page.** `app/page.tsx` template content is replaced with a small
  `<Placeholder>` component (heading + empty-state card).
- **Icon / favicon.** New `app/icon.svg` — a house silhouette with a Euro sign
  cut out, on a deep-green (`#15803d`) rounded tile — plus a regenerated
  multi-resolution `app/favicon.ico`. The default `app/favicon.ico` is deleted.
  The same mark is reused as the sidebar logo.
- **Cleanup.** Delete `public/{file,globe,next,vercel,window}.svg`. Remove the
  `body { font-family: Arial… }` override in `app/globals.css` that fights the
  Geist font wired up in `layout.tsx`. Set real `metadata` (`title`,
  `description`).
- **Remove `BACKEND_URL`.** Delete `frontend/.env.example` (its only content).
  Correct every doc that mentions it — `frontend/AGENTS.md`,
  `frontend/README.md`, root `AGENTS.md`, root `README.md`,
  `openspec/config.yaml` — to describe the frontend as a static bundle with no
  backend coupling yet. The mechanism for reaching the backend at runtime is
  left as an explicit open decision in `design.md`, to be settled with the first
  real data feature.

## Capabilities

### New Capabilities

- `web-client-shell`: The static Next.js web client — its build output contract
  (static export, no server), the application layout (collapsible sidebar with
  Home navigation), the home placeholder, and the app icon / favicon.

### Modified Capabilities

<!-- None. openspec/specs/ is empty; no existing requirements change. -->

## Impact

- **`frontend/` code**: `next.config.ts`, `app/layout.tsx`, `app/page.tsx`,
  `app/globals.css`, `package.json` (scripts); new `app/icon.svg`,
  `app/favicon.ico`, sidebar + placeholder components.
- **`frontend/` assets**: `public/*.svg` and `app/favicon.ico` deleted;
  `frontend/.env.example` deleted.
- **Docs**: `frontend/AGENTS.md`, `frontend/README.md`, root `AGENTS.md`, root
  `README.md`, `openspec/config.yaml`.
- **Deployment**: production model becomes "nginx serves `out/` from a Docker
  container." nginx config / compose wiring and the runtime backend-access model
  are described in `design.md`; whether to implement the `/api` proxy now or
  defer it is an open decision.
- **No backend changes.** No dependency changes (`output: "export"` is built in).

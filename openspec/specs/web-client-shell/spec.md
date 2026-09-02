# web-client-shell Specification

## Purpose

The client-only web app (Vite + React + TanStack Router) — its build-output
contract (static bundle, no server, embedded and served by the Go backend),
the application layout (collapsible sidebar with Home navigation and a
user control in its footer, plus a top bar holding the sidebar-collapse and
colour-theme controls), the home placeholder, the colour theme, the
app icon / favicon, and the no-`BACKEND_URL` rule.

## Requirements
### Requirement: Static build output

The web client SHALL build to a fully static bundle that runs without any
server runtime. `pnpm build` (Vite) SHALL emit the site to `frontend/out/`
(`build.outDir` set to `out`), and that directory SHALL be serveable by any
static file host with no additional runtime. The bundle SHALL be a single
`index.html` entry plus hashed assets; the application SHALL render entirely in
the browser. In production this output is embedded into and served by the Go
backend (see the `backend-static-serving` capability) — no separate static
host process is deployed.

The project MUST NOT depend on Next.js or any framework server: no route
handlers, no middleware, no server-side or build-time component rendering, no
image-optimization server. `frontend/` MUST NOT contain `next`, `next.config.*`,
`next-env.d.ts`, or an `app/` router directory.

#### Scenario: Build produces a static directory

- **WHEN** a developer runs `pnpm build` in `frontend/`
- **THEN** the build succeeds and writes `frontend/out/` containing
  `index.html` and hashed assets (e.g. under `assets/`)
- **AND** no Node server is required to view the site

#### Scenario: Static host serves the bundle

- **WHEN** the contents of `frontend/out/` are served by a plain static file
  server with a single-page-app fallback to `index.html`
- **THEN** the home page loads and the sidebar renders and functions

#### Scenario: No server-only start script

- **WHEN** a developer inspects `frontend/package.json`
- **THEN** it has no `next` dependency and no `start` script that runs a server
- **AND** `frontend/` contains a `vite.config.ts` and an `index.html` entry

### Requirement: Application layout with collapsible sidebar

The web client SHALL render every route inside a shared layout consisting of a
left sidebar, a top bar spanning the main content region, and a main content
region below the top bar.

The sidebar SHALL contain, from top to bottom: the application icon with a
"Family Finances" wordmark, a navigation list, and a footer holding the user /
sign-in control alone.

The top bar SHALL span the full width of the main content region, above the
active route's content. It SHALL hold the control to collapse or expand the
sidebar pinned to its left edge (visually adjacent to the sidebar), and the
colour-theme control pinned to its right edge. The top bar SHALL be present on
every route.

When collapsed, the sidebar SHALL show only the icon and the navigation item
glyphs (the footer user control rendering as its glyph or avatar alone); the
main content region SHALL expand to use the reclaimed width. The top bar and its
controls SHALL remain visible and functional regardless of the collapsed state.
The collapsed/expanded state SHALL persist across page loads in the browser via
`localStorage`, and the layout MUST render correctly on first load when no
stored value exists.

The layout SHALL be composed with the router's root route and an `<Outlet>` for
the active route; all components render in the browser (there are no
server-rendered components).

#### Scenario: Default expanded layout

- **WHEN** a visitor opens the site in a browser with no stored sidebar state
- **THEN** the sidebar renders expanded, showing the icon, the "Family Finances"
  wordmark, the navigation list with labels, and the user / sign-in control in
  the footer
- **AND** the top bar renders with the collapse control at its left edge and the
  theme control at its right edge

#### Scenario: Collapsing the sidebar

- **WHEN** the visitor activates the collapse control in the top bar
- **THEN** the sidebar narrows to show only the icon and navigation glyphs, with
  the footer user control shown as its glyph or avatar alone
- **AND** the main content region widens to fill the reclaimed space
- **AND** the top bar and its controls remain visible

#### Scenario: State persists across reloads

- **WHEN** the visitor collapses the sidebar and then reloads the page
- **THEN** the sidebar is still collapsed after the reload

#### Scenario: Controls are not in the sidebar footer

- **WHEN** the sidebar footer renders in either the expanded or collapsed state
- **THEN** it contains only the user / sign-in control
- **AND** neither the collapse control nor the theme control appears inside the
  sidebar

### Requirement: Selectable colour theme

The web client SHALL let the visitor choose one of three colour-theme states —
**System**, **Light**, or **Dark** — via a control at the right edge of the top
bar. The control SHALL be a single button that cycles through the three states
in a fixed order on activation, and its glyph SHALL indicate the currently
selected state (including distinguishing **System** from the resolved light or
dark appearance).

The selected state SHALL persist in the browser via `localStorage` under the
key `ff:theme`. When no stored value exists, the state SHALL be **System**.

When the state is **Light** or **Dark**, the page SHALL render in that scheme
regardless of the operating system's `prefers-color-scheme`. When the state is
**System**, the page SHALL follow the operating system's `prefers-color-scheme`,
and SHALL update immediately if that preference changes while the page is open.

The page MUST NOT show a flash of the wrong theme on load: the resolved theme
SHALL be applied before first paint by an inline script in `index.html` that
runs before the app mounts. The theme mechanism SHALL be entirely client-side,
built in-app (no `next-themes` or other framework theme runtime), and MUST NOT
require changes to existing `dark:` styling in individual components (the `dark`
variant keys off a `.dark` class on `<html>`).

#### Scenario: Default follows the OS

- **WHEN** a visitor opens the site with no stored theme value
- **THEN** the page renders in the scheme matching the OS `prefers-color-scheme`
- **AND** the theme control in the top bar indicates the **System** state

#### Scenario: Explicit choice overrides the OS

- **WHEN** the visitor cycles the theme control to **Dark** on a system whose
  `prefers-color-scheme` is light
- **THEN** the page renders in dark immediately
- **AND** after reloading the page it still renders in dark

#### Scenario: System state tracks the OS live

- **WHEN** the theme control is set to **System** and the visitor changes the OS
  colour scheme while the tab is open
- **THEN** the page re-renders in the new scheme without a reload

#### Scenario: Returning to System

- **WHEN** the visitor has an explicit **Light** or **Dark** choice stored and
  cycles the control back to **System**
- **THEN** the page resumes following the OS `prefers-color-scheme`
- **AND** the stored value reflects **System**

#### Scenario: No flash of the wrong theme

- **WHEN** the visitor has **Dark** stored and loads the site
- **THEN** the first painted frame is already dark, with no visible light flash

#### Scenario: Theme control available when the sidebar is collapsed

- **WHEN** the visitor collapses the sidebar
- **THEN** the theme control remains visible and operable at the right edge of
  the top bar

#### Scenario: Static build is preserved

- **WHEN** a developer runs `pnpm build` in `frontend/`
- **THEN** the build succeeds and emits the static site to `frontend/out/` with
  no Node server required to serve it

### Requirement: Home navigation

The sidebar navigation SHALL contain a single "Home" item linking to the site
root (`/`). The active navigation item SHALL be visually distinguished when its
route is the current route.

#### Scenario: Navigating home

- **WHEN** the visitor activates the "Home" navigation item
- **THEN** the browser navigates to `/` and the "Home" item is shown as active

### Requirement: Home placeholder content

The home page (`/`) SHALL render a self-contained placeholder component in the
main content region: a heading and an empty-state card indicating there is no
content yet. It MUST NOT contain any `create-next-app` template content, Next.js
or Vercel branding, or links to Next.js documentation or templates.

#### Scenario: Home shows the placeholder

- **WHEN** the visitor opens `/`
- **THEN** the main content region shows the placeholder heading and empty-state
  card
- **AND** no Next.js logo, Vercel logo, "Deploy Now" button, or template links
  are present

### Requirement: Application icon and favicon

The web client SHALL define its own icon: a house silhouette with a Euro sign
(€) cut out of it, on a deep-green (`#15803d`) rounded tile, provided as an SVG
at `frontend/public/icon.svg` and as a multi-resolution
`frontend/public/favicon.ico`. Browsers SHALL load this icon as the site
favicon via `<link>` elements in `index.html`, and the same mark SHALL be used
as the sidebar logo (an inline React component kept in sync with the SVG).
Because the build is a static bundle, the icon MUST be a static asset, not
generated at request time.

#### Scenario: Favicon is the custom icon

- **WHEN** the site is loaded in a browser
- **THEN** the browser tab shows the house-with-Euro icon, not a framework
  default

#### Scenario: Sidebar uses the same mark

- **WHEN** the sidebar renders
- **THEN** the icon shown at the top of the sidebar is the same house-with-Euro
  mark used as the favicon

### Requirement: No BACKEND_URL configuration

The web client SHALL NOT define, document, or read a `BACKEND_URL` environment
variable. `frontend/` SHALL contain no `.env.example`. Project documentation
(`frontend/AGENTS.md`, `frontend/README.md`, root `AGENTS.md`, root `README.md`,
`openspec/config.yaml`) SHALL describe the frontend as a static bundle with no
backend coupling, and MUST NOT reference `BACKEND_URL` or a server-side backend
call.

The frontend still MUST NOT open a database connection; the Go backend remains
the sole owner of persistence. At runtime the deployed client reaches the
backend same-origin (the Go backend serves the bundle — see the
`backend-static-serving` capability), so the client SHALL NOT need any base
URL, deployed proxy, or CORS handling; client code calls relative `/api/...`
paths directly. In local development only, the Vite dev server MAY proxy
`/api/*` to the backend (`server.proxy` in `vite.config.ts`); this is a
dev-only convenience and introduces no runtime configuration or `BACKEND_URL`.

#### Scenario: No BACKEND_URL in the repo

- **WHEN** the repository is searched for `BACKEND_URL`
- **THEN** there are no matches in code, configuration, or documentation
- **AND** `frontend/.env.example` does not exist

#### Scenario: Docs describe a static frontend

- **WHEN** a reader consults `frontend/AGENTS.md` or `frontend/README.md`
- **THEN** the frontend is described as a static bundle with no Node server and
  no `BACKEND_URL`, embedded in and served by the Go backend at runtime

#### Scenario: Client calls the backend with relative paths

- **WHEN** frontend code calls the backend
- **THEN** it uses a relative `/api/...` path with credentials included, and no
  base URL or CORS configuration is required


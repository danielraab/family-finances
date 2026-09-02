# web-client-shell Specification

## Purpose

The static Next.js web client — its build output contract (static export, no
server), the application layout (collapsible sidebar with Home navigation), the
home placeholder, and the app icon / favicon.

## Requirements

### Requirement: Static build output

The web client SHALL build to a fully static bundle that runs without a Node.js
server. `pnpm build` SHALL emit the site to `frontend/out/`, and that directory
SHALL be serveable by any static file host (e.g. nginx) with no additional
runtime.

The build MUST NOT depend on server-only Next.js features: no route handlers, no
middleware, no request-time Server Component rendering, no image optimization
server.

#### Scenario: Build produces a static directory

- **WHEN** a developer runs `pnpm build` in `frontend/`
- **THEN** the build succeeds and writes `frontend/out/` containing `index.html`
  and hashed static assets
- **AND** no invocation of `next start` or any Node server is required to view
  the site

#### Scenario: Static host serves the bundle

- **WHEN** the contents of `frontend/out/` are served by a plain static file
  server
- **THEN** the home page loads and the sidebar renders and functions

#### Scenario: No server-only start script

- **WHEN** a developer inspects `frontend/package.json` scripts
- **THEN** there is no `start` script that runs `next start`

### Requirement: Application layout with collapsible sidebar

The web client SHALL render every page inside a shared layout consisting of a
left sidebar and a main content region. The sidebar SHALL contain, from top to
bottom: the application icon with a "Family Finances" wordmark, a navigation
list, and a footer holding a colour-theme control and a control to collapse or
expand the sidebar.

When collapsed, the sidebar SHALL show only the icon, the navigation item
glyphs, and the footer control glyphs; the main content region SHALL expand to
use the reclaimed width. The collapsed/expanded state SHALL persist across page
loads in the browser via `localStorage`, and the layout MUST render correctly on
first load when no stored value exists.

The sidebar MAY be a Client Component; all other layout and page components
SHALL remain Server Components rendered at build time.

#### Scenario: Default expanded layout

- **WHEN** a visitor opens the site in a browser with no stored sidebar state
- **THEN** the sidebar renders expanded, showing the icon, the "Family Finances"
  wordmark, the navigation list with labels, the theme control, and the collapse
  control

#### Scenario: Collapsing the sidebar

- **WHEN** the visitor activates the collapse control
- **THEN** the sidebar narrows to show only the icon, navigation glyphs, and the
  theme and collapse control glyphs
- **AND** the main content region widens to fill the reclaimed space

#### Scenario: State persists across reloads

- **WHEN** the visitor collapses the sidebar and then reloads the page
- **THEN** the sidebar is still collapsed after the reload

### Requirement: Selectable colour theme

The web client SHALL let the visitor choose one of three colour-theme states —
**System**, **Light**, or **Dark** — via a control in the sidebar footer. The
control SHALL be a single button that cycles through the three states in a fixed
order on activation, and its glyph SHALL indicate the currently selected state
(including distinguishing **System** from the resolved light or dark appearance).

The selected state SHALL persist in the browser via `localStorage`. When no
stored value exists, the state SHALL be **System**.

When the state is **Light** or **Dark**, the page SHALL render in that scheme
regardless of the operating system's `prefers-color-scheme`. When the state is
**System**, the page SHALL follow the operating system's `prefers-color-scheme`,
and SHALL update immediately if that preference changes while the page is open.

The page MUST NOT show a flash of the wrong theme on load: the resolved theme
SHALL be applied before first paint. Introducing script-driven theming MUST NOT
require changes to existing `dark:` styling in individual components.

The theme mechanism SHALL be entirely client-side and MUST NOT depend on
server-only Next.js features; the static build output contract is unchanged.

#### Scenario: Default follows the OS

- **WHEN** a visitor opens the site with no stored theme value
- **THEN** the page renders in the scheme matching the OS `prefers-color-scheme`
- **AND** the theme control indicates the **System** state

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
at `app/icon.svg` and as a multi-resolution `app/favicon.ico`. The default
`create-next-app` favicon SHALL be removed. Browsers SHALL load this icon as the
site favicon, and the same mark SHALL be used as the sidebar logo. Because the
build is static, the icon MUST be provided as static asset files, not generated
by a route handler.

#### Scenario: Favicon is the custom icon

- **WHEN** the site is loaded in a browser
- **THEN** the browser tab shows the house-with-Euro icon, not the default
  Next.js favicon

#### Scenario: Sidebar uses the same mark

- **WHEN** the sidebar renders
- **THEN** the icon shown at the top of the sidebar is the same house-with-Euro
  mark used as the favicon

### Requirement: No BACKEND_URL configuration

The web client SHALL NOT define, document, or read a `BACKEND_URL` environment
variable. `frontend/.env.example` SHALL be removed. Project documentation
(`frontend/AGENTS.md`, `frontend/README.md`, root `AGENTS.md`, root `README.md`,
`openspec/config.yaml`) SHALL describe the frontend as a static bundle with no
backend coupling, and MUST NOT reference `BACKEND_URL` or a server-side backend
call.

The frontend still MUST NOT open a database connection; the Go backend remains
the sole owner of persistence. The mechanism by which the deployed static client
reaches the backend at runtime is intentionally unspecified here and is recorded
as an open decision in the change's `design.md`.

#### Scenario: No BACKEND_URL in the repo

- **WHEN** the repository is searched for `BACKEND_URL`
- **THEN** there are no matches in code, configuration, or documentation
- **AND** `frontend/.env.example` does not exist

#### Scenario: Docs describe a static frontend

- **WHEN** a reader consults `frontend/AGENTS.md` or `frontend/README.md`
- **THEN** the frontend is described as producing a static bundle served by a
  static host, with no Node server and no `BACKEND_URL`

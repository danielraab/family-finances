## ADDED Requirements

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
list, and a control to collapse or expand the sidebar.

When collapsed, the sidebar SHALL show only the icon and the navigation item
glyphs; the main content region SHALL expand to use the reclaimed width. The
collapsed/expanded state SHALL persist across page loads in the browser via
`localStorage`, and the layout MUST render correctly on first load when no
stored value exists.

The sidebar MAY be a Client Component; all other layout and page components
SHALL remain Server Components rendered at build time.

#### Scenario: Default expanded layout

- **WHEN** a visitor opens the site in a browser with no stored sidebar state
- **THEN** the sidebar renders expanded, showing the icon, the "Family Finances"
  wordmark, the navigation list with labels, and the collapse control

#### Scenario: Collapsing the sidebar

- **WHEN** the visitor activates the collapse control
- **THEN** the sidebar narrows to show only the icon and navigation glyphs
- **AND** the main content region widens to fill the reclaimed space

#### Scenario: State persists across reloads

- **WHEN** the visitor collapses the sidebar and then reloads the page
- **THEN** the sidebar is still collapsed after the reload

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

## MODIFIED Requirements

### Requirement: Static build output

The web client SHALL build to a fully static bundle that runs without a
Node.js server. `pnpm build` SHALL emit the site to `frontend/out/`, and that
directory SHALL be serveable by any static file host with no additional
runtime. In production, this output is embedded into and served by the Go
backend (see the `backend-static-serving` capability) — no separate static
host process is deployed.

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

### Requirement: No BACKEND_URL configuration

The web client SHALL NOT define, document, or read a `BACKEND_URL` environment
variable. `frontend/.env.example` SHALL be removed. Project documentation
(`frontend/AGENTS.md`, `frontend/README.md`, root `AGENTS.md`, root `README.md`,
`openspec/config.yaml`) SHALL describe the frontend as a static bundle with no
backend coupling, and MUST NOT reference `BACKEND_URL` or a server-side backend
call.

The frontend still MUST NOT open a database connection; the Go backend remains
the sole owner of persistence. The mechanism by which the deployed client
reaches the backend at runtime is resolved: the Go backend serves the
frontend from the same origin (see the `backend-static-serving` capability),
so the frontend SHALL NOT need any base URL, proxy configuration, or CORS
handling to call the backend — client code MAY call relative paths under
`/api/` directly.

#### Scenario: No BACKEND_URL in the repo

- **WHEN** the repository is searched for `BACKEND_URL`
- **THEN** there are no matches in code, configuration, or documentation
- **AND** `frontend/.env.example` does not exist

#### Scenario: Docs describe a static frontend

- **WHEN** a reader consults `frontend/AGENTS.md` or `frontend/README.md`
- **THEN** the frontend is described as producing a static bundle with no
  Node server and no `BACKEND_URL`, embedded in and served by the Go backend
  at runtime

## Why

The repo has no release artifact and no CI: `frontend/` and `backend/` each run
locally but nothing builds a deployable image, and nothing enforces lint/tests
before merge. `open-topic.md` also left open how the deployed browser reaches
the backend (option ① nginx proxy, ② Go serves the static export, ③ direct
public backend). We're resolving that now in favor of **② the Go backend
serves the frontend's static export directly**, embedded into the binary via
`go:embed`. This collapses the previously-planned two-container nginx+backend
deployment into one image and one process, needs no CORS, keeps the backend
off the public internet directly, and removes the deferred-decision debt so
CI/CD and a real Dockerfile can be built against a concrete architecture.

## What Changes

- **BREAKING** (docs/architecture only, no released behavior yet): retires the
  "independent deployables, no shared build" framing for the *production
  release artifact*. Local dev toolchains stay separate (`cd backend` /
  `cd frontend`); the release path now fuses them into one Docker image.
- Add static-file serving to the Go backend: embed `frontend/out/` via
  `//go:embed` and serve it from the mux, with a reserved `/api/` namespace for
  future backend endpoints so they never collide with static asset paths.
- Add a multi-stage `Dockerfile` at the repo root: build the frontend
  (`pnpm build`), copy its `out/` into the backend build context, `go build`
  the binary with the embed populated, and ship a minimal non-root runtime
  image containing only the compiled binary.
- Add GitHub Actions workflows:
  - Lint + test the frontend (`pnpm lint`, `pnpm build`) and backend
    (`gofmt -l .`, `go vet ./...`, `go test ./...`) on every push and PR.
  - On push to the default branch, after lint/test succeed: build the
    Dockerfile and push to GHCR, tagged `YYYYMMDD-N` where `N` is computed by
    querying GHCR's existing tags for today's date and incrementing.
- Resolve `open-topic.md` (O1): record option ② as the decision; retire the
  file per its own "when this is decided" instructions.
- Update root `AGENTS.md`/`README.md`, `backend/AGENTS.md`/`README.md`, and
  `frontend/AGENTS.md`/`README.md` to describe the single-image architecture
  instead of the deferred/nginx-leaning language.

## Capabilities

### New Capabilities

- `backend-static-serving`: the Go backend embeds and serves the frontend's
  static export, with a reserved `/api/` namespace for backend routes.
- `release-pipeline`: Dockerfile producing a single deployable image, and CI
  that lints/tests both packages and publishes that image to GHCR with a
  `YYYYMMDD-N` tag on successful builds to the default branch.

### Modified Capabilities

- `web-client-shell`: the "Static build output" requirement's static-host
  example changes from generic/nginx to "embedded in and served by the Go
  backend"; the "No BACKEND_URL configuration" requirement's runtime-path
  question is no longer open — the frontend is served same-origin by the
  backend, so no proxy/CORS/base-URL config is needed.

## Impact

- `backend/main.go` (or a new file): embed directive, static file handler,
  route namespace reserved for `/api/`.
- `backend/` build: requires `frontend/out/` to exist before `go build` in
  production; local `go run .`/`go test` must keep working without a built
  frontend present.
- New: root `Dockerfile`, `.dockerignore`, `.github/workflows/*.yml`.
- Docs: root `AGENTS.md`, root `README.md`, `backend/AGENTS.md`,
  `backend/README.md`, `frontend/AGENTS.md`, `frontend/README.md`,
  `openspec/config.yaml`, `open-topic.md` (resolved/retired).
- No change to persistence, auth, or existing HTTP behavior (`GET /` still
  returns `ok` unless superseded by the embedded `index.html` — see design.md
  for how the root route resolves).

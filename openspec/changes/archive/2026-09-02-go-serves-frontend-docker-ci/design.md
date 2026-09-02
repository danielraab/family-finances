## Context

See `proposal.md` - Why/What Changes. Relevant current state:

- `backend/main.go` is a stdlib-only `net/http` stub: one route,
  `GET /{$}` → `handleRoot` → `200 "ok"`. `main_test.go` builds its own local
  mux and calls `handleRoot` directly.
- `frontend/next.config.ts` already sets `output: "export"`; `pnpm build`
  writes `frontend/out/` (verified structure not yet built in this repo, but
  Next 16 App Router static export conventions apply): `index.html` at the
  root, hashed JS/CSS under `out/_next/`, `favicon.ico`, `icon.svg`, and (no
  `app/not-found.tsx` exists) a generic `404.html` Next generates by default
  for static export.
- `open-topic.md` / the archived `frontend-static-shell/design.md` (O1) framed
  three options; this change formally picks ② and supersedes that open
  question. The archived file itself is left untouched (historical record);
  `open-topic.md` at the repo root is retired.
- No CI exists (`.github/workflows/` does not exist yet). No `Dockerfile`
  exists.

## Goals / Non-Goals

**Goals:**
- Go backend serves the built frontend with zero runtime dependency on a
  filesystem path outside the compiled binary (fully self-contained artifact).
- `go run .` / `go test ./...` / `go build .` keep working from a fresh clone
  with no frontend build present — production embedding must not be a
  prerequisite for local backend development.
- One Dockerfile, one image, no docker-compose, no nginx.
- CI blocks the image build/push on both packages' lint+test passing.
- Image tags are strictly ordered per day (`YYYYMMDD-N`) and computed from
  what's actually already published, not from a run counter that can skip or
  duplicate across reruns.

**Non-Goals:**
- Any new backend API route beyond relocating the existing stub (see D2).
- Auth, TLS termination, reverse proxy, docker-compose, k8s manifests.
- Frontend test workflow beyond lint + a successful static build (no test
  script exists in `frontend/package.json`; inventing one is out of scope).
- Cache busting / CDN strategy for static assets — `_next/`'s hashed
  filenames already give this for free.

## Decisions

### D1: `//go:embed all:static/out`, populated by the Docker build

Add `backend/static/out/` as the embed root, with a committed
`backend/static/out/.gitkeep` (and `backend/.gitignore` entry ignoring
everything else under `static/out/`) so the directory always exists for local
builds. The Dockerfile's frontend stage produces `frontend/out/`; the backend
build stage removes the placeholder and copies that output's contents into
`backend/static/out/` **before** `go build`, so the compiled binary embeds the
real site.

`//go:embed` directive: `all:static/out` (not a bare `static/out`). Next's
static export writes `_next/` for hashed assets — Go's embed patterns silently
skip any file or directory whose name starts with `_` or `.` unless the
pattern carries the `all:` prefix. Without `all:`, the entire `_next/`
directory (all JS/CSS) would be silently dropped from the binary with no
build error — a bug that only surfaces at runtime as a broken page. This is
worth calling out explicitly since it's easy to get wrong silently.

Code sketch:

```go
//go:embed all:static/out
var embeddedStatic embed.FS

func staticHandler() http.Handler {
    sub, err := fs.Sub(embeddedStatic, "static/out")
    if err != nil {
        log.Fatal(err)
    }
    fileServer := http.FileServerFS(sub.(fs.ReadDirFS)) // or http.FileServer(http.FS(sub))
    return notFoundFallback(sub, fileServer)
}
```

(Exact `fs.Sub` / `http.FileServerFS` wiring is a task-level detail; the
constraint that matters here is the `all:` prefix and the `fs.Sub` rebasing.)

- **Why embed over copy-to-disk-and-serve:** decided with the user — a single
  self-contained binary, no runtime path configuration, easy to run the exact
  same artifact locally (`go build && ./server`) as ships in the image.
- **Why a committed placeholder directory instead of a build tag /
  conditional compile:** `//go:embed` fails at compile time if the referenced
  path doesn't exist — there is no way to make the directive itself optional.
  A checked-in `.gitkeep` is the smallest way to keep `go build` working
  before any frontend build has run (fresh clone, local dev, `go test` in
  CI's backend-lint-test job). The embedded placeholder-only binary is never
  what ships — the Docker build always overwrites it first.
- **Alternative — `os.DirFS` reading from a sibling directory at runtime:**
  rejected per the user's decision; would need a runtime path/env var and
  makes the binary alone non-functional, which is what embedding avoids.

### D2: Route split — `/` and everything else to static, `/api/` reserved

The mux gets two registrations:
- `/api/` (prefix) — reserved for backend routes. The existing stub health
  check moves here as `GET /api/healthz` (still returns `200 "ok"`), so the
  namespace split is real from day one rather than aspirational.
- `/` (catch-all) — the static handler (D1), serving `index.html` at the root
  and falling back to the embedded `404.html` (via `fs.Stat`/`Open` check
  before delegating to `http.FileServer`, since Go's stdlib file server does
  not know about Next's `404.html` convention) with a `404` status for
  unmatched paths.

`main_test.go`'s `TestHandleRoot` moves with the handler: it becomes a test of
`GET /api/healthz`, still built the same way (local mux + direct handler
call), just re-pointed at the new path and possibly renamed
(`handleHealthz`).

- **Why relocate rather than drop the stub:** the proposal's non-goals keep
  "no new real API endpoints" — renaming/moving the existing one satisfies
  "reserve `/api/` for backend routes" from the proposal without inventing new
  behavior.
- **Alternative — keep `GET /{$}` returning `"ok"` and serve static under a
  different path:** rejected; the frontend's `index.html` must be at `/` for
  routing/relative asset paths to work under static export.

### D3: Dockerfile — three stages, distroless final image

```
Stage "frontend" (node:22-alpine + corepack)
  → pnpm install --frozen-lockfile && pnpm build   =>  /frontend/out

Stage "backend" (golang:1.26-alpine)
  → copy backend/ source
  → rm -rf static/out/* && copy --from=frontend /frontend/out/. static/out/
  → CGO_ENABLED=0 go build -o /app/server .

Stage "final" (gcr.io/distroless/static-debian12, nonroot)
  → copy --from=backend /app/server /app/server
  → USER nonroot
  → EXPOSE 8080
  → ENTRYPOINT ["/app/server"]
```

- **Why distroless over alpine for the final stage:** no shell, no package
  manager, smallest attack surface; the binary is fully static
  (`CGO_ENABLED=0`, stdlib only, no cgo dependency) so distroless's minimal
  libc-less base is sufficient.
- **Why not Docker BuildKit cache mounts for pnpm/go module caches in this
  design doc:** an optimization, not a correctness concern — left to
  implementation/tasks, doesn't change the shape of the stages.
- **`.dockerignore`:** exclude `frontend/node_modules`, `frontend/out` (built
  fresh in-stage), `backend/static/out/*` except `.gitkeep`, `.git`,
  `openspec/`, and other non-build files, so the Docker build context stays
  small and the placeholder isn't accidentally treated as real content.

### D4: CI — one workflow, three jobs, tag computed from GHCR

`.github/workflows/ci.yml`:

- `frontend`: checkout → pnpm/node setup → `pnpm install --frozen-lockfile` →
  `pnpm lint` → `pnpm build` (a failed static export fails the job; doubles as
  a build-health check since there's no frontend test script).
- `backend`: checkout → Go setup → `gofmt -l .` (fail if non-empty output) →
  `go vet ./...` → `go test ./...`.
- `publish`: `needs: [frontend, backend]`, `if:` push to the default branch
  only (not PRs, not other branches). Steps:
  1. Log in to `ghcr.io` with `GITHUB_TOKEN` (`packages: write` permission).
  2. Compute the tag: query the GHCR Docker Registry v2 tag-list API
     (`GET https://ghcr.io/v2/<owner>/<repo>/tags/list`, bearer token from the
     GHCR token endpoint using `GITHUB_TOKEN`) for tags matching today's
     `YYYYMMDD` prefix; a `404`/empty result means no image has been pushed
     yet today. Take the max existing `N` and increment (start at `1` if
     none). This queries what's actually published, so it's correct across
     reruns/retries/skipped jobs — unlike using `github.run_number`, which
     increments per workflow run regardless of whether a previous run's
     publish step actually completed.
  3. `docker/build-push-action` building the root `Dockerfile`, pushing
     `ghcr.io/<owner>/<repo>:YYYYMMDD-N` and `ghcr.io/<owner>/<repo>:latest`.
- **Why one workflow file with job dependencies over two separate workflow
  files:** `needs:` gives an explicit, visible gate (publish literally cannot
  run if lint/test fail) without cross-workflow triggers or artifact passing.
- **Why query GHCR instead of `github.run_number` or a committed counter
  file:** decided with the user — correctness (dense, gap-free daily
  counter reflecting what's actually published) over simplicity.

## Risks / Trade-offs

- **`//go:embed` without `all:` silently drops `_next/`** → called out
  explicitly in D1 and in `backend/AGENTS.md`; a task-level smoke test (curl
  the built image's `/` and confirm a `_next/static/...` asset 200s, not just
  that `index.html` loads) catches this before it ships.
- **Committed placeholder directory could bit-rot / be forgotten when editing
  the embed path** → document the two-file relationship
  (`backend/static/out/.gitkeep` + the `//go:embed all:static/out` directive)
  together in `backend/AGENTS.md`.
- **GHCR tag-list query race: two concurrent `publish` runs on the same day
  could compute the same `N`** → acceptable for this repo's single-pusher,
  sequential-CI reality; not solved here. If it becomes a real problem later,
  GitHub Actions' `concurrency:` key on the `publish` job serializes it
  cheaply — worth adding proactively in tasks even though the design doesn't
  strictly require it for correctness under the current usage pattern.
- **GHCR package may not exist yet on the very first push** (tags-list 404)
  → treat 404 as "zero existing tags", start at `N=1`; first
  `docker push` creates the package.
- **Distroless final image has no shell** → no `docker exec sh` debugging in
  production; acceptable trade-off for the security/size win, standard
  practice.
- **Docs now assert a decision that reverses an earlier documented lean (O1
  → ①)** → every doc file that mentioned the open question is in the
  proposal's Impact list; nothing should be left saying "not yet decided."

## Migration Plan

1. Backend: add the embed static handler + `/api/healthz` relocation +
   placeholder directory + `.gitignore` entry; update `main_test.go`. Verify
   `go build .` / `go test ./...` still pass with only the placeholder
   present (no real frontend build).
2. Add root `Dockerfile` and `.dockerignore`. Build locally
   (`docker build -t family-finances:local .`), run it, curl `/`, a hashed
   `_next/` asset, an unknown path (expect the embedded 404 page), and
   `/api/healthz`.
3. Add `.github/workflows/ci.yml` with the three jobs. Push on a branch first
   to confirm `frontend`/`backend` jobs run and `publish` correctly does NOT
   run off the default branch; verify tag computation logic separately (e.g.
   via `workflow_dispatch` or a scratch run) before trusting it on `main`.
4. Update docs (root `AGENTS.md`/`README.md`, `backend/AGENTS.md`/`README.md`,
   `frontend/AGENTS.md`/`README.md`, `openspec/config.yaml`) and retire
   `open-topic.md`.

Rollback: revert the change commit; no data/infra touched. A previously
published GHCR image tag is not deleted by a revert — acceptable, images are
immutable artifacts by design.

## Open Questions

None — all decisions needed to write specs/tasks were resolved above or with
the user during exploration.

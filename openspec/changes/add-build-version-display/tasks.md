## 1. API contract

- [x] 1.1 In `openapi/openapi.yaml`, add `GET /api/version` next to
      `getHealthz`/`getOpenAPIDocument` (tag `meta`, `security: []`),
      returning a `BuildInfo` schema with required `version` and
      `commit` string fields (both may be empty).
- [x] 1.2 Regenerate both committed artifacts:
      `cd backend && go generate ./...` and
      `cd frontend && pnpm generate:api`.

## 2. Backend build info

- [x] 2.1 Add `backend/internal/buildinfo/buildinfo.go`: package-level
      `var version, commit string` (the `-ldflags -X` targets), and a
      `Get() (version, commit string)` that returns them when
      `version` or `commit` is non-empty, else falls back to
      `runtime/debug.ReadBuildInfo()`'s `vcs.revision` /
      `vcs.modified` settings for `commit` (appending `-dirty` when
      modified), leaving `version` empty in the fallback path.
- [x] 2.2 Unit-test `Get()`'s fallback logic in
      `buildinfo_test.go` by setting the package vars directly (stamped
      case) and unsetting them (fallback case, asserting it does not
      panic when build info is present/absent).

## 3. HTTP endpoint

- [x] 3.1 Add `backend/internal/httpapi/version.go`: a
      `versionHandler()` returning the current `buildinfo.Get()` values
      as `{"version": ..., "commit": ...}` via `writeJSON`.
- [x] 3.2 Register `GET /api/version` in `Routes` (`server.go`)
      alongside `/api/healthz` and `/api/openapi.yaml`, unauthenticated.
- [x] 3.3 `version_test.go`: assert the response shape and status, and
      check it against the OpenAPI contract with
      `openapicheck.AssertResponse`, mirroring `health_test.go`.

## 4. Docker & CI stamping

- [x] 4.1 In the root `Dockerfile`'s backend stage, add `ARG VERSION=""`
      and `ARG REVISION=""`, and pass
      `-ldflags "-X at.draab/familyfinances/internal/buildinfo.version=${VERSION} -X at.draab/familyfinances/internal/buildinfo.commit=${REVISION}"`
      to the `go build` invocation.
- [x] 4.2 In `.github/workflows/ci.yml`'s `publish` job, pass
      `build-args: | VERSION=${{ github.ref_name }}` and
      `REVISION=${{ github.sha }}` to the `docker/build-push-action`
      step.
- [x] 4.3 Make the two args self-deriving for a caller that doesn't
      supply them (Dokploy, building the `Dockerfile` directly with no
      build-arg wiring): in the same `RUN`, `--mount=type=bind,
      source=.,target=/tmp/ctx,ro` (the context *root*, not `.git`
      directly — see 4.4) and, when `VERSION`/`REVISION` is empty and
      `/tmp/ctx/.git` exists, `git describe --tags --exact-match` /
      `git rev-parse HEAD` against it (installing `git` in that `RUN`
      only when needed) before the `go build`. An explicit build arg
      still wins.
- [x] 4.4 **Fixed after shipping**: the first cut mounted the bind
      straight at `source=.git`, which broke Daniel's real Dokploy
      build (`"/.git": not found` — its build context doesn't carry
      `.git`). A bind-mount source that doesn't exist fails the `RUN`
      outright with no way to make the mount conditional, so re-target
      the mount at the context root (`.`, which always exists) and
      check `[ -d /tmp/ctx/.git ]` before using it — a context without
      one now just builds unstamped, as it did before this feature
      existed, instead of failing. See design.md.

## 5. Frontend

- [x] 5.1 New `frontend/src/components/SidebarVersion.tsx`: calls
      `api.GET("/api/version")` once on mount, renders nothing while
      pending/on error, otherwise the version (or first 7 chars of the
      commit, full hash in `title`) as a small muted line, taking the
      sidebar's `collapsed` prop like `SidebarUser`/`ThemeSwitch`.
- [x] 5.2 Render `<SidebarVersion collapsed={effectiveCollapsed} />`
      in `Sidebar.tsx`'s footer, below `SidebarUser`.
- [x] 5.3 Add the line's translation key(s) (e.g. `sidebar.buildVersion`
      title text) to `frontend/src/i18n/locales/en.json` and the
      matching German string to `de.json`.

## 6. Verification

- [x] 6.1 `cd backend && go build ./... && go vet ./... && go test ./...`.
- [x] 6.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 6.3 Manually verify in a browser (stubbed `/api/version`) that the
      line renders in both expanded and collapsed sidebar states and
      disappears when the endpoint returns empty strings.
- [x] 6.4 Verify the Dockerfile's git-derivation shell logic directly
      (no image build, since a full `docker build` could not be run in
      the environment this change was authored in — its egress policy
      blocks pulling Docker Hub base images, confirmed by running a
      real `dockerd` there and hitting the same policy denial): confirms
      empty `VERSION`/full-hash `REVISION` on an untagged commit, the
      exact tag when `HEAD` is tagged, and — after 4.4's fix — that a
      context with no `.git` at all skips the derivation cleanly rather
      than failing.
- [x] 6.5 Confirmed against Daniel's real Dokploy build: the
      `source=.git` version (pre-4.4) broke it outright; the
      `source=.` + `[ -d .../.git ]` version is the one actually
      shipped. Still worth a final confirmation on Dokploy's next
      deploy that the build now succeeds (it should show no version,
      since Dokploy's build context has no `.git` — the derivation is a
      no-op there, not a fix for that specific platform's missing
      version line).

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

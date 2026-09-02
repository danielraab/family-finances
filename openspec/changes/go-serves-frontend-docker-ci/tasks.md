## 1. Backend static serving

- [ ] 1.1 Create `backend/static/out/.gitkeep` and add `backend/static/out/*`
      (except `.gitkeep`) to `backend/.gitignore`; verify `go build .` in
      `backend/` succeeds with only the placeholder present
- [ ] 1.2 Add `//go:embed all:static/out` (embed.FS) and wire an
      `fs.Sub`-rebased handler onto `/` using `http.FileServerFS`/
      `http.FileServer(http.FS(...))`, falling back to the embedded
      `404.html` (if present) with a `404` status for unmatched paths;
      verify with a unit test serving a fake file placed under
      `static/out/` in a test-local `embed.FS`/temp dir
- [ ] 1.3 Move the existing stub handler to `GET /api/healthz` (rename
      `handleRoot` → `handleHealthz` or similar), update `main_test.go`
      accordingly, and confirm the mux registers `/api/` before the static
      catch-all so `/api/...` never falls through to the file server;
      verify with `go test ./...`
- [ ] 1.4 Run `gofmt -l .` and `go vet ./...` in `backend/` and confirm both
      are clean

## 2. Dockerfile

- [ ] 2.1 Add root `Dockerfile`: stage "frontend" (node + pnpm, `pnpm install
      --frozen-lockfile && pnpm build`), stage "backend" (golang, copies the
      frontend stage's `out/` into `backend/static/out/` replacing the
      placeholder, `CGO_ENABLED=0 go build`), stage "final" (distroless
      static, non-root, `EXPOSE 8080`, `ENTRYPOINT`)
- [ ] 2.2 Add root `.dockerignore` excluding `frontend/node_modules`,
      `frontend/out`, `backend/static/out/*` (keep `.gitkeep` reachable via
      the build's own COPY, not the ignore file), `.git`, `openspec/`
- [ ] 2.3 Build locally: `docker build -t family-finances:local .` succeeds
- [ ] 2.4 Run the built image locally and verify: `GET /` returns the home
      page HTML, a `_next/static/...` asset returns `200`, `GET /api/healthz`
      returns `200`, and an unknown path returns `404`

## 3. GitHub Actions CI

- [ ] 3.1 Add `.github/workflows/ci.yml` job `frontend`: checkout, pnpm/node
      setup matching `frontend/package.json`'s `packageManager` field,
      `pnpm install --frozen-lockfile`, `pnpm lint`, `pnpm build`; verify by
      pushing a branch and confirming the job runs and passes
- [ ] 3.2 Add job `backend`: checkout, Go setup matching `backend/go.mod`'s
      Go version, `gofmt -l .` (fail the step if output is non-empty),
      `go vet ./...`, `go test ./...`; verify on the same branch push
- [ ] 3.3 Add job `publish` with `needs: [frontend, backend]` and an `if:`
      condition restricting it to pushes on the default branch;
      `permissions: packages: write`; verify the job is skipped on a
      non-default-branch push and on pull requests
- [ ] 3.4 In `publish`, add a step that queries the GHCR tag list
      (`GET https://ghcr.io/v2/<owner>/<repo>/tags/list`, or the GitHub
      Packages REST API) for tags matching today's `YYYYMMDD` prefix,
      treats a 404/empty result as zero existing tags, and computes the next
      `N`; verify by running the step in isolation (e.g. `workflow_dispatch`)
      against the (possibly still-empty) package before wiring it to the
      build/push step
- [ ] 3.5 Add `docker/login-action` (ghcr.io, `GITHUB_TOKEN`) and
      `docker/build-push-action` building the root `Dockerfile`, pushing
      `ghcr.io/<owner>/<repo>:YYYYMMDD-N` and `:latest`; verify by merging to
      the default branch and confirming the image appears in the repo's GHCR
      packages with the expected tag

## 4. Documentation

- [ ] 4.1 Update root `AGENTS.md`: architecture section describes the
      production release as one Docker image (Go backend embeds and serves
      the frontend) while keeping "separate toolchains, cd into the package"
      guidance for local dev; verify by re-reading the file for consistency
      with no leftover "independent deployables in production" wording
- [ ] 4.2 Update root `README.md` architecture section similarly, and add a
      short "Build & run the container" section referencing the Dockerfile
- [ ] 4.3 Update `backend/AGENTS.md`: document the `//go:embed all:static/out`
      mechanism, the `.gitkeep` placeholder purpose, the `/api/` vs static
      route split, and that the Docker build (not `go build` alone) is what
      produces the production artifact with the real frontend embedded
- [ ] 4.4 Update `backend/README.md` to mention the Docker image / build
      command and the `/api/healthz` endpoint (replacing the old `/` → `ok`
      entry in its endpoints table)
- [ ] 4.5 Update `frontend/AGENTS.md` "Data" section and `frontend/README.md`
      "Backend" section: replace the "not yet decided, see O1" language with
      "served by the Go backend at runtime; same-origin `/api/...` calls need
      no base URL or CORS config"
- [ ] 4.6 Update `openspec/config.yaml` context block to match (single
      release artifact; runtime path resolved)
- [ ] 4.7 Resolve and remove `open-topic.md` per its own "When this is
      decided" instructions (decision recorded, docs listed there updated);
      verify `git grep -n "open-topic"` and `git grep -n "BACKEND_URL"` in
      the repo return nothing referencing a still-open decision

## 5. Final verification

- [ ] 5.1 `git grep -rn "nginx"` across the repo (excluding this change's own
      `openspec/changes/` history) returns no remaining references implying
      nginx is the deployment plan
- [ ] 5.2 Full local pass: `cd backend && gofmt -l . && go vet ./... && go
      test ./...`; `cd frontend && pnpm lint && pnpm build`; `docker build
      -t family-finances:local .` from the repo root — all succeed

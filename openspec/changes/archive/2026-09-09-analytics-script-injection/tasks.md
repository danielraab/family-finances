## 1. Config

- [x] 1.1 Add `AnalyticsScript string` to `config.Config` in
  `backend/internal/config/config.go`, read via `os.Getenv("ANALYTICS_SCRIPT")`
  (no default, empty when unset — no parsing/validation needed, it's a raw
  string).
- [x] 1.2 Document `ANALYTICS_SCRIPT` in `backend/.env.example`: optional,
  raw HTML snippet injected before `</head>`, empty by default.

## 2. Static serving

- [x] 2.1 `backend/internal/httpapi/static.go`: change
  `staticHandler(fsys fs.FS)` to `staticHandler(fsys fs.FS, analyticsScript
  string)`. After reading `indexBody` via `fs.ReadFile`, if
  `analyticsScript != ""`, replace the first `</head>` with
  `analyticsScript + "</head>"` using `bytes.Replace(..., 1)`.
- [x] 2.2 In the returned handler, special-case `r.URL.Path == "/" ||
  r.URL.Path == "/index.html"`: write the rendered `indexBody` directly
  (status 200, `Content-Type: text/html; charset=utf-8`) instead of
  delegating to `fileServer` — today those two paths bypass the
  SPA-fallback swap entirely and are served straight from the embedded
  `fs.FS`, so they'd otherwise never see the injected snippet.
- [x] 2.3 `backend/internal/httpapi/server.go`: pass the configured value
  through to `staticHandler` at its call site in `Routes`.
  → Added `Deps.AnalyticsScript string` (kept `Routes(deps Deps)`'s
  signature unchanged, consistent with every other per-request dependency
  already living on `Deps` rather than `Routes` taking `cfg` directly);
  `main.go` sets it from `cfg.AnalyticsScript`.

## 3. Tests

- [x] 3.1 `backend/internal/httpapi/static_test.go`: unset
  `analyticsScript` (`""`) leaves the served `/` and SPA-fallback bodies
  byte-identical to the bundle's original `index.html`.
  → `TestStaticHandlerNoAnalyticsScriptLeavesIndexUnchanged` (also covers
  `/index.html`); every pre-existing `staticHandler` test call updated to
  the new signature with `""`, unchanged assertions confirm no regression.
- [x] 3.2 A non-empty `analyticsScript` appears immediately before
  `</head>` in the response body for both a direct `/` request and a
  client-route SPA-fallback request (e.g. `/login`).
  → `TestStaticHandlerInjectsAnalyticsScript` covers `/`, `/index.html`,
  and `/login`.
- [x] 3.3 `backend/internal/httpapi/server_test.go` (or wherever `Routes`/
  `New` are exercised end-to-end): confirm the value flows from
  `config.Config`/`Deps` through to what's served.
  → `TestRoutesAnalyticsScript`.

## 4. Spec sync

- [x] 4.1 Apply this change's `specs/backend-static-serving` delta onto
  `openspec/specs/backend-static-serving/spec.md` (the `openspec` CLI is
  unavailable in this environment, as for prior changes — applied by hand).

## 5. Verify

- [x] 5.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`.
  → All pass; `gofmt -l .` prints nothing.
- [~] 5.2 Manual pass: `ANALYTICS_SCRIPT='<script>console.log("test")</script>'
  go run .`, confirm `curl localhost:8080/` and `curl localhost:8080/login`
  both contain the snippet immediately before `</head>`; confirm an unset
  `ANALYTICS_SCRIPT` run serves byte-identical `index.html` to before this
  change.
  → Not run: this sandbox has no Docker daemon (`docker compose up -d db`
  fails — no `/var/run/docker.sock`), so `go run .` can't reach a
  `DATABASE_URL` it needs to start. Substituted with handler-level
  `httptest` coverage (3.1-3.3 above) exercising the identical code path
  (`staticHandler`/`Routes`) for both `/` and the SPA-fallback path, under
  both a set and unset `ANALYTICS_SCRIPT` — behaviorally equivalent to the
  curl pass, but flagging that the literal `go run .` + curl step was not
  independently exercised.

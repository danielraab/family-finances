## Why

There is currently no way to add a third-party script snippet (e.g. an
analytics or consent-management tag) to the served frontend without baking
it into the Docker image at build time. `frontend/index.html` is a static
file with no templating hooks, embedded into the Go binary at compile time
(`//go:embed`) and served byte-for-byte (`backend/internal/httpapi/static.go`).
A snippet whose value (a tracking ID, a vendor script tag) legitimately
differs per deployment — or that an operator wants to change or remove
without rebuilding — has nowhere to live today.

## What Changes

- Add an `ANALYTICS_SCRIPT` environment variable: an optional, raw HTML
  snippet (e.g. `<script>...</script>` or `<script src="..."></script>`)
  injected into the served `index.html` just before `</head>`. Empty/unset
  (the default) leaves `index.html` byte-identical to today.
- Read it once in `internal/config` (`config.Config.AnalyticsScript`), the
  backend's sole `os.Getenv` boundary — same pattern as every other
  deployment-varying value (`SMTP_*`, `AUTH_BASE_URL`).
- Render it into `index.html` **once, at server startup** (not per request):
  `staticHandler` gains the snippet as a parameter and does a single
  `bytes.Replace` on the embedded `index.html` bytes before they're ever
  served, for both places those bytes currently reach a client — the direct
  `/` (and `/index.html`) request and the SPA-fallback-on-404 swap.
- No database, no admin endpoint, no per-request cost: this is a
  deploy-time value, changed by restarting the container with a different
  `ANALYTICS_SCRIPT`, not something an admin edits at runtime through the
  UI.

## Capabilities

### Modified Capabilities

- `backend-static-serving`: the "Backend serves the frontend static export"
  requirement gains an optional startup-time HTML injection point into
  `index.html`, sourced from an environment variable — the served bundle is
  otherwise unchanged.

## Impact

- **Code**: `backend/internal/config/config.go` (+`AnalyticsScript` field),
  `backend/.env.example` (document `ANALYTICS_SCRIPT`),
  `backend/internal/httpapi/static.go` (`staticHandler` takes the snippet,
  renders `index.html` once at construction, and serves the rendered bytes
  for both the direct `/`/`/index.html` request and the SPA-fallback swap),
  `backend/internal/httpapi/server.go` (thread the value from `Deps`/`cfg`
  through to `staticHandler`).
- **Spec**: delta on `backend-static-serving`.
- No frontend, API contract, or database changes — the frontend ships no
  awareness of this mechanism; it is purely a backend serving concern.

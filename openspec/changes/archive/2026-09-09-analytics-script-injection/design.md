## Context

`frontend/index.html` is a static file (no template placeholders) that ships
inside the compiled backend binary: `backend/embed.go`'s
`//go:embed all:static/out` embeds the Vite build at compile time, and
`backend/internal/httpapi/static.go`'s `staticHandler` serves it via
`http.FileServerFS`, byte-for-byte. There are exactly two places those bytes
currently reach a client:

1. A direct request for `/` or `/index.html` — matched and served straight
   from the embedded `fs.FS` by `http.FileServerFS`, never touching the
   `indexBody` variable `staticHandler` already reads.
2. The SPA fallback: a `GET`/`HEAD` request for an extension-less path that
   misses the bundle (a client route like `/login`) — `fileServer` 404s,
   `staticInterceptor` swaps in `indexBody` with `200`.

Neither path offers a way to change `index.html`'s contents without a new
Docker image. An analytics/consent-tag snippet is a value that legitimately
varies per deployment (different tracking ID per environment, or no tag at
all for a self-hosted instance) and needs to be settable without a rebuild
— the same shape of problem `AUTH_BASE_URL`/`SMTP_*` already solve via
`internal/config`.

## Goals / Non-Goals

**Goals:**

- An operator can set one environment variable to have an arbitrary HTML
  snippet appear in `<head>` on every page load, without rebuilding the
  image.
- Unset (the default) is byte-identical to today's `index.html` — zero
  behavior change for every existing deployment.
- No added per-request cost: the snippet is rendered into the bytes once,
  at server startup.

**Non-Goals:**

- No admin UI, no database-backed setting, no runtime toggle without a
  restart — this is a deploy-time value, matching every other
  `internal/config` field. (An admin-editable version is a different,
  larger change if ever wanted — see `openspec/specs/user-settings` for the
  precedent that per-user/app settings already have a distinct, heavier
  pattern.)
- No validation or sanitization of the snippet's contents — the operator
  who sets `ANALYTICS_SCRIPT` is trusted the same way `SMTP_PASSWORD` or
  `OIDC_CLIENT_SECRET` are: a backend env var, not user input.
- No change to what the frontend itself knows or does — `frontend/` stays
  entirely unaware of this mechanism.

## Decisions

### Decision: environment variable + startup-time byte substitution, not `html/template`

The snippet is a single opaque string substituted once via `bytes.Replace`
on the embedded `index.html`, not a Go `html/template`. `html/template`
exists to safely compose *structured* data into HTML with contextual
escaping; here the value is deliberately raw HTML supplied by whoever
controls the deployment (the same trust level as any other env var), so
escaping it would break the common case (a `<script>` tag) rather than
protect anything. A plain byte substitution keeps the change small and
avoids introducing templating machinery for a single insertion point.

### Decision: render once at `staticHandler` construction, not per request

`staticHandler` already reads `indexBody` and `notFoundBody` once via
`fs.ReadFile` when it's constructed (server startup) and closes over them
for the lifetime of the process — the snippet substitution slots into that
exact same spot. Rendering per-request would cost a `bytes.Replace` on every
page load for a value that never changes between requests; there is no
reason to pay that.

### Decision: `staticHandler` must special-case `/` and `/index.html`, not just the SPA-fallback swap

The existing `indexBody` read is used *only* for the SPA-fallback 404 swap
— a direct `/` or `/index.html` request is answered by `http.FileServerFS`
straight from the embedded `fs.FS` and never touches `indexBody` at all
(the file exists in the bundle, so `FileServerFS` serves it as a normal
200 before the interceptor's 404-swap logic can engage). Without handling
this, the snippet would appear on client-route fallbacks (`/login`, refresh
on a deep link) but not on a first load of `/` — the opposite of what's
useful, since `/` is the common case. `staticHandler` therefore checks for
`r.URL.Path == "/" || r.URL.Path == "/index.html"` up front and writes the
already-rendered bytes directly for those two paths, before falling through
to `fileServer` for everything else.

### Decision: `ANALYTICS_SCRIPT`'s value is raw HTML, not a bare tracking ID

Different analytics/consent vendors want different markup (some need a
`<script>` block with inline config, others just a `src=` tag, others a
`<noscript>` fallback too). Accepting the full snippet as one opaque string
covers all of them with no vendor-specific code in this backend, at the
cost of the operator having to paste the vendor's exact tag rather than
just an ID. This mirrors treating it as configuration, not a new feature
the backend understands.

## Risks / Trade-offs

- **No sanitization.** Whoever sets `ANALYTICS_SCRIPT` can inject arbitrary
  HTML/JS into every page. Accepted: this is an env var only the deployer
  controls, the same trust boundary as every other `internal/config` value
  (a malicious `SMTP_HOST` or `AUTH_BASE_URL` is equally capable of harm).
  Never sourced from user input or request data.
- **Startup-only.** Changing the snippet requires a container restart, not
  a hot reload. Accepted as a Non-Goal — see above.
- **`bytes.Replace` targets `</head>` textually.** If a future
  `frontend/index.html` edit removes or restructures the `<head>` block in
  a way that removes the literal `</head>` string, injection silently
  becomes a no-op (the snippet is simply never inserted) rather than
  failing loudly. Accepted: `index.html` is a small, stable file
  (`frontend/AGENTS.md`); a static/handler test asserting the injected
  snippet appears in the served body when `ANALYTICS_SCRIPT` is set catches
  this in CI if it ever happens.

## Migration Plan

1. Add `AnalyticsScript string` to `config.Config`, read from
   `ANALYTICS_SCRIPT` (no default, empty string when unset) in
   `config.Load()`. Document it in `backend/.env.example`.
2. Change `staticHandler(fsys fs.FS)` to
   `staticHandler(fsys fs.FS, analyticsScript string)`: after reading
   `indexBody`, if `analyticsScript != ""`, replace the first `</head>`
   occurrence with `analyticsScript + "</head>"`. Add the `/` /
   `/index.html` special-case in the handler's `ServeHTTP` to write the
   rendered `indexBody` directly, ahead of delegating to `fileServer`.
3. Thread the value through: `Deps` gains a field (or `Routes`/`New`
   already receives `cfg config.Config` — read `cfg.AnalyticsScript`
   directly at the `staticHandler(deps.Static, cfg.AnalyticsScript)` call
   site in `server.go`, whichever keeps `Deps` least redundant with `cfg`).
4. Update `backend-static-serving`'s delta spec with the new optional
   behavior and scenarios.
5. Tests: `staticHandler`/`Routes` handler tests asserting (a) unset
   `ANALYTICS_SCRIPT` leaves `index.html` byte-identical to the bundle's
   original, for both `/` and the SPA-fallback path; (b) a set value
   appears before `</head>` in the response body, for both paths.

Rollback: revert the commit — additive, opt-in via an unset-by-default env
var; no migration, no data, nothing to unwind.

## Open Questions

None blocking.

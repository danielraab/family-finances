# AGENTS.md — backend

Go HTTP API for family-finances. Module `at.draab/familyfinances`.

## Stack

- Go 1.26. `net/http` (Go 1.22+ pattern routing), `log/slog`, `os`.
  **No web framework, no router library, no ORM.**
- Third-party dependencies are allowed but deliberate: add one only through an
  OpenSpec proposal that justifies it, and keep the set small. Current
  dependencies: `github.com/jackc/pgx/v5` (PostgreSQL driver + pool),
  `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2` (OIDC discovery,
  `id_token` verification, and the authorization-code exchange — see
  `internal/oidcauth`), and `github.com/getkin/kin-openapi` — **test-support
  only**: `internal/openapicheck` validates handler-test responses against
  `openapi/openapi.yaml`. A guard test (`openapi_guard_test.go`) asserts it
  never enters the server binary's dependency graph.
- Persistence is **PostgreSQL only**, reached through a `pgxpool.Pool` built
  from `DATABASE_URL` (required, no default). See "Persistence" below.

## Package layout

The backend is a single Go module. Application code lives under `internal/` so
nothing outside this module can import it and the packages stay free to change.
`package main` at the module root is wiring only.

```
backend/
├── main.go              # package main — compose the app, start/stop the server
├── healthcheck.go       # package main — `server healthcheck` self-probe (Docker HEALTHCHECK)
├── embed.go             # package main — //go:embed all:static/out  (must stay here; see below)
├── static/out/          # embed target — placeholder .gitkeep in git; Docker overwrites with the real build
├── go.mod
├── .env.example
└── internal/
    ├── config/          # Config struct + Load() — the only place os.Getenv is called
    ├── httpapi/         # HTTP wiring shared by every endpoint
    │   ├── server.go    #   New(cfg, deps) *http.Server; Routes(deps) *http.ServeMux
    │   ├── middleware.go #   request logging, panic recovery, request id
    │   ├── auth.go      #   Authenticator interface; bearer/cookie → request-scoped user; RequireAuth
    │   ├── respond.go   #   writeJSON / writeError (+ exported WriteJSON/WriteError); sentinel-error → status mapping
    │   ├── static.go    #   staticHandler(fs.FS) + notFoundInterceptor
    │   └── health.go    #   GET /api/healthz
    ├── auth/            # first domain package — accounts, identities, sessions, invites, OIDC login state
    │   ├── auth.go      #   domain types + email normalization + the identity-linking decision (§D1)
    │   ├── service.go   #   use-case logic; declares Store, Mailer, OIDCClient, LanguageLookup interfaces
    │   ├── store.go     #   Store interface + sentinel errors (ErrSignupDisabled, ErrTokenExpired…)
    │   ├── handler.go   #   http.Handler for /api/auth/…; RenderError injected so it needn't import httpapi
    │   ├── httpctx.go   #   CookieName, WithUser / UserFromContext
    │   └── *_test.go
    ├── settings/        # per-user preferences — display language, timezone, default currency
    │   ├── settings.go  #   domain type + hardcoded defaults (en/UTC/EUR) + validation
    │   ├── service.go   #   Get/Update (resolved) + Language (raw, for auth.WithLanguageLookup)
    │   ├── store.go     #   Store interface + sentinel error (ErrInvalidValue)
    │   ├── handler.go   #   http.Handler for GET/PUT /api/settings; imports auth only for UserFromContext
    │   └── *_test.go
    ├── mailer/          # auth.Mailer over net/smtp — STARTTLS/implicit/none, hand-built MIME
    ├── oidcauth/        # auth.OIDCClient over coreos/go-oidc/v3 + x/oauth2 — discovery, PKCE, id_token verify
    ├── cli/             # `admin grant|revoke|list` — dispatched from main.go beside `healthcheck`
    ├── account/         # one package per product noun (account, transaction, budget, …)
    │   ├── account.go   #   domain type + validation + invariants — no HTTP, no SQL
    │   ├── service.go   #   use-case logic; depends on the Store interface below
    │   ├── store.go     #   Store interface + sentinel errors (ErrNotFound, ErrConflict…)
    │   ├── handler.go   #   http.Handler for /api/accounts…; maps HTTP ⇄ service
    │   └── *_test.go
    └── storage/
        ├── memory/      #   in-memory Store implementations — default for tests (memory.NewAuthStore)
        └── postgres/    #   real persistence — pgxpool.Pool (NewPool), embedded
            #   migrations/*.sql applied at startup (Migrate), Store impls (postgres.NewAuthStore)
```

`internal/httpapi` may resolve a request-scoped user, but only through the
`Authenticator` interface (satisfied by `auth.Service`) — it imports neither
`internal/storage/...` nor a driver (enforced by a test). The `auth` handler is
kept free of an `internal/httpapi` import by taking a `RenderError` function
(wired to `httpapi.WriteError` in `main`), so auth's sentinel errors still map
to status codes in the one place — `httpapi/respond.go`.

### Rules that keep this layout honest

- **Dependencies flow one way.** `main` → `httpapi` + `config` + `storage/*` →
  domain packages (`account`, …). Domain packages import none of the others —
  no `httpapi`, no `storage`, no DB driver. `storage/*` packages implement the
  `Store` interface that the domain package *declares*; `main` picks an
  implementation and injects it. If you find yourself needing a domain package
  to import `httpapi`, the type or helper you want belongs in the domain
  package instead.
- **Package per noun, not per layer.** New feature = new `internal/<noun>/`
  with the four-file shape (`<noun>.go`, `service.go`, `store.go`,
  `handler.go`). Do not create `handlers/`, `services/`, `models/` buckets —
  they force every feature to touch every package and invite import cycles.
- **No HTTP status codes or `net/http` imports in domain code.** Domain and
  service code returns sentinel errors; `httpapi/respond.go` translates them to
  status codes in one place.
- **No `os.Getenv` outside `internal/config`.** Everything configurable is a
  field on `config.Config`, documented in `.env.example`, passed down as a
  value.
- **`main.go` is wiring only** — no route strings, no business logic:
  1. Subcommand dispatch: `os.Args[1] == "healthcheck"` → run the probe, exit;
     `os.Args[1] == "admin"` → `os.Exit(cli.Admin(ctx, os.Args[2:]))`. The probe
     stays in `healthcheck.go`; every other subcommand's logic lives in
     `internal/cli`.
  2. `config.Load()` (now returns an error — a bad duration/bool/`SMTP_TLS`
     fails fast); fail fast if `DATABASE_URL` is empty.
  3. `postgres.NewPool(ctx, cfg.DatabaseURL)`, then `postgres.Migrate(ctx, pool)`;
     either error → log and exit. `defer pool.Close()`.
  4. Build each domain service + handler with a `storage/postgres` Store; hand
     them to `httpapi.New` (along with the pool for the health probe). For auth
     that means: `postgres.NewAuthStore(pool)`, `mailer.New(cfg.SMTP)`, an
     `oidcauth.New(ctx, …)` client **only when `OIDC_ISSUER` and
     `OIDC_CLIENT_ID` are both set**,
     `auth.NewService(…)`, `auth.NewHandler(svc, RenderError: httpapi.WriteError)`;
     pass the service as `Deps.Auth` (the `Authenticator`) and the handler as
     `Deps.AuthHandler` (mounted at `/api/auth/`).
  5. `fs.Sub` the embedded FS, pass it in.
  6. `http.Server` with SIGINT/SIGTERM → `srv.Shutdown(ctx)`.
- **Routing lives in `internal/httpapi`.** `Routes()` builds the `*http.ServeMux`
  and mounts each domain package's `http.Handler` under its `/api/<noun>/`
  prefix. Backend routes are always under `/api/`; everything else falls
  through to the static handler.

### Testing

- Domain and service tests use `storage/memory` — no HTTP, no network.
- Handler tests use `net/http/httptest` plus the in-memory store; assert status
  code and JSON body. Prefer black-box `package <noun>_test`.
- `httpapi` middleware is tested on its own with a stub handler.
- Keep `_test.go` beside the code it exercises.

## Conventions

- Routing uses Go 1.22+ pattern syntax: `mux.HandleFunc("GET /api/accounts/{id}", h)`.
- Configuration comes from environment variables, loaded once in
  `internal/config`. Current vars are in `.env.example` (`PORT` default `8080`;
  `DATABASE_URL` required, no default; the `AUTH_*`, `SMTP_*`, `OIDC_*` groups
  below). Go does not auto-load `.env` — export the vars or use direnv.
  `config.Load()` returns `(Config, error)` and rejects a malformed duration,
  bool, or `SMTP_TLS` value.
- Logging via `log/slog` to stderr; one structured line per request from the
  logging middleware.

## Persistence

- **PostgreSQL only.** `internal/storage/postgres` owns the `pgxpool.Pool`
  (`NewPool`, connectivity-checked at startup) and the migration runner
  (`Migrate`): `.sql` files under `migrations/`, embedded with `//go:embed`,
  named `NNNN_slug.sql`, applied in order at startup inside a transaction each,
  tracked in `schema_migrations`. Forward-only — a bad migration is fixed by a
  new one. No migration-tool dependency.
- `internal/storage/memory` stays as the fast unit/handler-test store.
- Domain packages still import no `storage` package and no driver: they declare
  a `Store` interface, `internal/storage/postgres` implements it, `main`
  injects it.
- `GET /api/healthz` pings the database; it returns `503` when the DB is down.
- `internal/storage/postgres` integration tests need a real database: they read
  `DATABASE_URL` and **skip** when it is unset, so `go test ./...` passes on a
  bare checkout. CI supplies a `postgres` service container.

## Authentication

`internal/auth` is the first domain package. It owns `users`, `identities`
(`kind ∈ {email, oidc}`), `sessions`, `magic_link_tokens`, `invites`, and
`oidc_login_state` (migration `0002_auth.sql`). `users` also carries
`disabled` and `deleted_at` (migration `0004_user_administration.sql`, see
"User administration" below).

- **Sessions are opaque bearer tokens** — 256 bits from `crypto/rand`, stored
  only as `sha256(token)`. Never a JWT. Browsers get an
  `HttpOnly; Secure; SameSite=Lax` cookie named `ff_session`; API/mobile
  clients (`Accept: application/json` or `?client=api` on the callback) get the
  token in the JSON body for `Authorization: Bearer`. One `sessions` table
  backs both. Sliding expiry (`AUTH_SESSION_TTL`, bumped past half-life) under a
  hard cap (`AUTH_SESSION_MAX_TTL`). Logout deletes the row.
- **Two sign-in methods, one account.** Magic link (`POST /api/auth/email/start`
  always `200` — no enumeration) and one OIDC provider
  (`GET /api/auth/oidc/start` → code + PKCE + nonce). They link to the same
  `user` by verified email (magic link always proves it; OIDC only on
  `email_verified: true`), or explicitly while authenticated.
- **`GET /api/auth/config`** — unauthenticated; reports which sign-in methods
  the client should show. Today: `{ "oidc": { "label", "start_path" } }` when an
  OIDC provider is configured, else `{ "oidc": null }`. `label` is `OIDC_LABEL`.
  `Service.OIDCLogin() (label string, ok bool)` is the accessor.
- **Registration policy, all env-driven** (`config.AuthConfig`): `AUTH_SIGNUP_ENABLED`,
  `AUTH_ALLOWED_EMAIL_DOMAINS` (comma list, empty = any, checked only at account
  creation), `AUTH_INVITE_ENABLED` (only bites once signup is off; any
  authenticated user may invite otherwise). When `users` is empty the system is
  in bootstrap mode: signup is forced open and the first account is an admin
  (enforced inside the account-creation transaction under an advisory lock).
- **`admin` CLI:** `server admin grant <email>`, `server admin revoke <email>`,
  `server admin list` — in `internal/cli`, dispatched from `main.go`. `is_admin`
  gates the user-administration endpoints below (its first real use as an
  authorization check).
- Env groups: `AUTH_BASE_URL` (builds magic-link URLs and the OIDC
  `redirect_uri`), `AUTH_SESSION_TTL`/`AUTH_SESSION_MAX_TTL`,
  `AUTH_COOKIE_SECURE`, `AUTH_SIGNUP_ENABLED`, `AUTH_ALLOWED_EMAIL_DOMAINS`,
  `AUTH_INVITE_ENABLED`, `AUTH_INVITE_TTL`, `AUTH_MAGIC_LINK_TTL`;
  `SMTP_HOST`/`SMTP_PORT`/`SMTP_USERNAME`/`SMTP_PASSWORD`/`SMTP_FROM`/`SMTP_TLS`
  (`starttls|implicit|none`);
  `OIDC_ISSUER`/`OIDC_CLIENT_ID`/`OIDC_CLIENT_SECRET`/`OIDC_SCOPES`/`OIDC_LABEL`
  (button text, default `Single sign-on`).
  OIDC is optional — the client is built only when `OIDC_ISSUER` **and**
  `OIDC_CLIENT_ID` are both set; otherwise the `/api/auth/oidc/*` routes are
  disabled and `/api/auth/config` reports `oidc: null`.

## User administration

Admin-only endpoints on `auth.Handler`, each gated on `user.IsAdmin` (`403`
for a non-admin, `401` unauthenticated): `GET /api/auth/users` (every
non-soft-deleted user), `GET /api/auth/invites` (every non-soft-deleted
invitation, any status, with the inviter's identity),
`POST /api/auth/users/{id}/disable`, `POST /api/auth/users/{id}/enable`,
`DELETE /api/auth/users/{id}` (soft delete — one-way, no undelete endpoint).
Disable and delete both call `Store.DeleteSessionsByUserID` to revoke every
session belonging to the target **immediately**, not just rely on expiry;
the auth middleware (`Service.Authenticate`) also re-checks
`disabled`/`deleted_at` on every request as a belt-and-suspenders guard
against a session row that somehow outlives the revocation. A disabled or
soft-deleted account is rejected by both sign-in flows too
(`ErrAccountDisabled`, `403`) — magic-link `POST /api/auth/email/start`
treats it like "no account" (still `200`, no mail sent).

**No self-lockout guard, deliberately.** An admin may disable or delete their
own account, including as the only remaining admin — there is no
server-side check preventing it. The frontend's confirmation dialog is the
only mitigation; see `openspec/specs/user-administration/spec.md`.

**Invite revocation and soft-delete** (`invites` gains `revoked_at` and
`deleted_at`, migration `0005_invite_revocation.sql`): `invites` is not
purely admin-managed — `GET /api/auth/invites/mine` lets any authenticated
user list the invitations *they* created (`invited_by = self`), and
`POST /api/auth/invites/{id}/revoke` is permitted for that same person or
for an admin (`403` for anyone else), not gated behind `requireAdmin` the
way every other admin endpoint is. Revoking is idempotent — a repeat call
leaves `revoked_at` at its original value and still returns `200` — and is
allowed on an invite in any state (pending, accepted, or expired); a revoked
invite is never removed from a listing, only `deleted_at` does that.
`DELETE /api/auth/invites/{id}` (admin-only, `204`) requires `revoked_at` to
already be set, `409` otherwise — it's cleanup for a revocation, not a
general-purpose delete. `Store.ConsumeInvite`'s atomic acceptance query
additionally excludes `revoked_at IS NOT NULL`, so a revoked invite's
acceptance link stops working immediately.

## Settings

`internal/settings` is a small domain package for per-user preferences:
display language, timezone, default currency (`user_settings` table,
migration `0003_user_settings.sql` — one row per user, every column
nullable; a missing row and a `NULL` column both mean "use the hardcoded
default," resolved once in `Service.Get`/`Update`). `GET`/`PUT /api/settings`
require authentication; `PUT` is a partial update (`ON CONFLICT (user_id) DO
UPDATE`, touching only the fields present in the body). Validation:
`language ∈ {en, de}`, `timezone` via `time.LoadLocation`, `default_currency`
by ISO-4217 shape (three uppercase letters) — not a canonical list.

`internal/settings` imports `internal/auth` only for `auth.UserFromContext`
(the same read-only context accessor `internal/httpapi` uses) — never its
`Store` or a driver. The reverse dependency runs the other way for the
language preference specifically: `auth.Service` declares its own narrow
`LanguageLookup` interface (`Language(ctx, userID) (*string, error)`,
satisfied structurally by `settings.Service`) and `main.go` wires it in via
`auth.WithLanguageLookup` — so `GET /api/auth/me` can embed the *raw*,
unresolved language preference for the web client's i18n precedence, without
`internal/auth` importing `internal/settings`. See
`openspec/specs/user-settings/spec.md` for why the raw value (not the
resolved one from `GET /api/settings`) has to be the one on `/me`.

## Serving the frontend

The compiled binary embeds and serves the frontend's Vite bundle — there is
no separate static host in production. The frontend is a client-only SPA
(React + TanStack Router, `frontend/`); see `frontend/AGENTS.md`. The
mechanics are load-bearing:

- `embed.go` stays in `package main` at the module root with
  `//go:embed all:static/out`. `//go:embed` cannot reference parent
  directories, and the Docker build does `COPY … static/out/` relative to
  `backend/` — moving the directive into `internal/` would break both. The
  *serving logic* (`staticHandler`, `staticInterceptor`) is ordinary code and
  lives in `internal/httpapi/static.go`, taking an `fs.FS`; `main.go` does
  `fs.Sub(embedded, "static/out")` and passes the result in.
- The `all:` prefix is kept but **no longer required**: Vite writes hashed
  assets under `assets/` (no leading `_` or `.`), which plain `//go:embed`
  already includes. It's left in as a cheap guard against a future asset
  directory that starts with `_`.
- `backend/static/out/` holds only a committed `.gitkeep` in this repo —
  `//go:embed` requires the path to exist at compile time, so the placeholder
  keeps `go build` / `go test` / `go run .` working from a fresh clone with no
  frontend build present. **It is not what ships**: the Docker build (root
  `Dockerfile`) overwrites this directory with the real `frontend/out/` output
  before compiling. A local `go build` produces a binary that serves an empty
  site — build the Docker image (or copy a `pnpm build` output into
  `static/out/` first) to get the real thing.
- Routes split at `/api/`: backend routes live under that prefix
  (`GET /api/healthz`, `/api/auth/…`). An unmatched path under `/api/` gets a
  JSON `404` — the reserved namespace never falls through to the static site.
- Every non-`/api/` path goes to the static handler. It serves a bundled file
  when one matches. On a miss, a **GET/HEAD request whose path has no file
  extension** is treated as a client route and answered with `index.html` and
  `200` (so the in-browser router renders `/login`, `/account/234/edit`, … on a
  direct load or refresh). A miss whose path *has* an extension (a real asset
  request) still returns `404` — the embedded `404.html` if the bundle ships
  one, otherwise the default. Add new backend endpoints under `/api/` so they
  never collide with a frontend route.

## Serving the API contract

`embed.go` carries a **second** `//go:embed` — `openapi.yaml` — plus a
`//go:generate cp ../openapi/openapi.yaml ./openapi.yaml` that syncs the
committed copy from the repo-root source of truth (`//go:embed` can't cross
`..`; run `go generate ./...` after editing the spec, and CI fails on drift).
`main.go` passes the bytes to `httpapi.Deps.OpenAPISpec`; `internal/httpapi`
serves them verbatim at `GET /api/openapi.yaml` (`application/yaml`, no auth),
registered ahead of the `/api/` catch-all. See root `AGENTS.md` and
`openapi/README.md`. New JSON endpoints are added to `openapi/openapi.yaml` in
the same change, with a response-conformance assertion
(`internal/openapicheck.AssertResponse`) in their handler test.

## Layout status

The `internal/` layout above is in place (`internal/config`,
`internal/httpapi`, `internal/storage/memory`, `internal/storage/postgres`).
`package main` is `main.go` (wiring + graceful shutdown + subcommand dispatch),
`healthcheck.go`, and `embed.go`. `internal/auth` is the first domain package,
in the four-file shape, with `Store` implementations in both `storage/memory`
and `storage/postgres` and migration `0002_auth.sql`; `internal/mailer`,
`internal/oidcauth`, and `internal/cli` are its supporting leaf packages.
`internal/settings` is the second domain package, same four-file shape,
`Store` implementations in both storage backends, migration
`0003_user_settings.sql`; `0004_user_administration.sql` extends `users`
for `internal/auth`'s admin endpoints and `0005_invite_revocation.sql`
extends `invites` the same way. The next product noun adds another
`internal/<noun>/` the same way.

## Before you're done

```bash
gofmt -l .        # must print nothing
go vet ./...
go test ./...
```

## Commands

```bash
docker compose up -d db          # from the repo root — start PostgreSQL
export DATABASE_URL=postgres://familyfinances:familyfinances@localhost:5432/familyfinances?sslmode=disable
go run .          # start on :8080 (or $PORT)
go build .        # production binary (serves an empty site without a frontend build)
```

`go run .` / `go build .` fail fast without a reachable `DATABASE_URL`. The
whole stack (app image + db) comes up with `docker compose up --build` from the
repo root.

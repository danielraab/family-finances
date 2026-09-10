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
- **Integration tests do not run by default.** `internal/storage/postgres` is
  the only integration-test package (see "Persistence" below) — everything
  else is a fast unit/handler test with no external dependency. Don't wire
  integration tests into a path that always executes them (a default `go
  test ./...` locally, a pre-commit hook, the main CI job that gates every
  push); they belong behind an explicit `DATABASE_URL` and, in CI, in the
  dedicated `backend-integration` job described below.

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
  bare checkout with no database running. They are intentionally excluded from
  the main `backend` CI job (no `DATABASE_URL` there, so they self-skip) and
  instead run in their own `backend-integration` job
  (`.github/workflows/ci.yml`), which starts a `postgres` service container,
  sets `DATABASE_URL`, and runs `go test ./internal/storage/postgres/...`. That
  job is scoped to `pull_request` events — integration tests run once per PR,
  not on every push to a branch with no open PR. Run them locally the same way
  the job does: start `docker compose up -d db`, export `DATABASE_URL` (see
  "Commands" below), then `go test ./internal/storage/postgres/...`.

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

## Account types

An account's type is a **plain required `text` column on `accounts`**
(`accounts.type`, migration `0021_account_type_text.sql`) — not a separate
entity. `internal/account`'s domain/service/store only trims it of
surrounding whitespace before storing; it is never case-folded or checked
against a list. A blank (or whitespace-only) `type` on
`POST /api/accounts` / `PATCH /api/accounts/{id}` is `ErrInvalidValue`
(`400`), the same mapping a blank `title` gets. A shared `owner`-tier
caller may change `type` like any other account field.

`GET /api/account-types` is repurposed to a read-only autocomplete source:
it returns a `[]string` of the caller's own distinct, non-empty, trimmed
`type` values across their non-deleted accounts, sorted case-insensitively
(ties broken by raw value). `Store.ListInUseTypes(ctx, ownerID)` backs it
(`SELECT DISTINCT type … ORDER BY lower(type), type` in Postgres). There is
no create/update/delete/disable/enable endpoint, no `account_types` table,
no `Type` struct, no seeding hook, and no `ErrTypeInUse`/`ErrTypeDisabled`
sentinel anymore.

The former default titles (Checking, Savings, Cash, Credit Card, Loan,
Investment) now live only in the frontend (`DEFAULT_ACCOUNT_TYPES` in
`AccountForm.tsx`), merged with the `GET /api/account-types` response as the
type field's `<datalist>` suggestions. `main.go` wires only
`categorySvc` into `auth.WithNewUserHooks(...)`; a brand-new user gets
seeded starter categories but no account types.

## Categories

`internal/category`'s tree-structured `categories` lookup is **per-user**,
not admin-managed — every operation is scoped to `owner_id = <authenticated
caller>` (`Store` methods take an explicit `ownerID` parameter, mirroring
`internal/account`), and a category belonging to a different owner reads as
`ErrNotFound` (`404`), never `403`. There is no `is_admin` gate on any
category endpoint. `parent_id` self-references form the tree, no depth
limit; a self/descendant reparent is rejected (`ErrCycle`, `422`).

- **`disabled`** (reversible): toggled via
  `POST /api/categories/{id}/disable` / `/enable`. It blocks the category
  from being **newly** selected on an entry (`internal/entry`'s
  `CategoryLookup.Usable` checks it) without touching any entry or child
  category already referencing it — an update that doesn't touch
  `category_id` never re-checks the entry's current category.
- **Soft delete** (`deleted_at`, one-way, no undelete) replaces what used to
  be a hard delete: `DELETE /api/categories/{id}` is rejected (`409`
  `ErrInUse`) while the category has a non-deleted child or is referenced
  by a non-deleted entry. `disabled` and delete are independent — a
  category need not be disabled first, and disabling never affects whether
  it can subsequently be deleted.
- **`sort_order`** orders siblings only (same `owner_id` + `parent_id`), not
  the whole tree. A new category is appended to the end of its sibling
  group; so is a reparented one, into its new parent's group.
  `POST /api/categories/{id}/move-up` / `/move-down` each swap `sort_order`
  with the immediate previous/next sibling — a no-op (`200`, unchanged) at
  either end of the sibling list, never an error.
- No naming-uniqueness constraint — deliberately dropped rather than
  reworked into an owner-scoped one; duplicate sibling names are allowed.
- **`icon` / `color`** (migration `0019_account_category_icon_color.sql`,
  which also adds the pair to `accounts`) are two independent optional
  fields — short opaque presentation tokens (`^[a-z0-9-]{1,40}$` or empty,
  validated for shape only in `category.go` / `account.go`; a bad value is
  `ErrInvalidValue` → `400`). The backend never interprets them — the web
  client owns the icon set and the colour palette. On update a `*string`
  distinguishes absent (untouched) from `""` (clear) from a value (set);
  the Postgres stores use `CASE WHEN $n::text IS NULL THEN col ELSE
  NULLIF($n, '') END`. Account **types** and tags do **not** carry them.

`internal/entry`'s `CategoryLookup` interface (`*category.Service` satisfies
it structurally) is `Usable(ctx, ownerID, categoryID) (bool, error)` —
exists, owned by `ownerID`, and not disabled, consulted only when a
category is being newly set (creation, or an update that explicitly
supplies `category_id`) — and `Subtree(ctx, ownerID, categoryID)
([]string, error)`, scoped to that owner's own tree.

## Tags

`internal/tag`'s per-user `tags` lookup is a flat list, not admin-managed —
every operation is scoped to `owner_id = <authenticated caller>`, and a tag
belonging to a different owner reads as `ErrNotFound` (`404`), never `403`.
Unlike categories, `DELETE /api/tags/{id}` is **unconditional**
— always `204`, detaching the tag from every entry it was attached to via
`entry_tags.tag_id`'s `ON DELETE CASCADE` — there is no in-use block and
never has been.

- **`disabled`** (reversible, mirroring `categories.disabled`): toggled via
  `POST /api/tags/{id}/disable` / `/enable`. It blocks the tag from being
  **newly** attached to an entry without touching any entry already carrying
  it, and — because `tag_ids` is always a full replacement array rather than
  a delta — specifically means only ids *not already on the entry* are
  checked; resubmitting the same `tag_ids` array on an otherwise-unrelated
  edit never trips it. See `internal/entry`'s `TagLookup` below.
- **`entry_count`**: every `Tag` response carries the number of the owner's
  own non-deleted entries currently carrying it, computed by
  `internal/storage/postgres/tag.go` as a correlated subquery against
  `entry_tags`/`entries` (`… AND e.deleted_at IS NULL`) folded into the same
  column list every tag query already selects — no separate endpoint, no
  `JOIN … GROUP BY` row-duplication risk. This mirrors how
  `internal/storage/postgres/category.go`'s `Delete` already reaches into
  `entries` by raw SQL for its in-use check, with no Go import of
  `internal/entry` either way — domain packages don't import each other, but
  their Postgres stores may still name each other's tables.
  `internal/storage/memory`'s `TagStore` has no visibility into entries (the
  same accepted gap `memory.CategoryStore` already has for its own in-use
  check), so `EntryCount` there always reads `0` — fine, since
  `storage/memory` is test/local-dev infrastructure, never what ships.

`internal/entry`'s `TagLookup` interface (`*tag.Service` satisfies it
structurally) has two methods: `OwnedBy(ctx, owner, tagIDs) (bool, error)` —
existence + ownership only, checked against the *entire* resubmitted
`tag_ids` array on every create/update that supplies one — and
`Usable(ctx, owner, tagIDs) (bool, error)` — existence + ownership + not
disabled, checked only against the ids a create/update is *newly adding*
(every id, for `Create`; the set difference against the entry's current
`TagIDs`, for `Update`). This two-method split is what lets a since-disabled
tag stay attached across unrelated edits while still blocking it from being
picked for the first time.

## Accounts and sharing

`internal/account` owns `accounts` plus `account_shares` (migration
`0020_account_sharing.sql`) — a many-to-many grant of one of four
`Permission` tiers (`view`, `append`, `entry_admin`, `owner`, ranked in that
order; `Permission.AtLeast(other)` is the comparison every authorization
check uses) from an account's real owner to another user.

- **`Access(ctx, accountID, callerID)`** is the one place that resolves a
  caller's effective tier: the real owner always resolves to `owner`;
  otherwise it's whatever `account_shares` row matches, or the empty string
  for no access at all. Handlers and `internal/entry` both branch on this —
  an empty permission is `ErrNotFound` (`404`, indistinguishable from the
  account not existing); a real but insufficient tier is `ErrForbidden`
  (`403`).
- **A shared `owner`-tier grant has full parity with the real owner**,
  including disabling/deleting the account, changing every metadata field
  (`type` included), and managing other shares.
- **`GET`/`POST /api/accounts/{id}/shares`,
  `PATCH`/`DELETE /api/accounts/{id}/shares/{userId}`,
  `POST /api/accounts/{id}/shares/leave`** — list requires any tier; invite/
  update/revoke require `owner`, except a caller may always self-leave
  regardless of tier. The real owner can never be a target of update, revoke,
  or self-leave (`ErrInvalidValue`, `400`).
- **Sharing is by email, resolved through `UserLookup`** (`internal/auth`'s
  `*Service` satisfies it structurally via `ByEmail`/`InvitingEnabled`, wired
  in `main.go` with `account.WithUserLookup` since it's a post-construction
  dependency loop — `auth` doesn't import `account`). A match creates/
  updates the share row and sends a notification email
  (`mailer.SendAccountShare`); no match returns `{matched: false,
  invite_allowed}` without writing anything, leaving the frontend to offer
  sending a real invite when invites are enabled.
- **Revoking a share is immediate and total**: the revoked user loses read/
  write access to the account and to every entry they created on it (their
  entries stay intact and fully visible/editable to everyone who retains
  access — see `created_by` below, not an ownership transfer).
- **`GET /api/accounts` and the caller's visible-account set widen to
  "owned or shared"** (`Access.Permission != ""`), and the `Account`
  response carries the caller's own `permission` plus, only when the caller
  isn't the real owner, `shared: true` and `owner_name`.

## Entries

`internal/entry` owns `entries` — transactions and balance adjustments
recorded against an account — plus their live balance computation and a
filterable/searchable/sortable, cursor-paginated listing (`GET`/
`POST /api/entries`, `GET`/`PATCH`/`DELETE /api/entries/{id}`,
`GET /api/accounts/{id}/balance`).

- **`created_by`** (renamed from `owner_id` in migration `0020`) records
  which user logged an entry, always the caller at `Create` time, and is
  returned to every viewer as `created_by`/`created_by_name` — not scoped to
  "only shown when it's someone else's entry." Write access is gated by the
  caller's `account.Access` tier on the entry's account
  (`entry.AccountLookup.Access`, a flat-value mirror of `account.Access` to
  avoid an import cycle — see the `TagLookup`/`CategoryLookup` pattern
  above): `append` may only edit/delete entries where `created_by` is the
  caller (`ErrForbidden` otherwise); `entry_admin` and `owner` may edit/
  delete any entry on the account; `view` may never write.

- **`amount` is always a signed delta applied to the running balance,
  for both kinds.** For a `transaction` it's exactly what the caller
  submits. For a `balance_adjustment` it is never client-supplied — the
  caller instead submits the absolute reading as `balance`
  (`entries.balance_reading` in Postgres, `CHECK`-constrained to be
  non-null exactly when `kind = 'balance_adjustment'`), and `amount` is
  computed as that reading minus the account's balance strictly before
  the entry's own `(booking_timestamp, id)` position. `POST`/`PATCH`
  reject `amount` on a `balance_adjustment` and `balance` on a
  `transaction` (`ErrInvalidValue`, `400`).
- **`Balance(asOf)` is a single `SUM(amount) WHERE … <= asOf`** — no
  kind-branching. This reproduces exactly the same values as "reset to
  the latest balance adjustment, then sum only the transactions after
  it," because a balance adjustment's `amount` is *defined* to make that
  running sum land on its reading, by construction.
- **Recompute is synchronous, inside the same Create/Update/SoftDelete
  operation, never async** — this backend has no job queue, and the
  numbers must be right immediately, not eventually. Only the earliest
  non-deleted balance adjustment at or after the mutated position (`A1`),
  and the one immediately after it (`A2`), can ever need a new `amount`;
  recomputing exactly those two is always sufficient — see
  `internal/storage/postgres/entry.go`'s `recomputeFrom`/`findAdjustment`/
  `setAmount` (and the mirrored Go-loop version in
  `internal/storage/memory`) for the algorithm and its worked example.
  An `Update` that moves an entry's `account_id` and/or
  `booking_timestamp` recomputes at both the vacated position (excluding
  the entry's own, already-updated row from that search) and the new one.
- **`GET /api/entries/flow-summary`** buckets the caller's matching
  entries into per-month or per-day income/outcome totals (`unit=month|
  day`, repeatable `account_id`, `year`, `month` — required for `day`,
  rejected for `month`), using the caller's resolved timezone
  (`internal/settings`, default UTC) to decide bucket boundaries. Unlike
  `GET /api/entries/summary` (which always excludes `balance_adjustment`
  — an absolute reading has no place in a spend-by-category report),
  flow-summary *includes* a balance adjustment's computed delta, since it
  answers a different question: how the balance actually moved. Every
  period in the requested range is present even when empty (`income: []`,
  `outcome: []`) — no special-casing. `internal/settings.Service.
  Timezone` (parallel to its existing `Language`) is wired in via
  `entry.WithTimezoneLookup`, the same optional-dependency pattern
  `auth.WithLanguageLookup` uses, so `internal/entry` still doesn't
  import `internal/settings`.

## Seeding fake data

`server seed --yes [--entries N] <email1,email2,...>` (`internal/cli.Seed`,
dispatched from `main.go` alongside `healthcheck`/`admin`) resets the
database and creates one user per given email with randomly generated
accounts and entries — for local/demo use, not anything a product feature
depends on. Args are parsed with the stdlib `flag` package (`parseSeedFlags`):
`--yes` (bool), `--entries N` (optional, `int`), and one trailing positional
— the comma-separated email list. The emails are entirely CLI-supplied —
nothing is hardcoded — via `parseSeedEmails` (splits on `,`,
`auth.NormalizeEmail`s and `auth.ValidateEmail`s each, rejects an empty
list, an invalid address, or a duplicate before touching the database).

- **`--entries N`** sets an exact per-user transaction-entry target,
  spread as evenly as possible across that user's 2-4 accounts (the first
  `N % numAccounts` accounts take one extra). `0` / omitted keeps the
  default 15-60 random entries per account, and leaves the fixed-seed
  fixture byte-identical to before this flag existed. `generateFixtures`
  returns the count it created; `seedTesters` prints it (`entries=N`) on
  each user's summary line. A negative `--entries`, an unknown flag, or an
  extra positional is a usage error (exit `2`, nothing written); ~10k
  entries takes roughly 30s since each goes through `entry.Service.Create`
  (one insert + a cheap recompute lookup — there are no balance
  adjustments in the fixture).

- **It is a full, irreversible reset**, not scoped to the given users:
  every table except `schema_migrations` is `TRUNCATE`d
  (`postgres.ResetAll`/`postgres.TableCounts`, `internal/storage/postgres`)
  before anything is recreated. Bare `server seed`, or any invocation
  without `--yes` and an email list, only prints the table/row-count list
  via `TableCounts` and exits `2` — it never opens a write transaction.
  `--yes` is required precisely because this ships in the same binary as
  production and `admin`'s no-confirmation style isn't strict enough for a
  whole-database wipe.
- Users are created, in the given order, directly via
  `AuthStore.CreateUserWithIdentity` (pre-verified email, no magic-link
  round-trip) — the same mechanism `internal/cli`'s own tests already used.
  Because the reset just emptied `users`, the first given email lands as
  the bootstrap admin through the ordinary zero-users-means-admin path.
- Creating each user this way bypasses `auth.Service.resolveIdentity`
  entirely, so `internal/auth`'s `NewUserHook`s (see "Account types" above)
  never fire — `internal/cli` already imports `internal/category` directly
  (the same way `main.go`'s builder does), so `Seed` calls each user's
  `categorySvc.SeedDefaults` itself instead of relying on that indirection,
  which exists only so `internal/auth` — which cannot import `internal/
  category` — can still notify it. (Account types are no longer seeded: a
  type is just a text label written on each generated account.)
- `auth.Service.IssueSession(ctx, userID)` mints a session directly,
  without a sign-in flow — added for this command, never exposed over
  HTTP. It reuses the same TTL-driven expiry every other session gets, so
  it needs `auth.Params.SessionTTL`/`SessionMaxTTL` populated from real
  config, not a zero-value `Params{}` (a zero `SessionTTL` mints a session
  whose `ExpiresAt` is already in the past).
- Fixture shape (`internal/cli/fixtures.go`): a fixed pool of tags
  (`tagNamePool`) created once per user; 2-4 accounts per user (random
  `type` label from `fixtureAccountTypes` + currency + a random
  `financial_institute` from `financialInstitutePool`), 15-60 transaction
  entries per account (random
  seeded category, amount sign/magnitude keyed to category — `Salary`
  positive, everything else negative — and 0-2 random tags from that
  user's pool) — or exactly `--entries N` of them, spread across the
  accounts, when that flag is given. A single fixed-seed `math/rand/v2`
  generator drives the whole run, so a given argument set produces the
  same fixture every time; categories are sorted locally before being
  indexed by the RNG, since a `Store` is only contracted to return "every
  category," not in a particular order — the account-type labels
  (`fixtureAccountTypes`) and tags don't need this, being fixed local
  slices in a stable order rather than read back via `List`.

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

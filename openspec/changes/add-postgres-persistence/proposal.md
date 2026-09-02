## Why

The backend has no persistence. `internal/storage/memory` is the only store
home, `backend/AGENTS.md` names a `sqlite` sibling as the anticipated real
store, and every product feature past the health check is blocked on a durable
database. The first such feature — authentication — needs users, identities,
sessions, and tokens that survive a restart.

We are choosing **PostgreSQL, not SQLite**, and dropping the "standard library
only" constraint. Postgres is picked for headroom the app will want as it grows
past a single-file toy: real concurrency, `timestamptz`/`uuid`/`citext`,
partial indexes, extensions, replication, and a clean path to a hosted
multi-family deployment. The `Store` interface pattern stays, so a second
backend remains an additive change if it is ever needed — but we are not
building the multi-database abstraction now.

## What Changes

- **BREAKING** (docs/policy, no released behavior yet): retire the "standard
  library only" rule in `backend/AGENTS.md`. The stance becomes: **no web
  framework, no router library, no ORM**, but vetted third-party dependencies
  are allowed, each added through an OpenSpec proposal that justifies it.
- Add `internal/storage/postgres/`: a `pgxpool`-backed connection pool built
  from `DATABASE_URL`, its lifecycle owned by `main.go` (open on startup, close
  on shutdown). The package ships essentially empty of `Store` implementations
  — like `internal/storage/memory` today — because no domain package has
  declared a `Store` yet. The first one lands with the authentication change.
- Add a migration runner: `.sql` files embedded in the binary, applied in
  filename order at startup inside a transaction, tracked in a
  `schema_migrations` table. Hand-rolled, no migration-tool dependency.
- Add `DATABASE_URL` to `internal/config` (`config.Config`), documented in
  `backend/.env.example`. No default — the backend fails fast at startup if it
  is unset or unreachable.
- Add `github.com/jackc/pgx/v5` as the sole new dependency. It is pure Go, so
  the `CGO_ENABLED=0` distroless build is unchanged.
- Add a root `compose.yaml`: the app image plus a `postgres:17` service and a
  named volume for its data. This is both the local-development database and
  the reference production topology (previously "single image + a volume").
- Extend CI (`.github/workflows/ci.yml`): a `postgres` service container for
  the backend job so `internal/storage/postgres` integration tests run against
  a real database. Those tests skip when `DATABASE_URL` is unset so
  `go test ./...` still passes on a bare checkout.
- Add a DB ping to `GET /api/healthz` so the health check fails when the
  database is down.
- Update docs: `backend/AGENTS.md` (Stack, layout diagram `storage/sqlite/` →
  `storage/postgres/`, Commands, dependency policy), root `AGENTS.md`
  (Architecture now names an external Postgres alongside the single app image),
  `README.md`, `openspec/config.yaml`.

## Capabilities

### New Capabilities

- `backend-persistence`: PostgreSQL as the backend's database — the
  `pgxpool` connection built from `DATABASE_URL` and its startup/shutdown
  lifecycle, the embedded `.sql` migration runner and `schema_migrations`
  ledger, `internal/storage/postgres/` as the real `Store` home, the
  `GET /api/healthz` database probe, and the `compose.yaml` topology used for
  local dev, CI, and production.

### Modified Capabilities

- `backend-package-architecture`: the "In-memory storage is the default store
  home" requirement changes — the real-persistence sibling is
  `internal/storage/postgres/` (not `sqlite/`), and the dependency rule shifts
  from "standard library only, DB driver the sole exception" to "no
  framework/router/ORM; vetted dependencies allowed via proposal".

## Impact

- `backend/main.go` — build the `pgxpool` from `config.DATABASE_URL`, run
  migrations, inject the pool, close it on shutdown.
- `backend/internal/config/` — `DATABASE_URL` field + load; test for
  required-value behavior.
- New: `backend/internal/storage/postgres/` (pool constructor, migration
  runner, embedded `migrations/*.sql`, integration tests), root `compose.yaml`.
- `backend/internal/httpapi/health.go` — health check gains a pool `Ping`.
- `backend/go.mod` / `go.sum` — add `github.com/jackc/pgx/v5`.
- `.github/workflows/ci.yml` — `postgres` service container + `DATABASE_URL`
  for the backend job.
- `backend/.env.example` — `DATABASE_URL`.
- Docs: `backend/AGENTS.md`, `backend/README.md`, root `AGENTS.md`, root
  `README.md`, `openspec/config.yaml`.
- No change to the frontend, the `Dockerfile` build stages, or existing HTTP
  behavior beyond the health check.

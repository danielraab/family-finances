# backend-persistence Specification

## Purpose
TBD - created by archiving change add-postgres-persistence. Update Purpose after archive.
## Requirements
### Requirement: PostgreSQL is the backend database

The Go backend SHALL use PostgreSQL as its only database. Persistence SHALL be
reached through a `github.com/jackc/pgx/v5` `pgxpool.Pool` constructed once at
startup from configuration and shared by every `Store` implementation. No other
database engine SHALL be supported, and no abstraction over multiple engines
SHALL be built; portability across SQL dialects is explicitly a non-goal.

The `pgx` dependency SHALL be pure Go so that the release image continues to
build with `CGO_ENABLED=0`.

#### Scenario: Pool built at startup and closed at shutdown

- **WHEN** the backend process starts
- **THEN** `main.go` constructs one `pgxpool.Pool` from `Config.DatabaseURL`,
  verifies connectivity before serving traffic, and injects the pool into the
  storage layer
- **AND** on `SIGINT`/`SIGTERM` the pool is closed as part of graceful
  shutdown

#### Scenario: Build stays CGO-free

- **WHEN** the backend is built with `CGO_ENABLED=0`
- **THEN** the build succeeds and the binary runs on the distroless runtime
  image

### Requirement: DATABASE_URL is required configuration

`internal/config` SHALL expose the database connection string as
`Config.DatabaseURL`, populated from the `DATABASE_URL` environment variable
and documented in `backend/.env.example`. It SHALL have no default. If
`DATABASE_URL` is unset or empty, or the database is unreachable at startup,
the process SHALL log a clear error and exit non-zero rather than start.

`os.Getenv` for `DATABASE_URL` SHALL occur only within `internal/config`.

#### Scenario: Missing DATABASE_URL fails fast

- **WHEN** the backend starts with `DATABASE_URL` unset
- **THEN** it logs an error naming the missing variable and exits non-zero
- **AND** no HTTP listener is opened

#### Scenario: Unreachable database fails fast

- **WHEN** `DATABASE_URL` is set but no database accepts the connection
- **THEN** startup fails with a logged error and a non-zero exit

### Requirement: Schema migrations are embedded and applied at startup

Database schema SHALL be defined by `.sql` files embedded in the binary via
`//go:embed` under `internal/storage/postgres/`. On startup, after the pool is
built and before HTTP traffic is served, the backend SHALL apply every
migration not yet recorded, in ascending filename order, each within a single
transaction, and record applied migrations in a `schema_migrations` table. A
failing migration SHALL abort startup with a non-zero exit and SHALL NOT leave
a partially-applied migration committed.

Migrations SHALL be applied by hand-written code in `internal/storage/postgres`;
no third-party migration tool or library SHALL be added.

#### Scenario: Fresh database is migrated to current schema

- **WHEN** the backend starts against an empty database
- **THEN** every embedded migration is applied in filename order
- **AND** `schema_migrations` contains one row per applied file

#### Scenario: Already-migrated database starts clean

- **WHEN** the backend starts against a database whose `schema_migrations`
  already lists every embedded file
- **THEN** no migration is re-applied and startup proceeds to serving

#### Scenario: Failing migration aborts startup atomically

- **WHEN** an embedded migration raises an error mid-file
- **THEN** its transaction is rolled back, no `schema_migrations` row is
  written for it, and the process exits non-zero

### Requirement: internal/storage/postgres is the real Store home

`internal/storage/postgres/` SHALL be the location for PostgreSQL `Store`
implementations of the interfaces that domain packages declare. Until a domain
package declares a `Store`, the package MAY contain only the pool constructor
and migration runner. `main.go` SHALL inject the Postgres implementations for
production and MAY use `internal/storage/memory` for tests.

#### Scenario: Postgres storage package builds

- **WHEN** `go build ./...` runs in `backend/`
- **THEN** `internal/storage/postgres` compiles

#### Scenario: main injects Postgres stores in production

- **WHEN** the production binary starts
- **THEN** each domain service is constructed with its
  `internal/storage/postgres` `Store`, not the in-memory one

### Requirement: Health check verifies database connectivity

`GET /api/healthz` SHALL report unhealthy when the database is unreachable. It
SHALL acquire and release a pooled connection (or issue `Pool.Ping`) with a
short timeout as part of answering the request, returning `200` with body `ok`
only when that succeeds and a `503` otherwise.

#### Scenario: Healthy database

- **WHEN** the database is reachable and `GET /api/healthz` is requested
- **THEN** the response is `200` with body `ok`

#### Scenario: Database down

- **WHEN** the database is unreachable and `GET /api/healthz` is requested
- **THEN** the response status is `503`

### Requirement: Compose topology for local dev, CI, and production

The repository root SHALL contain a `compose.yaml` defining the backend app
service and a `postgres` service with a named volume for its data, wired
together by `DATABASE_URL`. This file SHALL be the documented way to run the
stack locally and SHALL serve as the reference production topology. Backend
documentation SHALL describe starting the database via compose before running
`go run .` for local development.

#### Scenario: Compose brings up a working stack

- **WHEN** a developer runs the documented compose command from the repo root
- **THEN** the backend starts, connects to the `postgres` service, applies
  migrations, and serves `GET /api/healthz` as `200 ok`

#### Scenario: Postgres data survives a restart

- **WHEN** the compose stack is stopped and started again
- **THEN** the `postgres` service retains its data through the named volume

### Requirement: Integration tests run against real Postgres and skip without it

`internal/storage/postgres` tests SHALL exercise a real PostgreSQL database
addressed by `DATABASE_URL`. When `DATABASE_URL` is unset, those tests SHALL
skip (not fail) so that `go test ./...` passes on a bare checkout. CI SHALL
provide a `postgres` service container and set `DATABASE_URL` for the backend
job so the integration tests execute on every push and pull request.

#### Scenario: Bare checkout test run

- **WHEN** `go test ./...` runs in `backend/` with `DATABASE_URL` unset
- **THEN** the suite passes and the Postgres integration tests report as
  skipped

#### Scenario: CI runs the integration tests

- **WHEN** the CI backend job runs
- **THEN** a `postgres` service container is available, `DATABASE_URL` points
  at it, and the `internal/storage/postgres` tests execute rather than skip


## 1. Dependency

- [x] 1.1 Add `github.com/jackc/pgx/v5` to `backend/go.mod`; run `go mod tidy`
  and confirm `go.sum` updates and no cgo is pulled in.
- [x] 1.2 Confirm `CGO_ENABLED=0 go build .` in `backend/` still succeeds.

## 2. Config

- [x] 2.1 Add `DatabaseURL string` to `config.Config` in
  `backend/internal/config/config.go`, populated from `os.Getenv("DATABASE_URL")`
  with no default.
- [x] 2.2 In `config_test.go`, assert `DatabaseURL` is empty when the env var
  is unset and echoes the env var when set.
- [x] 2.3 Add `DATABASE_URL` to `backend/.env.example` with a
  `postgres://user:pass@localhost:5432/family_finances?sslmode=disable` sample
  and a comment that it is required (no default).

## 3. internal/storage/postgres — pool

- [x] 3.1 Create `backend/internal/storage/postgres/postgres.go` with
  `NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error)`
  that builds the pool and `Ping`s it with a ~5s timeout, returning a wrapped
  error on failure.
- [x] 3.2 Add a package doc comment describing this as the real `Store` home,
  empty of `Store` implementations until a domain package declares one.

## 4. internal/storage/postgres — migration runner

- [x] 4.1 Create `backend/internal/storage/postgres/migrations/0001_init.sql`
  (enable the `citext` extension; no product tables yet).
- [x] 4.2 Create `migrate.go` with `//go:embed migrations/*.sql` and
  `Migrate(ctx, pool) error`: create `schema_migrations (version text primary
  key, applied_at timestamptz not null default now())` if absent; read applied
  versions; for each embedded file whose version is not applied, in ascending
  filename order, run the file body and insert its `schema_migrations` row in a
  single transaction; return on first error.
- [x] 4.3 Parse the version from the filename prefix (`0001` from
  `0001_init.sql`); fail loudly on a malformed name or a duplicate version.

## 5. internal/storage/postgres — tests

- [x] 5.1 Add `TestMain` (or a helper) that reads `DATABASE_URL`, `t.Skip`s the
  package when unset, and otherwise creates a uniquely-named throwaway database
  (or isolated schema) per run and drops it on cleanup.
- [x] 5.2 Test: `Migrate` against a fresh database applies every embedded file
  in order and writes one `schema_migrations` row per file.
- [x] 5.3 Test: `Migrate` run twice is idempotent — the second run applies
  nothing.
- [x] 5.4 Test: a deliberately broken migration file causes `Migrate` to return
  an error, writes no `schema_migrations` row for it, and leaves no partial DDL
  committed.
- [x] 5.5 Test: `NewPool` returns an error for an unreachable `DATABASE_URL`.

## 6. Wire main.go

- [x] 6.1 In `backend/main.go`, after `config.Load()`: build the pool via
  `postgres.NewPool`; on error log and `os.Exit(1)` before opening the
  listener.
- [x] 6.2 Call `postgres.Migrate(ctx, pool)`; on error log and `os.Exit(1)`.
- [x] 6.3 `defer pool.Close()` and ensure it runs as part of graceful shutdown.
- [x] 6.4 Pass a DB pinger into `httpapi.New` (see task 7).

## 7. Health check probe

- [x] 7.1 Add a `DBPinger` interface (`Ping(context.Context) error`) to
  `httpapi.Deps` in `backend/internal/httpapi/server.go`.
- [x] 7.2 Update `handleHealthz` in `internal/httpapi/health.go` to call
  `Ping` with a ~2s context: `200` `ok` on success, `503` on failure.
- [x] 7.3 Update `health` tests: healthy pinger → `200 ok`; failing pinger →
  `503`. Use a stub pinger, no real DB.
- [x] 7.4 Confirm `main.go` passes `pool` (which satisfies `DBPinger`) into
  `httpapi.New`.

## 8. Compose

- [x] 8.1 Create root `compose.yaml`: `db` service (`postgres:17`, `POSTGRES_*`
  env, named volume `pgdata`, a `pg_isready` healthcheck) and `app` service
  (built from the root `Dockerfile`, `DATABASE_URL` pointing at `db`,
  `depends_on: db: condition: service_healthy`, port mapping).
- [x] 8.2 Verify `docker compose up` brings the stack up and
  `curl localhost:PORT/api/healthz` returns `200 ok`.
- [x] 8.3 Verify `docker compose down && docker compose up` retains data via
  the `pgdata` volume.

## 9. CI

- [x] 9.1 In `.github/workflows/ci.yml`, add a `postgres:17` service container
  to the backend job with health options and credentials.
- [x] 9.2 Set `DATABASE_URL` in the backend job env to point at the service
  container.
- [x] 9.3 Confirm the `internal/storage/postgres` tests execute (not skip) in
  CI and that a bare `go test ./...` locally without `DATABASE_URL` still
  passes with those tests skipped.

## 10. Docs

- [x] 10.1 `backend/AGENTS.md` §Stack: remove "standard library only" and the
  `modernc.org/sqlite` exception line; state "no web framework, no router, no
  ORM; vetted dependencies via OpenSpec proposal"; name `pgx/v5` and
  `DATABASE_URL`.
- [x] 10.2 `backend/AGENTS.md` package-layout diagram: `storage/sqlite/` →
  `storage/postgres/`; note the migration runner and embedded `migrations/`.
- [x] 10.3 `backend/AGENTS.md` §Commands: `docker compose up -d db`, export
  `DATABASE_URL`, then `go run .`.
- [x] 10.4 root `AGENTS.md` §Architecture: the single app image now runs
  alongside an external Postgres (`compose.yaml` is the reference topology).
- [x] 10.5 `backend/README.md` and root `README.md`: quick-start now includes
  Postgres via compose.
- [x] 10.6 `openspec/config.yaml` context block: "no database yet" →
  "PostgreSQL via `pgx/v5`".

## 11. Verify

- [x] 11.1 `gofmt -l backend/` prints nothing; `go vet ./...` clean in
  `backend/`.
- [x] 11.2 `go test ./...` passes in `backend/` both with and without
  `DATABASE_URL` set.
- [x] 11.3 `grep -rn "os.Getenv" backend/` still shows only
  `internal/config`.
- [x] 11.4 `grep -rn "standard library only" .` returns nothing outside
  `openspec/changes/archive/`.

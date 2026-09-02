## Context

The backend is stateless today. `internal/storage/memory` is a documented
placeholder, `backend/AGENTS.md` §Stack says "standard library only" with a DB
*driver* named as the one anticipated exception, and the
`backend-package-architecture` spec says a `sqlite` sibling "MAY be added later
through its own change". Authentication is the first feature that needs durable
state, and it is blocked on this.

Constraints that shape the design:

- `main.go` is wiring only; `os.Getenv` lives only in `internal/config`;
  domain packages import no storage package and no driver (the `Store`
  interface is declared by the domain package, implemented in `storage/*`,
  injected by `main`).
- The release artifact is one distroless image built `CGO_ENABLED=0`. Anything
  that forces cgo is off the table.
- CI lints/tests both packages on every push/PR and must keep passing on a
  bare checkout with no database present.

The engine decision (Postgres, drop SQLite, keep the `Store` seam) was settled
during exploration: Postgres for concurrency, `timestamptz`/`uuid`/`citext`,
partial indexes, extensions, and a credible path to a hosted multi-family
deployment. SQLite's single-writer model and typeless columns are a poor
foundation for that even though the current workload would fit it.

## Goals / Non-Goals

**Goals:**

- One `pgxpool.Pool` built at startup from `DATABASE_URL`, injected into the
  storage layer, closed on shutdown.
- An embedded, hand-written migration runner: `.sql` files applied in order at
  startup, tracked in `schema_migrations`, transactional per file.
- `internal/storage/postgres/` as the real `Store` home (empty of `Store`
  implementations until authentication declares the first interface).
- A root `compose.yaml` that is simultaneously the local-dev database, the CI
  topology, and the reference production topology.
- `GET /api/healthz` fails when the database is down.
- CI runs `internal/storage/postgres` integration tests against a real
  Postgres; the same tests skip on a bare checkout.
- Retire "standard library only" in the docs; replace it with "no
  framework/router/ORM, vetted deps via proposal".

**Non-Goals:**

- Any domain table or `Store` implementation — those come with the
  authentication change. This change may land with zero product tables beyond
  `schema_migrations`.
- A multi-engine abstraction, SQL-dialect portability, or keeping SQLite as an
  option.
- A migration CLI, down-migrations, or a third-party migration tool.
- Read replicas, connection-pool tuning knobs, `LISTEN/NOTIFY`, or
  observability beyond the existing slog request line and the health probe.
- Changing the `Dockerfile` build stages or the GHCR publish step.

## Decisions

### D1: PostgreSQL only; keep the `Store` interface as the seam

Commit to Postgres. Do not implement a second backend or an abstraction that
would allow one. The `Store` interface already isolates domain/service code
from the database, so adding `internal/storage/postgres2` or similar later
would be additive if it were ever needed — but designing for it now would force
lowest-common-denominator schemas (no `timestamptz`, no partial indexes, `?`
vs `$1` placeholder shims) on every future feature.

*Alternatives considered:* SQLite-only (simpler ops, single file, but the
single-writer model and lack of real types/`timestamptz` are a bad base for
where this is heading); multi-engine (both SQLite and Postgres selectable —
rejected: doubles every `Store` implementation, doubles CI, caps the schema at
SQLite's feature set, for a second backend we do not expect to ship).

### D2: `github.com/jackc/pgx/v5` with the native `pgxpool`, not `database/sql`

Use pgx's native interface (`pgxpool.Pool`, `pgx.Rows`, native scanning) rather
than the `database/sql` adapter. Portability to `database/sql` was the only
reason to prefer the adapter, and D1 removes that reason. Native pgx gives
built-in pooling, native decoding of `uuid`/`timestamptz`/`inet`/arrays/`jsonb`,
and `LISTEN/NOTIFY` if a later feature wants it. pgx is pure Go, so
`CGO_ENABLED=0` is unaffected.

*Alternatives considered:* `database/sql` + `pgx/v5/stdlib` (keeps a familiar
API and a theoretical swap path — not worth the lost native types now);
`lib/pq` (in maintenance mode, discouraged upstream).

This is the **only** new module. `backend/AGENTS.md` §Stack changes from
"standard library only" to: no web framework, no router library, no ORM; vetted
third-party dependencies allowed, each via an OpenSpec proposal.

### D3: Hand-written migration runner, `.sql` embedded via `go:embed`

`internal/storage/postgres/migrations/NNNN_slug.sql` (zero-padded, e.g.
`0001_init.sql`), embedded with `//go:embed migrations/*.sql`. At startup:
`CREATE TABLE IF NOT EXISTS schema_migrations (version text primary key,
applied_at timestamptz not null default now())`; read applied versions; for
each embedded file whose version is absent, in ascending order, run the file
body and insert its `schema_migrations` row **in one transaction**; abort
startup non-zero on the first failure. Runs before the HTTP listener opens.

Roughly 60 lines. A finance app of this size will accumulate on the order of
tens of migrations over its life; `goose`/`golang-migrate`/`tern` each bring a
dependency and a CLI surface for functionality this loop already covers. Matches
the new "keep the dependency set small" ethos. No down-migrations: forward-only
is simpler and safe to operate; a bad migration is fixed by a new one.

*Alternatives considered:* `pressly/goose` as a library (popular, embeds `.sql`,
but a dependency for ~60 lines of value); `jackc/tern` (pgx-family, Postgres-only
— viable, still a dependency + config file); running DDL from application code
on demand (loses the ordered, recorded ledger).

### D4: `DATABASE_URL`, required, fail-fast

Single connection string on `Config.DatabaseURL` from `DATABASE_URL`, no
default, documented in `.env.example`. `main.go` builds the pool, then
`pool.Ping(ctx)` with a short timeout (≈5s); either failure logs a clear
message and exits non-zero before the listener opens. A backend that cannot
reach its database should not accept traffic, and a missing-config crash at
boot is easier to diagnose than lazy per-request failures.

Pool sizing stays on pgxpool defaults; a `DATABASE_URL` can carry `pool_max_conns`
etc. if an operator needs it, so no extra env var now.

### D5: `compose.yaml` at the repo root is the one topology

One file: `app` (built from the existing root `Dockerfile`) + `db`
(`postgres:17`, named volume `pgdata`, `POSTGRES_*` env), `app.DATABASE_URL`
pointing at `db`, `app` `depends_on` `db` with a healthcheck condition. Used
for local dev (`docker compose up db` for just the database while running
`go run .` on the host, or the whole stack), mirrored by CI's service
container, and handed to operators as the production reference. One definition,
no drift.

`backend/AGENTS.md` §Commands gains: start the database first
(`docker compose up -d db`), export `DATABASE_URL`, then `go run .`. Root
`AGENTS.md` §Architecture notes that the single app image now runs alongside a
Postgres it does not contain — the "single Docker image" statement stays true
for the app, with persistence as a named external dependency.

### D6: Health check acquires a real connection

`GET /api/healthz` calls `pool.Ping(ctx)` (context deadline ≈2s) as part of
handling the request: success → `200 ok` (unchanged body), failure → `503`.
This makes the Docker `HEALTHCHECK` and any orchestrator readiness probe
reflect database reachability, not just process liveness. The handler needs the
pool, so `internal/httpapi` gains a dependency in `Deps` (a small
`DBPinger` interface — `Ping(context.Context) error` — so `httpapi` does not
import pgx). No separate `/api/readyz`: one probe is enough at this scale.

### D7: Integration tests skip without `DATABASE_URL`

`internal/storage/postgres` tests read `DATABASE_URL`; if empty, `t.Skip`. Each
test (or `TestMain`) provisions an isolated schema or a uniquely-named
throwaway database and runs the migration runner against it, so runs are
hermetic and parallel-safe. CI's backend job gets a `postgres` service
container and sets `DATABASE_URL`. No `testcontainers-go`: it is a heavy
test-only dependency tree, and a service container (CI) plus
`docker compose up db` (local) covers the same ground with zero deps.

*Alternatives considered:* `testcontainers-go` (nicer DX — ephemeral PG per run
with no external setup — but a large dependency and needs a Docker daemon in
the test environment anyway); an in-process Postgres (`embedded-postgres`
downloads a binary — fragile, still a dep).

## Risks / Trade-offs

- **Local dev now needs a running Postgres** → `compose.yaml` with a one-line
  `docker compose up -d db`; `storage/memory` still backs all domain/service
  unit tests, so only `storage/postgres` tests and a real run need the DB.
- **Hand-rolled migration runner has no down-migrations and no CLI** →
  forward-only is a deliberate operational choice; a bad migration is corrected
  by a follow-up migration. The runner is small and covered by its own tests
  (fresh DB, idempotent re-run, mid-file failure rolls back).
- **Fail-fast on an unreachable DB means a Postgres blip during a deploy can
  crash-loop the app** → acceptable and standard; the orchestrator restarts it,
  and `compose` `depends_on: condition: service_healthy` sequences the common
  case. Startup retry/backoff can be added later if it proves necessary.
- **`pgx` is a new supply-chain surface** → it is the de-facto standard Go
  Postgres driver, widely used and audited; it is pure Go (no cgo); it is the
  single dependency this change adds, gated by this proposal.
- **`internal/httpapi` gains a DB dependency for the health probe** → contained
  behind a one-method `DBPinger` interface in `Deps`; `httpapi` still does not
  import `pgx`.
- **Policy change ("standard library only" retired)** → deliberate and
  documented in the `backend-package-architecture` delta and the AGENTS.md
  edits; the "no framework/router/ORM" half of the rule is kept and
  strengthened.

## Migration Plan

1. Add `pgx/v5` to `go.mod`; `go mod tidy`.
2. Land `internal/storage/postgres`: pool constructor, migration runner,
   `migrations/0001_init.sql` (just `schema_migrations` is created by code; the
   first real migration can be empty or a `citext` extension enable if desired),
   runner tests.
3. Add `DATABASE_URL` to `internal/config` + `.env.example` + config test.
4. Wire `main.go`: build pool → run migrations → inject → `defer pool.Close()`.
5. Add the `DBPinger` to `httpapi.Deps` and the ping to `handleHealthz`;
   update health tests.
6. Add root `compose.yaml`.
7. Extend `.github/workflows/ci.yml` backend job with the `postgres` service +
   `DATABASE_URL`.
8. Doc edits: `backend/AGENTS.md`, `backend/README.md`, root `AGENTS.md`, root
   `README.md`, `openspec/config.yaml`.

**Rollback:** nothing in production consumes persistence yet, so reverting is a
straight `git revert` of the change plus removing the `pgx` require. No data
migration to unwind.

## Open Questions

_None._ Engine (Postgres), driver (`pgx/v5` native), migration approach
(hand-rolled embedded `.sql`, forward-only), test strategy (skip without
`DATABASE_URL`, CI service container), and PK convention (`uuid` via
`gen_random_uuid()`, established by the first real table in the authentication
change) were all settled during exploration.

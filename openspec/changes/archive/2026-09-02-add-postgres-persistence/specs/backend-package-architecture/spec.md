## MODIFIED Requirements

### Requirement: In-memory storage is the default store home

`internal/storage/memory/` SHALL be the location for in-memory `Store`
implementations, used as the default for local development and for tests.
`internal/storage/postgres/` SHALL be the location for the real,
PostgreSQL-backed `Store` implementations, injected by `main.go` for
production. No third store engine SHALL be added and no cross-engine
abstraction SHALL be built.

Third-party dependencies are permitted where they earn their place: the backend
SHALL use **no web framework, no router library, and no ORM**, but vetted
libraries (for example the PostgreSQL driver `github.com/jackc/pgx/v5`) MAY be
added, each through its own OpenSpec proposal that justifies it. Domain
packages SHALL still import no storage package and no database driver.

#### Scenario: Memory storage package exists and builds

- **WHEN** `go build ./...` runs in `backend/`
- **THEN** `internal/storage/memory` compiles

#### Scenario: Postgres storage package exists and builds

- **WHEN** `go build ./...` runs in `backend/`
- **THEN** `internal/storage/postgres` compiles

#### Scenario: Domain package still imports no storage or driver

- **WHEN** the import graph of any `internal/<noun>/` domain package is
  inspected
- **THEN** it imports neither `internal/storage/...` nor `github.com/jackc/pgx/...`

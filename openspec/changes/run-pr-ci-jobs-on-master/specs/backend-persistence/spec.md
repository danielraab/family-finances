# Spec Delta

## MODIFIED Requirements

### Requirement: Integration tests run against real Postgres and skip without it

`internal/storage/postgres` tests SHALL exercise a real PostgreSQL database
addressed by `DATABASE_URL`. When `DATABASE_URL` is unset, those tests SHALL
skip (not fail) so that `go test ./...` passes on a bare checkout. CI SHALL
provide a `postgres` service container and set `DATABASE_URL` so the integration
tests execute on every pull request and every push to `master`.

#### Scenario: Bare checkout test run
- **WHEN** `go test ./...` runs in `backend/` with `DATABASE_URL` unset
- **THEN** the suite passes and the Postgres integration tests report as skipped

#### Scenario: CI runs the integration tests
- **WHEN** CI runs for a pull request or a push to `master`
- **THEN** a `postgres` service container is available, `DATABASE_URL` points at it, and the integration tests execute rather than skip

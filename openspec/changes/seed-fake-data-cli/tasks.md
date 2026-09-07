## 1. Backend: `auth.Service.IssueSession`

- [ ] 1.1 `internal/auth/service.go`: add
  `IssueSession(ctx, userID string) (token string, err error)` — looks up
  the user via `store.UserByID`, then calls the existing (unexported)
  `issueSessionToken` with `SessionContext{Client: ClientAPI}`. Never
  routed through `internal/httpapi` — CLI-only.
- [ ] 1.2 Unit test: `IssueSession` returns a token `Authenticate` resolves
  back to the same user; unknown `userID` returns `ErrNotFound`.

## 2. Backend: `internal/cli` reset + user creation

- [ ] 2.1 `internal/cli/seed.go` (new): `Seed(ctx, args) int`, mirroring
  `Admin`'s shape — load `config`, build a `postgres.Pool`, run
  `postgres.Migrate`, then dispatch on `--yes`.
- [ ] 2.2 Bare `server seed` (no `--yes`): query
  `SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND
  tablename <> 'schema_migrations'` plus a row count per table, print the
  list to stdout, return a non-zero, non-1 exit code (`2`, matching the
  existing usage-error convention) without opening a write transaction.
- [ ] 2.3 `server seed --yes`: `TRUNCATE <every table from 2.2> RESTART
  IDENTITY CASCADE` in one transaction.
- [ ] 2.4 After truncation, build `postgres.AuthStore`, an `auth.Service`
  wired with `auth.WithNewUserHooks(accountSvc, categorySvc)` (built per
  task 3.1), and create `tester1@draab.at`, `tester2@draab.at`,
  `tester3@draab.at` in that order via `CreateUserWithIdentity` with a
  pre-verified email identity each (mirroring `cli_test.go`'s existing
  `seedUser` helper). Confirm `tester1@draab.at` lands as admin (empty
  `users` table ⇒ existing bootstrap path).
- [ ] 2.5 For each created user, call `svc.IssueSession` and collect
  `(email, isAdmin, token)`.

## 3. Backend: fixture generator

- [ ] 3.1 `internal/cli/fixtures.go` (new): build `account.Service`,
  `category.Service`, `entry.Service` over `postgres.NewAccountStore`,
  `postgres.NewCategoryStore`, `postgres.NewEntryStore` — the same
  construction `main.go`'s `buildAccount`/`buildCategory`/`buildEntry`
  already do.
- [ ] 3.2 A single `rand.New(rand.NewPCG(fixedSeed, fixedSeed))` (fixed
  constant) shared across the whole run — never the package-level
  `math/rand/v2` functions, which aren't reproducible.
- [ ] 3.3 `seedUserFixtures(ctx, ownerID string, accountSvc *account.Service,
  categorySvc *category.Service, entrySvc *entry.Service, rng *rand.Rand)
  error`:
  - List `ownerID`'s seeded account types (from the `NewUserHook`) and
    categories; pick 2–4 accounts, each a random type + a random currency
    from `{EUR, USD, GBP}` + a random `opening_date` within the past 2
    years, via `accountSvc.Create`.
  - Per account, create 15–60 transaction entries via `entrySvc.Create`:
    random `booking_timestamp` in the past 12 months, random category
    from the seeded set, amount sign/magnitude keyed to category (Salary
    positive/larger, everything else negative/smaller — see design.md),
    title from a small per-category string pool.
- [ ] 3.4 Unit tests (against `storage/memory`): the generator produces
  the expected account/entry count ranges; two runs with the same fixed
  seed produce identical output; category/amount-sign pairing holds
  (Salary entries are always positive, others always negative).

## 4. Wiring

- [ ] 4.1 `backend/main.go`: dispatch `os.Args[1] == "seed"` to
  `os.Exit(cli.Seed(ctx, os.Args[2:]))`, alongside the existing
  `healthcheck`/`admin` dispatch.
- [ ] 4.2 `internal/cli/seed.go`: after 2.5 and the fixtures in section 3
  complete, print one line per tester to stdout:
  `tester1@draab.at (admin) session=<token>` (and the same without
  `(admin)` for the other two).

## 5. Verify

- [ ] 5.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`.
- [ ] 5.2 `internal/storage/postgres` integration test (needs
  `DATABASE_URL`): seed unrelated rows into a throwaway database, run the
  `TRUNCATE` step directly, confirm every listed table is empty except
  `schema_migrations`.
- [ ] 5.3 Manual pass: `go run . seed` (bare) against a local Postgres —
  confirm it prints the table/row-count list and the database is
  unchanged; `go run . seed --yes` — confirm exactly three users exist,
  `tester1@draab.at` is admin, each has starter account types/categories
  plus generated accounts/entries; paste one printed token as
  `Authorization: Bearer` and confirm `GET /api/auth/me`,
  `GET /api/accounts`, `GET /api/entries` all return that tester's data;
  run `go run . seed --yes` a second time and confirm the same three
  users/fixture shape result (no duplicates, same counts).

## 6. Docs

- [ ] 6.1 `backend/AGENTS.md`: add a short "Seeding fake data" note
  (mirroring the existing `admin` CLI note) — command, what `--yes` does,
  that it's a full destructive reset, and that it's built on the same
  `NewUserHook`/`CreateUserWithIdentity` mechanisms `account-types-per-user`
  introduced.

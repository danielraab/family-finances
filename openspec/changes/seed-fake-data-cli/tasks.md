## 1. Backend: `auth.Service.IssueSession`

- [x] 1.1 `internal/auth/service.go`: add
  `IssueSession(ctx, userID string) (token string, err error)` — looks up
  the user via `store.UserByID`, then calls the existing (unexported)
  `issueSessionToken` with `SessionContext{Client: ClientAPI}`. Never
  routed through `internal/httpapi` — CLI-only.
- [x] 1.2 Unit test: `IssueSession` returns a token `Authenticate` resolves
  back to the same user; unknown `userID` returns `ErrNotFound`.

## 2. Backend: `internal/cli` reset + user creation

- [x] 2.1 `internal/cli/seed.go` (new): `Seed(ctx, args) int`, mirroring
  `Admin`'s shape — load `config`, build a `postgres.Pool`, run
  `postgres.Migrate`, then dispatch on `--yes`.
- [x] 2.2 Bare `server seed` (no `--yes`): `postgres.TableCounts` lists
  every data table + row count, printed to stdout; returns `2` without
  opening a write transaction.
- [x] 2.3 `server seed --yes`: `postgres.ResetAll` — `TRUNCATE` every data
  table (read at runtime via `pg_tables`, not hardcoded) in one
  transaction. Both `TableCounts`/`ResetAll` landed in
  `internal/storage/postgres/reset.go` rather than inline SQL in
  `internal/cli`, keeping SQL details in the package that already owns
  them.
- [x] 2.4 After truncation, build `postgres.AuthStore` and the three
  testers via `CreateUserWithIdentity` with a pre-verified email identity
  each (mirroring `cli_test.go`'s existing `seedUser` helper).
  `tester1@draab.at` lands as admin via the existing bootstrap path.
  **Revised from the original plan**: rather than wiring
  `auth.WithNewUserHooks` into `Seed`'s own `auth.Service` (which would
  never fire — `CreateUserWithIdentity` bypasses `resolveIdentity`
  entirely), `Seed` calls `accountSvc.SeedDefaults`/
  `categorySvc.SeedDefaults` directly for each tester, since
  `internal/cli` already imports both packages (see design.md).
- [x] 2.5 For each created user, call `svc.IssueSession` and print
  `(email, isAdmin, token)`.

## 3. Backend: fixture generator

- [x] 3.1 `internal/cli/fixtures.go` (new): build `account.Service`,
  `category.Service`, `entry.Service` over `postgres.NewAccountStore`,
  `postgres.NewCategoryStore`, `postgres.NewEntryStore` — the same
  construction `main.go`'s `buildAccount`/`buildCategory`/`buildEntry`
  already do.
- [x] 3.2 A single `rand.New(rand.NewPCG(fixedSeed, fixedSeed))` (fixed
  constants `seedRNGSeed1`/`seedRNGSeed2`) shared across the whole run.
- [x] 3.3 `generateFixtures`: lists `ownerID`'s seeded account types and
  categories (sorted locally by title/name before being indexed by the
  RNG — a `Store` only guarantees "every type/category," not a
  particular order, and `storage/memory`'s `ListTypes` ties on
  `CreatedAt` for a freshly seeded set, which surfaced as a real
  nondeterminism bug during testing); picks 2–4 accounts, each a random
  type + currency + `opening_date`, via `accountSvc.Create`; per account,
  15–60 transaction entries via `entrySvc.Create` with random category,
  category-shaped amount sign/magnitude, and a title from a small
  per-category pool.
- [x] 3.4 Unit tests (against `storage/memory`): account/entry counts
  land in range; two runs with the same fixed seed produce identical
  account titles; `Salary` entries are always positive, every other
  category always negative.

## 4. Wiring

- [x] 4.1 `backend/main.go`: dispatch `os.Args[1] == "seed"` to
  `os.Exit(cli.Seed(ctx, os.Args[2:]))`, alongside the existing
  `healthcheck`/`admin` dispatch.
- [x] 4.2 `internal/cli/seed.go` / `fixtures.go`: after user creation +
  fixture generation, print one line per tester to stdout:
  `tester1@draab.at (admin) session=<token>` (and the same without
  `(admin)` for the other two).

## 5. Verify

- [x] 5.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`.
- [x] 5.2 `internal/storage/postgres` integration test (needs
  `DATABASE_URL`): `TestResetAllTruncatesEveryDataTable` seeds unrelated
  rows into a throwaway database, runs `ResetAll`, confirms every listed
  table is empty; `TestTableCountsExcludesSchemaMigrations` confirms
  `schema_migrations` is never in the list.
- [x] 5.3 Manual pass (built binary + real Postgres): `go run . seed`
  (bare) printed the table/row-count list and left the database
  untouched; `go run . seed --yes` created exactly three users,
  `tester1@draab.at` admin, each with the full starter account
  types/categories plus 2–4 generated accounts and dozens of entries;
  pasted a printed token as `Authorization: Bearer` and confirmed
  `GET /api/auth/me`, `GET /api/accounts`, `GET /api/entries` all
  returned that tester's own data, correctly isolated from the other two;
  confirmed `Salary` entries were positive and every other category's
  entries negative. **This pass caught a real bug** — see task 1's
  `IssueSession` note and design.md — fixed and re-verified against a
  freshly rebuilt binary before considering this task done.

## 6. Docs

- [x] 6.1 `backend/AGENTS.md`: added a "Seeding fake data" section
  (mirroring the existing `admin` CLI note) — the command, what `--yes`
  does, the direct-`SeedDefaults`-call decision (and why `NewUserHook`
  doesn't apply here), the `IssueSession`/`SessionTTL` gotcha, and the
  fixture shape.

## 7. Follow-up: emails come from the CLI, not hardcoded

- [x] 7.1 `internal/cli/seed.go`: replaced the hardcoded
  `seedEmails = []string{"tester1@draab.at", ...}` with a required
  comma-separated argument after `--yes` (`server seed --yes
  <email1,email2,...>`); added `parseSeedEmails` — splits on `,`,
  `auth.NormalizeEmail`s and `auth.ValidateEmail`s each, rejects an empty
  list or a duplicate — called before `postgres.ResetAll`, so an invalid
  invocation never touches the database. Updated `seedUsage` and
  `printResetPlan`'s copy to match; any invocation other than exactly
  `--yes` plus a valid email list still falls through to the inert,
  table-listing path.
- [x] 7.2 `internal/cli/fixtures.go`: `seedTesters` takes `emails []string`
  as a parameter instead of reading the package-level var.
- [x] 7.3 Tests: `parseSeedEmails` — trims/normalizes/validates, rejects
  empty/invalid/duplicate; existing `seedTesters`/`generateFixtures` tests
  updated to pass an explicit email slice.
- [x] 7.4 `backend/AGENTS.md` and this change's `proposal.md`/`design.md`
  updated to describe CLI-supplied emails instead of the hardcoded three.
- [x] 7.5 `cd backend && gofmt -l . && go vet ./... && go test ./...`;
  manual pass against a real Postgres — bare `--yes` with no emails stays
  inert; an invalid email in the list is rejected before the reset runs
  (confirmed via direct DB query that nothing was touched); a valid
  two-email list (`alice@example.com, Bob@Example.com`) resets, creates
  both users (first as admin, email case-normalized), and each printed
  token authenticates against a running server.

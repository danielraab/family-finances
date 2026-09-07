## Context

`internal/cli.Admin` is the template: it loads `config`, builds a
`postgres.Pool`, runs `postgres.Migrate`, builds a `postgres.AuthStore`
directly (not through `main.go`), and dispatches on `args[0]` — returning a
process exit code, writing to injected `stdout`/`stderr` so `cli_test.go`
can assert on output without a real process. `internal/cli` already imports
`internal/auth` (a domain package) directly rather than going through
`main.go`'s composition — the precedent this change extends to
`internal/account`, `internal/category`, `internal/entry`.

`account-types-per-user` added `auth.NewUserHook` and
`auth.WithNewUserHooks`, invoked once inside `resolveIdentity`'s
`ActionCreate` branch right after `Store.CreateUserWithIdentity` succeeds.
Whatever builds the `auth.Service` this command uses needs the same hooks
wired in, or the three testers would get empty account-type/category lists
same as before that change shipped.

Every table this reset needs to clear:

```
users, identities, sessions, magic_link_tokens, invites, oidc_login_state,
user_settings, account_types, accounts, categories, tags, entries,
entry_tags
```

`schema_migrations` (created by `Migrate`, not a numbered migration) is
deliberately excluded — this is a data reset, not a request to re-run
migrations from an empty tracking table.

## Goals / Non-Goals

**Goals:**

- One command gets a local (or any `DATABASE_URL`-pointed) instance from
  empty to "three users, each with realistic-looking accounts and months
  of entries" — no manual clicking, no SMTP required to log in afterward.
- The reset is a genuine reset: every table above ends up empty before the
  three testers and their fixtures are created, not an additive top-up.
- Running it twice produces the same fixture, not accumulating duplicates
  or drifting output.
- The blast radius (full-table truncate) is never one flag or one
  keystroke away from firing unintentionally.

**Non-Goals:**

- Configurable volume/shape via flags (`--accounts=N`, `--months=N`, a
  `--seed=` RNG override). Fixed constants are enough for what this is —
  add flags later if a fixed shape turns out to be too rigid.
- Distinct scenarios per tester (a "simple" user vs a "heavy volume" user
  vs a "multi-currency" user) — explicitly declined during exploration in
  favor of the same random generator for all three.
- `balance_adjustment` entries, tags on entries, or disabled/closed
  accounts in the generated fixtures — every generated account stays open
  and enabled, every generated entry is a plain transaction. Keeps the
  generator simple; nothing about the seeded data needs to exercise those
  paths specifically.
- Any protection against pointing this at a real production
  `DATABASE_URL` beyond the `--yes` confirmation (no hostname allow-list,
  no "are you sure" prompt reading from stdin). `admin grant/revoke/list`
  carries the same trust level today — whoever has `DATABASE_URL` and a
  shell already has full read/write access to every table this would
  truncate.

## Decisions

### Decision: `TRUNCATE ... CASCADE`, not drop/recreate the database

The reset is `TRUNCATE <every table above> CASCADE` inside one transaction
(`RESTART IDENTITY` included, harmless even though every id here is a
`uuid` default, not a sequence — cheap insurance against a future table
that does use one). This only needs ordinary DML privileges the app's own
`DATABASE_URL` role already has to function at all — no `CREATEDB` or
superuser grant, so it works identically against the docker-compose
Postgres and a hosted one. The table list is read at runtime via
`SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND
tablename <> 'schema_migrations'` rather than hardcoded — a future
migration that adds a table is picked up automatically instead of silently
left out of the reset.

### Decision: bare `server seed` is inert; `--yes <emails>` is required to actually run

`server seed` with no args, or anything other than exactly `--yes`
followed by a comma-separated email list, queries and prints the table
list (and current row counts, so it's obvious how much would be lost) and
exits `2` without opening a transaction — this includes `--yes` with no
email argument, or one that fails `parseSeedEmails` (empty, malformed, a
duplicate). `server seed --yes <emails>` performs the reset and seeding.
This is a heavier confirmation than `admin grant/revoke/list` requires,
deliberately: those are narrow, reversible, single-row edits; this is a
total, irreversible wipe of every table in whatever database
`DATABASE_URL` points at, shipped in the same binary that runs in
production. No env-based or hostname-based guard is added on top (see
Non-Goals) — `--yes` plus a valid email list is the one deliberate step
between "ran the binary" and "the database is empty."

**Revised after the first implementation**: emails were originally
hardcoded to `tester1/2/3@draab.at`. They're now a required, CLI-supplied,
comma-separated argument — `parseSeedEmails` (`internal/cli/seed.go`)
splits on `,`, applies `auth.NormalizeEmail` and `auth.ValidateEmail` to
each, and rejects the whole invocation (before any write) on an empty
list, an invalid address, or a case-insensitive duplicate. The rest of
this document's examples still say `tester1@draab.at` etc. where they're
illustrative — nothing about the mechanism cares what the emails are, only
that `parseSeedEmails` accepted them.

### Decision: `internal/cli` composes domain services directly, same as `main.go`

`Seed` builds `account.Service`, `category.Service`, `entry.Service` (and
their Postgres stores) itself, the same construction `main.go`'s
`buildAccount`/`buildCategory`/`buildEntry` already do — not by importing
`main`, which isn't possible, but by duplicating that same three-line
construction inline. This isn't a new architectural allowance:
`internal/cli` already imports `internal/auth` directly for `Admin`, so
`internal/cli` depending on domain-package `Service` types is already the
established shape, just extended to three more packages.

**Revised during implementation**: the three testers are created via
`AuthStore.CreateUserWithIdentity` directly (see the next decision), which
bypasses `auth.Service.resolveIdentity` entirely — the one place
`NewUserHook`s fire. Wiring `auth.WithNewUserHooks(accountSvc,
categorySvc)` into `Seed`'s own `auth.Service` would therefore do nothing.
`NewUserHook` exists so `internal/auth` — which cannot import
`internal/account`/`internal/category` — can still notify them; `internal/cli`
has no such restriction, since it already imports both directly. So `Seed`
calls each user's `accountSvc.SeedDefaults(ctx, userID)` and
`categorySvc.SeedDefaults(ctx, userID)` itself, right after creating them —
simpler than routing through an indirection built to solve a
dependency-direction problem `internal/cli` doesn't have. `Seed`'s
`auth.Service` is used only for `IssueSession`.

### Decision: users are created in a fixed order so tester1 is always the bootstrap admin

`CreateUserWithIdentity`'s existing bootstrap check (`users` empty ⇒ next
account is admin, under an advisory lock) already does the right thing
here unmodified: since the reset just truncated `users`, creating
`tester1@draab.at` first makes it the admin, and `tester2@draab.at` /
`tester3@draab.at` land as ordinary users. No new admin-assignment logic —
this is the same path bootstrap has always used, just deliberately ordered
so which tester becomes admin is not left to chance.

### Decision: a new `Service.IssueSession`, not duplicated token logic in `internal/cli`

Minting a session for a user the caller already has (rather than through a
sign-in flow) needs the same 256-bit-token-plus-hash logic
`issueSessionToken` already implements — but that method, and `newToken`,
are unexported. Rather than reimplementing token generation in
`internal/cli` (a second place that would need to stay in sync with
`auth`'s hashing scheme), `auth.Service` gains one small exported method:

```go
// IssueSession mints a new session for userID directly, without a sign-in
// flow — used by the seed CLI to hand back an immediately usable session
// token for a user it just created. Never exposed over HTTP.
func (s *Service) IssueSession(ctx context.Context, userID string) (token string, err error) {
    user, err := s.store.UserByID(ctx, userID)
    if err != nil {
        return "", err
    }
    return s.issueSessionToken(ctx, user, SessionContext{Client: ClientAPI})
}
```

`SessionContext{Client: ClientAPI}` mirrors what a `client=api` sign-in
already produces (a bearer token, not a cookie) — the right shape for
something printed to a terminal.

**Caught during manual verification**: `IssueSession`'s expiry comes from
`s.p.SessionTTL` (`issueSessionToken` sets `ExpiresAt: now.Add(s.p.SessionTTL)`).
`Seed`'s first draft built its `auth.Service` with a zero-value
`auth.Params{}` — fine for `Admin`, which never mints a session, but for
`Seed` it meant every printed token had `ExpiresAt == the instant it was
created`, so pasting it into a request came back `401` immediately. Fixed
by passing `auth.Params{SessionTTL: cfg.Auth.SessionTTL, SessionMaxTTL:
cfg.Auth.SessionMaxTTL}` — the real configured values, matching what
`main.go`'s `buildAuth` uses. A regression test
(`TestSeedTestersTokensAreImmediatelyUsable`) now asserts each printed
token actually authenticates, not just that stdout contains the string
"session=".

### Decision: fixture shape — fixed constants, deterministic RNG

```
Per user:                              Per account:
  2–4 accounts (uniform random)          type: random pick from that
  currency: random from                        user's 6 seeded types
    {EUR, USD, GBP}                      opening_date: random within
                                                 the past 2 years
                                          15–60 entries (uniform random)

Per entry:
  kind: always "transaction"
  booking_timestamp: random within the past 12 months
  category_id: random pick from that user's 8 seeded categories
  amount: category "Salary" → positive, drawn from a plausible salary
          range; every other category → negative, drawn from a smaller
          plausible-expense range (still random, just sign- and
          magnitude-shaped by which category it landed on, matching how
          real transaction history actually looks)
  title: a short label drawn from a small per-category pool of plausible
         strings (e.g. Groceries → "Supermarket", "Grocery run"), so a
         listing doesn't read as obviously synthetic
```

`math/rand/v2` with a single fixed seed constant for the whole run (not
per-user or per-account) — Go 1.20+ auto-seeds the package-level generator
randomly by default, so a local `rand.New(rand.NewPCG(fixedSeed,
fixedSeed))` (or equivalent) is required for reproducibility, not the
package-level functions.

## Verification

- `internal/cli`: table-driven test over `run`-style dispatch (mirroring
  `cli_test.go`'s existing shape) against `storage/memory` where the
  in-memory stores support it, or a thin fake — bare `server seed` prints
  the table list and returns a non-zero-but-not-error exit without
  mutating anything; `server seed --yes` against a pre-seeded fixture
  results in exactly three users, `tester1@draab.at` admin, the other two
  not; each has the expected account-type/category starter set (via the
  `NewUserHook` wiring) plus 2–4 generated accounts and 15–60 entries per
  account; running it twice produces byte-identical account counts/titles
  (deterministic RNG) and no duplicate users (full reset first).
- `internal/storage/postgres` integration test (needs `DATABASE_URL`,
  matching this package's existing pattern): the real `TRUNCATE` against a
  throwaway database, seeded with unrelated rows first, ends up with every
  listed table empty except `schema_migrations`.
- `internal/auth`: `IssueSession` returns a token that `Authenticate`
  accepts for the right user, and is rejected once the CLI process's
  connection to the store closes (no lingering state beyond the row
  itself).
- Manual pass: `go run . seed` (bare) prints the table/row-count list and
  exits without touching the database (`psql` count check before/after
  matches); `go run . seed --yes` against a local Postgres, then use one
  of the three printed tokens as `Authorization: Bearer` against
  `GET /api/auth/me`, `GET /api/accounts`, `GET /api/entries` and confirm
  the data looks like a real user's — varied categories, plausible
  amounts, dates spread across the past year.

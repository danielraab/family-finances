## Why

There is no fast way to get a locally-running instance into a state worth
looking at. Building up even one realistic account with a few months of
categorized entries means clicking through the UI by hand — creating an
account, then dozens of individual entries — since the only other way data
enters the system is the full sign-in flow followed by manual entry. This
also means the per-user account-type/category starter set shipped in
`account-types-per-user` has never been exercised at any real volume (a
handful of accounts, months of transaction history), only the bare seeded
set itself.

`internal/cli` already has the exact shape this needs: `Admin()` builds its
own config/pool/store from `DATABASE_URL` and dispatches on subcommand
name, the same way a new `seed` subcommand would.
`AuthStore.CreateUserWithIdentity` already proves that creating a
pre-verified user without a magic-link round-trip is a supported path
(today only exercised by `internal/cli`'s own tests) — this change is the
first thing to use it for something other than a test fixture.

## What Changes

- New `server seed --yes <email1,email2,...>` CLI subcommand
  (`internal/cli.Seed`, dispatched from `main.go` alongside `healthcheck`
  and `admin`). **Revised after the first implementation** (which
  hardcoded `tester1/2/3@draab.at`): the emails are entirely CLI-supplied,
  comma-separated, so the command works for any set of test accounts, not
  just those three. `parseSeedEmails` splits, normalizes
  (`auth.NormalizeEmail`), and validates (`auth.ValidateEmail`) each one,
  rejecting an empty list, an invalid address, or a duplicate before the
  database is touched.
- **Full reset, not scoped to the given users**: every data table is
  `TRUNCATE`d (not `schema_migrations` — migration history is untouched,
  so this is a data reset, not a re-migration) before anything is
  (re)created. Because this wipes the whole database it ships in, and the
  same binary runs in production, the command requires exactly `--yes`
  followed by the email list; any other invocation (bare, a typo, an
  empty/invalid list) prints the list of tables it would truncate and
  exits without touching anything.
- Creates one user per given email, in that order, via
  `AuthStore.CreateUserWithIdentity` with a pre-verified email identity
  each, the same mechanism `internal/cli`'s own tests already use. Because
  the reset just emptied `users`, the first given email becomes the
  bootstrap admin through the existing zero-users-means-admin path — no
  new bootstrap logic.
- Each user's starter account types and categories are seeded directly
  (`accountSvc.SeedDefaults`/`categorySvc.SeedDefaults`) rather than via
  the `NewUserHook`s wired in `account-types-per-user` — creating a user
  this way bypasses `auth.Service.resolveIdentity` entirely, which is the
  only place those hooks fire; see design.md.
- For each given email, generates a small set of accounts across their
  seeded types and a mix of currencies, then several dozen transaction
  entries per account spread over the past year, drawn from their seeded
  categories — fully random (no distinct per-user scenario), but
  deterministically seeded so the output is the same on every run.
- Prints each user's email and a freshly minted, ready-to-use session
  token to stdout once seeding completes — script/copy-paste friendly, the
  same plain-stdout style `admin list` already uses, and needs no SMTP
  catcher to actually sign in as a seeded user locally.

## Capabilities

No existing capability document changes. `backend-package-architecture`'s
"CLI subcommands live in `internal/cli`" requirement is already written
generically ("for example `admin …`") and already covers a `seed`
subcommand without modification. This is a developer-facing tool with no
HTTP surface, no OpenAPI change, and no frontend change, so it does not
warrant a capability doc of its own.

## Impact

- **Dependencies**: none new.
- **Code**:
  - `backend/internal/cli/seed.go` (new): `Seed(ctx, args) int` — arg
    parsing (`--yes` plus a comma-separated email list, via
    `parseSeedEmails`), the `TRUNCATE` reset, dispatch into `fixtures.go`
    for the per-email `CreateUserWithIdentity` calls, session minting, and
    stdout output.
  - `backend/internal/cli/fixtures.go` (new): the random account/entry
    generator — constructs `account.Service`, `category.Service`,
    `entry.Service` over `internal/storage/postgres` stores (the same way
    `main.go`'s `buildAccount`/`buildCategory`/`buildEntry` already do;
    `internal/cli` importing domain-package services directly is already
    established by its existing `internal/auth` import) and calls their
    exported `Create` methods with randomly generated `New` values.
  - `backend/internal/auth/service.go`: a small new exported
    `Service.IssueSession(ctx, userID string) (token string, err error)`
    — a thin wrapper around the existing private `issueSessionToken`, used
    only by the seed CLI (never exposed over HTTP), so token
    generation/hashing stays in the one place that already owns it.
  - `backend/main.go`: dispatch `os.Args[1] == "seed"` to
    `cli.Seed(ctx, os.Args[2:])`, mirroring the `admin` dispatch.
- **API contract**: none — no HTTP endpoint.
- **Spec**: none — see Capabilities above.

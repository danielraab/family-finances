## Context

`internal/account`'s `account_types` slice (`account.go`, `store.go`,
`service.go`, `handler.go` — all sharing the package with `accounts`
themselves) is currently the only admin-managed, instance-global piece of
this domain: `id`, `title`, optional `description`, `disabled`
(`account-types-settings-tab`). `GET /api/account-types` is open to any
authenticated user; every write requires `is_admin` via the handler's
`requireAdmin` helper. `Service.resolveAssignableType` checks a `type_id`
exists and isn't disabled before letting `Create`/`Update` (re)assign it.

`internal/category` is the template for what this change turns
`account_types` into: `category-management` already moved categories from
global/admin-managed to owner-scoped, with every `Store` method taking an
explicit `ownerID` parameter (`category.Store`, not an implicit
context-derived filter), a cross-owner access reading as `ErrNotFound`, and
`disabled` as a reversible flag independent of hard/soft delete. Categories
kept their hard-vs-soft delete question separate from account types' — this
change does **not** revisit that: `account_types` keeps its existing hard
delete with the existing `ErrTypeInUse` (`409`) guard, since the user
explicitly asked to keep that shape.

Neither existing feature seeds anything: a new user's category tree and
(today) account-type list both start empty. This change adds seeding to
both, since the whole idea of "self-managed instead of global" for account
types has a colder start than the admin-curated list it replaces — without
a starter set, a brand-new user hits "add account" with an empty type
dropdown and has to leave the form to define one first.

## Goals / Non-Goals

**Goals:**

- Every account-type operation is scoped to its owner; `is_admin` is gone
  from every account-type code path.
- Cross-owner access to an account type (including as another user's
  `type_id`) reads as `ErrNotFound`, exactly like categories already do.
- Delete keeps its existing hard-delete-with-409-if-referenced shape,
  unchanged in behavior — only its scope changes (an owner's delete can
  only ever be blocked by that same owner's own accounts, since types and
  accounts share an owner now).
- Both account types and categories start with a small starter set instead
  of empty, for new users going forward and for existing users once, via
  this change's migration.

**Non-Goals:**

- Any household/shared-ownership concept — account types are private to
  one `user`, the same decision `category-management` already made for
  categories.
- Switching account types to soft delete, or adding `sort_order` to them —
  explicitly declined; account types keep exactly the lifecycle they have
  today, only the ownership scope changes.
- Preserving the *content* of today's global `account_types` rows through
  the ownership switch, or reconciling which account used which global row
  — this is a non-production app with no data to protect, so the migration
  reseeds everyone with the same starter set rather than cloning existing
  rows per owner (see "Migration Plan").
- Any locale-aware or user-language-aware seeding. The starter sets are
  seeded once in English and are immediately the user's own editable data
  — renaming a seeded row costs nothing more than renaming one they created
  themselves, so getting the seed language "right" isn't worth threading a
  language parameter through the signup flow (including the OIDC path,
  which has no natural place to source one from at account-creation time).
- Any change to how disabling an account type interacts with an account's
  `type_id` (`ErrTypeDisabled`, the effective-type-id-on-PATCH rule) — all
  of that is unchanged, just now operating within one owner's own data.

## Decisions

### Decision: ownership scoping mirrors `category.Store`'s explicit-`ownerID` shape

Every `Store` method for types gains an explicit `ownerID` parameter —
`ListTypes(ctx, ownerID)`, `GetType(ctx, ownerID, id)`,
`CreateType(ctx, ownerID, title, description)`,
`UpdateType(ctx, ownerID, id, title, description)`,
`SetTypeDisabled(ctx, ownerID, id, disabled)`,
`DeleteType(ctx, ownerID, id)` — the same explicit-parameter shape
`category.Store` uses (chosen there, and reused here, over an
implicit-context filter, so a recursive or cross-table query can't
accidentally forget whose data is in scope). `GetType` returns
`ErrNotFound` for a type owned by someone else, not a distinct
"forbidden" error — there's no scenario where a user should learn another
user's account type exists, matching `category`'s existing rule.

`Service.resolveAssignableType(ctx, ownerID, id)` gains the same
`ownerID` parameter: a `type_id` that exists but belongs to a different
owner resolves exactly like a nonexistent one (`ErrInvalidValue`, the
existing contract for caller-supplied input), never a distinct error.

### Decision: delete's behavior doesn't change, only its scope does

`DeleteType` keeps its existing hard `DELETE`, rejected `409`
(`ErrTypeInUse`) while any non-deleted account references it. The only
change is that `ownerID` narrows both which type can be targeted and which
accounts count toward "in use" — since accounts and their types now always
share an owner, this guard behaves identically to today's, just naturally
partitioned per user instead of instance-wide.

### Decision: seeding is delivered through a new-user hook, not a cross-package call

`internal/auth`'s `CreateUserWithIdentity` is where a user row is born, but
`internal/auth` cannot import `internal/account` or `internal/category` —
domain packages import none of each other (`backend/AGENTS.md`). This
change adds a narrow hook interface to `internal/auth`, the same pattern
`LanguageLookup` already establishes for `settings`:

```go
// internal/auth/service.go
type NewUserHook interface {
    SeedDefaults(ctx context.Context, ownerID string) error
}
```

`auth.Service` accepts zero or more via `auth.WithNewUserHooks(hooks ...NewUserHook)`
(mirroring `auth.WithLanguageLookup`'s functional-option shape) and calls
each, in order, right after `CreateUserWithIdentity` succeeds — best-effort
in the sense that a hook failure is logged, not surfaced as a signup
failure (a user who successfully created an account should never be told
signup failed because their starter account types couldn't be inserted).
`account.Service` and `category.Service` each gain a
`SeedDefaults(ctx, ownerID) error` method that structurally satisfies
`NewUserHook`; `main.go` wires
`auth.WithNewUserHooks(accountService, categoryService)` — no new
interface needs to be hand-written per package, and neither `account` nor
`category` needs to know it's being used as a hook.

This fires for both signup paths (magic-link and OIDC) since both end at
the same `CreateUserWithIdentity` call — no divergent seeding logic per
sign-in method.

### Decision: starter sets, seeded once, English only

```
Account types (SeedDefaultTypes)     Categories (SeedDefaults)
──────────────────────────────       ──────────────────────────
Checking                              Salary
Savings                               Groceries
Cash                                  Rent
Credit Card                           Utilities
Loan                                  Transportation
Investment                            Entertainment
                                       Health
                                       Other
```

Both lists are plain Go slices local to their own package (`account.go`,
`category.go`) — not configuration, not database rows describing
"defaults." `SeedDefaultTypes`/`SeedDefaults` insert them unconditionally
for a brand-new user (there is nothing to collide with yet). Categories get
one extra guard for the migration's existing-user backfill only (see
below): skip a user who already has at least one category, so an existing
user's already-built tree is never touched. Account types don't need the
equivalent guard because the migration clears `account_types` first — see
below.

### Decision: the migration reseeds instead of reconciling old global rows

The instruction driving this change was explicit: this app carries no
production data, so the migration doesn't need to preserve which account
used which global account type through the ownership switch. Rather than
writing reconciliation logic (clone each existing global row once per
owner, remap every account's `type_id` to that owner's copy of the same
row), the migration:

1. Deletes every existing `account_types` row outright.
2. Seeds every *existing* user with the same starter set `SeedDefaultTypes`
   uses going forward.
3. Repoints every existing account's `type_id` at its own owner's freshly
   seeded "Checking" row — not an attempt to preserve what that account's
   type used to mean, just a mechanical way to leave `accounts.type_id`
   non-null and valid. Any account that "should" be Savings/Credit
   Card/etc. is a one-field edit away in the UI post-migration, same as
   editing anything else.

Categories don't need step 1 or 3 — they've been owner-scoped since
`category-management`, with real (if sparse) per-user data already
possible in any environment that ran that migration. The categories
backfill is purely additive: seed the starter set only for a user with zero
existing categories, touching nothing for a user who already has some.

## Migration Plan

1. `backend/internal/storage/postgres/migrations/0015_account_types_ownership.sql`:
   ```sql
   -- Drop existing rows outright — no production data to preserve through
   -- the ownership switch (see design.md).
   DELETE FROM account_types;

   ALTER TABLE account_types
     ADD COLUMN owner_id uuid REFERENCES users(id);

   -- account_types_name_key predates the 0013 title rename (RENAME COLUMN
   -- does not rename the constraint) — this is the instance-wide
   -- uniqueness this change drops, matching categories' precedent of
   -- dropping naming uniqueness rather than reworking it per-owner.
   ALTER TABLE account_types DROP CONSTRAINT account_types_name_key;

   INSERT INTO account_types (owner_id, title)
   SELECT u.id, v.title
   FROM users u
   CROSS JOIN (VALUES ('Checking'), ('Savings'), ('Cash'),
                       ('Credit Card'), ('Loan'), ('Investment')) AS v(title);

   UPDATE accounts a
   SET type_id = t.id
   FROM account_types t
   WHERE t.owner_id = a.owner_id AND t.title = 'Checking';

   ALTER TABLE account_types ALTER COLUMN owner_id SET NOT NULL;
   CREATE INDEX ON account_types (owner_id);
   ```
2. `backend/internal/storage/postgres/migrations/0016_category_seed_defaults.sql`:
   ```sql
   INSERT INTO categories (owner_id, name, sort_order)
   SELECT u.id, v.name, v.ord
   FROM users u
   CROSS JOIN (VALUES ('Salary', 0), ('Groceries', 1), ('Rent', 2),
                       ('Utilities', 3), ('Transportation', 4),
                       ('Entertainment', 5), ('Health', 6), ('Other', 7))
     AS v(name, ord)
   WHERE NOT EXISTS (
     SELECT 1 FROM categories c WHERE c.owner_id = u.id
   );
   ```
3. `internal/account`: `Type` gains `OwnerID string` (`json:"-"`, same
   treatment as `Account.OwnerID`); every `Store`/`Service` type method
   gains `ownerID`; add `SeedDefaultTypes(ctx, ownerID) error` to both;
   `handler.go` drops `requireAdmin` from every account-type route, reading
   `ownerID` from `auth.UserFromContext` like the account routes beside it.
4. `internal/storage/postgres/account.go`: implement the new signatures
   (`WHERE owner_id = $1 AND ...` added to every query); remove the
   `isUniqueViolation` handling in `CreateType`/`UpdateType` — dead code
   once the title-uniqueness constraint is gone; implement
   `SeedDefaultTypes` as a single multi-row `INSERT`.
5. `internal/storage/memory`: mirror the same signature and behavior
   changes for the in-memory `Store`.
6. `internal/category`: add `SeedDefaults(ctx, ownerID) error` to
   `Store`/`Service`, implemented in both storage backends the same way as
   `SeedDefaultTypes`.
7. `internal/auth`: add the `NewUserHook` interface and
   `WithNewUserHooks` option described above; call each hook after
   `CreateUserWithIdentity` succeeds, logging (not failing signup on) an
   error.
8. `main.go`: wire `auth.WithNewUserHooks(accountService, categoryService)`.
9. `openapi/openapi.yaml`: drop "(admin only)" from every account-type
   endpoint's summary/description and remove their now-unreachable `403`
   responses (mirroring what `category-management` already did for
   categories' equivalent endpoints). No schema field changes. Then
   `cd backend && go generate ./...` and `cd frontend && pnpm generate:api`.
10. Frontend: `settings.account-types.tsx` drops its `is_admin` redirect
    guard and early return; `settings.tsx` moves the Account Types tab out
    of the admin-only group, alongside Common and My Invitations.

## Verification

- Backend: unit + handler tests — cross-owner access to an account type
  reads as `404` on every verb, including as another user's `type_id` on
  `POST`/`PATCH /api/accounts`; any authenticated (non-admin) user can
  create/update/disable/enable/delete their own account types; deleting a
  type referenced by a non-deleted account is rejected (`409`) regardless
  of which user's account it is (still scoped — a different user's
  unrelated account never affects this); a disabled type still can't be
  newly (re)assigned, exactly as before; a freshly created user has the
  full starter set of account types and categories immediately after
  signup (both magic-link and OIDC); the migration's existing-user backfill
  leaves a user who already had categories untouched, and repoints every
  pre-migration account's `type_id` at a valid, owned "Checking" row;
  `internal/storage/postgres` integration tests for both new migrations
  and the updated store methods.
- Frontend: `pnpm lint && pnpm exec tsc && pnpm build`; manual pass — a
  non-admin user now sees and fully manages the Account Types tab; a
  freshly created account already has account types to pick from on the
  first "add account" without needing to visit the tab first; admin-only
  behavior is gone (no redirect for a non-admin visiting
  `/settings/account-types` anymore).

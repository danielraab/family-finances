## Context

Two domain packages exist today: `internal/auth` (users, identities,
sessions, invites) and `internal/settings` (per-user language/timezone/
default-currency, resolved against hardcoded defaults). Neither stores any
money or references anything account-shaped. This change adds the first
"product" domain — a family's accounts and the entries booked against them —
following the same package-per-noun shape and dependency direction
(`main` → `httpapi`/`config`/`storage/*` → domain packages; domain packages
import no infrastructure) documented in `backend/AGENTS.md` and
`backend-package-architecture`.

Constraints carried over from the rest of the codebase:

- No web framework/ORM; `Store` interfaces declared by the domain package,
  implemented by `storage/memory` (tests) and `storage/postgres` (real),
  injected by `main`.
- No `os.Getenv` outside `internal/config`.
- Forward-only SQL migrations, one file per schema change, embedded and
  applied at startup.
- A domain package may depend on another domain package only through a
  narrow, structurally-satisfied interface it declares itself — the
  established precedent is `auth.Service`'s `LanguageLookup` interface,
  satisfied by `settings.Service`, wired in `main.go` — not a direct import
  of the other package's `Store` or types.

## Goals / Non-Goals

**Goals:**

- A signed-in user can create an account (title, description, type,
  currency, financial institute, opening/closing date) that only they can
  see.
- That user can record entries against their own accounts: relative
  transactions and absolute balance adjustments, each with a booking
  timestamp, title, optional description, exactly one category (except a
  balance adjustment, where a category is optional), and any number of
  their own tags.
- An account's balance at any point in time can be computed correctly and
  live from its entries.
- Account types and categories are curated centrally (admin-managed,
  instance-global); tags are private, per-user organization.

**Non-Goals:**

- Sharing an account or entry with another user, or any permissions model
  beyond single-ownership. `owner_id` is deliberately its own column on
  `entries` (not derived transitively through `accounts`) so that a future
  sharing change can grant access at the entry level without a schema
  change — but no such access is granted by this change.
- Any frontend screen. This change is backend-only; `/accounts` and entry UI
  are a follow-up once this API exists.
- Undeleting a soft-deleted account or entry.
- Multi-currency entries, currency conversion, or an entry-level currency
  field — an entry's amount is always in its account's currency.
- Moving an entry to a different account, or changing an entry's `kind`
  after creation.
- Validating or rescaling stored amounts when `AMOUNT_DECIMAL_PLACES`
  changes — the operator's responsibility, not this change's.
- Caching, precomputing, or materializing account balances.
- A canonical currency list (unchanged from `user-settings`'s existing
  shape-only validation).

## Decisions

### Decision: `owner_id` on both `accounts` and `entries`, visibility scoped to the owner

Both tables carry a required `owner_id uuid REFERENCES users(id)`. Every
store method that lists or fetches a row filters by
`owner_id = <authenticated user>`; a row belonging to someone else behaves
identically to a nonexistent one (`404`, matching `ErrNotFound` — not `403`,
so a caller can't distinguish "not yours" from "doesn't exist"). `entries`
duplicates the owner rather than joining through `accounts` for every check:
today it is always set equal to the parent account's `owner_id` at creation
and never changes, but a future per-entry share only has to add a grants
table and relax this column's write path — it doesn't have to touch every
existing row or query shape.

### Decision: package boundaries — `entry` depends on `account`, `category`, and `tag` through narrow interfaces

`internal/entry` needs to: confirm the caller owns the target account (and
read its currency), confirm a `category_id` exists, and confirm every
`tag_id` on the entry exists and is owned by the caller. Rather than
importing `internal/account`, `internal/category`, and `internal/tag`
wholesale (their `Store`, their persistence concerns), `entry.Service`
declares the narrow interfaces it needs — the same shape as
`auth.Service`'s `LanguageLookup`:

```go
type AccountLookup interface {
    // Owner returns the account's owner and currency, or ErrNotFound.
    Owner(ctx context.Context, accountID uuid.UUID) (ownerID uuid.UUID, currency string, err error)
}

type CategoryLookup interface {
    Exists(ctx context.Context, categoryID uuid.UUID) (bool, error)
}

type TagLookup interface {
    // OwnedBy reports whether every given tag id exists and belongs to owner.
    OwnedBy(ctx context.Context, owner uuid.UUID, tagIDs []uuid.UUID) (bool, error)
}
```

`account.Service`, `category.Service`, and `tag.Service` satisfy these
structurally; `main.go` wires the concrete services in. This keeps the
one-way dependency rule intact (no domain package imports another domain
package's `Store` or driver-facing code) while still letting `entry`
enforce its own invariants without duplicating lookup logic.

### Decision: `category_id` nullability is conditional on `kind`, enforced by a `CHECK` constraint

A `transaction` entry SHALL have a non-null `category_id`; a
`balance_adjustment` entry MAY have a null one. Rather than two nullable
columns or two tables, this is one `category_id uuid NULL REFERENCES
categories(id)` column plus:

```sql
CHECK (
  (kind = 'transaction' AND category_id IS NOT NULL)
  OR (kind = 'balance_adjustment')
)
```

`entry.Service` validates the same rule before it ever reaches the
database, so the API returns a clear `422` rather than surfacing a raw
constraint violation — the `CHECK` is a backstop, not the primary
validation path.

### Decision: ordering and same-millisecond ties — `ORDER BY booking_timestamp, id`, no separate sequence column

`entries.id` is a `bigserial` (monotonic, assigned in insertion order).
Every place order matters — listing, and the balance computation's "latest
adjustment at or before this point" — sorts by `(booking_timestamp, id)`.
Two entries booked at the exact same millisecond therefore tie-break by
insertion order for free, with no extra column and no `INSERT ... RETURNING`
gymnastics.

### Decision: balance is computed live, every time, with no cached column

`entry.Service.Balance(ctx, accountID, asOf)`:

```sql
-- 1. latest balance_adjustment at or before asOf (by the tie-break above)
SELECT amount FROM entries
WHERE account_id = $1 AND kind = 'balance_adjustment'
  AND deleted_at IS NULL AND booking_timestamp <= $2
ORDER BY booking_timestamp DESC, id DESC
LIMIT 1;
-- absent -> base := 0, base_at := -infinity (every transaction counts)

-- 2. sum of transactions strictly after that adjustment (or after -infinity), up to asOf
SELECT COALESCE(SUM(amount), 0) FROM entries
WHERE account_id = $1 AND kind = 'transaction'
  AND deleted_at IS NULL AND booking_timestamp <= $2
  AND (booking_timestamp, id) > (base_at, base_id);
```

No `balances` table, no trigger-maintained running total, no
materialized view — explicitly deferred until entry volumes make live
computation too slow (Risks). `GET /api/accounts/{id}/balance?as_of=` (default
`now`) is the only new read path this requires.

### Decision: soft delete matches the existing `deleted_at` convention; deletion does not cascade

Both `accounts` and `entries` get `deleted_at timestamptz`, set once,
never cleared — identical semantics to `users.deleted_at` /
`invites.deleted_at` (`0004_user_administration.sql`,
`0005_invite_revocation.sql`): excluded from listings, no undelete
endpoint. Soft-deleting an account does **not** write `deleted_at` on its
entries — instead, every entry read path (listing, balance computation)
joins to the parent account and requires it to be non-deleted too. This
avoids a cascading write across potentially many rows for what is, in
effect, a single flag flip.

### Decision: `account_types` and `categories` are hard-delete-if-unused, not soft-deleted

Only `accounts` and `entries` were asked to get a `deleted_at` column.
`account_types` and `categories` are admin-curated reference data, not
user records — deleting one that is still referenced by a non-deleted
account or entry (or, for a category, has child categories) is rejected
with `409` rather than performed or cascaded. An unused one is hard-deleted.
This mirrors the "protect referential integrity, don't silently orphan
data" instinct without adding a `deleted_at` nobody asked for on these
tables.

### Decision: deleting a tag cascades, deleting a category/account type does not

Tags are private, low-stakes organizational labels the owner fully
controls — deleting one just removes it from every entry it was attached to
(`ON DELETE CASCADE` on `entry_tags.tag_id`). Categories and account types
are shared classification data an admin curates for everyone; silently
detaching them from an entry on delete would quietly change what that
entry means to its owner, so those deletions are rejected instead when
in use (previous decision). Different blast radius, different behavior.

### Decision: amounts are integers at a fixed, env-configurable decimal-place count

`entries.amount` is `bigint`, minor units (e.g. cents at the default scale).
The decimal-place count is not a column — it is one instance-wide value,
`config.Config.AmountDecimalPlaces` (env `AMOUNT_DECIMAL_PLACES`, default
`2`), read once at startup and passed into `entry.NewService(...)` the same
way `config.AuthConfig` fields are passed into `auth.NewService`. There is
no per-currency scale and no migration path for changing it: this is an
accepted limitation (Risks), not solved here.

### Decision: account currency validation reuses `settings.ValidateCurrency`

`accounts.currency` needs the same ISO-4217-shape check (three uppercase
letters, not a canonical list) `user-settings` already validates
`default_currency` with. `internal/account` calls the exported
`settings.ValidateCurrency` rather than duplicating the regexp — a
one-direction dependency on a pure validation function, no `Store` or
driver involved, consistent with how `internal/settings` itself depends on
`internal/auth` only for a context accessor.

### Decision: API surface — flat `/api/entries`, not nested under accounts; `kind` and `account_id` immutable

`POST /api/entries` takes `account_id` in the body; `GET /api/entries`
filters by the required `account_id` query parameter. A nested
`/api/accounts/{id}/entries` was considered and rejected: entries are never
listed except scoped to one account today, so nesting buys nothing, and a
flat resource keeps `entry.Handler` self-contained rather than needing
`account`'s path parameter threaded through. `PATCH /api/entries/{id}` can
change `title`, `description`, `amount`, `booking_timestamp`, `category_id`,
and `tags`, but not `account_id` or `kind` — switching either changes what
the entry fundamentally means (which account's balance it affects, whether
it's relative or absolute) closely enough to a delete-and-recreate that
allowing an in-place change would need its own set of invariant checks for
no real benefit.

## Risks / Trade-offs

- **Live balance computation has no ceiling on entry count.** Every call
  scans (at most) one index lookup for the latest adjustment plus a ranged
  sum over transactions after it — fine at family-ledger scale, but there is
  no cache and no materialized total. Flagged as a likely future
  optimization once real usage data exists, not solved here.
- **`AMOUNT_DECIMAL_PLACES` is a silent reinterpretation knob.** Changing it
  after data exists does not rescale or validate anything already stored —
  an amount written as `1050` at 2 decimal places (10.50) reads as `10.50`
  units differently at 3 decimal places (1.050) with no warning. Accepted
  per explicit instruction; an operator-facing warning belongs in a future
  change if this becomes a real footgun.
- **No sharing model yet, but the schema already carries `entries.owner_id`
  independently of `accounts.owner_id`** in anticipation of it. Until a
  sharing change actually uses that independence, it's a column that is
  always redundant with its parent account's owner — a small, deliberate
  head start rather than dead weight, but worth remembering when a future
  change touches entry ownership.
- **404-not-403 for cross-owner access** means a client genuinely cannot
  tell "this account doesn't exist" from "it's not yours" — intentional
  (existence non-disclosure), but worth knowing if a future support/debug
  tool needs to tell the difference (it would need a privileged path, not a
  relaxed check on the normal one).

## Migration Plan

Five new forward-only migrations, in dependency order:

1. `0006_account_types.sql` — `account_types(id, name, created_at)`.
2. `0007_accounts.sql` — `accounts(id, owner_id, title, description,
   type_id, currency, financial_institute, opening_date, closing_date,
   deleted_at, created_at, updated_at)`.
3. `0008_categories.sql` — `categories(id, parent_id, name, created_at)`.
4. `0009_tags.sql` — `tags(id, owner_id, name, created_at)`, unique
   `(owner_id, name)`.
5. `0010_entries.sql` — `entries(id bigserial, owner_id, account_id, kind,
   amount, booking_timestamp, title, description, category_id, deleted_at,
   created_at, updated_at)` plus the `kind`/`category_id` `CHECK` and
   `entry_tags(entry_id, tag_id)` with `ON DELETE CASCADE` on both sides
   from `entries` and from `tags`.

No existing table is altered.

## Open Questions

None outstanding — ownership, visibility, soft delete, tie-breaking,
balance computation, category nullability, account-type scope, and amount
storage were all resolved during exploration before this proposal.

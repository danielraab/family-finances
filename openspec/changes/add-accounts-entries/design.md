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
  see, and can disable it (blocking new entries, reversibly) or soft-delete
  it.
- That user can record entries against their own accounts: relative
  transactions and absolute balance adjustments, each with a booking
  timestamp, title, optional description, exactly one category (except a
  balance adjustment, where a category is optional), and any number of
  their own tags (creatable inline).
- An account's balance at any point in time can be computed correctly and
  live from its entries.
- Account types and categories are curated centrally (admin-managed,
  instance-global); tags are private, per-user organization.
- That user can browse an accounts overview, drill into one account's
  details and recent activity, and work the full entry ledger across every
  account they own — filtered, searched, sorted, and paginated — entirely
  through the frontend, with the view's state reflected in the URL.

**Non-Goals:**

- Sharing an account or entry with another user, or any permissions model
  beyond single-ownership. `owner_id` is deliberately its own column on
  `entries` (not derived transitively through `accounts`) so that a future
  sharing change can grant access at the entry level without a schema
  change — but no such access is granted by this change.
- Undeleting a soft-deleted account or entry.
- Multi-currency entries, currency conversion, or an entry-level currency
  field — an entry's amount is always in its account's currency.
- Moving an entry to a different account, or changing an entry's `kind`
  after creation.
- Caching, precomputing, or materializing account balances.
- A canonical currency list (unchanged from `user-settings`'s existing
  shape-only validation).
- A dedicated category, account-type, or tag *management* UI. Categories
  and account types stay admin-only via the API (no admin frontend surface
  in this change, same as today); tags get exactly one creation surface —
  inline, from the entry form — and nothing else (no rename/delete UI).
- Sorting the entry ledger by anything beyond booking timestamp and amount
  (e.g. by account, category, or title) — those would need a join-backed
  keyset the first pass doesn't build.
- Bulk actions on the entry ledger (multi-select delete/retag/etc).
- Offline support or optimistic updates beyond ordinary loading/error
  states — data fetching stays plain `fetch` in effects, matching the rest
  of the frontend; no client-side cache/query library is introduced.

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
read its currency and `disabled` state), confirm a `category_id` exists,
and confirm every `tag_id` on the entry exists and is owned by the caller.
Rather than importing `internal/account`, `internal/category`, and
`internal/tag` wholesale (their `Store`, their persistence concerns),
`entry.Service` declares the narrow interfaces it needs — the same shape as
`auth.Service`'s `LanguageLookup`:

```go
type AccountLookup interface {
    // Owner returns the account's owner, currency, and whether new entries
    // are currently blocked, or ErrNotFound.
    Owner(ctx context.Context, accountID uuid.UUID) (ownerID uuid.UUID, currency string, disabled bool, err error)
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

### Decision: amounts are integers at a fixed 4 decimal places; display precision is a separate, per-user setting

`entries.amount` is `bigint`, always scaled by **4** decimal places
instance-wide — a Go constant (`entry.AmountScale = 4`) baked into
`internal/entry`, not read from configuration. An earlier draft of this
proposal made the decimal-place count an `AMOUNT_DECIMAL_PLACES`
environment variable; that's dropped — a single fixed constant needs no
config plumbing, no "what happens when it changes" footgun, and 4 places
comfortably covers every real currency's minor unit (including the
3-decimal outliers like `BHD`) with headroom to spare, so there's nothing
for an instance operator to tune.

What *is* configurable, per user, is display: `user-settings` gains
`displayed_decimal_places` (nullable smallint, default `2`, validated
`0..4` — never more than what's actually stored) alongside language/
timezone/default-currency, resolved the same way. It affects only how the
frontend *rounds an amount for display* (account balances, entry list
rows) — never how much precision the edit form accepts or the backend
stores; the create/edit form always works in the full 4-decimal
`amount`/`10000` fixed-point representation, so a display rounding
preference can never quietly truncate what's saved.

### Decision: `accounts.disabled` — a reversible flag distinct from `closing_date` and `deleted_at`

Accounts get a third, independent piece of state: `disabled boolean NOT
NULL DEFAULT false`. Disabling an account is reversible (an Enable action
flips it back) and has exactly one effect: `entry.Service.Create` rejects
(`422`) a new entry whose `account_id` names a disabled account. A disabled
account is otherwise unaffected — still listed, still readable, still
editable, its existing entries still fully usable (list, edit, delete,
count toward balance). This is deliberately a separate column from
`closing_date` (a bookkeeping fact about when an account was closed at the
institution — informational only, does not gate anything in this change)
and from `deleted_at` (one-way soft delete, invisible everywhere). The
three can combine freely (e.g. a closed-and-disabled account), and only
`deleted_at` removes an account from view.

### Decision: entry listing is filterable, searchable, sortable, and cursor-paginated

`GET /api/entries` grows real query parameters instead of just
`account_id`:

- **Filters**: `account_id` (repeatable — omitted means every account the
  caller owns), `category_id` (matches that category and, since categories
  form a tree, every descendant — computed by resolving the subtree once
  per request, not a recursive SQL query per row), `tag_id`, `kind`, `from`/
  `to` (inclusive `booking_timestamp` range).
- **Search**: `q`, matched against `title` and `description`
  (`ILIKE '%...%'` — no full-text index in this change; revisit if this
  becomes a real workload).
- **Sort**: `sort ∈ {booking_timestamp, amount}` (default
  `booking_timestamp`), `dir ∈ {asc, desc}` (default `desc` — newest
  first).
- **Pagination**: cursor-based, matching the tie-break already established
  for ordering. `after` is an opaque, base64-encoded `(sort_value, id)` pair
  for the last row of the previous page; the query becomes `WHERE
  (sort_column, id) < ($sort_value, $id)` (or `>` for `asc`) `ORDER BY
  sort_column, id LIMIT $page_size`. Every filter above still applies
  before the keyset comparison. The response is `{ items: Entry[],
  next_cursor: string | null }` — `next_cursor` is `null` once a page comes
  back shorter than `page_size`.

No `OFFSET`, no total count, no "page 3 of 12" — the frontend's infinite
scroll only ever asks for "the next page after the last thing I have,"
which is exactly what a keyset cursor is for and avoids the
consistency/performance problems `OFFSET` has on a table that keeps
growing while someone scrolls it.

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

### Decision: frontend routing, search-param, and data-fetching approach

Two new top-level `Sidebar` entries, "Accounts" and "Entries" (a second
hardcoded row in `NAV`, alongside a new icon each — the sidebar has only
ever had "Home" so far). Routes follow the existing file-based
`src/routes/` convention (`accounts.tsx`/`accounts.index.tsx`/
`accounts.$accountId.tsx`/`accounts.new.tsx`/`accounts.$accountId.edit.tsx`,
and the equivalent `entries.*` files), each gated the same way `/settings`
already is (redirect an anonymous visitor to `/login`).

`/entries`' filter/search/sort/cursor state lives in the URL via TanStack
Router's typed `validateSearch` / `useSearch` / `Link search={}}` — no new
dependency, and it's exactly what "filterable, searchable, sortable with
URL parameters, with pagination" asks for: a shared link reproduces the
exact same view, and the back button un-applies a filter change. The
running list of loaded entries (across "scroll to load more" pages) is
local component state, reset whenever the URL's filter/sort/search params
change but *not* when only scroll position changes — matching how the rest
of the frontend has no client-side cache (`AuthProvider`, `InviteList`):
plain `fetch` in an effect, keyed off the params that should refetch from
scratch.

Category selection (a tree) is a flat, indented list in a
`@headlessui/react` `Combobox`/`Listbox` (already a dependency, used for
`SidebarUser`'s menu) rather than a bespoke tree widget — the category set
is admin-curated and expected to be small, so a searchable flat list with
visual indentation reads the tree without needing expand/collapse
interaction. Tag input is a free-text field that matches against the
caller's existing tags as they type and, on submit, creates
(`POST /api/tags`) any typed value that didn't match an existing tag
before attaching it — the "inline creation" this proposal calls for,
without a separate tag-management surface.

## Risks / Trade-offs

- **Live balance computation has no ceiling on entry count.** Every call
  scans (at most) one index lookup for the latest adjustment plus a ranged
  sum over transactions after it — fine at family-ledger scale, but there is
  no cache and no materialized total. Flagged as a likely future
  optimization once real usage data exists, not solved here.
- **Filtered/searched cursor pagination needs an index that matches the
  query shape.** A keyset query with several optional filters plus `(sort,
  id)` ordering wants a composite index covering the common cases
  (`account_id, booking_timestamp, id` at minimum, per the original
  migration plan); `q`'s `ILIKE` is not indexed at all and degrades to a
  sequential scan for large ledgers. Acceptable at family-ledger scale;
  revisit (a trigram index, or real full-text search) if search becomes
  slow in practice.
- **Infinite scroll with no snapshot can show duplicate or shifted rows**
  if an entry is created, edited, or deleted while a long scroll session is
  in progress (keyset pagination has no stable "as of" point the way an
  offset+snapshot would). Accepted as an ordinary, low-stakes eventual-
  consistency artifact of a live ledger — a page refresh always
  self-corrects — not solved with a snapshot token in this change.
- **Two independent status switches on an account (`disabled`,
  `closing_date`) plus soft delete is three overlapping states to
  communicate clearly in the UI** (e.g. a closed-and-still-enabled account
  can still accept new entries, which may surprise a user who reads
  "closed" as "done"). The account overview/detail pages need a status
  presentation that shows all that apply, not just one badge — a frontend
  design detail to get right during implementation, not a schema risk.
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

Seven new forward-only migrations, in dependency order:

1. `0006_account_types.sql` — `account_types(id, name, created_at)`.
2. `0007_accounts.sql` — `accounts(id, owner_id, title, description,
   type_id, currency, financial_institute, opening_date, closing_date,
   disabled, deleted_at, created_at, updated_at)`.
3. `0008_categories.sql` — `categories(id, parent_id, name, created_at)`.
4. `0009_tags.sql` — `tags(id, owner_id, name, created_at)`, unique
   `(owner_id, name)`.
5. `0010_entries.sql` — `entries(id bigserial, owner_id, account_id, kind,
   amount, booking_timestamp, title, description, category_id, deleted_at,
   created_at, updated_at)` plus the `kind`/`category_id` `CHECK`, an index
   on `(account_id, booking_timestamp, id)` for the listing/balance keyset
   queries, and `entry_tags(entry_id, tag_id)` with `ON DELETE CASCADE` on
   both sides from `entries` and from `tags`.
6. `0011_account_disabled.sql` — `ALTER TABLE accounts ADD COLUMN disabled
   boolean NOT NULL DEFAULT false`. Kept as its own migration (rather than
   folded into `0007`) because it's a distinct concern added during
   exploration after the accounts table shape was first drafted — same
   spirit as `0004`/`0005` each being one focused `ALTER`.
7. `0012_user_settings_displayed_decimal_places.sql` — `ALTER TABLE
   user_settings ADD COLUMN displayed_decimal_places smallint CHECK
   (displayed_decimal_places BETWEEN 0 AND 4)`. The only migration in this
   change that touches a table from a previous change (`user_settings`,
   `0003_user_settings.sql`).

## Open Questions

None outstanding — ownership, visibility, soft delete, tie-breaking,
balance computation, category nullability, account-type scope, amount
storage (including the disabled flag, fixed decimal precision, and
frontend pagination/filtering approach added during a later exploration
round) were all resolved before writing this design.

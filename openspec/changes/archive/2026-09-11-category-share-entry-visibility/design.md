## Context

`internal/entry/service.go`'s `resolveFilter` (shared by `List` and
`Sum`) always narrows `Filter.AccountIDs` to `AccountLookup.VisibleIDs`
(the caller's own owned-or-shared accounts) before the store sees it;
`internal/storage/postgres/entry.go`'s `buildWhere` then ANDs
`entries.account_id = ANY(f.AccountIDs)` unconditionally, and both
`List` and `Sum` short-circuit to an empty result when `AccountIDs` is
empty (an optimization, since `= ANY('{}')` would return nothing
anyway). `category.Service.Subtree` — already called by `resolveFilter`
to resolve a `category_id` filter's descendant set — already encodes
exactly the permission check this change needs: it returns the full
owned subtree for a category the caller owns, `[id]` alone for one
visible only via a share (any tier), and `nil` for a category the
caller has no permission on at all (`internal/category/service.go:240`).

## Goals / Non-Goals

**Goals:**
- When a `category_id` filter is present and the caller holds any
  permission on that category, matching entries are returned
  regardless of the caller's account access.
- Leave every other read path (unfiltered `List`/`Sum`, an explicit
  `account_id` filter, `FlowSummary`, `BalanceSeries`, entry
  create/update/delete, single-entry `Get`) exactly as scoped today.
- Keep the permission check unforgeable: widening must be driven by a
  real permission resolution (`Subtree`'s non-empty result), never by
  the mere presence of a caller-supplied `category_id` value.

**Non-Goals:**
- Widening unfiltered listings or reports to include every
  category-shared entry automatically (rejected during scoping — see
  proposal's "Why"; that's a materially bigger, more surprising
  visibility grant the requester explicitly didn't want).
- Granting any account-level access (balance, other entries on that
  account, the account's own detail page) — only the entries matching
  the filtered category itself become visible.
- Any frontend change — the ledger's existing "not shared" fallback for
  an unresolvable `account_id` already renders this correctly.

## Decisions

### Widening signal: a new `Filter.AllAccounts bool`, not an empty `AccountIDs`

Reusing "empty `AccountIDs`" to mean "unrestricted" would collide with
its existing meaning ("caller has zero visible accounts — return
nothing"), which both storage backends already short-circuit on. A
distinct `AllAccounts bool` field keeps the two states unambiguous:
`buildWhere` (postgres) and `matchingRows` (memory) skip the
account-membership clause/check only when it's `true`; `List`/`Sum`'s
early-return guards become `if !f.AllAccounts && len(f.AccountIDs) == 0`.

### Where the permission check happens: `resolveFilter`, via the existing `Subtree` call

`resolveFilter` already calls `s.categories.Subtree(ctx, callerID,
*f.CategoryID)` for the default (non-exact) `CategoryMode` to resolve
descendants — its return value already *is* the permission check
(non-empty iff the caller owns or holds a share on that exact id).
`ModeExact` currently skips this call entirely (it just sets
`f.CategoryIDs = []string{*f.CategoryID}` with no permission
resolution — safe today only because the account restriction still
fully gates what's returned). For widening, `resolveFilter` now always
calls `Subtree` once (regardless of mode) to get that permission
signal; `ModeExact` still builds `f.CategoryIDs` from the raw id alone
(unchanged filtering semantics — exact never cascades), and separately
uses `Subtree`'s non-empty-ness purely to decide `AllAccounts`. This
keeps `ModeExact`'s pre-existing behavior (filter by any id, still
account-scoped, no new permission requirement to merely filter) while
adding a real permission gate specifically for the new widening.

### Explicit `account_id` filter suppresses widening

When the caller also supplies an explicit `AccountIDs` filter alongside
`category_id`, `resolveFilter` keeps intersecting it with `visible` as
today — `AllAccounts` is only set when no explicit account filter was
given. Rationale: a caller who explicitly names an account is asking
"entries in this account, in this category" — a narrower question than
"every entry in this category" — and silently reinterpreting it as the
latter (by ignoring the account they named unless it happens to be
visible) would be surprising. This also sidesteps a harder question
(should an explicit, inaccessible `account_id` combined with a
permitted category become visible too?) that the requester didn't ask
for.

### `Sum`'s per-account currency resolution needs no change

`AccountLookup.Access` (`internal/account/service.go:128`, backed by
`AccountStore.Access`'s SQL in `internal/storage/postgres/account.go:201`)
selects `currency`/`disabled` unconditionally from `accounts` by id
alone; the caller's permission is computed as an auxiliary `CASE`
column in the same query, never used to filter the row. `Sum` already
discards that permission value (`currency, _, _, err :=
s.accounts.Access(...)`). So a newly-widened cross-account entry's
currency resolves correctly with no code change — confirmed by reading
the implementation, not assumed.

### `Entry` gains `account_currency` — discovered during manual verification, not anticipated

The original version of this design assumed the frontend needed no
changes at all, reasoning that the ledger's existing "not shared"
fallback for an unresolvable `entry.account_id` was sufficient. Manual
verification (task 4.3) found that assumption half wrong: the ledger's
*account name* cell degrades gracefully, but its *amount* cell doesn't
— `accountCurrency(entry.account_id)` (a lookup against the caller's
own fetched `accounts` list) returns `""` for an account outside that
list, and `Intl.NumberFormat({ currency: "" })` throws, silently
falling back to a bare, unlabeled number. `reports.tsx` had the same
problem plus a worse one (the raw account UUID rendered in place of
any label at all, fixed separately, not by this decision).

Two shapes were available: (a) teach the frontend to tolerate a missing
currency by hiding the amount too, alongside the account name, or (b)
resolve `account_currency` server-side and put it directly on `Entry`,
mirroring `created_by_name`'s existing precedent. (b) is what got
built, based on later product direction (the amount should always be
shown, with the correct currency, even when the account name is
withheld — a currency code alone doesn't identify an account the way a
name would). It's also strictly simpler for the frontend: both
`entries.index.tsx` and `reports.tsx` had a local `accountCurrency`
helper that existed *only* to work around this gap by cross-referencing
the caller's own account list; with the field always present and
always correct, that helper is deleted outright rather than kept as a
fallback path, one less thing for those components to get wrong.

## Risks / Trade-offs

- **[Risk] A future edit to `ModeExact` or to `Subtree`'s semantics
  could silently break the permission gate** (e.g., if `Subtree` ever
  started returning non-empty for a category the caller has zero
  access to, for some unrelated reason) → Mitigation: a dedicated test
  asserts that filtering by a category id the caller has no permission
  on at all never sets `AllAccounts` and never surfaces another user's
  entries, independent of the "happy path" widening tests.
- **[Risk] Divergence between the postgres and memory store
  implementations** (one honors `AllAccounts`, the other doesn't) →
  Mitigation: the same table-driven service-level tests run against
  both backends (matching this package's existing pattern), so a
  missed implementation fails CI immediately rather than only in
  production.
- **[Trade-off] Currency resolution for a widened entry silently
  succeeds via a permission-blind lookup** — acceptable because
  currency alone (a 3-letter code) reveals nothing about the account's
  identity, balance, or other contents, and the alternative (a second,
  parallel "unchecked" lookup method) would add surface area for no
  behavioral difference.

## Migration Plan

No data migration. Pure application-logic change behind existing
endpoints/parameters; ships in the next normal deploy. No rollback
concerns beyond reverting the change — no schema or stored-data shape
changes.

## Open Questions

None outstanding — visibility scope (filter-only, not unfiltered) and
account-info exposure (none beyond currency, already-existing frontend
fallback) were confirmed with the requester before this design was
written.

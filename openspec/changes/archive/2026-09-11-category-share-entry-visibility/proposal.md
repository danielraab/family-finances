## Why

Sharing a category currently lets the recipient use it as an entry-list/
report filter, but the filter only ever searches the recipient's *own*
visible accounts. If the entries actually categorized under that shared
category live on an account the recipient has no separate account-level
access to (the common case — a category is shared precisely so someone
without account access can still see spending in one category), the
filter silently returns nothing. The feature is effectively inert for
its main use case. Sharing a category should make that category's own
entries visible when explicitly filtering by it, regardless of account
access — the account itself stays as private as before.

## What Changes

- `GET /api/entries` and its summary/report variants (`Sum`, i.e. the
  reports totals): when a `category_id` filter is supplied and the
  caller holds any permission on that exact category (real ownership or
  a `view`/`append` share), matching entries are returned regardless of
  whether the caller has account-level access to the entry's account.
  This widening applies only to that specific, explicit category filter
  query — it never changes what an unfiltered entries list or report
  shows, and never applies when the caller has no permission on the
  filtered category at all (unchanged `400`/empty-subtree behavior).
- An explicit `account_id` filter supplied alongside a category filter
  keeps today's behavior (scoped to the caller's own visible accounts) —
  the widening only fires when the caller isn't also narrowing by
  account.
- One API contract addition: `Entry` gains `account_currency`, its
  account's currency resolved server-side regardless of the caller's
  account-level access — the same "resolve it server-side, don't make
  the client re-derive it" precedent `created_by_name` already
  established. Without this, a client has no way to render a
  category-widened, account-inaccessible entry's `amount` with the
  right currency symbol at all (it isn't in the caller's own
  `GET /api/accounts`), which the frontend smoke test surfaced as a
  currency-less amount (`entries.index.tsx`) and a raw account UUID
  (`reports.tsx`, fixed separately below). No other field, endpoint, or
  filter parameter changes.
- Almost no frontend change needed beyond consuming that new field. The
  entries ledger
  (`entries.index.tsx`) already falls back to a "not shared" placeholder
  (`entries.notShared`) whenever `entry.account_id` isn't present in the
  caller's own fetched `GET /api/accounts` list — that's existing,
  generic defensive rendering, not new code, and it renders a
  now-reachable cross-account entry correctly with no change. The
  reports results table (`reports.tsx`) did **not** have the same
  fallback — it rendered the raw `entry.account_id` string instead —
  because this code path was previously unreachable (the backend never
  returned a cross-account entry to a report). Manual verification
  against the running dev stack (task 4.3) caught this; it's fixed to
  use the same placeholder as the ledger.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `entry-categories`: the existing "An entry-list or report category
  filter resolves a category shared with the caller" requirement is
  extended — resolving the filter no longer implies the result set
  stays scoped to the caller's own visible accounts; it now explicitly
  states that matching entries are returned regardless of account
  access, for that specific category-filtered query only.
- `account-entries`: a new requirement that every entry response
  includes `account_currency`, resolved server-side regardless of the
  caller's account-level access on it.

## Impact

- `backend/internal/entry/entry.go` — `Filter` gains a field marking
  "no account_id restriction for this query."
- `backend/internal/entry/service.go` — `resolveFilter` (shared by
  `List` and `Sum`) resolves the caller's permission on the filtered
  category and, when present and no explicit account filter was also
  supplied, sets that new field instead of narrowing to visible
  accounts.
- `backend/internal/storage/postgres/entry.go` and
  `backend/internal/storage/memory/entry.go` — both `List`/`Sum`
  (postgres) and `matchingRows` (memory) need to honor the new field:
  skip the account-membership restriction, and not short-circuit on an
  empty `AccountIDs` when it's set.
- No change to `internal/entry/service.go`'s currency resolution in
  `Sum` — `AccountLookup.Access` already returns an account's currency
  regardless of the caller's permission on it (permission is an
  auxiliary result, never a row filter), so summing a newly-visible
  cross-account entry's currency for report totals already works.
- `FlowSummary`/`BalanceSeries` take no category filter — out of scope,
  unaffected.
- Test coverage added in `backend/internal/entry` (service-level,
  exercised against both storage backends via the existing test
  harness) proving: a shared category's entries surface across
  accounts when filtered by it; an unfiltered list/report stays
  account-scoped; an explicit account+category filter combo stays
  account-scoped; a category the caller has no permission on never
  widens anything.
- `openapi/openapi.yaml` — `Entry` gains required `account_currency`;
  regenerate `backend/openapi.yaml` (`go generate ./...`) and
  `frontend/src/api/schema.d.ts` (`pnpm generate:api`) in the same
  change, per this repo's API-contract workflow.
- `backend/internal/entry/entry.go` — `Entry` struct gains
  `AccountCurrency string`.
- `backend/internal/storage/postgres/entry.go` — `entryCols`/
  `scanEntry` resolve it via a correlated subquery into `accounts`
  (`accounts.currency`), the same reach-into-another-domain's-table
  precedent `created_by_name` already establishes in this file, so
  every row carries it with no extra query.
- `backend/internal/storage/memory/entry.go` — left unpopulated (stays
  `""`), matching `CreatedByName`'s existing precedent there: the
  memory store backs domain/handler unit tests, never production, and
  none of those tests assert a resolved display value for either field.
- `frontend/src/routes/entries.index.tsx` and `reports.tsx` — the
  local `accountCurrency(accountId)` helper (a lookup against the
  caller's own fetched `accounts` list, `""` when not found) is
  replaced by reading `entry.account_currency` directly, since it's now
  always correct server-side data and removes the dependency on the
  caller's own account list entirely.

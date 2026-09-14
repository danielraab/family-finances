## Context

See `proposal.md` for motivation. Relevant existing shape (`backend/AGENTS.md`,
`backend/internal/entry`):

- `internal/entry` is a four-file domain package (`entry.go`, `service.go`,
  `store.go`, `handler.go`) declaring narrow `AccountLookup`/
  `CategoryLookup`/`TagLookup` interfaces, satisfied structurally by
  `*account.Service`, `*category.Service`, `*tag.Service` — no cross-package
  imports of `Store` types or drivers.
- `account.Permission` (`view`/`append`/`entry_admin`/`owner`) and
  `Access(ctx, accountID, callerID)` is the one place a caller's tier is
  resolved; `entry` mirrors it locally as `entry.Permission` to avoid an
  import cycle.
- Entries are cursor-paginated, filterable, and every response resolves
  `account_currency`/`created_by_name` server-side rather than making the
  client re-derive them.
- The backend has no job queue or scheduler — every side effect happens
  synchronously inside the HTTP request that triggers it.

## Goals / Non-Goals

**Goals:**
- A new `internal/recurringtransaction` package, structurally identical in
  shape and permission model to `internal/entry`, so it composes with
  existing account-sharing without new concepts.
- Compute `per_year_amount`, `ended`, and `next_suggested_date` server-side,
  on every read, never stored — consistent with how the backend already
  treats `account_currency`, `created_by_name`, and (for entries) the live
  balance.

**Non-Goals:**
- No scheduler, cron, or background job of any kind — "next suggested date"
  is a hint for prefilling a form, not a trigger for anything.
- No occurrence/schedule-compliance tracking (no "3 payments overdue"
  concept) — only a single next-suggested-date computation.
- No recurring balance adjustments.

## Decisions

### New package: `internal/recurringtransaction`

Follows the four-file shape exactly, declaring its own
`AccountLookup`/`CategoryLookup`/`TagLookup` interfaces the same way
`internal/entry` does (satisfied structurally by the same `*account.Service`/
`*category.Service`/`*tag.Service` — no new interface surface on those
packages). It additionally declares a narrow `EntryLookup` interface
(`LatestLinkedBookingTime(ctx, recurringTransactionID) (*time.Time, bool,
error)`, `LinkedCount(ctx, recurringTransactionID) (int, error)`),
satisfied structurally by `*entry.Service`, wired in `main.go` the same
post-construction way `account.WithUserLookup` is wired — `entry` doesn't
need to know `recurringtransaction` exists at all; the dependency points
one way, from the new package toward `entry`.

Rejected alternative: put recurring transactions inside `internal/entry`
itself (e.g. a `Kind: "recurring"` entry). Rejected because a recurring
transaction isn't a booked transaction — it has no `booking_timestamp`,
participates in no balance computation, and mixing it into `entry.List`'s
filters/sort/cursor logic would complicate a package that's already
handling a lot (balance recompute, flow-summary, balance-series). A
separate package with a narrow read-only dependency on `entry` for the
two lookups above is a much smaller surface.

### `recurring_transactions` table + `entries.recurring_transaction_id`

One migration adds both: `recurring_transactions` (mirroring `entries`'
shape minus `kind`/`balance`/`booking_timestamp`, plus `interval_unit`,
`interval_count`, `starts_on`, `ends_on`, `deleted_at`) and
`entries.recurring_transaction_id` (nullable FK, `ON DELETE SET NULL` is
irrelevant in practice since delete is blocked while any entry references
it — but included for defense in depth against a direct DB-level delete
bypassing the service layer, and to make the column's intent unambiguous
in the schema itself).

### `per_year_amount` / `ended` / `next_suggested_date` are always computed, never persisted

Same reasoning `entry.Balance` already established: a value derived from
other rows (here, the recurrence rule and the linked entries' booking
dates) drifts the moment anything it depends on changes, unless it's
recomputed on every read. There's no performance concern serious enough to
justify caching it — a caller's recurring-transaction list is small
(dozens, not thousands, of rows), and `next_suggested_date` needs one
indexed query per row for "the latest linked entry's booking_timestamp"
(`MAX(booking_timestamp) WHERE recurring_transaction_id = $1 AND
deleted_at IS NULL`), cheap the same way `entry.Service.Sum` already is.

### Calendar-aware date stepping for `month`/`year`, fixed day-count for `week`/`day`

Implemented once, in `internal/recurringtransaction`, as a small
`advance(t time.Time, unit Unit, count int) time.Time` using Go's
`time.AddDate` for `month`/`year` (which already clamps end-of-month
overflow the way the spec describes — `time.Date(2026, 1, 31, ...).AddDate(0,
1, 0)` yields March 3 in Go's raw arithmetic, so the implementation must
explicitly clamp to the target month's last day rather than relying on
`AddDate`'s overflow behavior — this is a one-function detail, not a
design fork) and plain `AddDate(0, 0, days)` for `week`/`day`.

### Frontend preset picker is a pure UI mapping, not new API surface

`interval_unit`/`interval_count` are the only fields the API knows about.
The preset dropdown (Weekly, Every 2 weeks, Monthly, Every 2 months,
Quarterly, Every 6 months, Yearly, Custom) is a static lookup table in the
frontend mapping a preset to a `{unit, count}` pair and back (for rendering
an existing recurring transaction's current preset, or falling back to
"Custom" when its `{unit, count}` doesn't match any preset exactly).

### Sidebar: a peer nav entry, not a new nesting concept

Confirmed with the user: "Recurring" is added to `Sidebar.tsx`'s existing
flat `NAV` array as one more entry, own glyph, own route — no expand/
collapse submenu concept is introduced. This is why the capability list
has no `web-client-shell` delta: per the existing pattern (see
`web-client-entries`'s "Entries link in the sidebar",
`web-client-reports`'s "Reports link in the sidebar", etc.), each
nav-bearing page capability specifies its own sidebar-link requirement
rather than `web-client-shell` enumerating every item.

## Risks / Trade-offs

- **[Risk]** `365.25`-based annualization for `day`/`week` units is an
  approximation, not exact — a user could in principle notice the total
  drift by a few cents against a manual calculation using `365`.
  **Mitigation**: this is inherent to any day-count-based annualization
  (there's no exact answer), and `365.25` is the more standard,
  leap-year-aware convention; `month`/`year` units (which cover the common
  cases — rent, subscriptions, salary, insurance) are exact.
- **[Risk]** A recurring transaction's `category_id`/`tag_ids` can go stale
  (disabled, unshared) the same way an entry's can. **Mitigation**: reuse
  `account-entries`' exact existing rule — already-set values are never
  re-validated on unrelated edits, only newly-set ones are checked against
  `Usable`.
- **[Risk]** `next_suggested_date`'s "latest linked entry" query runs once
  per recurring transaction on every list fetch. **Mitigation**: scoped to
  one account's recurring transactions (typically single digits to low
  tens of rows), indexed on `recurring_transaction_id`; no different in
  shape from `entry_count` already being computed per-row for categories
  and tags.

## Migration Plan

Additive only: new table, new nullable column, new package, new routes, new
frontend pages/nav item. No existing endpoint's request/response shape
changes except `Entry` gaining one new optional field. No backfill needed —
every existing entry's `recurring_transaction_id` starts `null`.

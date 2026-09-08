## Context

See proposal.md - Why. Relevant existing code this reuses or must stay
consistent with:

- `backend/internal/storage/postgres/entry.go`'s `Balance()`: "the latest
  non-deleted `balance_adjustment` at or before `asOf` (or `0` if none),
  plus every non-deleted `transaction` strictly after it (by the same
  `(booking_timestamp, id)` tie-break), up to `asOf`." This is the
  behavior this change must preserve exactly — only how it's computed
  changes.
- `backend/internal/entry/entry.go`'s `Kind` (`transaction` /
  `balance_adjustment`) and the existing `OptionalID` pattern for
  distinguishing "field absent" from "field present as null" — the same
  kind of explicit-contract instinct this change's `balance` field follows
  (a wire rename, not an overloaded meaning for `amount`).
- `internal/entry`'s `Sum` (`GET /api/entries/summary`): always forces
  `kind = 'transaction'`, deliberately excluding `balance_adjustment` —
  **unchanged by this change**. That endpoint answers "how much did I
  spend/earn in this category," where an absolute reading has no place.
  The new `flow-summary` endpoint answers a different question ("how did
  this account's balance actually move"), where a `balance_adjustment`'s
  delta belongs by design.
- `internal/settings.Service` already exposes a narrow structural
  interface to another domain package for exactly one field
  (`auth.LanguageLookup`, wired via `auth.WithLanguageLookup` in
  `main.go`, see `backend/AGENTS.md`'s "Settings" section) without
  `internal/auth` importing `internal/settings`. This change needs the
  same shape for timezone.
- `frontend/src/routes/accounts.$accountId.index.tsx` (recent entries) and
  `frontend/src/routes/entries.index.tsx` (full list) both already render
  an entry's amount via `amountColorClass`/`formatAmount` from
  `frontend/src/lib/amount.ts`, underlined when `kind ===
  "balance_adjustment"` — see `web-client-accounts`/`web-client-entries`'s
  existing "colored by sign" requirements. This change adds to both call
  sites identically.
- No chart library exists in the frontend today (checked
  `frontend/package.json`) — `BarChart3` in `FeatureOverview.tsx` is a
  `lucide-react` icon, not a chart.

## Goals / Non-Goals

**Goals:**

- A `balance_adjustment`'s delta is a real, stored value, correct after any
  create/update/delete anywhere in the account's history, and visible
  wherever the entry itself is rendered.
- `Balance()` keeps returning exactly the values it returns today — this
  change is an internal recomputation strategy, not a behavior change.
- One reusable bar-chart primitive, not a one-off for this page.
- The recompute cost of any single write stays O(1) (one or two row
  lookups + updates), never a scan of the account's whole history.

**Non-Goals:**

- No change to `GET /api/entries/summary`'s existing exclusion of
  `balance_adjustment`.
- No chart library. No async/background job processing — see "Decision:
  synchronous, in-transaction recompute" below.
- No bound on how far forward the account details page's year switcher can
  go.
- No UI copy change to the entry form's amount/balance field label — only
  the wire field name changes for a `balance_adjustment`.
- No multi-currency rendering in the account-details chart itself (the page
  always queries a single `account_id`, hence a single currency in
  practice) — the endpoint still groups by currency for a future
  multi-account caller.

## Decisions

### Decision: `amount` becomes a uniform delta; the absolute reading moves to a new `balance` field

`entries` gains `balance_reading bigint` (nullable), with a `CHECK`
constraint tying it to `kind`:

```sql
ALTER TABLE entries ADD COLUMN balance_reading bigint;
ALTER TABLE entries ADD CONSTRAINT balance_reading_matches_kind
  CHECK ((kind = 'balance_adjustment') = (balance_reading IS NOT NULL));
```

For `kind = 'transaction'`, `amount` is unchanged — the signed value the
user enters, `balance_reading` stays `NULL`. For `kind =
'balance_adjustment'`:

- `balance_reading` is what the client sends (as `balance`, not `amount` —
  see "Decision: wire contract" below) — the absolute reading, exactly what
  `amount` means today.
- `amount` becomes **computed**: the change from the balance immediately
  before this entry. Never client-supplied.

`Balance(asOf)` becomes a single query, no kind-branching:

```sql
SELECT COALESCE(SUM(amount), 0) FROM entries
WHERE account_id = $1 AND deleted_at IS NULL AND booking_timestamp <= $2
```

This is not a behavioral change: a `balance_adjustment`'s `amount` is
defined precisely as whatever value makes this sum land exactly on its
`balance_reading`, so it reproduces today's "ignore everything before the
latest adjustment" reset behavior by construction — see the worked example
below for why that holds even when something *before* an adjustment
changes later.

### Decision: only the entries immediately adjacent to a mutation ever need recomputing — worked example

An adjustment's delta depends on exactly two things: its own
`balance_reading`, and the running sum immediately before it. That running
sum is, in turn, pinned by the *previous* adjustment's own delta the same
way. This creates a firewall: recomputing one adjustment correctly makes
everything *after* it (up to the next adjustment) automatically correct too,
with no further cascade.

```
Adj1 (reading 100)   Txn2 (-20)   Adj3 (reading 200)   Txn4 (-30)   Adj5 (reading 300)
  amount = 100          amount=-20   amount = 200-80=120  amount=-30   amount = 300-170=130
  cum = 100             cum = 80     cum = 200             cum = 170    cum = 300
```

Now edit Txn2's amount from `-20` to `-50` (a change strictly before Adj3,
strictly after Adj1):

```
Adj1 (unchanged)   Txn2 (-50, edited)   Adj3 recomputed        Txn4 (unchanged)   Adj5 (unchanged!)
  amount = 100        amount = -50        amount = 200-50=150     amount = -30        amount = 130
  cum = 100            cum = 50            cum = 200               cum = 170           cum = 300
```

Adj3's `amount` must change (120 → 150) because the sum immediately before
it changed. But once Adj3 is recomputed, the cumulative sum *at* Adj3 is
back to exactly `200` — identical to before the edit — so Txn4's and Adj5's
stored `amount` values are still correct without touching them. The same
firewall argument applies to a create, update, or delete of *any* entry
(transaction or adjustment): only the nearest adjustment at-or-after the
changed position can possibly need its `amount` recomputed.

**Algorithm**, run synchronously inside the same DB transaction as the
mutation, after it's applied:

1. Let `P` be the mutated entry's `(booking_timestamp, id)` — for an
   `UPDATE` that changes `booking_timestamp`, this runs twice, once for the
   old `P` and once for the new one (they may resolve to the same row,
   which is harmless — recompute is idempotent).
2. Find `A1`, the earliest non-deleted `balance_adjustment` with
   `(booking_timestamp, id) >= P` (`LIMIT 1 FOR UPDATE` to lock it against
   concurrent recompute). If the mutated entry is itself a live
   `balance_adjustment`, `A1` is that same row — its own delta gets
   (re)computed against *its* predecessor, correctly bootstrapping it.
3. If `A1` exists, recompute `A1.amount = A1.balance_reading -
   BalanceBefore(A1)`, where `BalanceBefore(x)` sums `amount` for every
   non-deleted entry strictly before `x`'s `(booking_timestamp, id)`.
4. Find `A2`, the earliest non-deleted `balance_adjustment` strictly after
   `A1` (only relevant when `A1` exists), and recompute it the same way —
   this is the one entry whose *predecessor's identity or amount* just
   changed. Nothing after `A2` needs touching (see worked example).

A delete only removes a row from consideration, using the same
before/after positions; no different logic needed. Domain- and
service-layer tests should cover: inserting a transaction before an
existing adjustment, deleting one, editing an adjustment's own
`balance_reading`, editing an adjustment's `booking_timestamp` past another
adjustment, and the multi-adjustment chain above.

### Decision: synchronous, in-transaction recompute — not async

Considered a background job instead. Rejected: the recompute is one indexed
lookup plus one `UPDATE`, not a heavy aggregate — there's no latency problem
to justify it. This backend has no job queue or async worker anywhere
(`backend/AGENTS.md`: single binary, `net/http`, synchronous
request/response) — adding one is a much larger architectural change than
this feature warrants. Most importantly, staleness would be visibly wrong
exactly where it matters: editing an entry and immediately looking at the
affected adjustment's delta, or the chart, would show a number that hasn't
caught up yet, in a tool whose entire point is trustworthy numbers.

### Decision: wire contract sends `balance`, not `amount`, for a `balance_adjustment`

`EntryCreate`/`EntryUpdate` gain `balance` (integer, same 4-decimal-place
minor-unit scale as `amount`). For `kind: balance_adjustment`, `balance` is
required on create (replaces `amount` being required, which now applies
only to `transaction`); `amount` in the request body is rejected for a
`balance_adjustment` create/update (`400`) — it is never client-settable
for that kind, avoiding a silent "which one wins" ambiguity. For `kind:
transaction`, `amount` is required as today and `balance` is rejected the
same way. `Entry` responses always include the computed `amount` (the
delta, for both kinds) and, only for a `balance_adjustment`, `balance` (the
reading). This is a deliberate rename over reusing `amount` with a
per-kind meaning, matching this codebase's existing preference for
explicit contracts (`OptionalID`, immutable `kind`) over an implicit,
easy-to-misuse one.

The frontend's entry form UI is otherwise unchanged: the same field, same
label, same input — `entries.new.tsx`/`entries.$entryId.edit.tsx` just read
from and write to `balance` instead of `amount` when `kind ===
"balance_adjustment"`.

### Decision: `GET /api/entries/flow-summary` — one endpoint, two granularities

```
GET /api/entries/flow-summary
  ?account_id=...        (repeatable, same as /api/entries and /api/entries/summary)
  &unit=month|day
  &year=2026
  &month=1               (required when unit=day, rejected when unit=month)
```

Response:

```jsonc
{
  "buckets": [
    { "period": "2026-01-01", "income": [{"currency":"EUR","amount":250000}],
                               "outcome": [{"currency":"EUR","amount":180000}] }
    // one entry per month (unit=month) or per day of the given month (unit=day)
  ]
}
```

`period` is always the bucket's first calendar day, `YYYY-MM-DD`, regardless
of `unit` — a `day`-unit response just has finer-grained periods. `income`/
`outcome` reuse the existing `CurrencySum` shape (`{currency, amount}`),
grouped per currency the same way `Sum` already resolves each account's
currency (needed only because `account_id` is repeatable — the account
details page always passes exactly one, so it will only ever see one
currency in practice). `outcome` amounts are non-negative magnitudes, so a
client can plot both as upward bars. A currency with only income (or only
outcome) entries in a bucket is listed in just that one array — it is never
also listed on the other side with an amount of `0`, mirroring
`GET /api/entries/summary`'s "one entry per currency actually present," now
split per direction. A bucket with no matching entries at all is
`{"income": [], "outcome": []}` — no special-casing.

Per-entry contribution to a bucket: `amount` (already always the delta, per
above) — positive contributes to `income`, negative to `outcome`, zero
contributes to neither. `deleted_at IS NOT NULL` entries are excluded, same
as everywhere else.

**Timezone**: bucket boundaries (what counts as "January 2026," or "the
5th") are computed in the *caller's* resolved timezone setting (`GET
/api/settings`'s `timezone`, defaulting to UTC — see `user-settings`), not
UTC unconditionally and not the account's. `internal/entry` gains a narrow
structural interface, mirroring `auth.LanguageLookup`:

```go
// internal/entry/store.go
type TimezoneLookup interface {
    Timezone(ctx context.Context, ownerID string) (string, error)
}
```

satisfied by `internal/settings.Service`, wired in `main.go` the same way
`auth.WithLanguageLookup` is (`entry.WithTimezoneLookup(settingsSvc)`), so
`internal/entry` still doesn't import `internal/settings`. The Postgres
query converts with `timezone($tz, booking_timestamp)` before truncating to
month/day.

### Decision: reusable, hand-rolled bar chart — no charting library

Per discovery: no chart library dependency; a new `src/components/charts/`
directory holds presentational, data-in-props-out components (no fetching
inside them), so a future per-day-of-month chart or another page's chart
reuses the same primitive instead of hand-rolling SVG again. Colors follow
the `dataviz` skill's guidance (categorical palette, light/dark aware) at
implementation time rather than being fixed here. `frontend/AGENTS.md`
gains a short rule recording this convention (no chart library; hand-rolled
SVG/Tailwind under `src/components/charts/`; consult the `dataviz` skill)
so a future chart doesn't have to rediscover it.

The account details page owns fetching (`GET /api/entries/flow-summary`),
the selected-year state (component state, not persisted or in the URL —
matches the page having no other URL-driven state today), and the
previous/next-year buttons; it passes the fetched buckets down to the chart
component as props. Placed above "Recent entries," per discovery.

### Decision: the delta annotation shows the stored delta, unstyled by sign

The existing main display (the `balance_reading` value, colored by sign,
underlined for `balance_adjustment`) is unchanged — it now reads from
`entry.balance` instead of `entry.amount` for that kind, but renders
identically. The new annotation is a small, gray, sign-**un**colored
addition next to it showing `entry.amount` (the delta) with an explicit
`+`/`−` — gray regardless of whether the delta is positive or negative, per
discovery ("smaller font and gray color"). Both `accounts.$accountId.
index.tsx`'s recent-entries list and `entries.index.tsx`'s full list get
the identical addition, mirroring how both already duplicate the existing
color/underline rule.

## Migration Plan

1. `backend/internal/storage/postgres/migrations/00NN_entry_balance_reading.sql`:
   - `ALTER TABLE entries ADD COLUMN balance_reading bigint;`
   - Backfill, in two passes so the second can still read every
     adjustment's pre-migration absolute value via `balance_reading` even
     after `amount` starts being overwritten:
     a. `UPDATE entries SET balance_reading = amount WHERE kind =
        'balance_adjustment';` (copies today's absolute reading).
     b. For each account, walk its non-deleted `balance_adjustment` rows in
        `(booking_timestamp, id)` order (a window function over `adj` with
        `LAG` for the previous adjustment's `balance_reading`/position, then
        a correlated sum of non-deleted `transaction` rows strictly between
        the previous adjustment and this one) and set `amount =
        balance_reading - previous_balance_reading - transactions_between`
        (previous defaults to `0`/none). This reproduces exactly the
        pre-migration `Balance()` formula for every historical adjustment.
   - Add the `balance_reading_matches_kind` `CHECK` constraint (added after
     backfill so it doesn't fight the two-pass update).
   - Needs a dedicated integration test in
     `internal/storage/postgres/entry_test.go` asserting `Balance()` returns
     identical values before and after, against a fixture with multiple
     accounts, interleaved transactions and adjustments, and a soft-deleted
     entry.
2. `internal/entry`: `entry.go` (`Entry` gains `BalanceReading *int64`
   json:"balance,omitempty"`); `store.go` (the recompute algorithm above,
   `TimezoneLookup`, a `FlowSummary` method); `service.go` (delta
   recompute wired into `Create`/`Update`/`Delete`, `FlowSummary` using the
   caller's timezone); `handler.go` (new route, request validation for
   `unit`/`year`/`month`, and rejecting `amount`/`balance` on the wrong
   `kind`).
3. `internal/storage/memory` and `internal/storage/postgres`: implement the
   recompute algorithm identically (not a "good enough for tests"
   shortcut — `service_scenarios_test.go` asserts real balance behavior
   against the memory store) and `FlowSummary`.
4. `openapi/openapi.yaml`: `Entry`/`EntryCreate`/`EntryUpdate` gain
   `balance`; new `GET /api/entries/flow-summary` path + `FlowSummary`/
   `FlowBucket` schemas reusing `CurrencySum`. Then `go generate ./...` and
   `pnpm generate:api`.
5. Frontend: entry form field rename, delta annotations on both list call
   sites, `src/components/charts/`, the account details page's chart +
   year switcher, `frontend/AGENTS.md` rule, i18n.

## Verification

- Backend: `internal/entry` service tests for the recompute algorithm
  (each scenario in the worked example, plus edits that move an
  adjustment's `booking_timestamp` across another adjustment); handler
  tests for `flow-summary` (month and day units, multiple currencies via
  multiple accounts, empty buckets, invalid `unit`/`month` combinations,
  timezone affecting which bucket a boundary-crossing entry falls in);
  `internal/storage/postgres` integration tests for the migration backfill
  and for `FlowSummary`'s SQL directly. `internal/openapicheck.AssertResponse`
  on the new/changed handler tests.
- Frontend: `pnpm lint && pnpm exec tsc && pnpm build`; manual pass —
  create a balance adjustment and confirm the delta annotation on both
  list pages; edit an earlier transaction and confirm a later adjustment's
  displayed delta updates; switch years on the account details chart and
  confirm bars match `GET /api/entries/flow-summary`'s response; confirm a
  month with no entries renders (not errors).

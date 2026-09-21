## Context

Two independent pieces land in one change because they touch the same
page and the same request cycle, but they're separable: sorting is
frontend-only (the backend already accepts `sort`/`dir` on
`GET /api/entries`); Income/Outcome requires widening
`GET /api/entries/summary`'s response and the `Sum` code path behind it.

## Decisions

### Sort fires immediately, outside the "Generate report" gate

`/reports` deliberately never fetches on a filter change — every control
(category, tag, account, date range) only updates the URL and marks the
displayed report stale until "Generate report" is activated again
(`web-client-reports`' "A report is only generated on explicit action").

Sort does not go through that gate. Clicking a column header re-fetches
`GET /api/entries` immediately with the new `sort`/`dir`, resetting
`items`/`nextCursor` the same way `generatedFilter` changing already does,
but it does **not** mark the report stale and does **not** require
"Generate report" to be clicked again. Two reasons:

1. It matches `/entries`, the page this whole pattern is copied from,
   where sorting has always been immediate.
2. It never changes *which* entries are included or the sums — only their
   order — so folding it into the same staleness machinery that exists
   specifically to warn "the results on screen don't match your current
   filters" would be misleading: the results still match, just in a
   different order.

Consequently `generatedFilter` is left alone; sort/dir are read straight
from the URL search params (`search.sort`, `search.dir`) at fetch time,
the same way `/entries` does, rather than being folded into
`GeneratedFilter`. A page with no report generated yet has no sort request
to make, so the column headers are inert (or simply not rendered) until
`hasGenerated`.

Before a report is generated, clicking a header only updates the URL (no
`generatedFilter` yet to re-fetch against) — consistent with every other
control on this page, and harmless since there's nothing on screen to
reorder yet.

### `EntrySummary` gains `income`/`outcome`, `CurrencySum` itself is untouched

`CurrencySum` (`{ currency, amount }`) is shared by `FlowBucket`,
`BalancePoint`, and `EntrySummary.sums`. Adding fields to it to carry
income/outcome would leak two meaningless zero fields onto
`BalancePoint`'s running-balance samples. Instead `EntrySummary` grows two
new arrays of the existing `CurrencySum` type — exactly how `FlowBucket`
already pairs `income`/`outcome` beside itself, just without a `period`:

```
EntrySummary {
  sums:    CurrencySum[]   // net — unchanged
  income:  CurrencySum[]   // new
  outcome: CurrencySum[]   // new
  count:   int64           // unchanged
}
```

Purely additive — no existing client (the dashboard `query_stat` card, any
other `sums` reader) needs to change.

**Discovered during implementation:** `EntrySummary` was also, coincidentally,
the response schema for `GET /api/recurring-transactions/summary` (an
unrelated domain that happens to return the same `{ sums, count }` shape).
Widening it in place would have forced `income`/`outcome` — a concept that
doesn't apply to recurring-transaction templates — onto that endpoint too.
Split instead: `GET /api/recurring-transactions/summary` now returns its own
`RecurringTransactionSummary` schema (identical `{ sums, count }` shape,
unchanged behavior), and `EntrySummary` is entries-only, free to grow
`income`/`outcome`. No Go code changes outside `openapi/openapi.yaml` and the
two generated artifacts — no Go type is bound to a schema name.

### `Store.Sum` widens to carry income/outcome per account

`Sum` today returns `perAccount map[string]int64` (net only) plus a count;
`Service.Sum` groups that by currency. The Postgres query already computes
one `SUM(amount)` per account in a single `GROUP BY account_id` — adding
the same `FILTER (WHERE amount > 0)` / `FILTER (WHERE amount < 0)` split
`FlowSummary`'s query already uses is one extra pair of columns in the
same query, not a second round trip:

```sql
SELECT account_id,
       SUM(amount) AS amount,
       COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0) AS income,
       COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0) AS outcome,
       COUNT(*)
FROM entry_legs WHERE … GROUP BY account_id
```

`Store.Sum`'s signature changes from `(map[string]int64, int, error)` to
`(map[string]AccountSum, int, error)`, where `AccountSum{ Amount, Income,
Outcome int64 }` is a new small struct in `entry.go`. `Sum` has exactly one
caller (`Service.Sum`) and no other implementation depends on the old
shape, so this is a clean signature change across both stores (Postgres
and the in-memory test double) rather than an additive one.

`Service.Sum` then builds three `[]CurrencySum` (sums, income, outcome)
from the same per-account/per-currency grouping loop it already runs once,
instead of three.

A self-transfer already contributes to `Sum` once per account in scope,
netting to zero across both legs when both are in scope (existing
behavior, unchanged). The same per-leg signed amount drives income/outcome
identically: a self-transfer's outgoing leg (negative) adds to that
currency's `outcome`, its incoming leg (positive) adds to `income` — so a
self-transfer with both accounts in scope contributes to *both* `income`
and `outcome` even though it nets to zero in `sums`. This is accepted,
not a bug: `income`/`outcome` answer "how much moved in / out," and a
transfer between two of the caller's own accounts genuinely moved money
out of one and into the other.

### Frontend: Income/Outcome render per currency, next to the net sum

The sums bar (`reports.tsx`, currently one `<span>` per currency) grows two
more figures per currency, reusing `formatAmount`/`amountColorClass`
exactly as the net sum already does. Labelled "Income"/"Outcome" via new
`reports.income`/`reports.outcome` i18n keys — new keys, not a reach into
`flowChart.income`/`flowChart.outcome`, matching this repo's convention of
per-page label keys (see `add-recurring-list-filters`'s design.md on why
`/recurring`'s labels don't reach into `entries`'s namespace either).

`amountColorClass` is already sign-aware (used for the net sum, which can
be negative); Outcome's `amount` is a positive magnitude by construction
(`-SUM(...) FILTER (amount < 0)`), so it renders in the same "negative"
color the ledger already uses for money leaving, achieved by negating it
for display purposes (`amountColorClass(-outcomeAmount)`,
`formatAmount(-outcomeAmount, …)`) rather than by the API returning a
signed value — keeping the wire format an unambiguous magnitude while the
UI still shows it as an outflow.

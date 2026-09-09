## Context

The account details page has one chart today: a hand-rolled grouped
`BarChart` (`frontend/src/components/charts/BarChart.tsx`) fed by
`GET /api/entries/flow-summary`, showing income/outcome per month of a year.
`frontend/AGENTS.md` codifies the convention: charts are dependency-free
SVG/Tailwind components under `src/components/charts/`, presentational only,
colors from the `dataviz` skill's validated palette.

This change adds a second chart — a per-day running-balance line over one
month — and the `LineChart` primitive behind it. The balance itself is
never stored: `internal/entry`'s `Service.Balance` /
`Store.Balance(accountID, asOf)` computes it live as `SUM(amount)` with
`balance_adjustment` rows acting as anchors (everything before the latest
adjustment at-or-before `asOf` is ignored; the sum lands exactly on that
adjustment's `balance` reading). `FlowSummary` already resolves the
caller's timezone through an optional `TimezoneLookup` (default UTC) and
returns per-currency buckets with every period present even when empty.

Constraints: no charting library; no new backend dependency; the frontend
never talks to the database; the API contract is spec-first
(`openapi/openapi.yaml` → committed `backend/openapi.yaml` +
`frontend/src/api/schema.d.ts`).

## Goals / Non-Goals

**Goals:**

- A `GET /api/entries/balance-series` endpoint returning the running
  balance of the caller's account(s) sampled at each local midnight of a
  given month, per currency, timezone-aware.
- A reusable, presentational `LineChart` primitive with a signed y-axis,
  zero rule line, and step interpolation, sharing axis / width / tooltip
  machinery with `BarChart`.
- The account details page shows that line for the account's own currency,
  with a month switcher, below the existing bar chart.

**Non-Goals:**

- Multi-account or filtered balance lines. Combining accounts means
  computing each series independently and summing per currency afterward —
  a separate change. The endpoint's per-currency response shape and
  repeatable `account_id` leave room for it; the account details page only
  ever passes one `account_id`.
- Category / tag / text filtering of the series (a filtered "balance" is not
  a balance — see the spec).
- Sub-day fidelity (intraday low points between two midnights). Sampling is
  daily, as requested.
- Caching or persisting balances — the series is computed live per request,
  same as `GET /api/accounts/{id}/balance`.

## Decisions

### A dedicated endpoint, not client-side accumulation from `/api/entries`

The alternative — fetch the month's entries plus an opening balance and
accumulate in the route — was considered and rejected:

- `GET /api/entries` is cursor-paged; a busy month needs multiple round
  trips or an arbitrary large `limit`. `summary` / `flow-summary` exist
  precisely to avoid paging for an aggregate.
- The client would have to reimplement the `balance_adjustment` anchor rule
  (for adjustments inside the window) — logic that currently lives once, in
  `Store.Balance`, across both storage backends. A TypeScript copy is a
  drift and correctness risk.
- Day bucketing would use the browser timezone, which can differ from the
  account setting; the server already resolves the correct one.
- It does not even avoid a backend change (the opening balance still needs
  the server).

The one real upside of the raw-entries approach — being able to draw a
vertex at every actual transaction — is better served, if wanted later, by
the endpoint emitting step vertices (see below), not by moving computation
to the client.

### Sample at each local midnight; reuse `Store.Balance` per boundary

`Service.BalanceSeries` resolves the timezone exactly as `FlowSummary`
does, computes the day boundaries (`00:00` on days 1..N of the month, plus
`00:00` on the 1st of the next month, all in the resolved location), and
for each boundary calls the existing `Store.Balance(ctx, accountID, asOf)`
per matching account, summing results per currency.

For a month that is ~29–32 boundaries × (usually 1) account = a few dozen
indexed `SUM` queries per request — acceptable for a live, uncached read,
and it reuses the exact, already-tested balance rule with zero
duplication. A single-pass alternative (one opening balance + a walk of the
month's entries, re-anchoring on adjustments in-window) is a possible
optimization if this ever shows up in profiling; it is deliberately *not*
done now because it re-implements the anchor logic that the per-boundary
approach gets for free.

Boundaries and period labels reuse / mirror `flowPeriods` (label =
`YYYY-MM-DD` of the point's local day; the closing point is labelled with
the 1st of the next month).

### Response shape mirrors `flow-summary`

```
{ "points": [ { "period": "2026-03-01",
                "balances": [ { "currency": "EUR", "amount": 128000 } ] },
              ... ,
              { "period": "2026-04-01", "balances": [ ... ] } ] }
```

`period` is a date string (the point's local day), `balances` a
per-currency list reusing the existing `CurrencySum` schema. A currency is
listed on every point (unlike `flow-summary`, which omits a zero side) —
a balance line needs a value at every x even when it is `0`. Every point in
the month is present; a no-activity month is a run of identical values.

`unit` is accepted and must be `day` (the only defined value) so the query
shape reads like `flow-summary`'s and leaves room for a future `week` etc.
`month` is required and validated 1–12; `year` ≥ 1. Supplying
`category_id`, `tag_id`, `from`, `to`, or `q` is a `400`.

### `LineChart` + a shared `charts/` internal

`BarChart` currently privately owns: the `ResizeObserver` container-width
measurement, `niceMax` axis-tick rounding, and the hover / click-to-pin /
Escape-to-release tooltip state machine. `LineChart` needs all of it, plus:

- a **signed** y-domain (`niceMax` generalized to `niceExtent(min, max)`),
- a **zero rule line** when the domain spans zero,
- a `<path>` per series with `stepAfter` geometry (horizontal to the next
  x, then vertical) — a balance holds flat between the entries that move
  it, so a straight diagonal between midnights would draw money moving on
  days nothing happened,
- a dot at each sample point, and the shared per-x tooltip listing every
  series' value.

Extract the shared pieces into `src/components/charts/` internals (e.g.
`useContainerWidth`, `niceExtent`, a `usePinnableHover` hook or a small
`<ChartTooltip>`), refactor `BarChart` to consume them with **no
behavioural change**, and build `LineChart` on the same base. Series
colors come from the `dataviz` palette as Tailwind arbitrary-value
`stroke-[#…]` classes (run `scripts/validate_palette.js` on any new
pairing). On the account details page there is exactly one series, in the
account's currency; `LineChart` supports N for the future multi-currency /
multi-account case.

### Page placement

Below the bar chart, its own `<section>` with a heading and a
previous/next month control mirroring the year switcher's markup and
"no bound in either direction" behaviour. Default month = current month.
Two independent switchers (year for bars, month for line) rather than a
drill-down — simpler, and the two charts answer different questions. A
separate `useEffect` keyed on `[accountId, chartMonth, chartYear]` fetches
the series, matching the existing flow-summary effect.

## Risks / Trade-offs

- **Per-request query count** (a few dozen `SUM`s) → acceptable for a live
  read at this scale; single-pass walk documented as the escape hatch if
  profiling ever disagrees.
- **Daily sampling hides intraday dips** (balance goes negative at noon,
  recovers by midnight — invisible) → accepted and called out as a
  non-goal; the response shape can later carry per-entry vertices without a
  breaking change.
- **Linear vs. step rendering** → step chosen; a straight line between
  midnights would misrepresent a balance as continuously changing. Trade-off
  is a slightly busier-looking line on active months.
- **Refactoring `BarChart` onto a shared base could regress its tooltip /
  pin behaviour** → the extraction is mechanical (move code, no logic
  change); its existing behaviour is covered by the `web-client-accounts`
  bar-chart scenarios and should be eyeballed in the running app.
- **`niceExtent` for an all-positive or all-negative month** → must still
  produce sane ticks and only draw the zero line when the domain genuinely
  spans zero; unit-test the boundary cases.
- **Closing point labelled with the next month's 1st** → the frontend
  formats x labels as day-of-month; ensure the 32nd point renders sensibly
  (e.g. as the last tick / "1" of next month) rather than looking like a
  duplicate.

## Migration Plan

No data migration — new read over existing rows. Ship in one change: edit
`openapi/openapi.yaml`, regenerate `backend/openapi.yaml`
(`cd backend && go generate ./...`) and `frontend/src/api/schema.d.ts`
(`cd frontend && pnpm generate:api`), implement the backend service +
handler + both stores, then the frontend primitive + page wiring. Rollback
is reverting the change; nothing persists.

## Open Questions

- Endpoint path: `GET /api/entries/balance-series` (sits beside
  `flow-summary`, entries-scoped) vs. `GET /api/accounts/balance-series`
  (it is fundamentally about accounts). Leaning `entries/balance-series` for
  symmetry with `flow-summary` and because it reuses the entries filter
  vocabulary's `account_id` shape — confirm during spec edit.
- Whether the closing 32nd point is worth it or the line should simply end
  at the last day's midnight. Kept for now so the last day's activity is
  visible.

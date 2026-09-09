## Why

The account details page shows *flows* over time — an income/outcome bar
chart, one pair of bars per month of a year — but never the *level*: how
much money the account actually held as the month went on. The live balance
is a single present-day number; there is no way to see it rise and fall
across a month, spot the low point before a paycheck, or see the effect of a
mid-month balance adjustment. The bar chart also proved the "hand-rolled,
reusable, presentational SVG chart" approach works; a second chart shape (a
line) is the natural next primitive.

## What Changes

- **New endpoint `GET /api/entries/balance-series`.** Given `account_id`
  (repeatable), `unit=day`, `year`, and `month`, it returns the running
  account balance sampled at each local midnight of that month — one point
  per day, from the 1st at 00:00 (the balance of all history before the
  month) through a closing point at the end of the last day. Values are
  grouped per currency, the same per-currency bucket shape as
  `GET /api/entries/flow-summary`. It honours **only** `account_id` and the
  month window — no `category_id` / `tag_id` / `q`: a category- or
  tag-filtered "balance" is not a balance (a `balance_adjustment` carries
  neither, so any such filter would silently drop the anchors the running
  sum depends on). The caller's resolved timezone setting decides the
  midnight boundaries, exactly as `flow-summary` does.
- **New `LineChart` primitive** under `frontend/src/components/charts/`,
  alongside the existing `BarChart` — hand-rolled SVG/Tailwind, no charting
  library, presentational only (data + series definitions in as props,
  nothing fetched inside). Signed y-domain with a zero rule line, `stepAfter`
  interpolation (a balance is a step function — it only moves at an entry),
  responsive width, and the same hover / click-to-pin tooltip behaviour as
  `BarChart`. The width-measuring, "nice" axis-tick, and pin-tooltip
  scaffolding currently private to `BarChart` is lifted into a shared
  `frontend/src/components/charts/` internal so the two charts stay
  consistent.
- **Account details page gains a running-balance line chart** below the
  income/outcome bar chart, with previous-month / next-month controls
  (mirroring the bar chart's year switcher, no bound in either direction).
  One line — the account's own currency.
- **`frontend/AGENTS.md`'s Charts section** is extended to name `LineChart`
  and the shared internal beside `BarChart`.

Not in scope, called out to bound the endpoint's shape: a multi-account or
filtered version of this chart. Combining several accounts means computing
each account's balance series independently and summing per currency after —
its own change, later. The endpoint's per-currency response shape leaves
room for it without a redesign.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `account-entries`: adds the `GET /api/entries/balance-series` endpoint —
  a per-day running-balance series for one or more of the caller's accounts
  over a single month, per currency, timezone-aware, `account_id`-only
  filtering.
- `web-client-accounts`: the account details page adds a running-balance
  line chart for the month, with a month switcher, below the existing
  income/outcome bar chart.

## Impact

- **API contract**: `openapi/openapi.yaml` gains the
  `GET /api/entries/balance-series` operation and a `BalancePoint` (or
  reuses `flow-summary`'s per-currency amount shape) response schema;
  regenerate `backend/openapi.yaml` and `frontend/src/api/schema.d.ts` in
  the same change. CI's `contract` job covers it.
- **Backend**: `internal/entry` gains the `balance-series` handler/service.
  It computes an opening balance at the month's first local midnight, then
  walks that month's non-deleted entries once in booking order, re-anchoring
  on any `balance_adjustment` inside the window per the existing
  `Balance()` rule, emitting a point at each day boundary — one opening
  query plus an in-memory walk, not one query per day. Both storage
  backends (`postgres`, `memory`) get the supporting read. Reuses the
  existing narrow read-only dependency on `internal/settings` for the
  caller's timezone.
- **Frontend**: new `src/components/charts/LineChart.tsx` and a shared
  `src/components/charts/` internal (extracted from `BarChart.tsx`, which is
  refactored to consume it — no behavioural change to the bar chart);
  `src/routes/accounts.$accountId.index.tsx` fetches
  `/api/entries/balance-series` and renders the line chart plus a
  month-switcher; new i18n keys in `en.json` / `de.json`; `frontend/AGENTS.md`
  Charts section updated.
- **No data migration** — the endpoint is a new read over existing rows.

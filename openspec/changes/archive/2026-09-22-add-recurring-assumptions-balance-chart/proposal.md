## Why

The recurring-transaction preview feature (`GET /api/recurring-transactions/
preview`, `add-recurring-transaction-preview`) already lets a visitor
overlay upcoming recurring occurrences on `/entries`, `/reports`, and the
dashboard's `entry_list`/`bar_chart` cards via a `show_recurring_preview`
flag. The one running-balance surface in the app — the account details
page's balance line chart, and its dashboard analogue, the `line_chart`
card — has no equivalent. A visitor can see *that* money is coming
(the Upcoming block, the stacked bar segment) but never *what it does to
the balance line they're actually looking at*.

`line-chart-overlay-and-card`'s design explicitly scoped this out ("No
`show_recurring_preview` on `line_chart` — a running balance isn't an
income/outcome bucket, so the projected-segment feature has no equivalent
shape here"), reasonably, since `LineChart` didn't yet have the tooltip
mechanism `BarChart`'s projected-segment feature leans on. That gap is
closed now — `LineChart` has its own floating tooltip and the shared
`useOverlayTooltipPosition` hook. This proposal revisits that non-goal: a
running balance *can* be projected forward, the same way a bar's income/
outcome can, by treating each upcoming recurring occurrence as one more
delta on top of the real running sum.

## What Changes

- **`show_recurring_preview` becomes valid on `line_chart` cards**
  (`backend/internal/dashboard`'s `validateShape`), alongside its existing
  `entry_list`/`bar_chart` support. No new config field — the same boolean,
  same default (`false`).
- **`LineChart` gains an optional projected line segment**: a series may
  carry a `projectedStrokeClassName`/`projectedLabel` pair (mirroring
  `BarChart`'s `projectedFillClassName`/`projectedLabel`), and a datum may
  carry a parallel `projectedValues` array. Where present, a second, dashed
  line is drawn alongside the real one, sharing its anchor point (the first
  point both a real and projected value are equal — "today" — so the dashed
  line visually continues from the solid one, not floats separately), with
  its own legend entry and tooltip row.
- **The account details page's balance chart** (`/accounts/{id}`) gains a
  "Show recurring assumptions" toggle next to the month pager. When on, it
  fetches `GET /api/recurring-transactions/preview` for that one account
  (bounded by the visitor's `recurring_preview_horizon` setting, the same
  cutoff `bar_chart`'s toggle already uses — no page-level date filter to
  intersect with) and renders the projected balance as the chart's dashed
  overlay.
- **The dashboard `line_chart` card** gains the same toggle in
  `CardFormDialog`, and `LineChartCard` renders the same projected overlay
  when `config.show_recurring_preview` is set.
- **A new shared helper**, `cumulativePreviewDeltaByPeriod` in
  `frontend/src/lib/recurringPreview.ts`, turns a flat list of preview items
  into a per-currency running total aligned to a balance chart's own day
  points — used by both new call sites instead of duplicating the math.
- No backend endpoint changes: `GET /api/entries/balance-series` is
  unchanged, and the projection is composed client-side from the existing
  preview endpoint's flat item list, the same pattern `BarChartCard`
  already established for stacked bar segments.

## Capabilities

### Modified Capabilities

- `dashboard-cards`: `show_recurring_preview` becomes valid on `line_chart`
  (in addition to `entry_list`/`bar_chart`).
- `web-client-accounts`: the running-balance line chart gains the toggle
  and its projected overlay.
- `web-client-home`: the `line_chart` card gains the same toggle and
  overlay, mirroring `bar_chart`'s existing recurring-preview affordance.

## Impact

- **Backend**: `internal/dashboard/dashboard.go`'s `validateShape` and its
  `Config` doc comment; `openapi/openapi.yaml`'s
  `DashboardCardConfig.show_recurring_preview` description (widened to
  mention `line_chart`'s effect), synced to `backend/openapi.yaml`, with
  `frontend/src/api/schema.d.ts` regenerated (description text only — no
  schema shape changes, so no other drift). New/updated unit tests in
  `backend/internal/dashboard/service_test.go`.
- **Frontend**: `charts/LineChart.tsx` gains the projected-segment
  capability; a new `cumulativePreviewDeltaByPeriod` helper in
  `lib/recurringPreview.ts`; `routes/accounts.$accountId.index.tsx` and
  `components/dashboard/LineChartCard.tsx` each gain the fetch/toggle/
  overlay wiring; `components/dashboard/CardFormDialog.tsx`'s existing
  `show_recurring_preview` toggle widens to include `line_chart`. New i18n
  keys in `en.json` then `de.json`.
- No new dependencies, no new endpoint, no migration.

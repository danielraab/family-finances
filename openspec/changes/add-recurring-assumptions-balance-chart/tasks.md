## 1. Backend: allow show_recurring_preview on line_chart

- [x] 1.1 In `backend/internal/dashboard/dashboard.go`, remove
  `c.ShowRecurringPreview != nil` from `validateShape`'s `CardTypeLineChart`
  rejection list (line ~177) so it's accepted like `AccountID`/`Title`
  already are.
- [x] 1.2 Update `Config`'s doc comment (~line 86-93): fold `line_chart`
  into the `ShowRecurringPreview` bullet ("entry_list, bar_chart, and
  line_chart accept ShowRecurringPreview") instead of listing it only on
  `entry_list`/`bar_chart`.
- [x] 1.3 In `backend/internal/dashboard/service_test.go`, extend
  `TestServiceCreateShowRecurringPreviewAllowedOnEntryListAndBarChart`
  (renamed to `...AndLineChart`) with a `line_chart` case
  (`Config{ShowRecurringPreview: boolPtr(true)}`, expect success). Added
  `TestServiceCreateLineChartWithShowRecurringPreviewStillRejectsForeignFields`
  asserting a `line_chart` card with both `ShowRecurringPreview` and a
  foreign field (`CategoryID`) set is still rejected.

## 2. API contract

- [x] 2.1 In `openapi/openapi.yaml`, widened `DashboardCardConfig.
  show_recurring_preview`'s description to say "entry_list/bar_chart/
  line_chart" and added a clause describing the line_chart effect (a
  projected balance line, not an Upcoming block or stacked bar segment).
- [x] 2.2 Ran `cd backend && go generate ./...` to sync
  `backend/openapi.yaml` — byte-identical to the source of truth.
- [x] 2.3 Ran `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`. Confirmed no shape diff beyond the description
  text.

## 3. Frontend: shared cumulative-delta helper

- [x] 3.1 In `frontend/src/lib/recurringPreview.ts`, added
  `cumulativePreviewDeltaByPeriod(items, periods, today, cutoff)`
  returning `Record<currency, (number | undefined)[]>`: for each currency
  present in `items`, index `i` is `undefined` when `periods[i] < today`
  or `periods[i] > cutoff`, else the sum of `item.amount` for that
  currency's non-overdue items with
  `today <= item.booking_timestamp < periods[i]` (see design.md's
  half-open-interval decision).
  **(Discovered during implementation.)** The `cutoff` parameter wasn't in
  the original design: without it, a period beyond the preview fetch's own
  `to` bound (e.g. paging the account page's month pager into a future
  month past the visitor's `recurring_preview_horizon`) held the last
  known cumulative value flat forever, implying "no further activity"
  rather than "no further data fetched" — verified live (see task 9.3) by
  paging a `line_chart` to a month beyond the horizon before this fix,
  where the dashed line wrongly continued flat instead of stopping. Both
  call sites (tasks 5, 6) now pass their resolved `previewCutoff` through.
- [x] 3.2 Verified via the manual browser checks in task 9: a period equal
  to today yields `0` for every currency with a matching non-overdue item
  dated today or later; a period before today or after the cutoff yields
  `undefined`; an overdue item is never included.

## 4. Frontend: LineChart projected-segment capability

- [x] 4.1 In `frontend/src/components/charts/LineChart.tsx`, added
  `projectedStrokeClassName?: string` and `projectedLabel?: string` to
  `LineChartSeries`, and `projectedValues?: number[]` to `LineChartDatum`
  — mirroring `BarChart.tsx`'s `projectedFillClassName`/`projectedLabel`/
  `BarChartDatum.projectedValues`.
- [x] 4.2 Rendered a second `<path>` per series with
  `projectedStrokeClassName`, built the same stepAfter way as the real
  path but only over the indices where that series' `projectedValues[i]`
  is defined, with `strokeDasharray` and `projectedStrokeClassName` in
  place of the solid path's props. Also widened the y-axis domain
  (`finiteValues`) to include `projectedValues` so the projected line is
  always in frame.
- [x] 4.3 Widened the legend's render condition from `series.length > 1`
  to `series.length > 1 || series.some((s) => s.projectedStrokeClassName
  && s.projectedLabel)`, and added a second legend pass (mirroring
  `BarChart.tsx`'s) listing each series with both fields set, using a
  dashed-line swatch.
- [x] 4.4 Added a projected tooltip row per such series
  (`tooltipDatum.projectedValues?.[i]`), shown only when defined —
  mirroring `BarChart.tsx`'s projected tooltip row.
- [x] 4.5 Extended each point's `aria-label` to append the projected value
  when present, mirroring `BarChart.tsx`'s group `aria-label`.

## 5. Frontend: account details page toggle

- [x] 5.1 In `frontend/src/routes/accounts.$accountId.index.tsx`, added
  `showRecurringAssumptions` local `useState(false)` (not a URL param —
  see design.md) and rendered its checkbox beside the month pager, visible
  only when `chartView === "balance"`.
- [x] 5.2 Added a `previewItems` fetch effect (mirroring `BarChartCard.tsx`'s
  shape): when `chartView === "balance"` and a `previewCutoff` is resolved
  (only when the toggle is on), fetch `GET /api/recurring-transactions/
  preview` via `fetchRecurringPreview` scoped to `[accountId]`. Clear
  `previewItems` when the toggle or chart view turns off.
- [x] 5.3 Built `balanceData`'s `projectedValues` using
  `cumulativePreviewDeltaByPeriod(previewItems, points.map(p => p.period),
  todayDateString(today), previewCutoff)`, added onto each point's real
  value for `account.currency`, only when the toggle is on.
- [x] 5.4 Extended `balanceSeries`'s single entry with
  `projectedStrokeClassName`/`projectedLabel` (the new
  `dashboard.lineChart.projectedBalance` key) only when the toggle is on.

## 6. Frontend: dashboard line_chart card

- [x] 6.1 In `frontend/src/components/dashboard/LineChartCard.tsx`, added
  the same preview fetch (`useRecurringPreviewHorizon`,
  `fetchRecurringPreview` scoped to `config.account_id ?
  [config.account_id] : undefined`, cutoff via `resolvePreviewCutoff(
  undefined, horizon, today)`) gated on `config.show_recurring_preview`,
  mirroring `BarChartCard.tsx`'s existing effect shape.
- [x] 6.2 Extended `dataFor(code)` to add `projectedValues` via
  `cumulativePreviewDeltaByPeriod`, and extended the per-currency `series`
  array with `projectedStrokeClassName`/`projectedLabel` when
  `config.show_recurring_preview` is set — mirroring `BarChartCard.tsx`'s
  `series`/`dataFor` conditional shape.

## 7. Frontend: wire the toggle into CardFormDialog

- [x] 7.1 In `frontend/src/components/dashboard/CardFormDialog.tsx`, widened
  both the render condition (~line 329) and the `submit()` config-builder
  condition (~line 143-146) for `show_recurring_preview` from
  `type === "entry_list" || type === "bar_chart"` to also include
  `type === "line_chart"`. Reused the existing
  `dashboard.addCard.showRecurringPreviewLabel` string — no new label
  needed here (see design.md).

## 8. i18n

- [x] 8.1 Added `accounts.details.balanceChart.showRecurringAssumptions`
  ("Show recurring assumptions") and `dashboard.lineChart.projectedBalance`
  ("Balance (projected)") to `frontend/src/i18n/locales/en.json` and
  `de.json`.

## 9. Verification

- [x] 9.1 Backend: `gofmt -l .`, `go vet ./...`, `go test ./...` (from
  `backend/`) — all pass.
- [x] 9.2 Frontend: `pnpm lint`, `pnpm exec tsc`, `pnpm build` (from
  `frontend/`) — all pass, lint/format back at the pre-existing baseline
  (0 errors).
- [x] 9.3 Manual check, live in a real browser (headless Chromium via
  Playwright) against the dev server with a local Postgres, seeded via
  `server seed --yes --entries 40 <email>` (whose fixtures always include
  an upcoming, non-overdue recurring transaction — see `backend/AGENTS.md`
  "Seeding fake data"): `/accounts/{id}`'s balance chart's "Show recurring
  assumptions" toggle draws a dashed "Balance (projected)" line from
  today's point that steps down exactly when a non-overdue recurring
  transaction's date passes (verified against the raw
  `GET /api/recurring-transactions/preview` and `GET /api/entries/
  balance-series` responses — the pixel-level step matched the expected
  cents delta precisely), with a legend distinguishing solid "Balance"
  from dashed "Balance (projected)", and reverts cleanly when toggled off.
  Paging the chart to a future month beyond the resolved horizon correctly
  shows no dashed line at all (see task 3.1's discovered cutoff fix) rather
  than continuing the last known projection. A `line_chart` dashboard card
  created via "Add card" with the toggle enabled shows the identical
  overlay; a `line_chart` card without the toggle is visually unchanged
  from before this change; the existing `bar_chart` projected-segment
  feature and the Upcoming block were confirmed unaffected.
- [x] 9.4 Confirmed no drift: `openapi/openapi.yaml` and
  `backend/openapi.yaml` are byte-identical, and re-running
  `pnpm generate:api` against the current spec produces no further diff in
  `schema.d.ts`.

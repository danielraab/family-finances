## 1. Backend: `line_chart` card type

- [ ] 1.1 In `backend/internal/dashboard/dashboard.go`, add
  `CardTypeLineChart CardType = "line_chart"` (doc comment: "shows a
  running-balance line chart for an account or every account") and add it
  to `CardType.valid()`.
- [ ] 1.2 Add a `case CardTypeLineChart` to `validateShape` rejecting
  `CategoryID`, `IncludeSubcategories`, `TagID`, `Range`, `Unit`,
  `Columns`, and `ShowRecurringPreview` whenever any is non-nil; an empty
  config (no `AccountID`, no `Title`) is valid.
- [ ] 1.3 Update `Config`'s doc comment table to add the `line_chart` row
  (`AccountID` and `Title` only, both optional).
- [ ] 1.4 Add/extend unit tests in
  `backend/internal/dashboard/dashboard_test.go` (or wherever
  `validateShape`/`CardType.valid()` are already tested): a `line_chart`
  card with no config is accepted; one with `category_id`, `tag_id`,
  `unit`, or `columns` set is rejected (`ErrInvalidValue`); one with
  `account_id` and/or `title` set is accepted; `POST /api/dashboard/cards`
  with `type: "line_chart"` end-to-end via the handler test, mirroring the
  existing `bar_chart` handler tests.

## 2. API contract

- [ ] 2.1 In `openapi/openapi.yaml`, add `line_chart` to
  `DashboardCardType`'s `enum` and extend its `description` with the same
  one-line summary `account_stat`/`query_stat`/`entry_list`/`bar_chart`
  each get.
- [ ] 2.2 Run `cd backend && go generate ./...` to sync
  `backend/openapi.yaml` from the edited source of truth.
- [ ] 2.3 Run `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`. Confirm no other generated file drifts.

## 3. Backend: seed fixtures

- [ ] 3.1 In `backend/internal/cli/fixtures.go`'s `generateDashboardCards`,
  add one more `dashboardSvc.Create` call after the `bar_chart` card:
  `Type: dashboard.CardTypeLineChart`, empty `Config{}` (every account,
  summed per currency). Update the function's doc comment to mention it.

## 4. Frontend: shared overlay-tooltip positioning hook

- [ ] 4.1 In `frontend/src/components/charts/internal.ts`, add
  `useOverlayTooltipPosition({ wrapperRef, tooltipRef, visible, anchorSelector })`
  — the `useLayoutEffect` currently inline in `BarChart.tsx` (measure the
  element matching `anchorSelector` via `wrapper.querySelector`, compute a
  horizontally-clamped `left` and a `top` anchored to its top edge relative
  to `wrapperRef`, re-run on `scroll`/`resize` while `visible`), returning
  `{ left, top } | null`. `anchorSelector: string | null` — `null` (and a
  `null` return) when nothing is active, mirroring the current
  `activeIndex === null` guard.

## 5. Frontend: `BarChart` uses the shared hook

- [ ] 5.1 In `frontend/src/components/charts/BarChart.tsx`, replace the
  inline `useLayoutEffect` (currently setting `tooltipPosition`) with a
  call to `useOverlayTooltipPosition`, passing
  `anchorSelector={activeIndex !== null ? \`[data-group-index="${activeIndex}"]\` : null}`
  and `visible={tooltipVisible}`. No behavior change — verify the tooltip
  still anchors, clamps, and tracks scroll/resize identically.

## 6. Frontend: `LineChart` adopts the overlay tooltip

- [ ] 6.1 Restructure `LineChart.tsx`'s markup to match `BarChart.tsx`'s
  wrapper shape: an outer `wrapperRef` div (`className="relative"`)
  containing the existing `containerRef`/`overflow-x-auto` div, with the
  new floating tooltip as its sibling inside `wrapperRef`.
- [ ] 6.2 Add `data-point-index={pointIndex}` to each point's `<g>`
  element (the one already carrying `tabIndex`/`role="button"`/the
  hover/click handlers).
- [ ] 6.3 Delete the always-rendered bottom value box (the `<div
  aria-hidden={!tooltipVisible} className="... invisible">` block and its
  contents).
- [ ] 6.4 Add a `tooltipRef` and render a floating tooltip in `BarChart`'s
  shape (`role="status"`, positioned via `left`/`top` from the hook minus
  `TOOLTIP_GAP`, `-translate-x-1/2 -translate-y-full`, `invisible` until
  positioned, `pointer-events-none`), conditionally rendered only when
  `tooltipVisible && tooltipDatum` (drop the old always-mounted
  placeholder pattern). Content: the category, then one row per series
  (swatch + label + `formatValue`), same data `tooltipDatum` already
  provides — no projected-segment handling needed (that's `BarChart`-only).
- [ ] 6.5 Call `useOverlayTooltipPosition` with
  `anchorSelector={activeIndex !== null ? \`[data-point-index="${activeIndex}"]\` : null}`
  and `visible={tooltipVisible}`.
- [ ] 6.6 Confirm the legend (series.length > 1 only) is untouched, and
  that `usePinnableSelection`'s pin/click-away/Escape behavior still
  drives `activeIndex` exactly as before.

## 7. Frontend: `LineChartCard` dashboard card

- [ ] 7.1 Create `frontend/src/components/dashboard/LineChartCard.tsx`,
  mirroring `BarChartCard.tsx`'s structure: props
  `{ config, accounts, categories, tags, displayedDecimalPlaces, locale }`,
  `MissingReferenceCard` when `cardReferencesResolve` fails.
- [ ] 7.2 State: `year`/`month` (default: current month/year), fetched
  `balancePoints`. Effect fetches `GET /api/entries/balance-series` with
  `account_id: config.account_id ? [config.account_id] : undefined`,
  `unit: "day"`, `year`, `month`.
- [ ] 7.3 Previous/next-month pager (roll the year at the Dec/Jan
  boundary), not persisted across reloads — mirror
  `accounts.$accountId.index.tsx`'s existing balance-chart pager.
- [ ] 7.4 Derive currencies from the response's `balances[].currency`;
  render one `<LineChart>` per currency (headed by its code when more than
  one), each with a single series
  `{ label: t("accounts.details.balanceChart.series"), strokeClassName: BALANCE_STROKE, dotClassName: BALANCE_DOT }`
  (redeclare the same color constants `accounts.$accountId.index.tsx`
  uses, matching how `BarChartCard.tsx` redeclares
  `INCOME_FILL`/`OUTCOME_FILL` rather than importing them).
- [ ] 7.5 Map `BalancePoint[]` to `LineChartDatum[]` the same way
  `accounts.$accountId.index.tsx` already does (day-of-month category
  label, closing-boundary point labelled with month+day).
- [ ] 7.6 Heading: `<CardTitleLink config={config} accounts={accounts} categories={categories} tags={tags} allLabel={t("dashboard.filterAllAccounts")} />`,
  identical usage to `BarChartCard`.

## 8. Frontend: wire the new type into the dashboard shell

- [ ] 8.1 In `frontend/src/components/dashboard/CardFormDialog.tsx`, add
  `"line_chart"` to `CARD_TYPES`.
- [ ] 8.2 Widen the category/tag/`include_subcategories` JSX block's
  condition from `type !== "account_stat"` to
  `type !== "account_stat" && type !== "line_chart"`.
- [ ] 8.3 In `submit()`'s `config` builder, apply the same
  `type !== "account_stat" && type !== "line_chart"` guard to
  `category_id`, `include_subcategories`, and `tag_id` so a `line_chart`
  submission never sends fields its type rejects. Leave `title`/
  `account_id` as they are (already correct for `line_chart` unchanged).
- [ ] 8.4 In `frontend/src/routes/home.tsx`, add a `case "line_chart"` to
  `renderCardContent` rendering `<LineChartCard>` with the same props
  `BarChartCard` receives.
- [ ] 8.5 Widen `segmentCards`'s full-width check from
  `card.type === "bar_chart"` to
  `card.type === "bar_chart" || card.type === "line_chart"`.

## 9. i18n

- [ ] 9.1 Add `dashboard.cardTypes.line_chart` (type-picker label) to
  `frontend/src/i18n/locales/en.json` and `de.json`.
- [ ] 9.2 Add `dashboard.lineChart.previous`/`dashboard.lineChart.next`
  (pager `aria-label`s, mirroring `dashboard.barChart.previous`/`.next`)
  to both locale files.

## 10. Verify

- [ ] 10.1 `cd backend && gofmt -l .` prints nothing.
- [ ] 10.2 `cd backend && go vet ./...` passes.
- [ ] 10.3 `cd backend && go test ./...` passes.
- [ ] 10.4 `cd frontend && pnpm lint` passes.
- [ ] 10.5 `cd frontend && pnpm exec tsc` passes.
- [ ] 10.6 `cd frontend && pnpm build` writes `out/index.html`.
- [ ] 10.7 Manual check: `/accounts/{id}`'s balance chart shows the
  floating tooltip on hover/tap, pins on click, releases on click-away/
  Escape/re-click, and no longer reserves a fixed box below the chart.
- [ ] 10.8 Manual check: `/home` in edit mode can add a `line_chart` card
  (unfiltered, and scoped to one account), it renders full width with a
  working month pager, and its heading links through to `/reports`.

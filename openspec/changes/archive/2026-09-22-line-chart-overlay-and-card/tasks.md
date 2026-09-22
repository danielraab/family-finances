## 1. Backend: `line_chart` card type

- [x] 1.1 In `backend/internal/dashboard/dashboard.go`, add
  `CardTypeLineChart CardType = "line_chart"` (doc comment: "shows a
  running-balance line chart for an account or every account") and add it
  to `CardType.valid()`.
- [x] 1.2 Add a `case CardTypeLineChart` to `validateShape` rejecting
  `CategoryID`, `IncludeSubcategories`, `TagID`, `Range`, `Unit`,
  `Columns`, and `ShowRecurringPreview` whenever any is non-nil; an empty
  config (no `AccountID`, no `Title`) is valid.
- [x] 1.3 Update `Config`'s doc comment table to add the `line_chart` row
  (`AccountID` and `Title` only, both optional).
- [x] 1.4 Add/extend unit tests in
  `backend/internal/dashboard/dashboard_test.go` (or wherever
  `validateShape`/`CardType.valid()` are already tested): a `line_chart`
  card with no config is accepted; one with `category_id`, `tag_id`,
  `unit`, or `columns` set is rejected (`ErrInvalidValue`); one with
  `account_id` and/or `title` set is accepted; `POST /api/dashboard/cards`
  with `type: "line_chart"` end-to-end via the handler test, mirroring the
  existing `bar_chart` handler tests.

- [x] 1.5 (Discovered during implementation.) `dashboard_cards.type` also
  carries a Postgres `CHECK` constraint
  (`backend/internal/storage/postgres/migrations/0027_dashboard_cards.sql`)
  independent of the Go-level enum — add a new forward-only migration
  (`0029_dashboard_cards_line_chart.sql`) dropping and re-adding it with
  `line_chart` included, per `backend/AGENTS.md`'s "forward-only" rule.

## 2. API contract

- [x] 2.1 In `openapi/openapi.yaml`, add `line_chart` to
  `DashboardCardType`'s `enum` and extend its `description` with the same
  one-line summary `account_stat`/`query_stat`/`entry_list`/`bar_chart`
  each get.
- [x] 2.2 Run `cd backend && go generate ./...` to sync
  `backend/openapi.yaml` from the edited source of truth.
- [x] 2.3 Run `cd frontend && pnpm generate:api` to regenerate
  `src/api/schema.d.ts`. Confirm no other generated file drifts.

## 3. Backend: seed fixtures

- [x] 3.1 In `backend/internal/cli/fixtures.go`'s `generateDashboardCards`,
  add one more `dashboardSvc.Create` call after the `bar_chart` card:
  `Type: dashboard.CardTypeLineChart`. Update the function's doc comment
  to mention it.
- [x] 3.2 (Refined after initial implementation.) Scope the seeded
  `line_chart` card to `accountIDs[0]` rather than leaving it unfiltered
  — a generated account's currency is drawn randomly from a 3-currency
  pool, so an unfiltered card often rendered two or three stacked
  per-currency line graphs instead of the single demonstrative one a
  fresh dashboard should show. Update `backend/AGENTS.md`'s "Seeding
  fake data" section and `TestGenerateFixturesCreatesDashboardCards` to
  match.

## 4. Frontend: shared overlay-tooltip positioning hook

- [x] 4.1 In `frontend/src/components/charts/internal.ts`, add
  `useOverlayTooltipPosition({ wrapperRef, tooltipRef, visible, anchorSelector })`
  — the `useLayoutEffect` currently inline in `BarChart.tsx` (measure the
  element matching `anchorSelector` via `wrapper.querySelector`, compute a
  horizontally-clamped `left` and a `top` anchored to its top edge relative
  to `wrapperRef`, re-run on `scroll`/`resize` while `visible`), returning
  `{ left, top } | null`. `anchorSelector: string | null` — `null` (and a
  `null` return) when nothing is active, mirroring the current
  `activeIndex === null` guard.

## 5. Frontend: `BarChart` uses the shared hook

- [x] 5.1 In `frontend/src/components/charts/BarChart.tsx`, replace the
  inline `useLayoutEffect` (currently setting `tooltipPosition`) with a
  call to `useOverlayTooltipPosition`, passing
  `anchorSelector={activeIndex !== null ? \`[data-group-index="${activeIndex}"]\` : null}`
  and `visible={tooltipVisible}`. No behavior change — verify the tooltip
  still anchors, clamps, and tracks scroll/resize identically.

## 6. Frontend: `LineChart` adopts the overlay tooltip

- [x] 6.1 Restructure `LineChart.tsx`'s markup to match `BarChart.tsx`'s
  wrapper shape: an outer `wrapperRef` div (`className="relative"`)
  containing the existing `containerRef`/`overflow-x-auto` div, with the
  new floating tooltip as its sibling inside `wrapperRef`.
- [x] 6.2 Add `data-point-index={pointIndex}` to each point's `<g>`
  element (the one already carrying `tabIndex`/`role="button"`/the
  hover/click handlers).
- [x] 6.3 Delete the always-rendered bottom value box (the `<div
  aria-hidden={!tooltipVisible} className="... invisible">` block and its
  contents).
- [x] 6.4 Add a `tooltipRef` and render a floating tooltip in `BarChart`'s
  shape (`role="status"`, positioned via `left`/`top` from the hook minus
  `TOOLTIP_GAP`, `-translate-x-1/2 -translate-y-full`, `invisible` until
  positioned, `pointer-events-none`), conditionally rendered only when
  `tooltipVisible && tooltipDatum` (drop the old always-mounted
  placeholder pattern). Content: the category, then one row per series
  (swatch + label + `formatValue`), same data `tooltipDatum` already
  provides — no projected-segment handling needed (that's `BarChart`-only).
- [x] 6.5 Call `useOverlayTooltipPosition` with
  `anchorSelector={activeIndex !== null ? \`[data-point-index="${activeIndex}"]\` : null}`
  and `visible={tooltipVisible}`.
- [x] 6.6 Confirm the legend (series.length > 1 only) is untouched, and
  that `usePinnableSelection`'s pin/click-away/Escape behavior still
  drives `activeIndex` exactly as before.

## 7. Frontend: `LineChartCard` dashboard card

- [x] 7.1 Create `frontend/src/components/dashboard/LineChartCard.tsx`,
  mirroring `BarChartCard.tsx`'s structure: props
  `{ config, accounts, categories, tags, displayedDecimalPlaces, locale }`,
  `MissingReferenceCard` when `cardReferencesResolve` fails.
- [x] 7.2 State: `year`/`month` (default: current month/year), fetched
  `balancePoints`. Effect fetches `GET /api/entries/balance-series` with
  `account_id: config.account_id ? [config.account_id] : undefined`,
  `unit: "day"`, `year`, `month`.
- [x] 7.3 Previous/next-month pager (roll the year at the Dec/Jan
  boundary), not persisted across reloads — mirror
  `accounts.$accountId.index.tsx`'s existing balance-chart pager.
- [x] 7.4 Derive currencies from the response's `balances[].currency`;
  render one `<LineChart>` per currency (headed by its code when more than
  one), each with a single series
  `{ label: t("accounts.details.balanceChart.series"), strokeClassName: BALANCE_STROKE, dotClassName: BALANCE_DOT }`
  (redeclare the same color constants `accounts.$accountId.index.tsx`
  uses, matching how `BarChartCard.tsx` redeclares
  `INCOME_FILL`/`OUTCOME_FILL` rather than importing them).
- [x] 7.5 Map `BalancePoint[]` to `LineChartDatum[]` the same way
  `accounts.$accountId.index.tsx` already does (day-of-month category
  label, closing-boundary point labelled with month+day).
- [x] 7.6 Heading: `<CardTitleLink config={config} accounts={accounts} categories={categories} tags={tags} allLabel={t("dashboard.filterAllAccounts")} />`,
  identical usage to `BarChartCard`.

## 8. Frontend: wire the new type into the dashboard shell

- [x] 8.1 In `frontend/src/components/dashboard/CardFormDialog.tsx`, add
  `"line_chart"` to `CARD_TYPES`.
- [x] 8.2 Widen the category/tag/`include_subcategories` JSX block's
  condition from `type !== "account_stat"` to
  `type !== "account_stat" && type !== "line_chart"`.
- [x] 8.3 In `submit()`'s `config` builder, apply the same
  `type !== "account_stat" && type !== "line_chart"` guard to
  `category_id`, `include_subcategories`, and `tag_id` so a `line_chart`
  submission never sends fields its type rejects. Leave `title`/
  `account_id` as they are (already correct for `line_chart` unchanged).
- [x] 8.4 In `frontend/src/routes/home.tsx`, add a `case "line_chart"` to
  `renderCardContent` rendering `<LineChartCard>` with the same props
  `BarChartCard` receives.
- [x] 8.5 Widen `segmentCards`'s full-width check from
  `card.type === "bar_chart"` to
  `card.type === "bar_chart" || card.type === "line_chart"`.

## 9. i18n

- [x] 9.1 Add `dashboard.cardTypes.line_chart` (type-picker label) to
  `frontend/src/i18n/locales/en.json` and `de.json`.
- [x] 9.2 Add `dashboard.lineChart.previous`/`dashboard.lineChart.next`
  (pager `aria-label`s, mirroring `dashboard.barChart.previous`/`.next`)
  to both locale files.

## 10. Verify

- [x] 10.1 `cd backend && gofmt -l .` prints nothing.
- [x] 10.2 `cd backend && go vet ./...` passes.
- [x] 10.3 `cd backend && go test ./...` passes.
- [x] 10.4 `cd frontend && pnpm lint` passes.
- [x] 10.5 `cd frontend && pnpm exec tsc` passes.
- [x] 10.6 `cd frontend && pnpm build` writes `out/index.html`.
- [x] 10.7 Manual check: `/accounts/{id}`'s balance chart shows the
  floating tooltip on hover/tap, pins on click, releases on click-away/
  Escape/re-click, and no longer reserves a fixed box below the chart.
  Verified live (local Postgres, seeded user, Playwright/Chromium):
  hover anchors the tooltip above the point; click pins it; clicking away
  releases it; hovering the last point clamps the tooltip horizontally
  within the chart instead of spilling off-screen.
- [x] 10.8 Manual check: `/home` in edit mode can add a `line_chart` card
  (unfiltered, and scoped to one account), it renders full width with a
  working month pager, and its heading links through to `/reports`.
  Verified live: the type picker offers "Line chart" with only
  title/account fields (no category/tag/date-range); the seeded and a
  newly-added unfiltered card both render full width, one per row, with
  EUR/USD stacked charts and a working month pager; activating the
  heading navigates to `/reports`.

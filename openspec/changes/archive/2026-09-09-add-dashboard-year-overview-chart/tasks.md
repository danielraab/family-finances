## 1. i18n key move

- [x] 1.1 In `frontend/src/i18n/locales/en.json`, add a top-level
  `flowChart` object with `title`, `income`, `outcome`, `previousYear`,
  `nextYear` copied from `accounts.details.chart.*`, then remove the
  `accounts.details.chart` sub-object.
- [x] 1.2 Do the same rename in `frontend/src/i18n/locales/de.json`
  (translations move verbatim, no wording change).

## 2. Extract the `<FlowChart>` component

- [x] 2.1 Create `frontend/src/components/FlowChart.tsx` with props
  `{ accountIds?: string[]; currency?: string; displayedDecimalPlaces: number }`.
  Move `INCOME_FILL` / `OUTCOME_FILL` here from
  `accounts.$accountId.index.tsx`.
- [x] 2.2 Implement `chartYear` state (default `new Date().getFullYear()`)
  and the previous/next-year pager UI, lifted from
  `accounts.$accountId.index.tsx`, using the new `flowChart.*` keys.
- [x] 2.3 Implement the fetch effect: `GET /api/entries/flow-summary` with
  `query: { unit: "month", year: chartYear, ...(accountIds ? { account_id: accountIds } : {}) }`.
  Depend on `(accountIds ?? []).join(",")` and `chartYear` — not the array
  identity. Track buckets in state, render `null` until first response.
- [x] 2.4 Implement the `FlowBucket`→`BarChartDatum` mapping (localized
  short-month `category`, `values: [income, outcome]`) as a helper that
  takes the target currency, reused for both modes.
- [x] 2.5 When `currency` is set, render one `BarChart` filtered to it
  (no heading). When `currency` is omitted, derive
  `[...new Set(buckets.flatMap(b => [...b.income, ...b.outcome].map(s => s.currency)))].sort()`
  and render one `BarChart` per currency, each under an `<h3>` currency-code
  heading. `formatValue` uses `formatAmount(v, currency, displayedDecimalPlaces, locale)`.
- [x] 2.6 Empty case: when `currency` is omitted and no currencies are
  present, render nothing (or the same `null` as pre-load).

## 3. Rewire the account-details page

- [x] 3.1 In `frontend/src/routes/accounts.$accountId.index.tsx`, delete
  the inlined flow-summary effect, `flowBuckets` state, `chartYear` state,
  `chartSeries` / `chartData`, `INCOME_FILL` / `OUTCOME_FILL`, and the
  year-pager JSX in the flow branch.
- [x] 3.2 Replace the `chartView === "flow"` render branch with
  `<FlowChart accountIds={[accountId]} currency={account.currency} displayedDecimalPlaces={displayedDecimalPlaces} />`.
  Keep the flow/balance toggle and the balance `LineChart` branch
  untouched.
- [x] 3.3 Confirm the toggle still gates rendering so `<FlowChart>` mounts
  only in the flow view.

## 4. Add the dashboard section

- [x] 4.1 In `frontend/src/routes/home.tsx`, call
  `useDisplayedDecimalPlaces()` in `HomeDashboard`.
- [x] 4.2 In the non-empty branch (after the cards grid `<div>`), add a
  `<section>` rendering `<FlowChart displayedDecimalPlaces={dp} />` with a
  heading using `flowChart.title` (or a new `dashboard.*` heading key if a
  distinct label reads better — add to both locale files if so).
- [x] 4.3 Verify the empty-accounts branch is unchanged (no chart, no year
  controls).

## 5. Verify

- [x] 5.1 `cd frontend && pnpm lint` passes.
- [x] 5.2 `cd frontend && pnpm exec tsc` passes.
- [x] 5.3 `cd frontend && pnpm build` writes `out/index.html`.
- [ ] 5.4 Manual check: `/accounts/{id}` flow chart looks and behaves as
  before (year pager, empty months at zero, account currency only).
  _Pending: needs a logged-in browser session. Vite compiles the route
  clean and the FlowBucket→bar mapping is unchanged from what shipped._
- [x] 5.5 Manual check: `/home` shows the year overview below the cards —
  one chart for a single-currency visitor, one per currency (alphabetical,
  headed by code) for a multi-currency visitor, year pager refetches all,
  and nothing renders for a visitor with no accounts.
  _Pending: needs a logged-in browser session (no browser harness in this
  repo). Vite compiles `home.tsx` + `FlowChart.tsx` with no errors._

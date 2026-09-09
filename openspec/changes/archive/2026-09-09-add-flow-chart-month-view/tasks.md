## 1. API contract

- [x] 1.1 Add `GET /api/entries/balance-series` to `openapi/openapi.yaml`:
  `operationId: getEntriesBalanceSeries`, tag `entries`, params `account_id`
  (repeatable, optional), `unit` (enum `[day]`, required), `year` (integer,
  required), `month` (integer 1–12, required); `200` returns
  `{ points: [ { period: date, balances: [CurrencySum] } ] }`; document that
  `category_id` / `tag_id` / `from` / `to` / `q` are not accepted and yield
  `400`. Reuse the existing `CurrencySum` schema.
- [x] 1.2 Regenerate the committed copies: `cd backend && go generate ./...`
  (updates `backend/openapi.yaml`) and `cd frontend && pnpm generate:api`
  (updates `frontend/src/api/schema.d.ts`). Commit both.
- [x] 1.3 `cd openapi` (or repo root) run the spec lint used by CI's
  `contract` job; confirm no drift.

## 2. Backend — service

- [x] 2.1 In `internal/entry`, add a `BalanceFilter` (AccountIDs, Year,
  Month, Unit, Timezone) and `BalancePoint` / per-currency result types
  beside the existing `FlowFilter` / `FlowBucket`.
- [x] 2.2 Add `Service.BalanceSeries(ctx, ownerID, f)`: validate `Unit ==
  day`, `Month` 1–12, `Year` ≥ 1 (`ErrInvalidValue` otherwise); resolve
  visible account IDs and intersect with `f.AccountIDs` exactly as
  `FlowSummary` does; resolve the caller's timezone via `s.timezones`
  (default `"UTC"`).
- [x] 2.3 Compute the day boundaries: `00:00` local on days 1..N of the
  month plus `00:00` local on the 1st of the following month. For each
  boundary and each matching account call the existing
  `Store.Balance(ctx, accountID, asOf)`; sum results per account currency
  (resolve currency via `s.accounts.Owner`, cached per account like
  `FlowSummary`).
- [x] 2.4 Build one `BalancePoint` per boundary, label = `YYYY-MM-DD` of the
  point's local day, `balances` listing every currency present (including a
  `0` — unlike `FlowSummary`, do not omit zero). Every point is present even
  for a no-activity month.
- [x] 2.5 Unit-test `BalanceSeries`: point count (31-day month → 32 points),
  first point = prior-history balance, a transaction shifts the next day's
  point, a mid-month `balance_adjustment` re-anchors every later point, a
  cross-midnight entry lands in the right day for a non-UTC timezone,
  multi-currency accounts produce separate per-currency values, no-activity
  month is a flat run, soft-deleted entry has no effect, invalid
  `unit` / missing `month` → `ErrInvalidValue`.

## 3. Backend — storage

- [x] 3.1 Confirm `Store.Balance(ctx, accountID, asOf)` already satisfies
  the per-boundary need in both `internal/storage/postgres` and
  `internal/storage/memory`; add no new store method if so. If a batched
  variant is added for query-count reasons, implement it in both backends
  with parity tests.
- [x] 3.2 Extend the storage parity / scenario tests to cover a balance
  series spanning a `balance_adjustment` inside the requested month.

## 4. Backend — handler

- [x] 4.1 Add the `balance-series` route to `internal/entry/handler.go`
  beside `flow-summary`: parse repeatable `account_id`, `unit`, `year`,
  `month`; reject `category_id` / `tag_id` / `from` / `to` / `q` with
  `400`; map `ErrInvalidValue` → `400`, unauth → `401`.
- [x] 4.2 Marshal the service result to the OpenAPI response shape
  (`points` / `period` / `balances`).
- [x] 4.3 Handler tests: happy path shape, `400` on a filtering param,
  `400` on missing `month`, `400` on `unit=month`, `401` unauthenticated,
  `account_id` for an account the caller does not own is silently ignored
  (same as `flow-summary`).

## 5. Frontend — shared chart internals

- [x] 5.1 Create `frontend/src/components/charts/` shared internals by
  extracting from `BarChart.tsx` with no behavioural change: container-width
  measurement (`ResizeObserver`), axis-tick rounding generalized to a
  signed `niceExtent(min, max)` (keep `niceMax` as a thin wrapper or
  replace its call site), and the hover / click-to-pin / Escape-to-release
  tooltip state (hook or small component).
- [x] 5.2 Refactor `BarChart.tsx` to consume the shared internals; verify
  `pnpm lint`, `pnpm exec tsc`, and the running bar chart (tooltip, pin,
  responsive width) are unchanged.

## 6. Frontend — LineChart primitive

- [x] 6.1 Add `frontend/src/components/charts/LineChart.tsx`: props
  `series: { label, strokeClassName }[]`, `data: { category, values:
  number[] }[]`, `formatValue`. Signed y-domain via `niceExtent`; zero rule
  line only when the domain spans zero; one `<path>` per series with
  `stepAfter` geometry; a dot per sample point; shared per-x tooltip
  listing every series value; responsive width; legend when 2+ series.
- [x] 6.2 Pick series stroke colors from the `dataviz` palette as Tailwind
  `stroke-[#…] dark:stroke-[#…]` classes; run
  `frontend/scripts/validate_palette.js` on the pairing.
- [x] 6.3 Update `frontend/AGENTS.md`'s Charts section to name `LineChart`
  and the shared internals alongside `BarChart`.

## 7. Frontend — account details page

- [x] 7.1 In `src/routes/accounts.$accountId.index.tsx` add `chartMonth`
  state (default current month) and a `useEffect` keyed on
  `[accountId, chartYear-for-line, chartMonth]` fetching
  `/api/entries/balance-series?account_id={id}&unit=day&year={y}&month={m}`.
- [x] 7.2 Map the response to `LineChart` data: one series in the account's
  currency, `category` = localized day-of-month label, `values` = that
  currency's balance per point; format with the existing `formatAmount` +
  `useDisplayedDecimalPlaces`.
- [x] 7.3 Render a new `<section>` below the bar chart: heading
  (`accounts.details.balanceChart.title`) plus previous/next-month controls
  mirroring the year switcher's markup and unbounded navigation (roll the
  year when crossing Jan/Dec). Render `LineChart` once the data resolves.
- [x] 7.4 Add i18n keys to `frontend/src/i18n/locales/en.json` and
  `de.json` (`accounts.details.balanceChart.title`, `.previousMonth`,
  `.nextMonth`, and any axis/legend label).

## Implementation notes

- Point count is **days-in-month + 1** (a 30-day month → 31 points), not a
  fixed 32 — the +1 is the closing boundary.
- Each daily point uses the instant `midnight − 1ns` so an entry booked
  exactly at a local midnight belongs to that day's activity (first point =
  strictly-prior history), matching `flow-summary`'s day bucketing.
- No batched `Store` method was added (task 3.1) — `Store.Balance` per
  boundary was sufficient; task 3.2's mid-month-adjustment case is covered
  by `TestBalanceSeriesMidMonthAdjustmentReanchorsEveryLaterPoint` at the
  service level, and `Store.Balance`'s own Postgres integration tests
  (`TestPGEntryBalanceLiveComputation` etc.) already exercise the
  adjustment-anchor rule the series relies on.
- `LineChart` is a single-series balance line on this page, so **no legend
  box** is shown (a legend renders only for 2+ series) — matches the
  dataviz rule "none for one series".
- `LineChartSeries` carries `strokeClassName` **and** `dotClassName` as
  separate literal strings so Tailwind's scanner emits both classes.
- The palette validator lives in the `dataviz` skill, not
  `frontend/scripts/`; blue slot 1 (`#2a78d6` / `#3987e5`) passed light and
  dark (lightness band, chroma floor, ≥3:1 contrast).
- Verified in the running app (seeded dev DB + a Sept 2026 mid-month
  `balance_adjustment` reading of €2,500): the line steps, re-anchors
  exactly to the reading, crosses zero with the stronger zero rule line,
  navigates past the current month and a no-activity month renders flat —
  light and dark.

## 8. Verification

- [x] 8.1 Backend: `cd backend && go test ./...` and the linter both pass;
  integration test against Postgres covers the new endpoint.
- [x] 8.2 Frontend: `cd frontend && pnpm lint && pnpm exec tsc && pnpm
  build` all pass.
- [x] 8.3 Run the app (root `compose.yaml` or the two dev servers), open an
  account with a mid-month balance adjustment, confirm the line steps and
  re-anchors correctly, the month switcher works past the current month and
  across a year boundary, and a no-activity month renders a flat line.
- [x] 8.4 Re-run the CI `contract` job locally (or confirm in CI) — no spec
  drift.
- [x] 8.5 `openspec validate add-flow-chart-month-view` passes.

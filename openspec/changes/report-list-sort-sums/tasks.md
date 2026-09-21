## 1. API contract

- [ ] 1.1 In `openapi/openapi.yaml`, add `income` and `outcome` (arrays of
      `CurrencySum`, required) to `EntrySummary`, worded like `FlowBucket`'s
      identical pair, noting they total the whole filtered result rather
      than a single bucket.
- [ ] 1.2 Regenerate both committed artifacts:
      `cd backend && go generate ./...` and
      `cd frontend && pnpm generate:api`.

## 2. Backend domain and service

- [ ] 2.1 In `entry.go`, add `AccountSum{ Amount, Income, Outcome int64 }`
      and change `Store.Sum`'s return type from `map[string]int64` to
      `map[string]AccountSum`, updating its doc comment.
- [ ] 2.2 Add `Income`/`Outcome []CurrencySum` to `Summary`.
- [ ] 2.3 In `service.go`, update `Sum` to build all three `[]CurrencySum`
      slices (sums, income, outcome) from the widened `perAccount` map in
      one pass over the same per-currency grouping it already does.

## 3. Stores

- [ ] 3.1 In `storage/postgres/entry.go`'s `Sum` query, add
      `COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0) AS income` and
      `COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0) AS outcome`
      alongside the existing `SUM(amount)`, and scan them into the widened
      return map.
- [ ] 3.2 Do the same in-process in `storage/memory/entry.go`'s `Sum`.

## 4. Backend tests

- [ ] 4.1 Service/scenario tests: income/outcome split for a single
      currency, split across multiple currencies, a self-transfer with
      both legs in scope contributing to both income and outcome while
      still netting `sums` to zero, balance adjustments excluded from all
      three (not just `sums`), and an empty result returning empty
      `income`/`outcome` arrays alongside `sums: []`.
- [ ] 4.2 Postgres store tests for the new query columns, alongside the
      existing `Sum` coverage (self-skip without `DATABASE_URL`, run via
      `backend-integration` in CI).
- [ ] 4.3 Handler test: `GET /api/entries/summary`'s response includes
      `income`/`outcome` and validates against the updated
      `openapi/openapi.yaml` via `internal/openapicheck`.

## 5. Frontend sorting

- [ ] 5.1 In `frontend/src/routes/reports.tsx`, extend `ReportsSearch`
      with `sort`/`dir` (same `Sort`/union shape `entries.index.tsx`
      already defines — reuse or mirror it) and read them in
      `buildEntriesQuery` instead of the hardcoded
      `sort: "booking_timestamp", dir: "desc"`.
- [ ] 5.2 Add a `toggleSort` matching `entries.index.tsx`'s exactly (click
      a column not currently sorted → that column, `dir=desc`; click the
      currently-sorted column again → same column, direction flipped),
      calling `patchSearch` and, when a report has already been generated
      (`hasGenerated`), also resetting `items`/`nextCursor` and re-fetching
      `GET /api/entries` immediately with the new sort — without touching
      `generatedFilter`, the stale-hint check, or the sums fetch.
- [ ] 5.3 Make the Date and Amount column headers buttons wired to
      `toggleSort`, with the same arrow indicator `entries.index.tsx` uses
      (`↑`/`↓`, shown only on the active sort column, defaulting to
      `booking_timestamp`/`desc` when `search.sort` is unset).

## 6. Frontend Income/Outcome display

- [ ] 6.1 In the sums bar (`reports.tsx`), render Income and Outcome per
      currency alongside the existing net sum, using `formatAmount`/
      `amountColorClass` — Outcome's positive-magnitude `amount` negated
      before formatting/coloring so it renders as an outflow (see
      design.md).
- [ ] 6.2 Add `reports.income`/`reports.outcome` i18n keys to `en.json`
      (English text matching `flowChart.income`/`flowChart.outcome`, as
      the page's own keys — not a reach into that namespace) and mirror
      in `de.json`, keeping both locales at 100% coverage.

## 7. Verification

- [ ] 7.1 `cd backend && go build ./... && go test ./... && go vet ./...`
      and the repo's Go linter.
- [ ] 7.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 7.3 Drive `/reports` in Chromium against a stubbed `/api`: generate a
      report, click the Date header (flips direction), click Amount
      (switches column, resets to desc, click again flips), confirm no
      "results are stale" hint appears from sorting alone and a real
      filter change still shows it; confirm the sums bar shows net/
      Income/Outcome per currency in both locales and both themes,
      including a multi-currency report and a self-transfer-containing
      report (income and outcome both non-zero, net unaffected).

## 1. Backend: schema

- [x] 1.1 Add migration
      `backend/internal/storage/postgres/migrations/0018_entry_balance_reading.sql`:
      `ALTER TABLE entries ADD COLUMN balance_reading bigint;`, backfill in
      two passes (copy existing `balance_adjustment.amount` into
      `balance_reading`, then recompute `amount` per account in
      `(booking_timestamp, id)` order as `balance_reading − previous
      balance_reading − transactions strictly between`), then add
      `CHECK ((kind = 'balance_adjustment') = (balance_reading IS NOT
      NULL))`.
- [x] 1.2 `internal/storage/postgres/entry_test.go`: integration test
      (`TestPGEntryMigration0018BackfillPreservesBalances`) asserting
      `Balance()` returns identical values before and after the migration,
      against a fixture with interleaved transactions/adjustments — run
      for real against a local Postgres in this environment, not just
      written.

## 2. Backend: `internal/entry` domain — delta recompute

- [x] 2.1 `entry.go`: `Entry` gains `Balance *int64` (`json:"balance,
      omitempty"`); `New`/`Update` gain `Amount *int64`/`Balance *int64`
      (pointers, so "not supplied" is distinguishable from "supplied as
      0"); doc comments describe `amount` as always the signed delta.
- [x] 2.2 Recompute lives inside each storage backend's own
      Create/Update/SoftDelete (not as new `Store` interface methods) —
      `internal/storage/postgres/entry.go`'s unexported `findAdjustment`/
      `setAmount`/`recomputeFrom`, and the mirrored Go-loop version in
      `internal/storage/memory/entry.go` (`findAdjustmentLocked`/
      `setAmountLocked`/`recomputeFromLocked`).
- [x] 2.3 `Create` recomputes from the new entry's position; `Update`
      recomputes from both the old and new `(account, position)` when
      either changed (excluding the entry's own now-stale old-position row
      from that search), or once when neither changed; `SoftDelete`
      recomputes from the deleted entry's position (already excluded by
      `deleted_at IS NULL`).
- [x] 2.4 `validateNew` and `Service.Update` reject `amount` on a
      `balance_adjustment` and `balance` on a `transaction`
      (`ErrInvalidValue`, `400`); `balance` is required on
      `balance_adjustment` create, `amount` on `transaction` create.
- [x] 2.5 Implemented identically (not a shortcut) in both
      `internal/storage/memory` and `internal/storage/postgres`.
- [x] 2.6 Service tests (memory) for every scenario in design.md's worked
      example, plus contract-rejection tests — all in
      `internal/entry/service_scenarios_test.go`. Mirrored as real
      Postgres integration tests in
      `internal/storage/postgres/entry_test.go`, including a cross-account
      move case not in the original test list. All run against a real
      local Postgres in this environment.

## 3. Backend: `flow-summary` endpoint

- [x] 3.1 `store.go`: `TimezoneLookup` interface
      (`Timezone(ctx, ownerID string) (string, error)`); `entry.go` gains
      `FlowUnit`, `FlowFilter`, `FlowRow`, `FlowBucket`; `Store` gains
      `FlowSummary(ctx, ownerID string, filter FlowFilter) ([]FlowRow,
      error)`.
- [x] 3.2 `service.go`'s `FlowSummary` validates `unit`/`year`/`month`,
      resolves the caller's visible accounts and timezone (default `UTC`),
      delegates to `store.FlowSummary`, then fills every period in the
      requested range (even empty ones) and resolves each row's account to
      its currency — mirroring `Sum`. A currency with only income (or only
      outcome) in a bucket is listed on just that one side, never also on
      the other with a `0`.
- [x] 3.3 `main.go` wires `entry.WithTimezoneLookup(settingsSvc)`, and
      `internal/settings.Service` gained a `Timezone` method (parallel to
      its existing `Language`) to satisfy it.
- [x] 3.4 `handler.go`: `GET /api/entries/flow-summary` parses/validates
      `account_id`/`unit`/`year`/`month`, `400` on invalid input.
- [x] 3.5 Implemented in both `internal/storage/memory` (Go loop,
      `time.LoadLocation` + `local.In(loc)`) and `internal/storage/postgres`
      (`timezone($tz, booking_timestamp)` + `date_trunc` + `FILTER`).
- [x] 3.6 Handler + service tests: month/day units, multiple currencies,
      empty buckets, balance-adjustment delta inclusion, invalid
      unit/month combinations, and a timezone boundary-crossing case
      (`America/New_York`) — memory-backed unit tests plus real Postgres
      integration tests for the SQL itself.
- [x] 3.7 `internal/storage/postgres/entry_test.go`: `TestPGEntryFlowSummary*`
      integration tests, run for real.

## 4. API contract

- [x] 4.1 `openapi/openapi.yaml`: `Entry`/`EntryCreate`/`EntryUpdate` gain
      `balance`; `EntryCreate.required` drops `amount`, with the
      kind-conditional requirement documented in each schema's
      description (400, not 422 — matches `entry.ErrInvalidValue`'s real
      HTTP mapping). Added `FlowBucket` schema and
      `GET /api/entries/flow-summary` with its params and responses.
- [x] 4.2 `go generate ./...` — `backend/openapi.yaml` resynced.
- [x] 4.3 `pnpm generate:api` — `frontend/src/api/schema.d.ts` resynced.
- [x] 4.4 `internal/openapicheck.AssertResponse` assertions added to the
      new flow-summary handler tests; `spectral lint` run for real via
      `npx @stoplight/spectral-cli` — no errors.

## 5. Frontend: entry form

- [x] 5.1 `entries.new.tsx` and `entries.$entryId.edit.tsx`: for
      `kind === "balance_adjustment"`, submit `balance` instead of
      `amount`, and populate the edit form's initial value from
      `entry.balance`. No UI/label change.

## 6. Frontend: delta annotation on entry lists

- [x] 6.1 `accounts.$accountId.index.tsx`: recent-entries list renders
      `entry.balance` (falling back to `entry.amount` for a transaction)
      as the main figure, plus a small gray `formatSignedAmount(entry.
      amount, …)` annotation for `kind === "balance_adjustment"` rows.
- [x] 6.2 `entries.index.tsx`: identical addition to its table cell.
      New `formatSignedAmount` helper added to `src/lib/amount.ts`
      (`Intl.NumberFormat` with `signDisplay: "always"`).

## 7. Frontend: reusable bar chart

- [x] 7.1 Read the `dataviz` skill before writing chart code; validated
      the income/green (`#008300`) and outcome/red (`#e34948`/`#e66767`,
      categorical slots 6 and 8) pairing with
      `scripts/validate_palette.js` for both light and dark modes.
- [x] 7.2 `frontend/src/components/charts/BarChart.tsx` — presentational,
      hand-rolled SVG/Tailwind, no charting library, generic N-category ×
      M-series props; legend, hairline gridlines with "nice" tick values,
      per-group hover/focus tooltip listing every series' value, dimming
      of non-hovered groups, theme-aware via Tailwind arbitrary-value fill
      classes.
- [x] 7.3 `frontend/AGENTS.md` gained a "Charts" section documenting the
      no-library convention, the `dataviz` skill consultation, and the
      validated-palette-over-`amountColorClass` rule for chart fills.

## 8. Frontend: account details page chart + year switcher

- [x] 8.1 `accounts.$accountId.index.tsx`: `chartYear` state (defaults to
      the current year), previous/next-year buttons (no bound), a
      dedicated effect fetching
      `GET /api/entries/flow-summary?account_id={id}&unit=month&year={year}`
      keyed on `[accountId, chartYear]`, and the `BarChart` rendered above
      "Recent entries".

## 9. i18n

- [x] 9.1 Added `accounts.details.chart.{title,income,outcome,
      previousYear,nextYear}` to `en.json` then `de.json` — 100% coverage
      confirmed via `node scripts/i18n-coverage.mjs`.

## 10. Verify

- [x] 10.1 `cd backend && gofmt -l . && go vet ./... && go test ./...` —
      clean, run against a real local Postgres (started in this
      environment) via `DATABASE_URL`, so the `internal/storage/postgres`
      integration tests actually executed rather than self-skipping.
- [x] 10.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build` —
      clean.
- [x] 10.3 Manual pass — actually performed in this environment (a local
      Postgres and a headless-Chromium-driven dev server were both
      available, unlike prior changes): seeded realistic data via
      `server seed --yes`, obtained a session directly from its printed
      token (no email round-trip needed), created a balance adjustment via
      the API, and drove the running app with Playwright. Confirmed: the
      chart renders 12 months × 2 bars with correct values, axis labels
      and gridlines render without clipping (fixed a real left/top margin
      bug found this way), the legend and per-bar hover tooltip both show
      correct values, the year switcher fetches and redraws (including a
      correctly all-empty year), the delta annotation renders correctly
      on the full `/entries` list (verified: `€12.35` reading, `-€5,990.48`
      gray delta, matching what was posted), and dark mode renders
      legibly.
- [x] 10.4 `backend/AGENTS.md` gained an "Entries" section (schema,
      recompute algorithm, flow-summary); `frontend/AGENTS.md` gained the
      "Charts" section from 7.3.

## 11. Spec sync

- [x] 11.1 Applied this change's `specs/account-entries`,
      `specs/web-client-accounts`, and `specs/web-client-entries` deltas
      onto `openspec/specs/` by hand (the `openspec` CLI is unavailable in
      this environment, as for prior changes).

## 1. Backend: schema

- [ ] 1.1 Add migration
      `backend/internal/storage/postgres/migrations/00NN_entry_balance_reading.sql`:
      `ALTER TABLE entries ADD COLUMN balance_reading bigint;`, backfill in
      two passes (copy existing `balance_adjustment.amount` into
      `balance_reading`, then recompute `amount` per account in
      `(booking_timestamp, id)` order as `balance_reading − previous
      balance_reading − transactions strictly between`), then add
      `CHECK ((kind = 'balance_adjustment') = (balance_reading IS NOT
      NULL))`.
- [ ] 1.2 `internal/storage/postgres/entry_test.go`: integration test
      asserting `Balance()` returns identical values before and after the
      migration, against a fixture with multiple accounts, interleaved
      transactions/adjustments, and a soft-deleted entry.

## 2. Backend: `internal/entry` domain — delta recompute

- [ ] 2.1 `entry.go`: add `BalanceReading *int64` to `Entry`
      (`json:"balance,omitempty"`); update doc comments describing `amount`
      as always the signed delta.
- [ ] 2.2 `store.go`: add the recompute algorithm's building blocks —
      `BalanceBefore(ctx, accountID string, pos Cursor) (int64, error)`
      (sums `amount` strictly before a `(booking_timestamp, id)` position)
      and `NextAdjustmentAtOrAfter(ctx, accountID string, pos Cursor)
      (*Entry, error)` (locked `FOR UPDATE`, earliest non-deleted
      `balance_adjustment` at/after `pos`).
- [ ] 2.3 `service.go`: `Create`/`Update`/`Delete` each call a shared
      `recomputeAffectedAdjustments` step after applying the mutation —
      for `Update`, run it for both the old and new
      `(booking_timestamp, id)` when `booking_timestamp` changes. Recompute
      finds `A1` via `NextAdjustmentAtOrAfter`, sets `A1.amount =
      A1.BalanceReading − BalanceBefore(A1)` if `A1` exists, then does the
      same for `A2` (earliest non-deleted adjustment strictly after `A1`).
- [ ] 2.4 Reject `amount` on a `balance_adjustment` create/update and
      `balance` on a `transaction` create/update (`ErrInvalidValue`, `400`).
      `balance` is required on `balance_adjustment` create.
- [ ] 2.5 Implement identically in `internal/storage/memory` and
      `internal/storage/postgres` (not a memory-store shortcut — balance
      correctness is asserted against the memory store in
      `service_scenarios_test.go`).
- [ ] 2.6 Service tests for every scenario in design.md's worked example:
      transaction inserted before an adjustment; edit between two
      adjustments only recomputes the following one; deleting a middle
      adjustment shifts the next one's baseline; moving an adjustment's
      `booking_timestamp` past another adjustment recomputes both affected
      neighbors; a chain of 3+ adjustments confirms no unnecessary
      recompute beyond the immediate neighbor.

## 3. Backend: `flow-summary` endpoint

- [ ] 3.1 `store.go`: add `TimezoneLookup` interface
      (`Timezone(ctx, ownerID string) (string, error)`) and a
      `FlowSummary(ctx, ownerID string, f FlowFilter) ([]FlowBucket,
      error)` method on `Store` — `FlowFilter` carries `AccountIDs`,
      `Unit`, `Year`, `Month`; buckets by `booking_timestamp` converted
      into the given timezone, summing positive `amount` into `income` and
      `|negative amount|` into `outcome`, grouped per currency (mirroring
      how `Sum` resolves each account's currency).
- [ ] 3.2 `service.go`: `FlowSummary` resolves the caller's timezone via
      `TimezoneLookup` (default `UTC` on lookup failure/empty, matching
      `user-settings`' own default), validates `unit`/`year`/`month`
      combinations, delegates to `store.FlowSummary`.
- [ ] 3.3 `main.go`: wire `entry.WithTimezoneLookup(settingsSvc)`, mirroring
      `auth.WithLanguageLookup`.
- [ ] 3.4 `handler.go`: `GET /api/entries/flow-summary` — parse/validate
      query params, `400` on invalid `unit`/missing or extraneous `month`.
- [ ] 3.5 Implement `FlowSummary` in both `internal/storage/memory` and
      `internal/storage/postgres` (Postgres: `timezone($tz,
      booking_timestamp)` before truncating to month/day).
- [ ] 3.6 Handler + service tests: month and day units; multiple accounts
      of different currencies; empty buckets present with `income: []`/
      `outcome: []`; a `balance_adjustment`'s delta included; invalid
      `unit=day` with no `month` and `unit=month` with `month` both `400`;
      a boundary-crossing entry lands in the correct bucket under a
      non-UTC timezone.
- [ ] 3.7 `internal/storage/postgres/entry_test.go`: integration test for
      `FlowSummary`'s SQL directly.

## 4. API contract

- [ ] 4.1 `openapi/openapi.yaml`: add `balance` to `Entry`, `EntryCreate`,
      `EntryUpdate`; adjust `EntryCreate.required` (drop unconditional
      `amount`, document the kind-conditional requirement in the
      description, matching how `category_id` is already documented as
      conditionally required). Add `FlowSummary`/`FlowBucket` schemas
      (`FlowBucket`: `period`, `income`/`outcome` arrays of
      `CurrencySum`). Add `GET /api/entries/flow-summary` with its
      `account_id`/`unit`/`year`/`month` params and `400`/`401` responses.
- [ ] 4.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [ ] 4.3 `cd frontend && pnpm generate:api` to regenerate
      `src/api/schema.d.ts`.
- [ ] 4.4 Add `internal/openapicheck.AssertResponse` assertions to the
      new/changed handler tests; lint the spec with spectral.

## 5. Frontend: entry form

- [ ] 5.1 `frontend/src/routes/entries.new.tsx` and `entries.$entryId.
      edit.tsx`: for `kind === "balance_adjustment"`, read/write `balance`
      instead of `amount` (submission payload and, on edit, the initial
      form value). No UI/label change.

## 6. Frontend: delta annotation on entry lists

- [ ] 6.1 `frontend/src/routes/accounts.$accountId.index.tsx`: the
      recent-entries list renders `entry.balance` (falling back to
      `entry.amount` for a transaction) as today's main figure, plus — for
      `kind === "balance_adjustment"` — a small, gray, unstyled-by-sign
      span showing `entry.amount` (the delta) with an explicit sign.
- [ ] 6.2 `frontend/src/routes/entries.index.tsx`: the same addition to its
      list rendering (currently lines ~360-375).

## 7. Frontend: reusable bar chart

- [ ] 7.1 Read the `dataviz` skill before writing any chart code.
- [ ] 7.2 New `frontend/src/components/charts/BarChart.tsx` (or similar) —
      presentational, hand-rolled SVG/Tailwind, no charting library, props
      in / no fetching — rendering N categories × 2 series (income/
      outcome), theme-aware (light/dark).
- [ ] 7.3 Add a rule to `frontend/AGENTS.md` documenting: no chart library;
      hand-rolled SVG/Tailwind components under `src/components/charts/`;
      consult the `dataviz` skill before building one.

## 8. Frontend: account details page chart + year switcher

- [ ] 8.1 `accounts.$accountId.index.tsx`: add year state (component
      state), previous/next-year buttons (no bound), and a fetch of
      `GET /api/entries/flow-summary?account_id={id}&unit=month&year={year}`
      on mount and on year change; render the chart above "Recent
      entries" using the new `BarChart` component.

## 9. i18n

- [ ] 9.1 Add new keys to `frontend/src/i18n/locales/en.json` first, then
      `de.json`: chart section heading, year-switcher labels/aria-labels,
      income/outcome series labels, the delta-annotation's accessible
      label if needed.

## 10. Verify

- [ ] 10.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`.
      `internal/storage/postgres` integration tests need a reachable
      `DATABASE_URL` — run them if available, otherwise they self-skip per
      `backend/AGENTS.md` and are exercised in CI's `backend-integration`
      job.
- [ ] 10.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 10.3 Manual pass: create a balance adjustment and confirm its delta
      annotation on both list pages; edit an earlier transaction and
      confirm a later adjustment's displayed delta updates accordingly;
      switch years on the account details chart and confirm bars match
      the API response; confirm an all-zero month renders without error.
- [ ] 10.4 Update `backend/AGENTS.md` ("Account entries"/balance section)
      and `frontend/AGENTS.md` (chart convention rule from 7.3, account
      details page description) so they don't go stale.

## 11. Spec sync

- [ ] 11.1 Apply this change's `specs/account-entries`,
      `specs/web-client-accounts`, and `specs/web-client-entries` deltas
      onto `openspec/specs/` by hand (the `openspec` CLI is unavailable in
      this environment, as for prior changes).

## 1. API contract

- [ ] 1.1 In `openapi/openapi.yaml`, add `GET /api/recurring-transactions
      /preview` (query: `account_id` repeatable, `category_id`,
      `category_mode`, `tag_id`, required `to`; response:
      `{ items: RecurringTransactionPreviewItem[] }`) per
      `recurring-transactions`'s new requirements.
- [ ] 1.2 Add `RecurringTransactionPreviewItem` schema (`recurring_transaction_id`,
      `account_id`, `account_currency`, `title`, `description`,
      `category_id`, `counterparty`, `location`, `tag_ids`, `amount`,
      `booking_timestamp`, `overdue`).
- [ ] 1.3 Add `recurring_preview_horizon` (enum: `1_month`, `2_months`,
      `3_months`, `end_of_this_month`, `end_of_next_month`,
      `end_of_this_year`) to the `Settings` schema and `PUT /api/settings`'s
      request body, per `user-settings`'s modified requirements.
- [ ] 1.4 Add `show_recurring_preview` (boolean) to `DashboardCardConfig`,
      per `dashboard-cards`'s modified requirement.
- [ ] 1.5 Sync `backend/openapi.yaml` (`cd backend && go generate ./...`)
      and regenerate `frontend/src/api/schema.d.ts`
      (`cd frontend && pnpm generate:api`); confirm no other diff.

## 2. Backend: recurring transaction preview projection

- [ ] 2.1 In `backend/internal/recurringtransaction/recurringtransaction.go`,
      factor the existing interval-advance logic (used by
      `NextSuggestedDate`) into a reusable function that advances a `Date`
      by one interval, usable in a loop.
- [ ] 2.2 Add a `PreviewFilter` type (`AccountIDs`, `CategoryID`,
      `CategoryMode`, `TagID`, `To`) and a `Service.Preview` method:
      resolves the caller's visible recurring transactions (reusing the
      existing `List` access/filter resolution), and for each, generates
      occurrences starting at its `NextSuggestedDate` up to
      `min(filter.To, template.EndsOn)`, applying the single-overdue-row
      rule (emit the anchor once regardless of being in the past; skip,
      don't emit, any subsequently-advanced date still before today; then
      emit normally) and the 366-occurrence-per-template cap.
- [ ] 2.3 Add `backend/internal/recurringtransaction/handler.go`'s
      `GET /preview` route: parses query params, calls `Service.Preview`,
      maps `ErrInvalidValue` (missing `to`) to `400`.
- [ ] 2.4 Unit tests: basic multi-occurrence projection; the single
      overdue-row rule (including the "still-past-after-one-advance"
      case); `ends_on` clamping tighter than the requested `to`; the
      366-occurrence cap; account/category/tag filtering; missing `to`
      rejected; empty result for a caller with no visible templates;
      confirms no entry or recurring transaction is ever mutated by
      calling it repeatedly.

## 3. Backend: user-settings horizon field

- [ ] 3.1 Add migration
      `backend/internal/storage/postgres/migrations/00NN_recurring_preview_horizon.sql`
      adding a nullable `recurring_preview_horizon` column to
      `user_settings`.
- [ ] 3.2 In `backend/internal/settings`, add the field to the resolved
      settings struct, its hardcoded default (`end_of_this_month`), and
      validation against the six allowed enum values.
- [ ] 3.3 Unit tests: default resolution (missing row / `NULL` column),
      valid values accepted, an invalid value rejected without touching
      other fields, per `user-settings`'s modified requirements. Postgres
      store tests for the new column.

## 4. Frontend: recurring-transaction preview data + cutoff resolution

- [ ] 4.1 Add a shared helper (e.g. `lib/recurringPreview.ts`) resolving a
      surface's effective cutoff: `min(that surface's own resolved date
      filter "to", if any; the caller's recurring_preview_horizon setting
      resolved to a concrete date via the same kind of preset math
      `dateRangePresets.ts` already has)`, plus a thin
      `GET /api/recurring-transactions/preview` fetch wrapper.
- [ ] 4.2 Add the horizon-preset resolution functions (1/2/3 months from
      now; end of this/next/current year) alongside the existing
      date-range preset math, reusing its date-arithmetic helpers where
      they overlap.

## 5. Frontend: entries.new date override

- [ ] 5.1 `entries.new.tsx`: accept an optional `booking_timestamp` search
      param; when present alongside `recurring_transaction_id`, use it as
      the prefilled booking timestamp instead of the fetched recurring
      transaction's `next_suggested_date`, per
      `web-client-recurring-transactions`'s modified requirement.

## 6. Frontend: shared Upcoming block component

- [ ] 6.1 Build a shared `UpcomingBlock` component (row: title,
      counterparty/category/account per the existing entry-row styling,
      sign-colored amount, "Create transaction" link to
      `/entries/new?recurring_transaction_id=...&booking_timestamp=...`,
      overdue rows tinted), taking preview items + a loading state as
      props — presentational only, no fetching inside it, matching this
      app's chart-component convention.
- [ ] 6.2 Add the new `dashboard.*`/`entries.*`/`reports.*` i18n keys this
      component needs (heading, empty state, overdue hint) to `en.json`
      first, then `de.json`.

## 7. Frontend: /entries wiring

- [ ] 7.1 `entries.index.tsx`: add the `show_recurring` boolean URL search
      param (default `false`) and its filter-row toggle control.
- [ ] 7.2 When on, fetch the preview using the ledger's current
      account/category/tag filters and the resolved cutoff (task 4.1);
      render `UpcomingBlock` above the real list, refetching when filters
      or the toggle change.

## 8. Frontend: /reports wiring

- [ ] 8.1 `reports.tsx`: add the `show_recurring` boolean URL search param
      and its toggle control alongside the existing filter controls —
      changing it updates the URL only, per the page's existing
      no-auto-fetch rule.
- [ ] 8.2 "Generate report" additionally fetches the preview (using the
      resolved cutoff) when the toggle is on at generation time; render
      `UpcomingBlock` alongside the results table. Confirm the displayed
      per-currency sum stays sourced only from `GET /api/entries/summary`,
      untouched by the toggle.

## 9. Frontend: dashboard entry_list card

- [ ] 9.1 `CardFormDialog.tsx`: add the `show_recurring_preview` toggle,
      shown only for `entry_list`/`bar_chart` card types, included in the
      submitted config.
- [ ] 9.2 `EntryListCard.tsx`: when `config.show_recurring_preview` is
      true, fetch the preview using the card's own filter + resolved
      cutoff and render `UpcomingBlock` above the card's fixed-size real
      list, without reducing that list's count.

## 10. Frontend: dashboard bar_chart card + BarChart stacking

- [ ] 10.1 Extend `BarChart`/`BarChartDatum` with an optional stacked
      sub-value per series, rendered as a second, muted-color segment on
      top of the real segment — additive only, no change to a datum with
      no stacked value.
- [ ] 10.2 Add muted `INCOME_FILL_PROJECTED`/`OUTCOME_FILL_PROJECTED`
      Tailwind fill classes; validate the pairing with the `dataviz`
      skill's `scripts/validate_palette.js` before wiring them in.
- [ ] 10.3 `BarChartCard.tsx`: when `config.show_recurring_preview` is
      true, fetch the preview, bucket items into the same month/day
      periods as the card's real `flow-summary` buckets, and pass the
      bucketed amounts as each series' stacked value for buckets between
      today and the resolved cutoff.

## 11. Frontend: settings horizon picker

- [ ] 11.1 `settings.index.tsx`'s Profile tab: add the recurring preview
      horizon `<select>` (six named presets), saving on change via
      `PUT /api/settings` with only that field, mirroring the week-start
      control's pattern.

## 12. i18n

- [ ] 12.1 Add every remaining new user-facing string (the `show_recurring`
      toggle labels on `/entries`/`/reports`, the dashboard config form's
      toggle label, the horizon `<select>`'s option labels, the Upcoming
      block's heading/empty-state/overdue-hint copy not already covered
      in task 6.2) to `en.json` first, then `de.json`.

## 13. Verification

- [ ] 13.1 Backend: `gofmt -l .`, `go vet ./...`, `go test ./...` (from
      `backend/`), including the postgres integration suite against the
      local compose DB.
- [ ] 13.2 Frontend: `pnpm lint`, `pnpm exec tsc`, `pnpm build` (from
      `frontend/`).
- [ ] 13.3 Manually exercise, in a real browser against the dev server
      with a seeded user: enabling the toggle on `/entries` and `/reports`
      shows the Upcoming block with correct cutoff behavior (filter `to`
      vs. horizon, whichever is earlier); an overdue recurring
      transaction shows exactly one tinted row in place; "Create
      transaction" from an Upcoming row prefills the correct date; an
      `entry_list` card and a `bar_chart` card with the preview enabled
      render correctly, including a bucket that mixes real and projected
      amounts; confirm `/reports`' sum and the dashboard's `query_stat`
      cards are unaffected by the toggle.
- [ ] 13.4 Confirm no drift: `openapi/openapi.yaml` and
      `backend/openapi.yaml` are byte-identical, and re-running
      `pnpm generate:api` against the current spec produces no further
      diff in `schema.d.ts`.

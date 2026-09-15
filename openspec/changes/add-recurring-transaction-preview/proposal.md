## Why

Recurring transactions are manual-only templates: `/recurring` shows each
one's `next_suggested_date`, but nothing else in the app surfaces what's
coming due. A visitor has to remember to check `/recurring` separately from
the entry ledger, reports, and the dashboard they actually use day to day.
A bounded, opt-in preview of upcoming occurrences — woven into those
existing views — lets a visitor see committed future cash flow in context,
without changing the deliberately manual "nothing is ever auto-created"
model recurring transactions already have.

## What Changes

- New backend endpoint, `GET /api/recurring-transactions/preview`, that
  computes virtual (never persisted) future occurrences of the caller's
  visible recurring transactions, bounded by a caller-supplied cutoff date.
  Never unbounded — a horizon is always required.
- New per-user setting, `recurring_preview_horizon`, one of six fixed
  presets (1/2/3 months from now; end of this/next month; end of this
  year), independent of any page's own date filter.
- The effective cutoff for a given recurring transaction is
  `min(the page's filter "to", if it has one; the horizon setting resolved
  to a date; that recurring transaction's own "ends_on", if set)`.
- `/entries`, `/reports`, and the dashboard's `entry_list` card gain an
  opt-in toggle that renders a dedicated "Upcoming" block — separate from
  the real, paginated list — listing virtual rows sorted by date, each
  offering the same "Create transaction" action the `/recurring` page
  already has, prefilled to that occurrence's specific projected date.
- A recurring transaction whose `next_suggested_date` is already in the
  past still surfaces as exactly one overdue virtual row (visually tinted),
  in its normal chronological position — never skipped, and never repeated
  for every interval it's fallen behind by.
- The dashboard's `bar_chart` card gains the same opt-in toggle: a bucket
  between today and the cutoff renders a second, distinctly-colored
  segment stacked on its real income/outcome bar for the projected amount.
- `entries.new`'s existing recurring-transaction prefill flow
  (`?recurring_transaction_id=`) gains an optional explicit
  `booking_timestamp` override, since a virtual row beyond a template's
  first occurrence needs to prefill that specific date rather than always
  the template's own `next_suggested_date`.
- Explicitly unchanged: the dashboard's `query_stat` card, `/reports`' own
  per-currency sum, and account balances. Projected amounts are never
  folded into a real total anywhere — only shown as their own rows or
  chart segments.

## Capabilities

### Modified Capabilities

- `recurring-transactions`: adds the preview-projection endpoint and its
  cutoff/overdue/cap rules.
- `user-settings`: adds the `recurring_preview_horizon` preference,
  storage default, and validation.
- `web-client-settings`: the Profile tab gains the horizon picker.
- `web-client-recurring-transactions`: the "Create transaction" flow
  accepts an explicit date override for a non-`next_suggested_date`
  occurrence.
- `web-client-entries`: adds the preview toggle and the Upcoming block.
- `web-client-reports`: adds the preview toggle and the Upcoming block,
  gated behind the existing explicit "Generate report" action.
- `dashboard-cards`: adds the `show_recurring_preview` config field,
  allowed only on `entry_list` and `bar_chart` cards.
- `web-client-home`: `entry_list` cards render the Upcoming block;
  `bar_chart` cards render stacked projected segments.

## Impact

- **Backend**: `internal/recurringtransaction` gains a new
  `Preview`/`GET /api/recurring-transactions/preview` path through
  service/store/handler, reusing the existing interval-advance math;
  `internal/settings` gains the new preference column, default, and
  validation; `openapi/openapi.yaml` (and its two generated copies) gain
  the new endpoint, the new settings field, and the new dashboard config
  field.
- **Frontend**: a new shared "Upcoming" block component used by
  `entries.index.tsx`, `reports.tsx`, and `EntryListCard.tsx`; `BarChart`
  gains an optional stacked-segment capability, used by `BarChartCard.tsx`;
  `entries.new.tsx` accepts the new `booking_timestamp` override;
  `settings.index.tsx`'s Profile tab gains the horizon picker;
  `CardFormDialog.tsx` gains the toggle for `entry_list`/`bar_chart`
  cards; `src/api/schema.d.ts` regenerated from the updated contract.
- **Data**: one new nullable `user_settings` column, no backfill. No new
  tables — preview rows are computed on read, never persisted.

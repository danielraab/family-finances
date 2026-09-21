## Why

`/reports` results are always ordered newest-first with no way to change
it, and the sums bar shows only one net figure per currency. The backend
already supports sorting the underlying list (`GET /api/entries` accepts
`sort=booking_timestamp|amount` and `dir`) — `/reports` just never exposes
it, unlike `/entries`, which already has clickable Date/Amount column
headers doing exactly this. And `GET /api/entries/summary` only ever
returns a net `sums` per currency, so a report can't answer "how much came
in" versus "how much went out" without opening every row — a split
`GET /api/entries/flow-summary` already computes for the dashboard's flow
chart, just bucketed by period instead of totalled for the whole report.

## What Changes

- `/reports`' results table gets clickable Date/Amount column headers,
  mirroring `/entries`' `toggleSort` behavior exactly: click sorts
  descending by that column, click again flips direction, and it fires an
  immediate fetch — sort is not part of the "Generate report" gate the
  other filters go through, since it reorders the same result set rather
  than choosing it.
- `GET /api/entries/summary`'s response gains `income` and `outcome`
  arrays of `{ currency, amount }`, alongside the existing net `sums` —
  the same shape `GET /api/entries/flow-summary`'s `FlowBucket` already
  uses for its own `income`/`outcome`, just unbucketed (one total per
  currency for the whole filtered result, not per period).
- `/reports`' sums bar shows, per currency, the net total plus Income and
  Outcome, using the same "Income"/"Outcome" labels the flow chart already
  established.

## Non-goals

- **No sort on the sums bar or the dashboard's `query_stat` card.** Only
  `GET /api/entries/summary` gains the new fields; `query_stat` cards keep
  reading `sums` and are unaffected (additive response fields).
- **No change to `/entries`' own sort UI.** `report-list-sort-sums` reuses
  its `toggleSort` pattern but does not touch `entries.index.tsx` itself.
- **No bucketing/period breakdown on `/reports`.** Income/Outcome are one
  total per currency for the whole generated report, not a chart.

## Capabilities

### Modified Capabilities

- `account-entries`: `GET /api/entries/summary` additionally returns
  `income` and `outcome` per-currency totals alongside `sums`.
- `web-client-reports`: the results table is sortable by date or amount,
  and the sums bar shows Income/Outcome alongside the net total.

## Impact

- `openapi/openapi.yaml` (+ the two generated artifacts) — `EntrySummary`
  gains `income`/`outcome`.
- `backend/internal/entry/entry.go` — `Summary` gains `Income`/`Outcome`
  fields; `Store.Sum`'s return shape widens to carry income/outcome per
  account, not just the net.
- `backend/internal/entry/service.go` — `Service.Sum` aggregates the
  income/outcome totals per currency the same way it already does for the
  net.
- `backend/internal/storage/{postgres,memory}/entry.go` — `Sum`'s query/
  loop computes income/outcome per account alongside the existing net sum,
  reusing the `FILTER (WHERE amount > 0 / < 0)` split `FlowSummary`
  already uses in Postgres.
- `frontend/src/routes/reports.tsx` — sortable column headers (URL-backed
  `sort`/`dir`, immediate re-fetch resetting items/cursor), and the sums
  bar renders Income/Outcome per currency.
- `frontend/src/i18n/locales/{en,de}.json` — new `reports.income`/
  `reports.outcome` keys (own-page keys, same English text
  `flowChart.income`/`flowChart.outcome` already use, per this repo's
  per-page label convention — see design.md).

## Context

Recurring transactions (`internal/recurringtransaction`) are templates
only — nothing about them is ever auto-materialized into a real `Entry`.
The one future-facing signal that exists today, `next_suggested_date`, is
computed fresh on every read by advancing one interval past the latest
linked entry's `booking_timestamp` (or `starts_on` if there is no linked
entry yet), and it is never auto-advanced by the passage of time — a
neglected template just sits with a `next_suggested_date` that drifts into
the past.

`/entries` and `/reports` both filter by an optional `from`/`to` range via
the shared `DateRangeFilter`; `/reports` in particular defaults to *no*
date filter at all. The dashboard's `entry_list`/`query_stat` cards reuse
the same `from`/`to` shape inline in their config; `bar_chart` cards have
no `from`/`to` at all — they page by calendar unit (a year of months, or a
month of days) instead.

This change adds a **preview** layer on top of all of that: a bounded,
read-only projection of a recurring transaction's next few occurrences,
shown as virtual rows/segments a visitor can act on (via the existing
"Create transaction" flow) but that are never stored and never counted in
any real total.

## Goals / Non-Goals

**Goals:**
- One cutoff formula, applied identically everywhere the preview appears:
  `min(page's filter "to" [if any], horizon setting resolved to a date,
  that recurring transaction's own "ends_on" [if set])`.
- A recurring transaction that's fallen behind schedule still gets exactly
  one overdue row, in its correct chronological spot, not one row per
  missed interval.
- Reuse the existing "Create transaction" flow (`/entries/new
  ?recurring_transaction_id=`) for turning a virtual row into a real entry,
  rather than inventing a second creation path.
- Keep every real aggregate (account balances, `/reports`' own sum,
  `query_stat`'s sum) exactly as accurate as it is today — a projected
  amount never enters one.

**Non-Goals:**
- No automatic or scheduled entry creation. This is strictly a display
  layer; the "manual materialization only" rule
  (`web-client-recurring-transactions`) is unchanged.
- No change to `query_stat` cards, account balances, or `/reports`'/
  `/entries`' own real-data totals — see "Projected data never enters a
  real total" below.
- No unbounded projection, ever — there is no state in which the preview
  computes "as far as possible."
- No per-occurrence editing of a virtual row. Acting on one always goes
  through the same edit-before-submit "Create transaction" form real
  occurrences already use.
- No new backend bucketing endpoint for the bar chart. The client buckets
  the flat preview list itself, into the same month/day periods
  `flow-summary`'s real buckets already use.

## Decisions

### One new endpoint, no persistence, shared by all three surfaces

`GET /api/recurring-transactions/preview` — query params `account_id`
(repeatable, default every account the caller has any permission on, same
default `GET /api/recurring-transactions` uses), `category_id`,
`category_mode` (`exact`|`subtree`, mirroring `GET /api/entries`),
`tag_id`, and a required `to` (RFC3339 timestamp — the caller always sends
an already-resolved cutoff; the backend never resolves a horizon preset
itself). Response: `{ items: [...] }`, each item carrying the same content
fields a materialized entry would (`account_id`, `account_currency`,
`title`, `description`, `category_id`, `counterparty`, `location`,
`tag_ids`, `amount`), plus `recurring_transaction_id`,
`booking_timestamp` (the projected date), and `overdue` (boolean). No
`id` — a virtual row isn't a stored entity; the frontend keys rows by
`recurring_transaction_id` + `booking_timestamp`.

One endpoint keeps the projection logic (the interval-advance math
`next_suggested_date` already implements) in exactly one place, reused
by `/entries`, `/reports`, and both dashboard card types — rather than
duplicating it per surface or per card type.

### The cutoff formula, and who resolves which input

| Input | Resolved by | Notes |
|---|---|---|
| Page's filter `to` | Client (already how `DateRangeFilter`/`resolveEffectiveRange` works) | Absent for `bar_chart` cards (no `from`/`to` concept) and for `/reports` with no date filter set |
| Horizon setting | Client, from the resolved `recurring_preview_horizon` setting | The backend never sees the preset key, only the final date |
| `ends_on` | Backend, per recurring transaction | Already stored per template; clamps independently of the other two |

The client always computes a single concrete `to` (`min(filter to, horizon
date)`, or just the horizon date when the surface has no filter `to`) and
sends it as the endpoint's `to`. The backend then additionally clamps per
template by that template's own `ends_on`. This mirrors how date-range
presets are already resolved client-side (`resolveEffectiveRange`) before
ever reaching the backend as concrete bounds.

If the surface's own filter `to` already resolves before today (e.g. a
`/reports` Custom range over a past month, or a dashboard card's
`range: last_month`), the computed cutoff is before today, `to` is sent as
that past date, and the endpoint naturally returns no rows for any
template — the Upcoming block or stacked segment simply doesn't render.
No special-case handling needed; it falls straight out of the formula.

### Exactly one overdue row per template, not one row per missed interval

Naively advancing forward from a stale `next_suggested_date` would emit
every intermediate occurrence still short of today — a daily recurring
transaction neglected for a month would flood the preview with thirty
overdue rows. Instead: the anchor (`next_suggested_date`) is always
emitted once, however far in the past it is. If advancing one interval
from the anchor is still before today, the generator keeps advancing
without emitting until it reaches a date that is on/after today (or past
the cutoff, in which case that template contributes only its single
overdue row); from there it emits normally, one row per interval, up to
the cutoff. This keeps the "one overdue row surfaces the fact you're
behind" signal without turning neglect into noise.

### A defensive per-template occurrence cap

A pathological combination (a daily/weekly interval with a far horizon,
e.g. `end_of_this_year` selected on January 1st) could still generate a
few hundred rows for one template. The service caps generation at 366
occurrences per template (comfortably above any real horizon × any
sensible interval) and silently stops — not a user-facing error, just a
defensive bound, the same spirit as `entry_list`'s fixed page size.

### Upcoming block, not interleaved pagination

`/entries` and the `entry_list` card are cursor-paginated against a real
DB query, sortable by `booking_timestamp` or `amount`, either direction.
Virtual rows aren't part of that query and are always few (bounded by the
cutoff and the occurrence cap) — merging them into the paginated stream
would mean re-ranking them against real pages the client hasn't fetched
yet under an `amount` or ascending sort, which cursor pagination can't do
without teaching the backend about virtual rows too. Instead, a dedicated
"Upcoming" block (its own small, non-paginated fetch) renders above the
real list, always sorted by date, regardless of whatever sort the real
list is currently using. `/reports`' results table gets the same
treatment. This also gives the single overdue row a natural, visible home
— the first entry in the block, tinted.

### `entries.new` gains an explicit date override

The existing flow (`/entries/new?recurring_transaction_id=X`) fetches the
recurring transaction and prefills `booking_timestamp` from its
`next_suggested_date`. A virtual row for a template's second or later
projected occurrence needs to prefill *that* occurrence's date instead.
Adding an optional `booking_timestamp` search param: when present
alongside `recurring_transaction_id`, it overrides the fetched
`next_suggested_date` as the prefilled value; when absent, behavior is
identical to today. `/recurring`'s own "Create transaction" link (which
never passes this param) is unaffected.

### Horizon is a new `user_settings` field, independent of any page's filter

Six fixed presets — `1_month`, `2_months`, `3_months`,
`end_of_this_month`, `end_of_next_month`, `end_of_this_year` — resolved
client-side, mirroring how `DateRangeFilter`'s own presets are resolved.
Stored as a nullable column with a hardcoded default
(`end_of_this_month`), following `user-settings`'s existing resolution
pattern exactly (missing row and `NULL` column both resolve to the
default). A short default keeps the common case's projection trustworthy
— the further out a projection reaches, the more it assumes every
intervening occurrence gets manually created on schedule, which nothing
enforces.

### `bar_chart` stacking is a new, additive capability of the shared `BarChart` component

`BarChart`/`BarChartSeries` today renders grouped bars only — one value
per series per category, no stacking. This change adds an optional
stacked sub-value per series (drawn as a second, muted-color segment on
top of the real segment within the same bar) purely additively: a
`BarChartDatum` with no stacked value renders exactly as it does today.
Only `BarChartCard` (when its `show_recurring_preview` config is on)
supplies the stacked value, computed by bucketing the flat preview-list
response into the same month/day periods the real `flow-summary` buckets
already use. The projected segment's color is a muted/lighter variant of
the existing `INCOME_FILL`/`OUTCOME_FILL` classes — validated against the
`dataviz` skill's palette guidance before shipping, per this repo's
existing chart-color convention.

### Projected data never enters a real total, anywhere

Generalizing beyond the dashboard's `query_stat` card (explicitly out of
scope per the request): no aggregate figure in this app — `query_stat`'s
sum, `/reports`' own per-currency sum, or an account's live balance —
ever includes a projected amount. Only row-level (Upcoming block) or
segment-level (`bar_chart`'s stacked bar) displays show one. This avoids
the double-counting risk a blended total would create the moment a
previewed occurrence is actually materialized, and keeps every number a
visitor already trusts exactly as trustworthy as it is today.

## Risks / Trade-offs

- **[Risk]** A heavily-neglected, short-interval recurring transaction
  could still surface as "very overdue" without context on *how* overdue
  beyond the one row's own (past) date. **Mitigation**: the row's own
  `booking_timestamp` already communicates this; no additional "N days
  overdue" copy is in scope for this change, but the date is always
  visible.
- **[Risk]** `BarChart`'s new stacked-segment capability changes a shared
  component other future charts also build on. **Mitigation**: strictly
  additive/optional — a `BarChartDatum` that supplies no stacked value
  renders identically to today's behavior; existing call sites
  (`BarChartCard` with the preview off) are unaffected.
- **[Trade-off]** There is no single "projected total" number anywhere
  (e.g. "your balance in 2 months will be roughly X") — only itemized
  rows/segments. Accepted: computing a trustworthy running total would
  require compounding every account's current balance with every
  template's projected occurrences, which reintroduces exactly the kind
  of "if you skip one, everything after is wrong" fragility the preview
  framing was chosen to avoid.

## Migration Plan

- New `user_settings.recurring_preview_horizon` column, nullable, hardcoded
  default — no backfill, matching every other preference in that table.
- No destructive changes anywhere. Every surface's preview defaults to
  **off** (no URL param, no card config field, means disabled) — existing
  users see zero behavior change until they explicitly opt in on a given
  page or card.
- Single deploy, no feature flag beyond the opt-in toggles themselves.
- Rollback: revert the deploy. The new column and endpoint are additive;
  a rolled-back binary simply stops serving `/api/recurring-transactions
  /preview` and ignores the unused column.

## Open Questions

- None outstanding — the cutoff formula, overdue handling, block
  placement vs. interleaving, bar-chart mechanics, and total/summary
  scope were all resolved during exploration before this proposal was
  written.

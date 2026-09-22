## Context

Two features already exist and this proposal composes them, changing
neither's own contract:

- `GET /api/entries/balance-series` (`account-entries`) returns one
  `BalancePoint` per local day of a month (plus a closing point), each a
  real running balance computed purely from persisted entries — no
  recurring-transaction awareness at all, and this proposal adds none:
  every future day's "real" value is simply flat at the last point any
  real entry could affect it (there are normally no real future-dated
  entries, so in practice this is "today's balance," but the formula below
  holds even if one exists).
- `GET /api/recurring-transactions/preview` (`recurring-transactions`)
  returns a flat, unsorted list of virtual occurrences
  (`RecurringTransactionPreviewItem`: `account_id`, `account_currency`,
  `amount`, `booking_timestamp` (a `Date`), `overdue`) bounded by a
  caller-supplied `to`. It computes nothing cumulative — every existing
  consumer (`UpcomingBlock`, `BarChartCard`'s `bucketPreviewItems`) sums it
  client-side into whatever shape that surface needs.

`BarChartCard` already proves the pattern this proposal extends to a line
chart: fetch the flat preview list once per card, bucket/sum it
client-side, and pass the result alongside the real series as a parallel,
optional field the chart component renders as a visually distinct overlay.
No backend change was needed for `bar_chart`'s stacked segment, and none is
needed here either — a balance is just as reachable by summing preview
deltas onto the real running total as a bucket total is by summing them
onto a real bucket.

## Goals / Non-Goals

**Goals:**

- Reuse the existing `show_recurring_preview` field verbatim — no new
  config field, no new query parameter, no new response field.
- One shared cumulative-delta helper, used by both new call sites, instead
  of two near-identical implementations.
- `LineChart`'s projected-overlay capability is generic (series-level
  props, datum-level parallel array) — like `BarChart`'s, not hardcoded to
  "balance."

**Non-Goals:**

- No change to `GET /api/entries/balance-series` or
  `GET /api/recurring-transactions/preview`'s contracts.
- No persistence of the toggle's state anywhere (`entry_list`/`bar_chart`
  cards persist it as part of the card's own `config`, which this proposal
  reuses unchanged for `line_chart`; the account-details page's own toggle
  is transient UI state, matching that page's existing `chartView`/
  `balanceMonth` local-only convention — see "Account page toggle is local
  state, not a URL param" below).
- No change to the 366-occurrence-per-template cap, the horizon setting, or
  any other part of the preview projection's own rules.

## Decisions

### The cumulative formula: `projected(D) = real(D) + Σ(preview items, today ≤ date < D)`

For a balance point `D` (a calendar day), the real value `real(D)` already
correctly reflects every persisted entry up to that day — including any
real entry that happens to be future-dated, an edge case the formula does
not need to special-case. Adding the sum of every non-overdue preview
item whose `booking_timestamp` falls in `[today, D)` on top of that real
value gives the balance *as if* every such occurrence had already
happened — exactly the running-balance analogue of `bar_chart`'s stacked
segment.

The half-open interval `[today, D)` mirrors `GET /api/entries/balance-
series`'s own point semantics (an entry booked exactly at a point's day
boundary belongs to the *next* point, not the one that opens it — see its
spec's "A transaction moves the line the following day" scenario): a
preview item dated exactly `D` has not yet moved the balance as of `D`'s
own midnight, only from `D+1` onward.

At `D = today`, the interval `[today, today)` is empty, so
`projected(today) = real(today)` always — the two lines are numerically
equal at that point by construction, not by a special case in the
rendering code. This is what lets `LineChart` draw the dashed segment as a
visual continuation of the solid one rather than a separately-anchored
line: the chart just draws the projected path over whatever indices carry
a defined `projectedValues`, and the caller's job is only to make sure
that set starts at (and includes) today.

Overdue items (`booking_timestamp < today`) are excluded from the sum
entirely, mirroring `BarChartCard`'s `bucketPreviewItems` exactly — an
occurrence already overdue reads as "should already be a real entry," not
as a future assumption to project forward.

**(Discovered during implementation.)** The formula also needs an upper
bound: `cumulativePreviewDeltaByPeriod` takes the same resolved cutoff
used to fetch `items` (`GET /api/recurring-transactions/preview`'s own
`to`) and returns `undefined` for any `period > cutoff`, the same way it
already does for `period < today`. Without this, paging a chart's month
view to a period beyond the horizon (a real case — the account page's
month pager is independent of the cutoff, per "Cutoff resolution" below)
held the last known cumulative delta flat indefinitely, since nothing
beyond the fetch's own bound is known. That reads as "no further activity
after this point," which is false — it's "no further data was fetched,"
a different claim. Clamping to the cutoff makes the dashed line simply
stop, the same way `bar_chart`'s stacked segments already implicitly stop
(no preview item exists to stack onto a bucket beyond the fetch's own
`to`).

*Alternative considered:* fold overdue amounts into `real(today)` itself
(the account "should" already reflect them). Rejected — this would make
the chart show a number that disagrees with `GET /api/accounts/{id}/
balance` and every other balance display in the app, for no benefit the
existing Upcoming-block/bar-chart treatment doesn't already give overdue
items (a tinted row, not a silent balance adjustment).

### `cumulativePreviewDeltaByPeriod`: one shared helper, not two implementations

```ts
// frontend/src/lib/recurringPreview.ts
export function cumulativePreviewDeltaByPeriod(
  items: RecurringTransactionPreviewItem[],
  periods: string[], // ascending "YYYY-MM-DD", e.g. BalancePoint.period
  today: string,
  cutoff: string, // the same cutoff items was fetched with
): Record<string /* currency */, (number | undefined)[]>
```

Parallel in shape to `periods`: index `i` is `undefined` when
`periods[i] < today` (nothing to project — the real and displayed value
are the same, and `LineChart` should draw no dashed segment there), and
otherwise the cumulative sum defined above for that currency. Both new
call sites (`LineChartCard`, `accounts.$accountId.index.tsx`) call this
once per fetch and add its result onto the real per-currency value to
build each `LineChartDatum.projectedValues`.

*Alternative considered:* keep the cumulative math local to each call site
(the same way `bucketPreviewItems` lives inside `BarChartCard.tsx`, not in
`lib/`). Rejected specifically here — `bucketPreviewItems` has exactly one
caller; this cumulative math has two from the start (`LineChartCard` and
the account page), so inlining it twice would duplicate the half-open-
interval logic above, the exact kind of shared math `frontend/AGENTS.md`
says belongs lifted out rather than copied.

### `LineChart`'s projected overlay mirrors `BarChart`'s shape exactly

`LineChartSeries` gains `projectedStrokeClassName?` and `projectedLabel?`
(vs. `BarChart`'s `projectedFillClassName`/`projectedLabel`);
`LineChartDatum` gains `projectedValues?: number[]`, parallel to `values`.
Rendering:

- **Path**: for each series with `projectedStrokeClassName`, draw a second
  `<path>` using the same step-after (`stepAfter`) construction the real
  line already uses, but only over the indices where that series'
  `projectedValues[i]` is defined (in practice always a trailing
  contiguous run starting at "today"), with `strokeDasharray` and the
  series' `projectedStrokeClassName` in place of `strokeClassName`. No
  bridging logic is needed in the chart — the cumulative formula above
  already guarantees the first such index's value equals the real line's
  value there, so the dashed path starts exactly on the solid path.
- **Legend**: `LineChart` currently only renders its legend when
  `series.length > 1` (a single-series chart, the common case for an
  account's own balance, shows no legend today). A single real series with
  a projected variant now needs one anyway — to label "solid = actual,
  dashed = projected" — so the condition widens to
  `series.length > 1 || series.some((s) => s.projectedStrokeClassName && s.projectedLabel)`,
  and a second legend pass (mirroring `BarChart`'s) lists each series that
  has both `projectedStrokeClassName` and `projectedLabel`, with a dashed
  swatch instead of a solid one.
- **Tooltip**: an additional row per projected series, `tooltipDatum.
  projectedValues?.[i]`, shown only when defined — same
  `s.projectedLabel`-gated pattern `BarChart`'s tooltip already uses for
  its stacked segment.
- **`aria-label`**: each point's accessible label appends the projected
  value when present, mirroring `BarChart`'s group `aria-label` exactly.

A datum/series with no projected data renders pixel-identical to before
this capability existed — purely additive, same guarantee `BarChart`'s own
projected-segment doc comment already states for itself.

### Account page toggle is local state, not a URL param

`/entries`' `show_recurring` toggle is a URL search param (so the state
survives a reload and is shareable/bookmarkable, matching every other
filter on that page). The account details page's chart view (`chartView`)
and displayed month (`balanceMonth`) are, by contrast, always local
`useState` that resets to its default (`"flow"`, the current month) on
every reload — there is no existing precedent on this page for persisting
chart state in the URL. The new "Show recurring assumptions" toggle
follows that page's own existing convention (local state, resets on
reload) rather than importing `/entries`' URL-param pattern — consistency
with the page it's added to outweighs consistency with a toggle on an
unrelated page.

### Cutoff resolution mirrors `bar_chart`'s, not `/entries`'

Neither the account page's balance chart nor the `line_chart` card has a
page-level date-range filter to intersect with (the chart's own month
pager is a *display* window, not a query filter passed to the preview
endpoint) — so both resolve the preview cutoff exactly the way
`BarChartCard` already does: `resolvePreviewCutoff(undefined, horizon,
today)`, the horizon setting alone. The `periods` array passed to
`cumulativePreviewDeltaByPeriod` is always the currently-displayed month's
points regardless of that cutoff; a period beyond the cutoff simply never
matches any preview item (the fetch itself is already bounded by `to`), so
no separate clamping is needed in the cumulative helper.

### `line_chart`'s toggle label is the existing generic one, not a new string

`CardFormDialog`'s `show_recurring_preview` checkbox already reads "Show
upcoming recurring occurrences" for `entry_list`/`bar_chart` — generic
enough (it names the feature, not the specific visual effect) to reuse
verbatim for `line_chart` rather than branching its label by type. The
account page's own toggle, being a distinct standalone control (not a
saved card config), gets its own new string, "Show recurring assumptions"
(`accounts.details.balanceChart.showRecurringAssumptions`), matching the
product framing this capability was requested under. Both surfaces reuse
one new `dashboard.lineChart.projectedBalance` key ("Balance (projected)")
as the projected series' legend/tooltip label — the same "Balance" label
`accounts.details.balanceChart.series` already provides for the real
series, reused by `LineChartCard` today.

## Risks / Trade-offs

- **A visitor could misread the dashed projection as guaranteed.** Same
  risk `bar_chart`'s existing stacked segment already carries, mitigated
  the same way — a visually distinct (dashed, muted-color) treatment and a
  "(projected)" label, not a new problem this proposal introduces.
- **Cross-month correctness depends on the flat-future-balance assumption.**
  If a real, already-booked future-dated entry exists on the account, the
  formula still holds (see "Decisions" above) — flagged here only because
  it's a departure from "there are no real future entries" being true in
  the common case, not because the formula breaks.

## Open Questions

None outstanding.

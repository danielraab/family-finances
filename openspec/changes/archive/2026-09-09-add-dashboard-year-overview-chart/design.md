## Context

`/accounts/{id}` (`accounts.$accountId.index.tsx`) already renders an
income/outcome bar chart: it holds `chartYear` state, fetches
`GET /api/entries/flow-summary?account_id={id}&unit=month&year={year}` in
an effect, maps each `FlowBucket` to a `{ category, values: [income,
outcome] }` datum filtered to the account's currency, renders a
previous/next-year pager, and passes the data to the presentational
`charts/BarChart`. That endpoint already treats an omitted `account_id`
as "every account the caller owns" and already returns income/outcome as
`CurrencySum[]` (one entry per currency), so the dashboard version needs
no backend work — only a second consumer of the same shape.

`frontend/src/components/charts/` is fetch-free by convention
(`frontend/AGENTS.md`): those components take data as props. The reusable
piece here is bigger than a chart — it includes a fetch and a pager — so
it can't live there.

## Goals / Non-Goals

**Goals:**

- Add an all-accounts year-overview chart to `/home` below the cards grid.
- Do it by extracting one reusable component, leaving the account-details
  page smaller than before.
- Handle multiple currencies by rendering one chart each.
- No backend, OpenAPI, or dependency changes.

**Non-Goals:**

- No running-balance line chart on the dashboard (no flow/balance toggle);
  a multi-account balance line is out of scope.
- No currency conversion or a combined "total" chart across currencies.
- No cross-page state sharing (each page's chart keeps its own year).
- No new data-fetching abstraction — plain `fetch`-in-effect, matching the
  rest of the frontend.

## Decisions

### Extract `<FlowChart>` at `src/components/FlowChart.tsx`, not under `charts/`

It owns the flow-summary fetch, the `chartYear` state + pager UI, the
`FlowBucket`→`BarChartDatum` mapping with localized `toLocaleDateString`
month labels, and the `INCOME_FILL` / `OUTCOME_FILL` constants. It renders
one or more `charts/BarChart`. Keeping it out of `charts/` preserves that
folder's "presentational only" invariant.

Props:

```
accountIds?: string[]          // omitted → all the caller's accounts
currency?: string              // given → one chart for it; omitted → one per currency
displayedDecimalPlaces: number // passed in, not hook'd internally
```

- Account-details: `<FlowChart accountIds={[accountId]} currency={account.currency} displayedDecimalPlaces={dp} />`
- Dashboard: `<FlowChart displayedDecimalPlaces={dp} />`

*Alternative considered:* leave the account page as-is and copy the
fetch/map/pager into `home.tsx`. Rejected — three near-identical blocks
(fetch, mapping, pager JSX) drifting independently, and the proposal's
own framing is that the second use is what justifies the extraction.

*Alternative considered:* put only the mapping helper in
`charts/internal.ts` and duplicate the fetch + pager. Rejected — the fetch
and pager are the bulk of the duplication; a lone helper barely helps.

### `displayedDecimalPlaces` is a prop, not an internal `useDisplayedDecimalPlaces()` call

The account-details page already calls that hook for its balance readout;
if `<FlowChart>` also called it, that page would issue `GET /api/settings`
twice. Passing the number in keeps the request count unchanged there, and
`home.tsx` adds the one hook call it needs.

### Currencies come from the response, not the accounts list

In the `currency`-omitted mode:
`[...new Set(buckets.flatMap(b => [...b.income, ...b.outcome].map(s => s.currency)))].sort()`.
Response-derived means no cross-referencing the accounts list and no
"account exists but had no activity this year" edge case — a currency with
zero flow in the selected year simply doesn't render a chart. Each chart
gets an `<h3>`-level heading with the currency code; headings show in this
mode even when only one currency is present (cheap, and consistent).

### Effect keys off a serialized account-id list

`accountIds={[accountId]}` is a fresh array every parent render. The fetch
effect depends on `(accountIds ?? []).join(",")` and `chartYear`, not the
array identity, so it refetches only on a real change.

### `<FlowChart>` owns `chartYear`; it resets on unmount

On `/accounts/{id}` the flow/balance toggle unmounts `<FlowChart>` when
"balance" is selected (the existing `chartView === "flow" ? … : …`
ternary), so switching away and back resets the year to the current one.
This matches today's behaviour closely enough (today the state persists,
but the year is rarely changed then toggled) and avoids lifting state into
both call sites. Noted as a minor, accepted change.

### i18n keys move to a shared `flowChart.*` namespace

`accounts.details.chart.{title,income,outcome,previousYear,nextYear}` →
`flowChart.*` in `en.json` and `de.json`, with the account page updated to
the new keys. `accounts.details.chart` had only those five keys, so the
sub-object is removed rather than left half-empty. `de.json` is allowed to
lag `en.json` per `frontend/AGENTS.md`, but since these are pure renames
of existing translations, both locales move together.

### Dashboard placement

A new `<section>` after the cards grid, inside the existing
`max-w-5xl` column, rendered only in the non-empty branch of `home.tsx`
(the empty state already `return`s its own markup for the no-accounts
case, so nothing extra is needed to hide the chart there). Charts stack
full-width; the per-currency `<h3>` headings separate them.

## Risks / Trade-offs

- **Tall page for multi-currency households** → each `BarChart` is
  ~260px; three currencies is a long scroll. Mitigation: single-currency
  (the common case) is unaffected; alphabetical order is at least
  predictable. A collapse-others disclosure was considered and judged
  over-engineered for a rare case.
- **Refactor touches a shipped, spec'd feature** (the account-details flow
  chart) → behaviour must stay identical. Mitigation: the
  `web-client-accounts` delta is MODIFIED-not-changed (same scenarios,
  reworded to name the component); manual check of both pages after the
  swap; `pnpm lint && pnpm exec tsc && pnpm build` before done.
- **Two `flow-summary` requests when both charts are visible** — not
  possible: they're on different routes. On `/accounts/{id}` only the
  flow *or* balance series loads at once, unchanged.
- **`chartYear` reset on toggle** (see Decisions) → accepted, documented.

## Open Questions

None outstanding — currency source, component extraction, no toggle, and
disabled-account inclusion were settled during exploration.

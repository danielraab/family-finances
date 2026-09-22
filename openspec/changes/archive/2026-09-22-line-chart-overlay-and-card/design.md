## Context

`BarChart.tsx` positions its tooltip with a `useLayoutEffect` that: finds
the active group's element via `wrapper.querySelector('[data-group-index="…"]')`,
measures it and the wrapper with `getBoundingClientRect()`, computes a
horizontally-clamped `left` and a `top` anchored to the element's own top
edge, and re-runs on `scroll`/`resize` while the tooltip is visible. It
renders inside a dedicated non-scrolling `wrapperRef` div that wraps the
`overflow-x-auto` container — not inside the scrolling container itself,
since `overflow-x-auto` alone forces `overflow-y` to compute as `auto`
too, which would clip a tooltip poking above the chart.

`LineChart.tsx` has none of this. It renders a second, always-mounted
block below the `<svg>` (`aria-hidden` and `invisible` when idle, purely
to hold the layout height) showing the active/pinned point's values. Both
charts already share `usePinnableSelection` (the hover/pin/Escape state)
from `internal.ts` — only the *positioning and rendering* of the value
display differ.

Separately, `dashboard-cards`' `bar_chart` type proves the pattern for
adding a chart-backed card: a type-specific config subset, a `*Card.tsx`
component owning its own fetch/pager, a `home.tsx` branch, and a
`CardFormDialog` branch. `line_chart` follows the same shape, but its data
source — `GET /api/entries/balance-series` — has a narrower contract than
`GET /api/entries/flow-summary`: it explicitly rejects `category_id`,
`tag_id`, `from`, `to`, and `q` (a category/tag-filtered "balance" isn't a
balance), and its `unit` parameter has only one defined value (`day`) — so
there is no `bar_chart`-style month/day toggle to expose.

## Goals / Non-Goals

**Goals:**

- One shared, tested-once positioning mechanism for both charts' overlay
  tooltips.
- `LineChart` visually and interactively matches `BarChart`'s tooltip
  behavior, including touch (click/tap-to-pin, click-away/Escape release).
- A `line_chart` dashboard card, scoped to a single optional account, with
  no new backend endpoint.

**Non-Goals:**

- No multi-account `line_chart` filter (explicit account list) — single
  account or "every account, summed per currency" only, matching this
  round's scope.
- No new `balance-series` granularity (a trailing 6/12-month view) — the
  card uses the existing month-at-a-time endpoint unchanged.
- No `show_recurring_preview` on `line_chart` — a running balance isn't an
  income/outcome bucket, so the projected-segment feature has no
  equivalent shape here.
- No spec changes to `web-client-accounts`'s existing running-balance-chart
  requirement — it describes the chart's axis/step-line/month-switcher
  behavior, never its tooltip's rendering mechanism, so replacing the
  mechanism doesn't change anything it specifies (see "Spec impact"
  below).

## Decisions

### `useOverlayTooltipPosition` takes a selector, not chart-specific knowledge

```ts
// frontend/src/components/charts/internal.ts
export function useOverlayTooltipPosition({
  wrapperRef,
  tooltipRef,
  visible,
  anchorSelector,
}: {
  wrapperRef: React.RefObject<HTMLDivElement | null>;
  tooltipRef: React.RefObject<HTMLDivElement | null>;
  visible: boolean;
  anchorSelector: string | null; // null when nothing is active
}): { left: number; top: number } | null
```

Body is `BarChart`'s existing effect verbatim, generalized: `wrapper.querySelector<Element>(anchorSelector)`
in place of the hardcoded `[data-group-index="…"]` template, guarded by
`anchorSelector === null` the same way the current code guards on
`activeIndex === null`. Each caller builds its own selector and passes
`null` when idle:

- `BarChart`: `activeIndex !== null ? \`[data-group-index="${activeIndex}"]\` : null`
- `LineChart`: `activeIndex !== null ? \`[data-point-index="${activeIndex}"]\` : null`

*Alternative considered:* pass the element directly (a ref or a resolved
`Element`) instead of a selector. Rejected — the active element changes
identity every render (a new `<g>` is whichever one currently matches
`activeIndex`), so the caller would need its own `querySelector` call
anyway; a selector keeps that lookup inside the hook, matching what
`BarChart` already does today.

The hook returns the same `{ left, top } | null` shape `BarChart`'s local
`tooltipPosition` state already has; both charts keep rendering their own
tooltip JSX (content differs — `LineChart`'s is simpler, one row per
series with no projected-segment handling) and keep applying the shared
`TOOLTIP_GAP`/`-translate-x-1/2 -translate-y-full`/`invisible`-until-
positioned convention `BarChart` established.

### `LineChart`'s DOM gains a `wrapperRef`, matching `BarChart`'s shape

Today `LineChart` renders `<div ref={containerRef} className="overflow-x-auto">`
directly with no outer wrapper. It needs the same two-layer structure
`BarChart` uses — an outer `position: relative` `wrapperRef` div containing
the scrolling `containerRef` div, with the floating tooltip as a sibling
of the scrolling div, inside the wrapper — for the same clipping reason
`BarChart`'s own comment documents (`overflow-x-auto` alone makes
`overflow-y` compute as `auto`, turning the scroll container into a
vertical clip box too).

### Each point's `<g>` gains `data-point-index`

`BarChart`'s group `<g>` already carries `data-group-index={groupIndex}`
for exactly this lookup. `LineChart`'s per-point `<g>` (the one holding the
hit-target `<rect>`, the active rule line, and the series dots) gains the
matching `data-point-index={pointIndex}` attribute — the only new markup
`LineChart` needs beyond removing the old bottom box and adding the
floating tooltip.

### The bottom value box is deleted, not repurposed

`LineChart`'s current bottom block is always mounted (just visually
hidden) specifically to reserve layout height so hovering doesn't shift
the page. The floating tooltip needs no such placeholder — like
`BarChart`'s, it's conditionally rendered only while
`tooltipVisible && tooltipDatum`, absolutely positioned, and
`pointer-events-none`, so removing the placeholder entirely (rather than
finding it a new job) is correct, not just simpler. A single-series
`LineChart` (the common case — an account's own balance) therefore loses
its only always-visible value readout and gains the same
hover/tap-to-reveal behavior a single-series `BarChart` already has; this
was confirmed acceptable rather than needing a replacement always-on
readout.

### Tooltip content and pin behavior are unchanged, only relocated

`LineChart` already computes `tooltipDatum` from `usePinnableSelection`'s
`activeIndex`; the new floating tooltip renders the same category +
per-series rows the old bottom box did, just in `BarChart`'s floating
markup shape instead. No change to what triggers a pin (click/tap a
point) or a release (click away, click the same point again, Escape) —
that's all already `usePinnableSelection`, untouched by this change.

### `line_chart` card: single optional `account_id`, no category/tag/range/unit

Mirrors `dashboard.go`'s existing per-type table:

| Field | `account_stat` | `bar_chart` | `line_chart` (new) |
|---|---|---|---|
| `account_id` | required | optional | optional |
| `category_id`/`include_subcategories`/`tag_id` | — | optional | — |
| `unit` | — | required | — |
| `title` | — | optional | optional |
| `show_recurring_preview` | — | optional | — |

`validateShape`'s new `CardTypeLineChart` case rejects `CategoryID`,
`IncludeSubcategories`, `TagID`, `Range`, `Unit`, `Columns`, and
`ShowRecurringPreview` whenever any is non-nil — an empty config (no
`account_id`, no `title`) is valid, meaning "every account, this month."
This is narrower than `bar_chart`'s allowed set, not a subset relation
inverted — `line_chart` simply has nothing to filter by beyond the account
itself, because `balance-series` has nothing to filter by beyond the
account itself.

*Alternative considered:* give `line_chart` a `Unit` field for future
parity with `bar_chart`, even though only one value (`day`) is defined
server-side today. Rejected as speculative — `balance-series`'s OpenAPI
`unit` parameter is `enum: [day]` with no second value even in the
endpoint's own contract; adding a client-side choice with one option is
pure ceremony. If a month-granularity balance endpoint is ever added,
that's a new capability with its own design, not a field sitting unused
until then.

### `LineChartCard.tsx` mirrors `BarChartCard.tsx`'s per-currency stacking and pager shape, scaled down

- State: `year`/`month` (default: current month), `balancePoints`.
- Fetch: `GET /api/entries/balance-series` with
  `account_id: config.account_id ? [config.account_id] : undefined`,
  `unit: "day"`, `year`, `month` — no `unit` branch to choose, unlike
  `BarChartCard`.
- Pager: previous/next **month** only (rolling the year at the Dec/Jan
  boundary), not `BarChartCard`'s year-or-month choice — there is only one
  granularity here.
- Currencies: derived from the response's `balances[].currency` the same
  way `BarChartCard` derives them from `income`/`outcome`; one `LineChart`
  rendered per currency, each headed by its code when more than one is
  present (same convention `BarChartCard`/`FlowChart` already use).
- Series: a single `{ label: t("accounts.details.balanceChart.series"), strokeClassName: BALANCE_STROKE, dotClassName: BALANCE_DOT }`
  — reusing the account-details page's existing i18n key (the label reads
  the same, "Balance", in both places) and redeclaring the same
  `BALANCE_STROKE`/`BALANCE_DOT` color constants locally, matching how
  `BarChartCard.tsx` already redeclares `INCOME_FILL`/`OUTCOME_FILL`
  rather than importing them from `FlowChart.tsx`.
- Heading: `<CardTitleLink>`, identical to `BarChartCard`'s usage — it's
  already generic over `config` and needs no type-awareness, so
  `line_chart` joins the title/heading-link behavior with no changes to
  that component.

### `home.tsx`/`CardFormDialog.tsx` changes are additive, not restructured

- `segmentCards`'s full-width-segment check
  (`card.type === "bar_chart"`) becomes
  `card.type === "bar_chart" || card.type === "line_chart"`.
- `renderCardContent` gets one more `case`.
- `CardFormDialog`'s category/tag/`include_subcategories` block, currently
  gated on `type !== "account_stat"`, becomes
  `type !== "account_stat" && type !== "line_chart"` — the only condition
  that needs widening, since `unit`/`columns`/`show_recurring_preview`/
  `DateRangeFilter` are already gated to the specific types that use them
  and none of those include `line_chart`. The account `<select>` and title
  `<input>` are already unconditional (beyond the `account_stat` title
  exclusion) and need no change — a `line_chart` card gets both for free.
- `submit()`'s `config` builder needs the same
  `type !== "account_stat" && type !== "line_chart"` guard added to
  `category_id`/`include_subcategories`/`tag_id`, so a `line_chart`
  submission never sends fields its own type rejects.

### No spec change to `web-client-accounts`

That capability's running-balance-chart requirement describes observable
behavior (axis framing, step interpolation, zero rule line, month
switcher, "a reusable presentational component that fetches nothing") —
none of which changes. The tooltip's rendering *mechanism* was never a
spec'd requirement (grepping every `specs/*.md` file for "tooltip"/
"legend" turns up nothing about either chart's hover/tooltip behavior —
it has only ever been an implementation detail, consistent with the
precedent `BarChart`'s own tooltip-positioning rework set: that change
shipped with no spec delta at all). This proposal follows the same
precedent for `LineChart`'s equivalent rework.

## Risks / Trade-offs

- **DOM restructure risk**: `LineChart` picking up `BarChart`'s
  wrapper/container split changes its outer markup; any external styling
  assumption about `LineChart`'s root node would need re-checking.
  Mitigation: `LineChart` has exactly one call site
  (`accounts.$accountId.index.tsx`) today plus the new `LineChartCard`,
  both of which just render `<LineChart .../>` inside their own flex
  layout — no external CSS reaches into its internals.
- **Losing the always-visible value box** could read as a regression for
  a visitor who expects to see the current/latest value without
  interacting. Mitigation: this exact trade-off already exists for
  `BarChart` today and hasn't been raised as a problem; explicitly
  accepted per this round's scope ("the floating tooltip alone is fine so
  far").
- **`line_chart`'s single-account scope reads as a smaller feature than
  `bar_chart`'s** (which can filter by category/tag too). Mitigation:
  this mirrors what `balance-series` itself supports — there is no
  category/tag-filtered "balance" to build a richer filter on top of;
  widening the card's filter surface would require widening the endpoint
  first, out of scope for this round.

## Open Questions

None outstanding — the hook's shape, the DOM restructure, dropping the
always-visible value box, and the card's single-account scope were all
settled during exploration.

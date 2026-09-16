## Why

`BarChart` and `LineChart` diverged in how they reveal per-point values.
`BarChart` floats a tooltip over the plot, anchored to the hovered/tapped
bar group's actual on-screen position, pinnable by tap so it works without
a mouse. `LineChart` instead reserves a fixed-height box glued below the
chart that always occupies space, even idle — a different mechanism doing
the same job, and the one place a visitor sees the two hand-rolled charts
behave inconsistently. `LineChart` should adopt `BarChart`'s floating,
click-to-pin overlay instead, so both charts read as one system. Lifting
the positioning math the two now share into `charts/internal.ts` avoids
duplicating it, matching `frontend/AGENTS.md`'s "extend a chart by lifting
anything genuinely shared into `internal.ts`" rule.

Separately, `LineChart` itself has only ever been reachable from
`/accounts/{id}` — there's no way to put a running-balance line on the
`/home` dashboard the way an income/outcome bar chart already can via the
`bar_chart` card. A `line_chart` card closes that gap, reusing the
existing `GET /api/entries/balance-series` endpoint (already scoped to a
single caller-chosen account, or every account summed per currency when
none is chosen) with no backend endpoint changes — only the dashboard
card type needs to learn about it.

## What Changes

- **New `useOverlayTooltipPosition` hook** in
  `frontend/src/components/charts/internal.ts`: the container-relative,
  scroll/resize-tracking, horizontally-clamped positioning logic
  `BarChart` already has, generalized to take a CSS selector for the
  active point/group's anchor element instead of assuming bars.
- **`BarChart` refactored** to call the shared hook instead of its own
  copy of the same effect — no behavior change.
- **`LineChart` adopts the same floating, click-to-pin tooltip** `BarChart`
  already has, in place of its current always-rendered, fixed-height value
  box below the chart. Each point's `<g>` gains a `data-point-index`
  attribute (mirroring `BarChart`'s `data-group-index`) so the shared hook
  can anchor to it. The chart's legend (shown only for 2+ series, already
  the existing rule) is unchanged; a single-series chart's only value
  display becomes the floating tooltip, same as a single-series `BarChart`
  would have today.
- **New `line_chart` dashboard card type**, alongside `account_stat` /
  `query_stat` / `entry_list` / `bar_chart`: a running-balance line chart
  via `GET /api/entries/balance-series`, scoped to an optional single
  `account_id` (omitted means every account the caller owns, summed per
  currency — one `LineChart` per currency present in the response, same
  per-currency stacking `bar_chart` cards already use), with a
  previous/next-month pager (that endpoint only ever samples one month at
  `unit: day` — there is no month/year mode to choose, unlike `bar_chart`).
  It accepts the same optional `title` `query_stat`/`entry_list`/
  `bar_chart` cards do, with the same generated-summary fallback and
  clickable link through to `/reports`. It always renders full width, one
  per row, like `bar_chart`.
- **`CardFormDialog`** gains `line_chart` in its type picker; its existing
  account/title fields already cover a `line_chart` card unchanged, and its
  category/tag fields are hidden for `line_chart` the same way they would
  need to be for a type with no such filter (its config accepts neither).
- **`home.tsx`** renders `line_chart` cards via a new `LineChartCard.tsx`
  (mirroring `BarChartCard.tsx`) and treats them as full-width segments
  the same way `bar_chart` cards already are.
- **Seed fixtures** (`backend/internal/cli/fixtures.go`) gain one
  unfiltered `line_chart` card in the starter dashboard layout, so a fresh
  seed demonstrates the new type without manual setup.

## Capabilities

### Modified Capabilities

- `dashboard-cards`: adds the `line_chart` card type and its config
  validation rules (optional `account_id`, optional `title`, no other
  field).
- `web-client-home`: adds `line_chart` to every card-type-enumerating
  requirement (rendering, full-width layout, the title/heading link, the
  edit-mode type picker).

## Impact

- **Backend**: `internal/dashboard/dashboard.go` gains `CardTypeLineChart`
  and its `validateShape` case; `openapi/openapi.yaml` (and its synced
  `backend/openapi.yaml` copy) gain `line_chart` in `DashboardCardType`'s
  enum and description. No new endpoint, no migration — validation is the
  only backend logic involved, and `GET /api/entries/balance-series`
  already supports every filter shape a `line_chart` card needs.
  `backend/internal/cli/fixtures.go`'s `generateDashboardCards` gains one
  more seeded card.
- **Frontend**: `charts/internal.ts` gains `useOverlayTooltipPosition`;
  `BarChart.tsx` and `LineChart.tsx` both change to use it (`LineChart.tsx`
  loses its fixed bottom value box and gains `data-point-index` attributes
  and the floating tooltip markup, mirrored from `BarChart.tsx`); new
  `components/dashboard/LineChartCard.tsx`; `CardFormDialog.tsx` and
  `home.tsx` each gain a small `line_chart` branch;
  `src/api/schema.d.ts` regenerated from the updated contract. New i18n
  keys for the card's type-picker label and month pager, added to both
  `en.json` and `de.json`.
- No new dependencies.

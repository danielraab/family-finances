## Why

`/home` today is a fixed, non-configurable layout — every account's card
plus one all-accounts income/outcome chart, in that order, for every
visitor. It cannot show a single account's own trend, a filtered
category/tag view, or a recent-activity list, and no two users can want a
different arrangement. A dashboard built from user-placed cards lets each
visitor surface the numbers they actually care about, without turning
`/reports` (still a one-shot filter form, not a saved concept) into the
dashboard's data model.

## What Changes

- Add a per-user, ordered set of dashboard cards, each one of four types:
  - **Account stat** — an account's live balance.
  - **Query stat** — the sum/count (`GET /api/entries/summary`-shaped)
    for an inline filter: optional account, category (+ include
    subcategories), tag, and date range.
  - **Entry list** — the most recent entries (fixed page size) matching
    an account or the same inline filter as a query stat.
  - **Bar chart** — an income/outcome bar chart, year or month view, for
    an account or an inline-filtered query, reusing the existing
    `BarChart`/`FlowChart` building blocks.
  - Each card's filter is a plain, inline value on the card itself —
    there is no separate saved-query entity, and `/reports` is untouched.
- Add an edit mode on `/home`: add a card (type picker → per-type config
  form reusing `/reports`' existing account/category/tag/date-range
  controls), remove a card, and reorder via ▲/▼ move buttons — no
  drag-and-drop, matching this app's existing house convention
  (`/categories`).
- Layout: bar-chart cards always render full width, one per row; every
  other card type flows in a responsive grid (3-4 per row depending on
  viewport).
- Two distinct empty states on `/home`: no accounts at all keeps today's
  "create your first account" prompt; accounts exist but no cards yet
  shows a new "add your first card" prompt.
- A card referencing an account/category/tag the visitor has lost access
  to (an unshared/deleted entity) SHALL render a "no longer accessible"
  placeholder instead of erroring, and stays removable like any other
  card.
- **BREAKING**: `/home`'s fixed account-cards-plus-chart view is removed.
  Every existing user's dashboard starts empty after this ships — nothing
  is auto-migrated from the old fixed view (deliberate: see design.md).
- Extend `GET /api/entries/flow-summary` to accept the same `category_id`
  (+ `category_mode`) and `tag_id` filters `GET /api/entries/summary`
  already accepts, so a query-scoped bar chart is possible.

## Capabilities

### New Capabilities

- `dashboard-cards`: backend domain package owning per-user dashboard
  cards — persistence, ordering (move-up/move-down), and validating each
  card's inline filter config against the caller's actual account/
  category/tag access.

### Modified Capabilities

- `web-client-home`: `/home` becomes the customizable card dashboard
  described above (edit mode, four card types, two empty states)
  in place of the fixed account-grid-plus-chart view.
- `account-entries`: `GET /api/entries/flow-summary` gains optional
  `category_id`/`category_mode`/`tag_id` filters, mirroring
  `GET /api/entries/summary`.

## Impact

- **Backend**: new `internal/dashboard` package (four-file shape) +
  migration adding a `dashboard_cards` table; `internal/entry`'s
  flow-summary handler/service/store gain category/tag filtering;
  `openapi/openapi.yaml` (and its two generated copies) gain the
  dashboard-cards endpoints and the extended flow-summary parameters.
- **Frontend**: `frontend/src/routes/home.tsx` rewritten around card
  rendering + edit mode; new per-card-type components (reusing
  `AccountCard`, `FlowChart`/`BarChart`, and `/reports`' filter controls
  where they fit); `src/api/schema.d.ts` regenerated from the updated
  contract.
- **Data**: new table only, no destructive migration; no data loss for
  existing accounts/entries — only the dashboard's own prior "layout"
  (which was never stored) goes away.

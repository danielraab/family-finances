## Why

`/recurring` has two filter controls — the pair of self-transfer
checkboxes — and nothing else. The entry ledger next door filters by
account, category, tag, kind, date range and free text; the recurring
list, which is the page a household actually plans from, cannot answer
"what are my subscriptions on the joint account?" or "what does
*Insurance* cost me per year?" without reading every row.

The backend is most of the way there already:

- `GET /api/recurring-transactions` and its `/summary` sibling already
  accept a repeatable `account_id`, but **no client sends it** — the list
  page never built a control for it.
- `GET /api/recurring-transactions/preview` already accepts
  `category_id`, `category_mode` and `tag_id`, resolved in
  `Service.Preview` against the caller's own category tree. The list and
  summary endpoints accept none of the three, so the same question asked
  of the upcoming block and of the list gets different answers.

So the list is the only one of the three recurring read endpoints that
cannot be narrowed by content, and the one page that reads them has a
control for neither.

## What Changes

- `GET /api/recurring-transactions` and
  `GET /api/recurring-transactions/summary` accept `category_id`,
  `category_mode` (`subtree` | `exact`, default `subtree`) and `tag_id`,
  resolved exactly the way the preview endpoint and the entry listing
  already resolve theirs. The summary keeps taking the same parameters as
  the listing, so the total below the rows is still the total *of* the
  rows.
- `Service.Preview` stops filtering by category and tag in its own loop
  and resolves the same `Filter` the listing does, so one resolution rule
  serves all three endpoints.
- `/recurring` gains a filter panel above the list carrying **Account**,
  **Category** and **Tag** selects beside the two existing self-transfer
  checkboxes. Every control's state lives in the URL, as the two
  checkboxes already do, and every filter is sent on both the list and
  the summary request.
- The panel is the one `/entries` already has: collapsible below `sm`
  with an active-filter count badge and a "Clear all filters" action.
  Its chrome is extracted into a shared `FilterPanel` component that
  `/entries` and `/recurring` both render, rather than a second copy of
  the same 50 lines.

## Non-goals

- **No free-text search, kind filter or date range on `/recurring`.** A
  recurring transaction has no booking timestamp to range over, and its
  kind is already what the self-transfer checkboxes select. Search is a
  separate question — the list is short enough to read.
- **No multi-select.** `account_id` stays repeatable on the wire (it
  already is), but the control is a single select, matching `/entries`.
- **No category- or tag-permission widening.** `/entries` lifts its
  account restriction when the caller holds permission on the filtered
  category or tag itself (`Filter.AllAccounts`). Recurring's listing has
  never done that, its preview endpoint does not either, and adding it
  here would change which templates a *shared* category exposes — a
  separate decision. The account scope stays "every account the caller
  can see", narrowed by `account_id`.
- **No new persistence, migration or view change.** The category and tag
  columns the filter reads already exist on
  `recurring_transaction_legs` and `recurring_transaction_tags`.

## Capabilities

### Modified Capabilities

- `recurring-transactions`: the list and summary endpoints accept the
  `category_id`/`category_mode` and `tag_id` filters the preview
  endpoint already accepts, resolved identically.
- `web-client-recurring-transactions`: `/recurring` offers account,
  category and tag filters alongside the self-transfer checkboxes, in a
  collapsible panel with an active count and a clear-all action, with
  every filter in the URL and applied to the summary as well as the
  list.

## Impact

- `backend/internal/recurringtransaction/recurringtransaction.go` —
  `Filter` gains the caller-supplied `CategoryID`/`CategoryMode`/`TagID`
  and the Service-resolved `CategoryIDs`; `PreviewFilter` drops the
  three it duplicates.
- `backend/internal/recurringtransaction/service.go` — `resolveFilter`
  replaces `resolveAccountIDs`; `Preview` reuses it.
- `backend/internal/recurringtransaction/handler.go` — `list` and
  `summary` parse the three new parameters.
- `backend/internal/storage/{postgres,memory}/recurringtransaction.go` —
  the two `List` implementations apply `CategoryIDs`/`TagID`.
- `openapi/openapi.yaml` (+ the two generated artifacts) — the new
  parameters on both operations.
- `frontend/src/components/FilterPanel.tsx` — new, shared.
- `frontend/src/routes/recurring.index.tsx` — the panel and its three
  controls.
- `frontend/src/routes/entries.index.tsx` — renders the extracted panel.
- `frontend/src/i18n/locales/{en,de}.json` — the panel's shared chrome
  keys move to a top-level `filters` namespace; `/recurring` gains its
  own control labels.

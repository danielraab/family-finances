## 1. API contract

- [x] 1.1 In `openapi/openapi.yaml`, add `DashboardCard` schema (`id`,
      `type` enum `account_stat`/`query_stat`/`entry_list`/`bar_chart`,
      `config` object, `sort_order`, timestamps) and the six
      `/api/dashboard/cards` operations (list, create, update, delete,
      move-up, move-down) per `dashboard-cards`'s spec.
- [x] 1.2 In `openapi/openapi.yaml`, add `category_id`, `category_mode`,
      `tag_id` query parameters to `GET /api/entries/flow-summary`, per
      `account-entries`'s modified requirement.
- [x] 1.3 Sync `backend/openapi.yaml` (`cd backend && go generate ./...`)
      and regenerate `frontend/src/api/schema.d.ts`
      (`cd frontend && pnpm generate:api`); confirm no other diff.

## 2. Backend: dashboard_cards persistence

- [x] 2.1 Add migration
      `backend/internal/storage/postgres/migrations/0027_dashboard_cards.sql`
      creating `dashboard_cards` (`id`, `user_id` FK, `type`, `config
      jsonb`, `sort_order`, `created_at`, `updated_at`).
- [x] 2.2 Create `backend/internal/dashboard/dashboard.go`: `Card` domain
      type, the four-type `CardType` enum, and per-type config field
      validation (required/allowed fields per type, per the spec's
      table).
- [x] 2.3 Create `backend/internal/dashboard/store.go`: `Store` interface
      (`List`, `Create`, `Update`, `Delete`, `MoveUp`, `MoveDown`, all
      scoped to `ownerID`) + sentinel errors (`ErrNotFound`,
      `ErrInvalidValue`).
- [x] 2.4 Create `backend/internal/dashboard/service.go`: use-case logic,
      declaring `AccountLookup`/`CategoryLookup`/`TagLookup` interfaces
      (flat-value, mirroring `internal/entry`'s pattern) for the
      view-tier-or-above access checks on `account_id`/`category_id`/
      `tag_id` in `config`. Added `category.Service.Visible` (new method,
      mirroring `tag.Service.OwnedBy`'s existing contract) since no
      existing category method returned a plain view-tier boolean;
      `account.Service.Access` and `tag.Service.OwnedBy` already fit the
      needed interfaces with no changes.
- [x] 2.5 Create `backend/internal/dashboard/handler.go`: `http.Handler`
      for `/api/dashboard/cards...`, mapping service errors to statuses.
- [x] 2.6 Implement `internal/storage/memory`'s `DashboardStore`.
- [x] 2.7 Implement `internal/storage/postgres`'s `DashboardStore`
      (position swap logic mirroring `category.go`'s move-up/move-down).
- [x] 2.8 Wire `internal/dashboard` into `main.go`: construct the service
      with `account.Service`/`category.Service`/`tag.Service` satisfying
      the lookup interfaces, mount the handler under `/api/dashboard/`.
- [x] 2.9 Unit tests: domain validation (per-type required/allowed
      fields, invalid `type`, invalid `unit`), service tests (access
      checks against owned/shared/inaccessible accounts, categories,
      tags), handler tests (status codes, `openapicheck` response
      conformance), postgres store tests, memory store tests. Also added
      `category.Service.Visible` tests. Postgres integration tests run
      and pass against the local compose DB.

## 3. Backend: flow-summary filter extension

- [x] 3.1 Extend `internal/entry`'s flow-summary handler to accept
      `category_id`/`category_mode`/`tag_id`, reusing the same resolution
      `GET /api/entries/summary` already applies (subtree expansion,
      `exact` mode). Factored the shared category/tag resolution out of
      `resolveFilter` into `Service.resolveCategoryAndTag`, used by both
      `Sum`/`List` and `FlowSummary`.
- [x] 3.2 Extend the flow-summary service/store query (both
      `storage/memory` and `storage/postgres`) to apply those filters
      before bucketing. Postgres reuses `buildWhere` (same clauses
      List/Sum apply); memory reuses `matchingRows`.
- [x] 3.3 Tests: category filter (with and without subtree/`exact`), tag
      filter, combined with `account_id`, per the spec's new scenarios —
      added at both the service layer (memory-backed) and the postgres
      integration layer; all pass against the local compose DB.

## 4. Frontend: per-card-type rendering

- [x] 4.1 Add `src/api/client.ts`-typed helpers for the
      `/api/dashboard/cards` endpoints (list/create/update/delete/
      move-up/move-down). Done inline in `home.tsx`/`AddCardDialog.tsx`
      via the typed `api` client directly — no separate wrapper module
      needed, matching how other pages call `api.GET`/`POST` inline.
- [x] 4.2 Build `AccountStatCard` (adapts existing `AccountCard`: title,
      institute, live balance, sign-colored, add-entry button gated on
      `append`+). Reuses `AccountCard` as-is.
- [x] 4.3 Build `QueryStatCard` (per-currency sums from
      `GET /api/entries/summary`, `range` resolved via
      `resolveEffectiveRange` at render time).
- [x] 4.4 Build `EntryListCard` (last 10 matching entries via
      `GET /api/entries`, reusing the row rendering `reports.tsx` already
      has: date, title, `AccountLabel`, sign-colored amount).
- [x] 4.5 Build `BarChartCard` (wraps `BarChart`, extended to pass
      `category_id`/`tag_id` through to `flow-summary`; year pager for
      `unit: month`, month pager for `unit: day`).
- [x] 4.6 Build a shared "no longer accessible" placeholder card
      (`MissingReferenceCard`), used by any of the above when its
      config's `account_id`/`category_id`/`tag_id` doesn't resolve
      against the caller's own `/api/accounts`/`/api/categories`/
      `/api/tags` response (`cardReferencesResolve` in
      `lib/dashboardFilter.ts`).

## 5. Frontend: `/home` rewrite

- [x] 5.1 Rewrite `frontend/src/routes/home.tsx`: fetch
      `GET /api/dashboard/cards` (plus accounts/categories/tags for
      reference resolution), render each card by `type` in saved order.
- [x] 5.2 Grid layout: `bar_chart` cards full width, one per row; other
      types in a responsive grid
      (`grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4`), via
      `segmentCards` chunking the ordered list into grid/chart runs.
- [x] 5.3 Empty state 1 (no accounts at all): keep the existing
      "create your first account" prompt, unchanged.
- [x] 5.4 Empty state 2 (accounts exist, zero cards): new "add your first
      card" prompt that opens the add-card flow.
- [x] 5.5 Delete the old fixed account-cards-grid + `FlowChart` rendering
      path from `home.tsx` (superseded, not dual-run per design.md).

## 6. Frontend: edit mode

- [x] 6.1 Add an edit-mode toggle to `/home`.
- [x] 6.2 Per-card, in edit mode: remove action (immediate, no
      confirmation) and move-up/move-down buttons (disabled at list
      ends), calling the new endpoints and refetching/reordering the
      list. (`DashboardCardFrame`.)
- [x] 6.3 "Add card" control: type picker, then a per-type config form —
      reuse `/reports`' account/category/tag `<select>`s and
      `DateRangeFilter` for `query_stat`/`entry_list`/`bar_chart`
      (`bar_chart` gets a `unit` month/day choice instead of `range`);
      submit calls `POST /api/dashboard/cards` and appends the new card.
      (`AddCardDialog`.)
- [x] 6.4 Inline validation on the config forms (e.g. `account_stat`
      requires an account) before submit, mirroring the account-required
      pattern in `AccountForm.tsx`.

## 7. i18n

- [x] 7.1 Add every new user-facing string (edit-mode toggle, add/remove/
      move actions, type picker labels, per-type config form labels, both
      empty-state prompts, "no longer accessible" placeholder text) to
      `frontend/src/i18n/locales/en.json` first, then `de.json`. Reused
      the existing `reports.filters.*`/`reports.columns.*` keys for the
      shared account/category/tag/date-range controls rather than
      duplicating them; removed the now-dead `dashboard.yearOverview` key.

## 8. Verification

- [x] 8.1 Backend: `gofmt -l .`, `go vet ./...`, `go test ./...` (from
      `backend/`) — all clean, including the postgres integration suite
      run against the local compose DB.
- [x] 8.2 Frontend: `pnpm lint`, `pnpm exec tsc`, `pnpm build` (from
      `frontend/`) — all clean.
- [x] 8.3 Manually exercised `/home` in a real browser (Playwright
      against the dev server + a seeded user): the empty-cards state and
      its "Add card" CTA, creating an account_stat/bar_chart/query_stat/
      entry_list card each, the full-width chart vs. grid layout, edit
      mode's move-up/move-down/remove — all worked with zero console
      errors. (Did not additionally test the revoked-share placeholder
      path live; it's covered by `cardReferencesResolve`'s unit-level
      logic and code review instead.)
- [x] 8.4 Confirmed no drift: `openapi/openapi.yaml` and
      `backend/openapi.yaml` are byte-identical, and re-running
      `pnpm generate:api` against the current spec produces no further
      diff in `schema.d.ts`.

## 9. Follow-up: configurable entry_list card width

Added after the initial implementation: an `entry_list` card's grid span
is user-configurable (2-4 columns, default 2) rather than fixed at one
column like `account_stat`/`query_stat`.

- [x] 9.1 Add `columns` (`2`-`4`) to `DashboardCardConfig` in
      `openapi/openapi.yaml`; regenerate `backend/openapi.yaml` and
      `frontend/src/api/schema.d.ts`.
- [x] 9.2 `internal/dashboard`: add `Config.Columns *int`; validate in
      `validateShape` — allowed only on `entry_list`, range 2-4 inclusive
      when present. Unit tests for in-range, out-of-range, and
      rejected-on-other-types.
- [x] 9.3 `AddCardDialog`: a named width `<select>` (Narrow/Wide/Full
      width → `columns` 2/3/4) shown only for `entry_list`, defaulting to
      2; included in the submitted config.
- [x] 9.4 `home.tsx`: `entryListSpanClass(columns)` applies the matching
      literal `sm:col-span-*/lg:col-span-*/xl:col-span-*` classes to an
      `entry_list` card's grid wrapper; every other type unaffected.
- [x] 9.5 Updated `design.md` and the `dashboard-cards`/`web-client-home`
      spec deltas to document `columns` and its scenarios.
- [x] 9.6 Re-ran the full verification suite (`gofmt`/`go vet`/
      `go test ./...` incl. postgres integration; `pnpm lint`/`tsc`/
      `build`) — all clean.

## 10. Follow-up: editable cards + visually attached edit-mode toolbar

Added after the initial implementation: a card's config can be edited in
place (not just delete-and-recreate), and the edit-mode control bar reads
as attached to its card rather than a separate floating box above it.

- [x] 10.1 Renamed `AddCardDialog` → `CardFormDialog`; added an
      `editingCard: DashboardCard | null` prop. When set: the type
      selector is disabled, every other field seeds from the card's
      current `config` (keyed on the card's id, not object identity, so
      an unrelated re-render never clobbers in-progress edits), the
      dialog title/submit label switch to "Edit card"/"Save", and
      submitting calls `PATCH /api/dashboard/cards/{id}` instead of
      `POST`.
- [x] 10.2 `home.tsx`: `editingCard` state + `openAddDialog`/
      `openEditDialog`/`handleSaved` — the latter replaces the matching
      card in place by id on a PATCH response, or appends on a POST
      response, from one shared callback.
- [x] 10.3 `DashboardCardFrame`: added an `onEdit` callback and a "✎" edit
      button (between the ▲/▼ group and the ✕ remove button); restyled
      the toolbar to share the card's own rounded-corner/border box
      (`overflow-hidden` wrapper) with zero gap, instead of a separate
      dashed box floating above it.
- [x] 10.4 Added `dashboard.edit.edit`/`dashboard.editCard.heading`/
      `dashboard.editCard.typeImmutableHint` to `en.json`/`de.json`.
- [x] 10.5 Updated the `web-client-home` spec delta: new "edit an
      existing card" requirement and a new "controls render attached to
      the card" requirement.
- [x] 10.6 Verified live (Playwright against the dev server + a seeded
      user): the attached toolbar renders with no gap; the edit action
      opens the form correctly pre-filled (type disabled, filter fields
      matching); saving updates the card in place with zero console
      errors. Re-ran `pnpm lint`/`tsc`/`build` — all clean (no backend
      changes this round).

## 11. Follow-up: customizable card title, linking through to /reports

Added after the initial implementation: `query_stat`/`entry_list`/
`bar_chart` cards can carry an optional custom title, and the card's
heading (custom title or the existing generated filter summary) is now a
link to `/reports` with the card's own filter prefilled.

- [x] 11.1 Add `title` (string) to `DashboardCardConfig` in
      `openapi/openapi.yaml`, settable only on `query_stat`/`entry_list`/
      `bar_chart`; regenerate `backend/openapi.yaml` and
      `frontend/src/api/schema.d.ts`.
- [x] 11.2 `internal/dashboard`: add `Config.Title *string`; reject it in
      `validateShape` for `account_stat` only (no other type restricts
      it). Unit tests for accepted-on-filter-types and
      rejected-on-account-stat.
- [x] 11.3 `lib/dashboardFilter.ts`: add `cardReportsSearch(config)`
      mapping a card's filter to `/reports`' own search-param shape
      (`category_id`, `include_subcategories`, `tag_id`, `account_id`,
      `range`/`from`/`to` from `config.range`) — `unit`/`columns` have no
      equivalent there and are dropped.
- [x] 11.4 New shared `CardTitleLink` component: renders
      `config.title?.trim() || describeCardFilter(...)` as a
      `<Link to="/reports" search={cardReportsSearch(config)}>`; wired
      into `QueryStatCard`/`EntryListCard`/`BarChartCard` in place of
      their previous plain-text heading. `account_stat` unaffected — its
      `AccountCard` heading still links to the account's own details
      page.
- [x] 11.5 `CardFormDialog`: added an optional title `<input>`, shown for
      every type except `account_stat`; seeded from `editingCard.config
      .title` when editing, included (trimmed, dropped if empty) in the
      submitted config.
- [x] 11.6 Added `dashboard.addCard.titleLabel`/`titlePlaceholder` to
      `en.json`/`de.json`.
- [x] 11.7 Updated `design.md` (new "Card heading links to /reports"
      decision) and the `dashboard-cards`/`web-client-home` spec deltas
      (new requirement + scenarios for the title and its click-through).
- [x] 11.8 Verified live (Playwright against the dev server + a seeded
      user): a `query_stat` card created with both a category filter and
      a custom title rendered that title as its heading; activating it
      navigated to `/reports?category_id=...` with the category
      preselected and subcategories included, report not yet generated —
      zero console errors. Re-ran `gofmt`/`go vet`/`go test ./...` and
      `pnpm lint`/`tsc`/`build` — all clean.

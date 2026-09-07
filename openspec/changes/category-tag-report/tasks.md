## 1. API contract

- [ ] 1.1 In `openapi/openapi.yaml`, add a `category_mode` query parameter
      (`enum: [exact, subtree]`, `default: subtree`) to `GET /api/entries`,
      and update `category_id`'s description to mention it.
- [ ] 1.2 Add `GET /api/entries/summary` to `openapi/openapi.yaml`: the
      same `account_id`, `category_id`/`category_mode`, `tag_id`,
      `from`/`to`, `q` parameters as `GET /api/entries` (no `sort`, `dir`,
      `after`, `limit`), a new `EntrySummary` schema
      (`{ sums: [{currency, amount}], count }`) as its `200` response, and
      `400`/`401` responses matching the sibling operation. Verify
      `openspec/openapi/README.md`'s lint command (or the `contract` CI
      job's) passes against the edited spec.
- [ ] 1.3 Run `cd backend && go generate ./...` to sync
      `backend/openapi.yaml` and verify it now matches
      `openapi/openapi.yaml` byte-for-byte.
- [ ] 1.4 Run `cd frontend && pnpm generate:api` and verify
      `frontend/src/api/schema.d.ts` picks up `category_mode` and the new
      `getEntriesSummary` operation with no manual edits needed.

## 2. Backend: category exact/subtree mode

- [ ] 2.1 In `backend/internal/entry/entry.go`, add a `CategoryMode` type
      (`ModeExact`, `ModeSubtree`) and a `CategoryMode` field on `Filter`.
- [ ] 2.2 In `backend/internal/entry/service.go`, factor `List`'s
      category-resolution block (currently always calling
      `categories.Subtree`) into a private helper that branches on
      `CategoryMode`: `exact` sets `CategoryIDs = []string{*f.CategoryID}`
      directly, `subtree` (or the zero value) keeps calling `Subtree` —
      preserving today's behavior when `CategoryMode` is unset. Verify
      with a unit test that a filter with `CategoryMode` unset behaves
      identically to before this change (existing tests in
      `service_test.go`/`service_scenarios_test.go` should still pass
      unmodified).
- [ ] 2.3 Add a test exercising `CategoryMode: ModeExact` against a
      category with children, asserting descendant-tagged entries are
      excluded — the inverse of the existing subtree-inclusion test.
- [ ] 2.4 In `backend/internal/entry/handler.go`, parse `category_mode`
      from the query string in `list`, rejecting an unrecognized value
      with `ErrInvalidValue` (`400`). Verify via a handler test asserting
      `?category_mode=bogus` returns `400`.

## 3. Backend: summary endpoint

- [ ] 3.1 In `backend/internal/entry/store.go`, add `Sum(ctx, ownerID
      string, f Filter) (sums map[string]int64, count int, err error)` to
      the `Store` interface.
- [ ] 3.2 Implement `Sum` in `backend/internal/storage/memory` by
      filtering the same way `List` does (forcing `Kind ==
      KindTransaction` regardless of `f.Kind`) and accumulating per
      account-currency totals in Go. Verify with a unit test covering:
      single currency, multiple currencies, balance adjustments excluded,
      and the empty-result `{sums: [], count: 0}` case.
- [ ] 3.3 Implement `Sum` in `backend/internal/storage/postgres` as a
      `SELECT accounts.currency, SUM(entries.amount), COUNT(*) ... GROUP
      BY accounts.currency` query, reusing the same filter-building
      helper `List`'s query construction uses (per design.md's shared
      WHERE/join decision) with `kind = 'transaction'` forced in
      unconditionally. Verify with an integration test (skipped without
      `DATABASE_URL`, per `backend/AGENTS.md`) covering the same cases as
      3.2 plus category exact/subtree resolution.
- [ ] 3.4 In `backend/internal/entry/service.go`, add `Service.Sum(ctx,
      ownerID string, f Filter) (map[string]int64, int, error)`, sharing
      the account-visibility and category/tag resolution `List` uses
      (same helper from 2.2), then calling `store.Sum`.
- [ ] 3.5 In `backend/internal/entry/handler.go`, add `GET
      /api/entries/summary`, parsing the same filter query parameters as
      `list` (reusing that parsing rather than duplicating it) and calling
      `Service.Sum`. Response body `{ sums: [{currency, amount}], count
      }`, with `sums` as `[]` (never `null`) when empty. Add a
      response-conformance assertion
      (`internal/openapicheck.AssertResponse`) per `backend/AGENTS.md`.
      Verify with a handler test covering the scenarios from the
      `account-entries` delta spec (single currency, multi-currency,
      balance adjustments excluded, no matches, exact-mode narrowing).
- [ ] 3.6 In `backend/internal/httpapi/server.go` (or wherever entry
      routes are registered), confirm the new route is reachable and
      run `gofmt -l . && go vet ./... && go test ./...` from `backend/`
      to verify the whole package still passes.

## 4. Frontend: reports route

- [ ] 4.1 Add `src/routes/reports.tsx` (file-based route → `/reports`),
      auth-gated like `/entries` (redirect an anonymous visitor to
      `/login`), with an "Reports" entry added to `Sidebar` navigating to
      it. Verify by running `pnpm dev`, confirming an anonymous visitor
      hitting `/reports` is redirected, and the sidebar link appears and
      is marked active on `/reports`.
- [ ] 4.2 Implement the filter controls: a category-or-tag exclusive
      selector (selecting one clears the other), the "include
      subcategories" checkbox (shown only for category, default checked),
      and account/date-range fields, all read from and written to typed
      URL search params via `validateSearch`, mirroring
      `entries.index.tsx`'s `EntriesSearch` pattern. Verify changing each
      control updates the URL and does not trigger any network request
      (inspect the Network tab in a manual `pnpm dev` check).
- [ ] 4.3 Add "Generate report" as an explicit action that snapshots the
      current filter state into a `generatedQuery` value (per design.md);
      only a change to `generatedQuery` triggers the `GET /api/entries`
      and `GET /api/entries/summary` fetches. Verify: loading `/reports`
      with filters already in the URL does not fetch until the button is
      clicked; changing a filter after a report is shown leaves the
      displayed results unchanged until clicked again.
- [ ] 4.4 Render the fetched entries using the same table/infinite-scroll
      markup style as `entries.index.tsx` (cursor-based `IntersectionObserver`
      loading), scoped read-only (no create/edit links needed). Verify
      scrolling past the initially loaded page fetches and appends further
      pages using `next_cursor`.
- [ ] 4.5 Render the per-currency sum banner from `GET
      /api/entries/summary`'s `sums`, and distinguish three states: no
      category/tag selected yet, a generated report with matches, and a
      generated report with zero matches. Verify each state's text is
      present in the DOM under the corresponding condition.
- [ ] 4.6 Add every new user-facing string to
      `frontend/src/i18n/locales/en.json` (and ideally `de.json`, though
      CI's `i18n-coverage` check is informational only per
      `frontend/AGENTS.md`), used via `useTranslation()`/`t()` — no
      literal strings.
- [ ] 4.7 Run `pnpm lint`, `pnpm exec tsc`, and `pnpm build` from
      `frontend/` and verify all three succeed.

## 5. End-to-end verification

- [ ] 5.1 With `docker compose up -d db` and a real `DATABASE_URL`, seed
      data via `server seed --yes <email>`, then manually exercise
      `/reports` in a browser: category with/without subcategories, a
      tag, account/date narrowing, and a multi-currency scenario (two
      accounts in different currencies sharing a category or tag) to
      confirm the sum splits per currency as designed.
- [ ] 5.2 Confirm `/entries`'s existing category filter behavior is
      unchanged (still always includes subtree, since it never sends
      `category_mode`) — re-run its existing test suite and manually spot
      check one filtered view.

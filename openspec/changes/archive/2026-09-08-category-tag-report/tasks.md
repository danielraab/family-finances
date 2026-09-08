## 1. API contract

- [x] 1.1 In `openapi/openapi.yaml`, add a `category_mode` query parameter
      (`enum: [exact, subtree]`, `default: subtree`) to `GET /api/entries`,
      and update `category_id`'s description to mention it.
- [x] 1.2 Add `GET /api/entries/summary` to `openapi/openapi.yaml`: the
      same `account_id`, `category_id`/`category_mode`, `tag_id`,
      `from`/`to`, `q` parameters as `GET /api/entries` (no `sort`, `dir`,
      `after`, `limit`), a new `EntrySummary` schema
      (`{ sums: [{currency, amount}], count }`) as its `200` response, and
      `400`/`401` responses matching the sibling operation. Verify
      `openspec/openapi/README.md`'s lint command (or the `contract` CI
      job's) passes against the edited spec.
- [x] 1.3 Run `cd backend && go generate ./...` to sync
      `backend/openapi.yaml` and verify it now matches
      `openapi/openapi.yaml` byte-for-byte.
- [x] 1.4 Run `cd frontend && pnpm generate:api` and verify
      `frontend/src/api/schema.d.ts` picks up `category_mode` and the new
      `getEntriesSummary` operation with no manual edits needed.

## 2. Backend: category exact/subtree mode

- [x] 2.1 In `backend/internal/entry/entry.go`, add a `CategoryMode` type
      (`ModeExact`, `ModeSubtree`) and a `CategoryMode` field on `Filter`.
- [x] 2.2 In `backend/internal/entry/service.go`, factor `List`'s
      category-resolution block (currently always calling
      `categories.Subtree`) into a private helper (`resolveFilter`,
      shared with `Sum`) that branches on `CategoryMode`: `exact` sets
      `CategoryIDs = []string{*f.CategoryID}` directly, `subtree` (or the
      zero value) keeps calling `Subtree` — preserving today's behavior
      when `CategoryMode` is unset. Verified: existing
      `service_test.go`/`service_scenarios_test.go` tests pass unmodified.
- [x] 2.3 Add a test exercising `CategoryMode: ModeExact` against a
      category with children, asserting descendant-tagged entries are
      excluded — the inverse of the existing subtree-inclusion test.
- [x] 2.4 In `backend/internal/entry/handler.go`, parse `category_mode`
      from the query string in `list`, rejecting an unrecognized value
      with `ErrInvalidValue` (`400`). Verify via a handler test asserting
      `?category_mode=bogus` returns `400`.

## 3. Backend: summary endpoint

- [x] 3.1 In `backend/internal/entry/store.go`, add `Sum(ctx, ownerID
      string, f Filter) (perAccount map[string]int64, count int, err
      error)` to the `Store` interface — totals keyed by account id, not
      currency (`Store` has no notion of currency; see design.md).
- [x] 3.2 Implement `Sum` in `backend/internal/storage/memory`, sharing
      `List`'s row-matching logic (a `matchingRows` helper) with `Kind`
      forced to `KindTransaction` regardless of `f.Kind`, accumulating
      per-account-id totals. Verify with a unit test covering: single
      account, multiple accounts, balance adjustments excluded, and the
      empty-result case.
- [x] 3.3 Implement `Sum` in `backend/internal/storage/postgres` as a
      `SELECT account_id, SUM(amount), COUNT(*) ... GROUP BY account_id`
      query, reusing the same `buildWhere` helper `List`'s query
      construction uses, with `kind = 'transaction'` forced in
      unconditionally. Verify with an integration test (skipped without
      `DATABASE_URL`, per `backend/AGENTS.md`) covering the same cases as
      3.2 plus category exact/subtree resolution.
- [x] 3.4 In `backend/internal/entry/service.go`, add `Service.Sum(ctx,
      ownerID string, f Filter) (Summary, error)`, sharing the
      account-visibility and category/tag resolution `List` uses (the
      `resolveFilter` helper from 2.2), calling `store.Sum`, then mapping
      each resulting account id to its currency via the existing
      `AccountLookup.Owner` and summing into `Summary.Sums`.
- [x] 3.5 In `backend/internal/entry/handler.go`, add `GET
      /api/entries/summary`, parsing the same filter query parameters as
      `list` (a shared `parseCommonFilter` helper) and calling
      `Service.Sum`. Response body `{ sums: [{currency, amount}], count
      }`, with `sums` as `[]` (never `null`) when empty. Add a
      response-conformance assertion
      (`internal/openapicheck.AssertResponse`) per `backend/AGENTS.md`.
      Verified with handler tests covering the scenarios from the
      `account-entries` delta spec (single currency, multi-currency,
      balance adjustments excluded, no matches, exact-mode narrowing).
- [x] 3.6 The route is registered directly on `entry.Handler`'s own mux
      (`GET /api/entries/summary`, mounted under `/api/entries` by
      `internal/httpapi` like every other entry route — no separate
      registration needed). Ran `gofmt -l . && go vet ./... && go test
      ./...` from `backend/` (including `internal/storage/postgres`'s
      integration suite against a real local PostgreSQL) — all pass.

## 4. Frontend: reports route

- [x] 4.1 Add `src/routes/reports.tsx` (file-based route → `/reports`),
      auth-gated like `/categories` (single self-contained route doing its
      own redirect-to-`/login` check), with a "Reports" entry added to
      `Sidebar` navigating to it. Verified in a real browser (Playwright
      against `pnpm dev` + the seeded backend): the sidebar link appears,
      is marked active on `/reports`, and a deep link with filters already
      in the URL restores the controls without redirecting.
- [x] 4.2 Implemented the filter controls: a category-or-tag exclusive
      selector (selecting one clears the other — verified: picking a tag
      cleared a previously-selected category), the "include
      subcategories" checkbox (shown only for category, default checked),
      and account/date-range fields, all read from and written to typed
      URL search params via `validateSearch`, mirroring
      `entries.index.tsx`'s `EntriesSearch` pattern. Verified changing
      controls updates the URL with zero `/api/entries*` requests fired.
- [x] 4.3 Added "Generate report" as an explicit action that snapshots the
      current filter state into a `generatedFilter` value (a fresh object
      each click, so it's always the correct effect dependency even for
      an unchanged selection); only that snapshot changing triggers the
      `GET /api/entries` and `GET /api/entries/summary` fetches. Verified:
      loading `/reports?category_id=...` fires zero entries-related
      requests until the button is clicked; changing a filter after a
      report is shown leaves the previously displayed results unchanged
      until clicked again.
- [x] 4.4 Rendered the fetched entries using the same
      table/infinite-scroll markup style as `entries.index.tsx`
      (cursor-based `IntersectionObserver` loading), read-only (plain text
      title, no create/edit links).
- [x] 4.5 Rendered the per-currency sum banner from `GET
      /api/entries/summary`'s `sums`, distinguishing three states —
      verified all three in the browser: no category/tag selected yet
      (prompt text, no table), a generated report with matches (sum
      banner + rows), and a generated report with zero matches (table
      header only + "no entries match" text, no sum banner).
- [x] 4.6 Added every new user-facing string to
      `frontend/src/i18n/locales/en.json` and `frontend/src/i18n/locales/de.json`,
      used via `useTranslation()`/`t()` — no literal strings.
- [x] 4.7 `pnpm lint`, `pnpm exec tsc`, and `pnpm build` all succeed from
      `frontend/` (after `pnpm generate-routes` to register the new route
      in `routeTree.gen.ts`).

## 5. End-to-end verification

- [x] 5.1 Ran the real stack locally (a local PostgreSQL instance,
      `server seed --yes demo@example.com`, `go run .`, `pnpm dev`) and
      exercised `/reports` in a real Chromium browser via Playwright:
      category with subcategories included (sum split EUR/GBP, a
      manually-added child-category entry included), the same category
      with "include subcategories" unchecked (the child entry excluded,
      sum dropped by exactly its amount), a tag (cleared the category
      selection), and the multi-currency split rendering as two separate
      totals rather than one combined figure — matches curl-level checks
      made directly against `GET /api/entries/summary`.
- [x] 5.2 `/entries`'s existing category filter behavior is unchanged: it
      never sends `category_mode`, so `Filter.CategoryMode`'s zero value
      keeps resolving to the full subtree exactly as before this change —
      confirmed both by the full `go test ./...` suite (including
      `storage/postgres`'s integration tests against a real database) and
      by design (see design.md's additive-parameter decision).

## 6. Follow-up: stale-results hint and home page card

- [x] 6.1 In `src/routes/reports.tsx`, add an `isStale` comparison between
      `generatedFilter` and the live `search` draft, and render a hint
      (distinct styling from the other status texts) whenever they
      diverge — disappearing again if the draft is changed back to match.
      Add the `reports.staleHint` key to `en.json`/`de.json`.
- [x] 6.2 Add a fifth card to `src/components/FeatureOverview.tsx`
      (`BarChart3` from `lucide-react`, already a dependency) introducing
      the reports feature, with `home.features.reports.title`/`.description`
      added to `en.json`/`de.json`.
- [x] 6.3 Ran `pnpm lint`, `pnpm exec tsc`, and `pnpm build` from
      `frontend/` — all pass.

## 1. API contract

- [x] 1.1 In `openapi/openapi.yaml`, add `category_id`, `category_mode`
      and `tag_id` parameters to `getRecurringTransactions` and
      `getRecurringTransactionsSummary`, worded as the preview
      operation's identical three already are, and note in both
      descriptions that a category or tag filter never widens the
      account scope.
- [x] 1.2 Regenerate both committed artifacts:
      `cd backend && go generate ./...` and
      `cd frontend && pnpm generate:api`.

## 2. Backend domain and service

- [x] 2.1 In `recurringtransaction.go`, add `CategoryID`,
      `CategoryMode`, `CategoryIDs` and `TagID` to `Filter`, documenting
      that a Store reads only `CategoryIDs`/`TagID` and that
      `CategoryIDs` is nil when unfiltered and non-nil-but-possibly-empty
      when filtered — see design.md.
- [x] 2.2 Keep `CategoryID`/`CategoryMode`/`TagID` on `PreviewFilter` as
      the preview endpoint's own input — `Preview` copies them into a
      `Filter`, so the resolution rule is shared without every
      `PreviewFilter{To: …}` call site growing a nested struct literal.
- [x] 2.3 In `service.go`, replace `resolveAccountIDs` with
      `resolveFilter`: the same account resolution plus `CategoryID` →
      `CategoryIDs` via `CategoryLookup.Subtree` (itself alone at
      `CategoryModeExact`), rejecting an invalid `CategoryMode` with
      `ErrInvalidValue`.
- [x] 2.4 Point `List` and `Summary` at `resolveFilter`.
- [x] 2.5 Rewrite `Preview` to build a `Filter` carrying the caller's
      category and tag and resolve it the same way, deleting its
      `allowedCategories` map, its `containsString` tag check and the
      now-unused helper. Keep its forced `SelfTransferBothLegs` and its
      ended-template exclusion.

## 3. Stores

- [x] 3.1 In `storage/postgres/recurringtransaction.go`, add a
      `category_id = ANY($n::uuid[])` clause when `CategoryIDs` is
      non-nil and a `recurring_transaction_tags` `EXISTS` clause when
      `TagID` is set, against `recurring_transaction_legs`.
- [x] 3.2 Do the same in-process in
      `storage/memory/recurringtransaction.go`.

## 4. Backend HTTP

- [x] 4.1 In `handler.go`, parse `category_id`, `category_mode` and
      `tag_id` in both `list` and `summary`, factoring the three out of
      `preview`'s own parsing so all three handlers read them the same
      way.

## 5. Backend tests

- [x] 5.1 Service tests: subtree vs `exact`, an uncategorized template
      never matching, a category the caller cannot see matching nothing
      (not everything), tag matching, category and tag combining, and an
      invalid `category_mode` returning `ErrInvalidValue`.
- [x] 5.2 A test that the summary under a category/tag filter totals
      exactly the rows the listing returns under the same filter.
- [x] 5.3 A test that a category filter does not expose a template on an
      account the caller has no permission on.
- [x] 5.4 Postgres store tests for the two new clauses, alongside the
      existing `List` coverage. (Written; they self-skip without
      `DATABASE_URL` and were not run locally — no database available in
      the dev container. CI's `backend-integration` job runs them.)

## 6. Shared filter panel

- [x] 6.1 Add `frontend/src/components/FilterPanel.tsx`: the heading,
      the active-count badge, the `sm`-and-below collapse toggle with
      `aria-expanded`/`aria-controls`, the clear-all button, and the
      controls grid, taking `activeCount`, `onClearAll` and children
      from the page.
- [x] 6.2 Render it from `frontend/src/routes/entries.index.tsx` in
      place of the inline markup, leaving `/entries`' behaviour
      unchanged.

## 7. Recurring list filters

- [x] 7.1 In `frontend/src/routes/recurring.index.tsx`, extend
      `RecurringSearch` and `validateSearch` with `account_id`,
      `category_id` and `tag_id`.
- [x] 7.2 Send all five filters on both the list and the summary
      request.
- [x] 7.3 Render the three selects inside `FilterPanel` beside the two
      self-transfer checkboxes, the category one flattened via
      `flattenCategoryTree` with its shared-owner suffix, the tag one
      likewise.
- [x] 7.4 Add the page's `activeFilterCount` and `clearAllFilters`,
      counting the self-transfer checkbox only when checked and leaving
      the revealed both-legs flag folded into it.

## 8. Translations

- [x] 8.1 Move `entries.filters.heading`, `activeCount_one`/`_other` and
      `clearAll` to a top-level `filters` namespace in `en.json`, and
      point `FilterPanel` at it.
- [x] 8.2 Add `recurring.filters.account`/`allAccounts`/`category`/
      `allCategories`/`tag`/`allTags` to `en.json`.
- [x] 8.3 Mirror both in `de.json` — both locales stay at 100% coverage.

## 9. Verification

- [x] 9.1 `cd backend && go build ./... && go test ./... && go vet ./...`
      and the repo's Go linter.
- [x] 9.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 9.3 Drive `/recurring` in Chromium against a stubbed `/api` at
      375px and 1280px, in both locales and both themes: each filter
      narrowing the rows and the totals, the subtree cascade, the count
      badge, clear-all, the filtered-empty message, and a reload
      restoring the filters from the URL. `/entries` re-checked for the
      panel extraction.

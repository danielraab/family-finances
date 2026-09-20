## 1. API contract

- [ ] 1.1 In `openapi/openapi.yaml`, add `category_id`, `category_mode`
      and `tag_id` parameters to `getRecurringTransactions` and
      `getRecurringTransactionsSummary`, worded as the preview
      operation's identical three already are, and note in both
      descriptions that a category or tag filter never widens the
      account scope.
- [ ] 1.2 Regenerate both committed artifacts:
      `cd backend && go generate ./...` and
      `cd frontend && pnpm generate:api`.

## 2. Backend domain and service

- [ ] 2.1 In `recurringtransaction.go`, add `CategoryID`,
      `CategoryMode`, `CategoryIDs` and `TagID` to `Filter`, documenting
      that a Store reads only `CategoryIDs`/`TagID` and that
      `CategoryIDs` is nil when unfiltered and non-nil-but-possibly-empty
      when filtered — see design.md.
- [ ] 2.2 Drop `CategoryID`, `CategoryMode` and `TagID` from
      `PreviewFilter`, which now carries only `AccountIDs` and `To`.
- [ ] 2.3 In `service.go`, replace `resolveAccountIDs` with
      `resolveFilter`: the same account resolution plus `CategoryID` →
      `CategoryIDs` via `CategoryLookup.Subtree` (itself alone at
      `CategoryModeExact`), rejecting an invalid `CategoryMode` with
      `ErrInvalidValue`.
- [ ] 2.4 Point `List` and `Summary` at `resolveFilter`.
- [ ] 2.5 Rewrite `Preview` to build a `Filter` carrying the caller's
      category and tag and resolve it the same way, deleting its
      `allowedCategories` map, its `containsString` tag check and the
      now-unused helper. Keep its forced `SelfTransferBothLegs` and its
      ended-template exclusion.

## 3. Stores

- [ ] 3.1 In `storage/postgres/recurringtransaction.go`, add a
      `category_id = ANY($n::uuid[])` clause when `CategoryIDs` is
      non-nil and a `recurring_transaction_tags` `EXISTS` clause when
      `TagID` is set, against `recurring_transaction_legs`.
- [ ] 3.2 Do the same in-process in
      `storage/memory/recurringtransaction.go`.

## 4. Backend HTTP

- [ ] 4.1 In `handler.go`, parse `category_id`, `category_mode` and
      `tag_id` in both `list` and `summary`, factoring the three out of
      `preview`'s own parsing so all three handlers read them the same
      way.

## 5. Backend tests

- [ ] 5.1 Service tests: subtree vs `exact`, an uncategorized template
      never matching, a category the caller cannot see matching nothing
      (not everything), tag matching, category and tag combining, and an
      invalid `category_mode` returning `ErrInvalidValue`.
- [ ] 5.2 A test that the summary under a category/tag filter totals
      exactly the rows the listing returns under the same filter.
- [ ] 5.3 A test that a category filter does not expose a template on an
      account the caller has no permission on.
- [ ] 5.4 Postgres store tests for the two new clauses, alongside the
      existing `List` coverage.

## 6. Shared filter panel

- [ ] 6.1 Add `frontend/src/components/FilterPanel.tsx`: the heading,
      the active-count badge, the `sm`-and-below collapse toggle with
      `aria-expanded`/`aria-controls`, the clear-all button, and the
      controls grid, taking `activeCount`, `onClearAll` and children
      from the page.
- [ ] 6.2 Render it from `frontend/src/routes/entries.index.tsx` in
      place of the inline markup, leaving `/entries`' behaviour
      unchanged.

## 7. Recurring list filters

- [ ] 7.1 In `frontend/src/routes/recurring.index.tsx`, extend
      `RecurringSearch` and `validateSearch` with `account_id`,
      `category_id` and `tag_id`.
- [ ] 7.2 Send all five filters on both the list and the summary
      request.
- [ ] 7.3 Render the three selects inside `FilterPanel` beside the two
      self-transfer checkboxes, the category one flattened via
      `flattenCategoryTree` with its shared-owner suffix, the tag one
      likewise.
- [ ] 7.4 Add the page's `activeFilterCount` and `clearAllFilters`,
      counting the self-transfer checkbox only when checked and leaving
      the revealed both-legs flag folded into it.

## 8. Translations

- [ ] 8.1 Move `entries.filters.heading`, `activeCount_one`/`_other` and
      `clearAll` to a top-level `filters` namespace in `en.json`, and
      point `FilterPanel` at it.
- [ ] 8.2 Add `recurring.filters.account`/`allAccounts`/`category`/
      `allCategories`/`tag`/`allTags` to `en.json`.
- [ ] 8.3 Mirror both in `de.json` — both locales stay at 100% coverage.

## 9. Verification

- [ ] 9.1 `cd backend && go build ./... && go test ./... && go vet ./...`
      and the repo's Go linter.
- [ ] 9.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 9.3 Drive `/recurring` in Chromium against a stubbed `/api` at
      375px and 1280px: each filter narrowing the rows and the totals,
      the count badge, clear-all, and a reload restoring the filters
      from the URL.

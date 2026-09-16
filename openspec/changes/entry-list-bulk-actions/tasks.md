## 1. Selection state and toolbar

- [ ] 1.1 `entries.index.tsx`: add `selectedIds: Set<string>` state; a
  header checkbox column that selects/deselects every id currently in
  `items`, and a per-row checkbox (always enabled) that toggles that one
  id. Selection persists across `loadMore()` appends; reset it whenever
  `searchKey` changes (a new filter effectively invalidates the previous
  loaded window) and whenever the bulk-run modal's "Reload list" fires.
- [ ] 1.2 Add a `BulkActionToolbar` component, rendered above the table
  only when `selectedIds.size > 0`: shows the count, a "Clear selection"
  button, and the six action triggers (Set category, Add tags, Remove
  tags, Set tags, Link to recurring transaction, Delete). "Link to
  recurring transaction" is disabled (with a hint) whenever the selected
  entries' `account_id`s aren't all equal.

## 2. Shared bulk-run modal

- [ ] 2.1 Add `frontend/src/lib/bulkRun.ts` (or similar): given a list of
  `{ id, title, run: () => Promise<{ok: boolean, reason?: string}> }`
  tasks, sequentially executes them, reporting progress and a
  `{succeeded, failed: {id, title, reason}[]}` result — the same
  sequential-loop/failure-collection shape `ImportRunStep.tsx` already
  implements, generalized to accept a caller-supplied per-task async
  function instead of being hardcoded to `POST /api/entries`.
- [ ] 2.2 Add `BulkActionRunModal` (or similar): opens with a task list,
  renders a progress bar while running (mirroring `ImportRunStep`'s), and
  on completion shows the success count plus a list of failed entries
  (title + reason), with **Reload list** (re-runs the ledger's existing
  `searchKey` fetch from page one, clears `items`/`nextCursor`/selection)
  and **Close** buttons.
- [ ] 2.3 Add a small per-entry failure-reason mapper (HTTP status/error
  body → a short translated string: forbidden, not found, invalid value,
  generic), reusable across every action's task list.

## 3. Set category action

- [ ] 3.1 Add a small `Dialog` (opened from the toolbar) with the same
  category `<select>` the entry edit form uses (`flattenCategoryTree`,
  filtered to not-disabled/append+ for the caller, plus a "No category"
  option) and an Apply button.
- [ ] 3.2 On Apply: build one task per selected entry —
  `PATCH /api/entries/{id} {category_id}`, skipped (counted as an
  immediate success) when the entry's current `category_id` already
  equals the chosen value — and open `BulkActionRunModal`.

## 4. Add tags / Remove tags / Set tags actions

- [ ] 4.1 Add tags: a `Dialog` with `TagInput` (existing-tags filter:
  not disabled, not view-tier — same as the entry edit form) and an Apply
  button. On Apply, resolve tag names to ids (`resolveTagIds`, same helper
  the entry forms/import already use) and build one task per selected
  entry: `PATCH {tag_ids: entry.tag_ids ∪ chosen}`, skipped when the
  entry's tags already cover every chosen tag.
- [ ] 4.2 Remove tags: a `Dialog` with `TagInput`, but its `existingTags`
  is computed as the set of tags present on at least one selected entry
  (from the already-loaded `items` + the ledger's own `tags` list), not
  the caller's full tag list. On Apply, build one task per selected entry:
  `PATCH {tag_ids: entry.tag_ids − chosen}`, skipped when the entry has
  none of the chosen tags.
- [ ] 4.3 Set tags: a `Dialog` with `TagInput` (same existing-tags filter
  as Add). On Apply, build one task per selected entry:
  `PATCH {tag_ids: chosen}` unconditionally (no skip — see design.md).

## 5. Link to recurring transaction action

- [ ] 5.1 When enabled (single shared `account_id` across the selection),
  fetch `GET /api/recurring-transactions?account_id=<that account>` on
  open, same scoped fetch `entries.$entryId.edit.tsx` already does.
- [ ] 5.2 A `Dialog` with a `<select>` of that account's recurring
  transactions plus a "Not linked" option (mirrors the entry edit form's
  picker) and an Apply button.
- [ ] 5.3 On Apply: build one task per selected entry:
  `PATCH {recurring_transaction_id: chosen || null}`, skipped when the
  entry's current value already matches.

## 6. Delete action

- [ ] 6.1 A destructive-confirm `Dialog` (mirrors the entry edit page's
  existing delete confirmation copy/shape), no value picker.
- [ ] 6.2 On confirm: build one task per selected entry:
  `DELETE /api/entries/{id}`, and open `BulkActionRunModal`.

## 7. i18n

- [ ] 7.1 Add `entries.bulk.*` keys to `frontend/src/i18n/locales/en.json`:
  selection count/clear, each action's label and dialog copy (category,
  add/remove/set tags, link recurring incl. the cross-account-disabled
  hint, delete confirm), and the run modal's progress/result/reload/close
  copy and failure-reason strings. Add German equivalents to `de.json`
  where practical (non-blocking in CI if it lags, per `frontend/AGENTS.md`).

## 8. Verification

- [ ] 8.1 Frontend: `pnpm lint`, `pnpm exec tsc`, `pnpm build` from
  `frontend/`.
- [ ] 8.2 Manually exercise the golden path in a browser: select a mix of
  loaded rows (including, if reachable, an entry the visitor can't edit),
  run each of the six actions, confirm the run modal's progress/result/
  failure list, confirm skip-if-unchanged avoids redundant writes on a
  re-run, confirm "Link to recurring transaction" is disabled for a
  multi-account selection and enabled/correctly scoped for a single-
  account one, confirm "Reload list" refreshes the ledger and clears
  selection, confirm header "select all" only touches loaded rows (scroll
  for more, then re-check select-all behavior).

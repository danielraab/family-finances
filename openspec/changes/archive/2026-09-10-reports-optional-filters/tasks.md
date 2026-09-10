## 1. Route logic (`frontend/src/routes/reports.tsx`)

- [x] 1.1 In `selectCategory`, stop clearing `tag_id`; keep resetting
  `include_subcategories` only when the category itself is being cleared.
- [x] 1.2 In `selectTag`, stop clearing `category_id` and
  `include_subcategories`.
- [x] 1.3 Remove the `canGenerate` binding and the early-return guard in
  `generateReport()` (still allow it to run with no `category_id`/`tag_id`).
- [x] 1.4 On the "Generate report" button, drop the `disabled` prop and the
  conditional `title`; remove the now-unused `disabled:*` classes if they
  serve no other state.
- [x] 1.5 Confirm the `include_subcategories` checkbox stays gated on
  `search.category_id` only (no tag condition) — verified, no code change.
- [x] 1.6 Update the pre-generation hint to render `reports.beforeGenerate`
  with its new wording (no other structural change).

## 2. i18n (`frontend/src/i18n/locales/`)

- [x] 2.1 `en.json`: reword `reports.beforeGenerate` to prompt for optional
  filters + "Generate report" without requiring a category or tag.
- [x] 2.2 `en.json`: remove the now-unused `reports.selectOneHint` key.
- [x] 2.3 `de.json`: apply the matching reword to `reports.beforeGenerate`
  and remove `reports.selectOneHint`.

## 3. Spec sync

- [ ] 3.1 Apply the `web-client-reports` delta to
  `openspec/specs/web-client-reports/spec.md` (remove the exclusivity
  requirement, update the three modified requirements) — done via
  `/opsx:archive` at completion, or manually if syncing earlier.

## 4. Verification

- [x] 4.1 `cd frontend && pnpm lint` passes.
- [x] 4.2 `cd frontend && pnpm exec tsc` passes.
- [x] 4.3 `cd frontend && pnpm build` writes `out/index.html`.
- [x] 4.4 Manual: on `/reports`, generate with no selection → all
  transaction entries + sum load; select a category *and* a tag → results
  are the intersection and the subcategories checkbox is visible; stale
  hint and revert-clears-hint still behave.
- [x] 4.5 `openspec validate reports-optional-filters --strict` passes.

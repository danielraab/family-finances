# Tasks

## 1. Consolidate the ledger metadata

- [x] 1.1 Add the `entries.columns.details` translation in English and German, and verify the ledger has no hardcoded Details header.
- [x] 1.2 In `frontend/src/routes/entries.index.tsx`, replace the Account, Category, and Tags headers and cells with one Details column immediately after Date; compose the existing account, category, and tag label paths in account → category → tags order, and verify shared and unavailable-entity treatment is unchanged.
- [ ] 1.3 Keep missing category and tags out of the Details stack and allow attached tag labels to wrap, then verify rows with only an account remain compact and rows with multiple tags keep every tag visible.

## 2. Verify the presentation

- [x] 2.1 From `frontend/`, run `pnpm lint`, `pnpm exec tsc --noEmit`, and `pnpm build`; verify all commands succeed.
- [ ] 2.2 Browser-check `/entries` at desktop and narrow widths with complete metadata, no optional metadata, shared/unavailable metadata, and many tags; verify the Date → Details → Title order, retained label semantics, tag wrapping, and usable horizontal scrolling.

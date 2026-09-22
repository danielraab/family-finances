## 1. Compact create action

- [x] 1.1 In `frontend/src/routes/entries.index.tsx`, render the
      `/entries/new` link's label inside a `hidden sm:inline` span beside
      a `lucide-react` `Plus` glyph, and give the link an `aria-label` of
      the same `entries.create` translation so its accessible name is
      unchanged below `sm`.
- [x] 1.2 Keep the Import action's label at every width — it is the
      secondary action and short enough not to crowd the header.

## 2. Clearable search input

- [x] 2.1 Wrap the search `input` in a `relative` container, add right
      padding for the button, and suppress the native cancel glyph with
      `[&::-webkit-search-cancel-button]:hidden`.
- [x] 2.2 Render a clear button (a `lucide-react` `X`, labelled via a new
      `entries.filters.clearSearch` key) inside the field only while the
      draft value is non-empty. On activation: clear the pending debounce
      timeout, set the draft to `""`, patch `q` to `undefined`, and
      refocus the input.

## 3. Filter panel, clear-all, and mobile collapse

- [x] 3.1 Add an `activeFilterCount(search)` helper counting
      `account_id`, `category_id`, `tag_id`, `kind`, `q`,
      `show_recurring`, and the date range as one — see design.md.
- [x] 3.2 Add a `clearAllFilters()` handler navigating to a search
      object carrying only `sort` and `dir`.
- [x] 3.3 Wrap the controls in a bordered panel with a header row: a
      `sm:hidden` toggle button (heading + active count badge +
      chevron, `aria-expanded`/`aria-controls`), a `hidden sm:block`
      static heading, and the "Clear all filters" button rendered only
      when the count is non-zero.
- [x] 3.4 Lay the controls out as `grid-cols-1 sm:grid-cols-2
      lg:grid-cols-3`, each control `w-full`, with the search field
      spanning every column. Hide the grid below `sm` while collapsed.
- [x] 3.5 Give `DateRangeFilter` a `fullWidth` prop (default `false`, so
      `/reports`' flex-wrap row is untouched) adding `w-full` to its
      trigger, and pass it from the ledger — a `<button>` sizes to its
      content even as a flex container, so without it the trigger was the
      one control not filling its grid cell.

## 4. Translations

- [x] 4.1 Add `entries.filters.heading`, `entries.filters.clearAll`,
      `entries.filters.clearSearch`, and the pluralised
      `entries.filters.activeCount` to `en.json`.
- [x] 4.2 Add the same keys to `de.json` — both locales stay at 100%
      coverage.

## 5. Verification

- [x] 5.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 5.2 Drive the built app in Chromium against a stubbed `/api` at
      375px and at 1280px: the plus-only button, the search clear
      button, the collapse toggle, the active-filter badge, and
      "Clear all filters" returning the URL to its sort-only state.

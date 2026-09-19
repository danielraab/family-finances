## 1. Row layout

- [x] 1.1 In `frontend/src/components/UpcomingBlock.tsx`, restructure the
      row into two lines: title + amount on the first, date/account meta
      + create action on the second — see design.md's "Two lines, not a
      narrower button".
- [x] 1.2 Give the meta line `min-w-0` and `overflow-hidden`, and make
      the date and its `·` separator their own non-shrinking spans so a
      narrow row can never orphan the separator onto a second line.
- [x] 1.3 Add `whitespace-nowrap` to the amount so a negative amount's
      leading `-` cannot wrap, matching the fix `/recurring` already
      carries.

## 2. Compact create action

- [x] 2.1 Make the row's `/entries/new` link an `inline-flex
      items-center whitespace-nowrap` control rendering a `lucide-react`
      `Plus` glyph with its `recurring.createTransaction` label in a
      `hidden sm:inline` span, and `aria-label` + `title` carrying that
      label at every width — the same control `/recurring`'s rows use.

## 3. Overdue marker

- [x] 3.1 Move the overdue marker out of the title's `truncate` span into
      a `shrink-0` amber pill beside it, so it is never ellipsised.

## 4. The title opens the recurring summary

- [x] 4.1 Add a required `onOpenRecurring: (recurringTransactionId:
      string) => void` prop to `UpcomingBlock` and render the title as a
      `<button>` calling it, with the ledger's own
      `truncate text-left font-medium underline-offset-2 hover:underline`
      treatment.
- [x] 4.2 Pass `openRecurring` from
      `frontend/src/components/dashboard/EntryListCard.tsx` (destructure
      it from the `useSummaryModals` call the card already makes).
- [x] 4.3 Pass `openRecurring` from `frontend/src/routes/entries.index.tsx`
      and `frontend/src/routes/reports.tsx`, which already hold it.

## 5. Embedded rendering inside a dashboard card

- [x] 5.1 Add an `embedded?: boolean` prop to `UpcomingBlock`: when set,
      render without the block's own border, rounding and padding, and
      with a bottom rule under it.
- [x] 5.2 Render the block with `embedded` from `EntryListCard`; leave
      `/entries` and `/reports` on the bordered box.

## 6. Translations

- [x] 6.1 Confirm no new i18n keys are needed (the action and the overdue
      marker reuse `recurring.createTransaction` and
      `recurringPreview.overdue`), and that `en.json`/`de.json` stay at
      100% coverage.

## 7. Verification

- [x] 7.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 7.2 Drive the app in Chromium against a stubbed `/api` at 390px and
      1280px, in both themes: the two-line rows with nothing overlapping,
      the plus-only action below `sm` and the labelled one above it, the
      overdue pill legible beside a long title, no orphan separator, and
      the title opening the recurring summary modal.

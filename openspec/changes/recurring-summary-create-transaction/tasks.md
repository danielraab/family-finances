## 1. A bordered secondary action style

- [x] 1.1 Add `summaryOutlineLinkClass` to
      `frontend/src/components/summary/SummaryField.tsx`, mirroring
      `/recurring`'s own row-level Create transaction control — see
      design.md's "Why a bordered action rather than a second filled one".

## 2. The action

- [x] 2.1 Add a `bookingTimestamp?: string` prop to
      `RecurringSummaryModal`, documented as the occurrence the summary
      was opened for.
- [x] 2.2 Render a "Create transaction" `Link` to `/entries/new` in
      `SummaryActions`, before Edit, carrying
      `recurring_transaction_id` always and `booking_timestamp` only when
      the prop is set, labelled with the existing
      `recurring.createTransaction` key and a `Plus` glyph.

## 3. Carrying the occurrence date

- [x] 3.1 Widen `useSummaryModals`' `openRecurring` to
      `(id: string, bookingTimestamp?: string)`, carry it on the open
      target, and pass it to the modal. The entry-summary cross-link
      passes `undefined` — that path names no occurrence.
- [x] 3.2 Make `UpcomingBlock`'s title pass its row's own
      `booking_timestamp` to `onOpenRecurring`.

## 4. Translations

- [x] 4.1 Confirm no new i18n keys are needed and both locales stay at
      100% coverage.

## 5. Verification

- [x] 5.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 5.2 Drive the app in Chromium against a stubbed `/api`: open the
      summary from an Upcoming row other than the first and confirm the
      action's href carries that row's date; open it from a `/recurring`
      row and from a linked entry's badge and confirm no
      `booking_timestamp` is sent; check the action row at 390px and
      1280px in both themes.

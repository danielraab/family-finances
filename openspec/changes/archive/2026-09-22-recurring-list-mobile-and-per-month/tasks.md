## 1. Compact create action

- [x] 1.1 In `frontend/src/routes/recurring.index.tsx`, render the
      `/recurring/new` link's label inside a `hidden sm:inline` span
      beside a `lucide-react` `Plus` glyph, and give the link an
      `aria-label` and `title` of the same `recurring.create`
      translation so its accessible name is unchanged below `sm`.

## 2. Fix the per-row create-transaction action

- [x] 2.1 Give the row's `/entries/new` link `inline-flex items-center`
      and `whitespace-nowrap` so it is one unbreakable box instead of an
      inline box that fragments across lines — see design.md.
- [x] 2.2 Render its `recurring.createTransaction` label inside a
      `hidden sm:inline` span beside a `Plus` glyph, with `aria-label`
      and `title` carrying the label at every width.

## 3. Amounts stay on one line

- [x] 3.1 Add `whitespace-nowrap` to the list's amount cells and to the
      totals block's figures, so the leading `-` of a negative amount no
      longer wraps to its own line — see design.md.

## 4. Per-month column and total

- [x] 4.1 Add a `perMonthAmount(perYearAmount)` helper to
      `frontend/src/lib/recurrence.ts` dividing by twelve and rounding
      half away from zero, documented against the backend's
      `PerYearAmount`.
- [x] 4.2 Add a `recurring.columns.perMonthAmount` header cell after the
      per-year one, and a matching body cell per row formatted with
      `formatAmount` and coloured with `amountColorClass`.
- [x] 4.3 Rework the totals block into two labelled groups — per year and
      per month — each listing its per-currency figures, stacking below
      `sm` and side by side from `sm` up.

## 5. Translations

- [x] 5.1 Add `recurring.columns.perMonthAmount` and
      `recurring.totalPerMonth` to `en.json`.
- [x] 5.2 Add the same keys to `de.json` — both locales stay at 100%
      coverage.

## 6. Verification

- [x] 6.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 6.2 Drive the app in Chromium against a stubbed `/api` at 375px and
      1280px, in both locales and both themes: the plus-only header
      button, the row action rendering as one unbroken box, the per-month
      column matching per-year ÷ 12, and both totals showing per
      currency.

## Why

`/recurring` has the same desktop-first problems the entry ledger just
shed, plus a missing figure:

- **The create action crowds the title.** "New recurring transaction" is
  the longest create label in the app. At 375px it wraps to two lines and
  takes nearly half the header, leaving the page title to wrap under it.
  `/entries` solved this a commit ago with a plus-glyph-only button below
  `sm`; `/recurring` never got it.
- **The row's "Create transaction" button renders broken on a phone.**
  The action is a `<Link>` with border and padding and no display set, so
  it is an *inline* box. Its label is two words, and at phone width the
  cell is narrow enough that the label wraps — an inline box that breaks
  across lines draws its border around each fragment and its horizontal
  padding only at the two ends, so the button renders as two overlapping,
  half-bordered pieces (confirmed in Chromium at 375px). It is not a
  rounding or truncation artifact; the element simply has the wrong
  display type for a control.
- **Negative amounts break across two lines.** Every amount cell is a
  formatted currency string, and a leading `-` is a line-break
  opportunity. The table's auto layout sizes the amount columns to their
  widest *unbreakable* run, which excludes that sign — so in the row with
  the longest amount, the sign lands on its own line above the digits.
  Visible at 375px and at 1280px alike, on every negative row.
- **There is no per-month figure.** Every row carries an entered amount
  and an annualized `per_year_amount`, and the page totals the latter per
  currency. Household budgeting is done per month, so today the reader
  divides by twelve in their head, once per row and again for the total.

## What Changes

- The "New recurring transaction" action renders as a plus glyph alone
  below the `sm` breakpoint and as the existing labelled button from `sm`
  up, with the same translated label as its accessible name at both
  sizes — the same treatment `/entries` uses.
- Each row's "Create transaction" action becomes an `inline-flex` control
  that never wraps, and below `sm` renders as a plus glyph alone with its
  translated label as its accessible name. This both fixes the broken
  rendering and gives back the width the new column needs.
- Every amount cell (entered, per year, per month, and the totals) stops
  wrapping, so a currency string and its sign always render on one line.
- The list gains a **Per month** column beside the existing Per year
  column, showing each row's `per_year_amount` divided by twelve, in the
  row's own currency and with the same sign colouring the other amount
  columns use.
- The per-currency total block gains a **Total per month** figure beside
  the existing Total per year, derived the same way from the summary
  endpoint's per-currency totals, and keeps the per-currency grouping —
  never a single combined figure across currencies.

## Non-goals

- **No API contract change.** Per month is an exact division of a figure
  the backend already computes; adding `per_month_amount` to the schema
  would mean a spec change, a backend change, and a regenerated contract
  for a division by twelve. See design.md.
- **No change to how per-year is computed.** The client divides what the
  backend sends; the annualization rules (calendar-exact for
  month/year, 365.25-day average for week/day) are untouched.
- **No card/stacked layout for the table on a phone.** The table keeps
  its existing `overflow-x-auto` horizontal scroll. Reshaping it into
  per-row cards below `sm` is a larger change and a separate decision.
- No change to the recurring summary modal, to `/recurring/new`, or to
  `/recurring/{id}/edit`.

## Capabilities

### Modified Capabilities

- `web-client-recurring-transactions`: the list's create action is
  icon-only on a phone, the per-row create-transaction action is a
  non-wrapping control that is icon-only on a phone, and both the rows
  and the totals carry a per-month figure beside the per-year one.

## Impact

- `frontend/src/routes/recurring.index.tsx` — the header action, the new
  column, the row action, and the totals block.
- `frontend/src/lib/recurrence.ts` — a `perMonthAmount` helper.
- `frontend/src/i18n/locales/{en,de}.json` — keys for the per-month
  column header and the per-month total.

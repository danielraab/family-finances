# Design

## Where the per-month figure is computed

Client-side, from `per_year_amount ÷ 12`. No new API field.

The alternative was a `per_month_amount` on `RecurringTransaction` and a
second sum on `GET /api/recurring-transactions/summary`. It was rejected
because the figure carries no information the client doesn't already
hold: `per_year_amount` is itself computed server-side and never stored,
and per month is that same number divided by a constant. The cost would
be a change to `openapi/openapi.yaml`, both generated artifacts, the Go
service, and its tests — for a division by twelve.

**The rounding this introduces is invisible.** Amounts travel as integers
at a fixed 4-decimal-place scale, so dividing by twelve and rounding to
the nearest integer is accurate to 0.0001 of a currency unit, two orders
of magnitude below the 2-decimal display default. The helper rounds half
away from zero so a negative (expense) row rounds symmetrically to a
positive one, matching the backend's own `PerYearAmount`.

## Which number the per-month total divides

The per-currency **total per month is that currency's total per year
divided by twelve**, not the sum of the rows' rounded per-month figures.

Two reasons. The summary endpoint covers the *non-ended* recurring
transactions while the table lists ended ones too (dimmed), so summing
the visible rows would either contradict the per-year total beside it or
force the client to re-derive the backend's "non-ended" rule. And a
reader checking the figure with a calculator will divide the per-year
total they can see, so that is the number that should come out.

The two approaches differ by at most half a unit at the 4-decimal scale
per row — under a hundredth of a cent across any realistic list, and
never enough to move a displayed digit.

## Why the row action renders broken today

`<Link className="rounded-md border px-2.5 py-1.5 …">` produces an `<a>`
with no `display` set, so it is an inline box. CSS renders an inline box
that breaks across lines as one fragment per line: the border is drawn
around each fragment and the left/right padding applied only at the very
start and the very end. With a two-word label in a narrow cell that is
exactly what happens — two boxes, each missing a side, overlapping the
row's text.

`inline-flex` makes it a single, unbreakable box; `whitespace-nowrap`
keeps its label on one line rather than growing the row's height. The
icon-only treatment below `sm` is what actually keeps the column narrow.

## Why negative amounts wrap

`Intl.NumberFormat`'s output for a negative amount starts with U+002D,
and a hyphen-minus is a line-break opportunity. The table uses the
browser's automatic layout, which sizes each column to the widest run of
text that *cannot* be broken — for an amount column that run is the
digits alone, with the sign left out of the measurement. So the column
comes out one character too narrow and every negative amount in it puts
its sign on a line of its own. It is not a phone-only problem: the same
row wraps at 1280px.

`whitespace-nowrap` on the amount cells removes the break opportunity, so
the column is measured with the sign included. The cost is a few pixels
of column width, which the horizontal scroll already absorbs.

## Table width on a phone

The table stays horizontally scrollable — a sixth data column does not
change that, and it was already scrolling at 375px. What the icon-only
row action gives back (roughly the width of "Create transaction") is
close to what the per-month column costs, so the scroll distance is
about what it is today rather than meaningfully worse.

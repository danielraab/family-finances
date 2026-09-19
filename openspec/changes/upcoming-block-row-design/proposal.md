## Why

The Upcoming block (`UpcomingBlock.tsx`, shared by the dashboard's
`entry_list` card, `/entries` and `/reports`) lays a row out as one flex
line: a shrinkable title/meta column on the left, and a `shrink-0` group
holding the amount and a full-text "Create transaction" button on the
right. The button is the widest thing in the row and it never gives way,
so in the narrowest container that uses the block — a dashboard card,
which nests the block's own bordered box inside the card's — the left
column is starved and the row falls apart. Measured in Chromium at 390px,
in a card whose row box is 274px wide:

- **The left column collapses to 43px.** The amount (84px) and the button
  (123px) plus gaps claim 231px of the 274, and the right group cannot
  shrink. The title truncates to "Miet…" while two thirds of the row is
  taken by a button.
- **The meta line overflows and collides with the amount.** The
  date/account line is a flex container with the default
  `min-width: auto`, so it refuses to shrink below its content and spills
  out of the 43px column instead of truncating — its account name is drawn
  from x=145 to x=253, straight through the amount that starts at x=113.
  Two pieces of text overlap on screen.
- **The separator drops onto a line of its own.** `9/16/2026 · ` is a
  single anonymous flex item; in a 43px box it wraps, leaving an orphan
  `·` under the date and making the row 65px tall.
- **The "Overdue" marker is clipped.** It lives inside the title's
  `truncate` span, so the ellipsis eats it: at 390px it disappears, at
  desktop it reads "O…". The one row that most needs attention is the one
  whose state cannot be read.
- **The block draws a bordered, padded box inside the card's own bordered,
  padded box.** Beyond reading as a box in a box, its `p-4` costs 32px of
  row width on the surface that has the least to spare.

Two of these are the same ask Daniel made for `/recurring` a change ago
(`recurring-list-mobile-and-per-month`): a row action whose full-text
label does not fit belongs behind a glyph on a phone.

Separately, the block's title is inert. Every other list of recurring
transactions in the app — `/recurring`'s rows, a linked entry's badge in
the ledger and in `/reports` — opens the read-only recurring summary
modal. The Upcoming block is the one place showing a recurring
transaction by name where the name does nothing.

## What Changes

- **Each Upcoming row becomes two lines**, so no element has to starve
  another: the title and the amount share the first line (the same shape
  the real entry rows beside them already use), the date/account meta and
  the create action share the second. Every shrinkable part carries
  `min-w-0` so it truncates instead of overflowing, and the date and its
  separator no longer wrap.
- **The create action renders as a plus glyph alone below `sm`** and as
  the existing labelled button from `sm` up, with the same translated
  label as its accessible name at both sizes — the treatment `/recurring`
  and `/entries` already use.
- **The overdue marker becomes a pill beside the title** rather than text
  inside the truncating title, so it is always legible; the row keeps its
  existing tint and its chronological position.
- **An Upcoming row's title opens the recurring transaction's read-only
  summary modal**, matching `/recurring`'s rows and the badge on a linked
  entry. The dashboard card wires in the `openRecurring` half of
  `useSummaryModals` it already renders for entries.
- **Inside a dashboard card the block drops its own border and padding**,
  keeping its heading and gaining a rule below it to separate it from the
  card's real entries. `/entries` and `/reports`, where the block stands
  on its own, keep the bordered box.

## Non-goals

- **No API contract change.** Nothing here needs a field the preview
  endpoint does not already return.
- **No change to what the block fetches, how it is bounded, or how it is
  sorted.** Cutoff resolution, the date-ascending order and the
  overdue-stays-in-place rule are untouched.
- **No change to the recurring summary modal itself**, only to what opens
  it. (Followed up by `recurring-summary-create-transaction`, which gives
  the summary its own Create transaction action.)
- **No change to the real entry rows** in the card, the ledger or the
  report table.

## Capabilities

### Modified Capabilities

- `web-client-entries`: the Upcoming block's rows lay out on two lines,
  the create action is icon-only on a phone, the overdue marker is a pill
  that never truncates, and a row's title opens the recurring summary.
- `web-client-home`: an `entry_list` card's Upcoming block renders
  without its own nested box, and its rows get the same row treatment.
- `web-client-reports`: the report's Upcoming block gets the same row
  treatment.

## Impact

- `frontend/src/components/UpcomingBlock.tsx` — the row layout, the
  action, the overdue pill, the title button, the embedded variant.
- `frontend/src/components/dashboard/EntryListCard.tsx` — renders the
  block embedded and passes `openRecurring`.
- `frontend/src/routes/entries.index.tsx`,
  `frontend/src/routes/reports.tsx` — pass `openRecurring`.
- `frontend/src/i18n/locales/{en,de}.json` — no new keys expected; the
  action and the overdue marker reuse the ones they already have.

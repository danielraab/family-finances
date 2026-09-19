# Design

## Why the row breaks, precisely

The row is

```
<li class="flex items-center justify-between gap-2 …">
  <div class="flex min-w-0 flex-col">       ← title + meta
  <div class="flex shrink-0 items-center gap-2">  ← amount + action
```

`shrink-0` on the right group is the whole problem. Flexbox distributes
free space, then shrinks items when there is none; a `shrink-0` item is
exempt, so *all* of the deficit lands on the left column. In a 274px row
the right group keeps its full 215px (an 84px amount, a 123px button and
their gaps) and the left column is handed the 43px that remain —
regardless of how long the title is.

The left column's own children then behave differently from each other.
The title span has `truncate`, which sets `overflow: hidden`, so it obeys
its 43px box and shows "Miet…". The meta span is `flex items-center`
with no `min-w-0`; a flex container's default `min-width: auto` floors it
at its content's size, and nothing clips it, so it renders at its natural
width and simply paints outside its parent — through the amount, which is
drawn at the same vertical center. The measurements above (left column
43px wide, account name spanning x=145–253, amount starting at x=113)
were read off the live DOM in Chromium at 390px.

The orphan `·` is the same cause one level down. `9/16/2026 · ` is a text
run, so flexbox wraps it in one anonymous flex item; that item is a block
box 43px wide, its text wraps inside it, and the `·` ends up on a second
line. It is not a stray element — it is the date's own line break.

## Two lines, not a narrower button

Making the button icon-only below `sm` is necessary but not sufficient:
it fixes the phone and leaves the tight desktop cases (a dashboard card
is at most ~460px wide at `xl`, and the block also renders in a
two-column card at `sm`) still fighting over one line, since the labelled
button returns at `sm` and up.

The two-line row removes the competition instead of rationing it. Line
one carries the title against the amount — exactly the shape of the real
entry rows directly beneath it in the same card, so the two lists read as
one thing. Line two carries the date/account meta against the action. The
widest fixed element on each line is now small (an 84px amount above, a
26px–123px action below), so the shrinkable half of each line gets the
rest. Worked through at the sizes that actually occur:

| container | meta line budget | meta needs | fits |
| --- | --- | --- | --- |
| dashboard card at 390px (icon action) | 306 − 8 − 26 = 272 | ~208 | yes |
| dashboard card at 1280px (labelled) | ~396 − 8 − 123 = 265 | ~208 | yes |
| `/entries` at 390px (icon action) | ~324 | ~208 | yes |

"~208" is the meta line's natural width: a 70px date, the separator, and
`AccountLabel`, whose name group carries a hard `min-w-32` (128px) floor
so its shared-owner badge wraps onto its own line rather than competing
with the name. That floor is why the meta line cannot simply be told to
truncate — below 128px it overflows whatever box it is in. The two-line
layout keeps it above that floor everywhere the block renders, and the
meta line additionally gets `min-w-0` + `overflow-hidden` so that a
container narrower than anything shipped today clips rather than paints
over its neighbour.

## Why the overdue marker moves out of the title

It is currently a `<span>` inside the title's `truncate` span. `truncate`
is `overflow: hidden; text-overflow: ellipsis; white-space: nowrap` on
that box, and the marker is part of the text flow inside it, so it is
clipped and ellipsised along with the title. The result is backwards:
the longer the title, the less of the word "Overdue" survives, and a long
title hides the marker completely.

As a sibling pill with `shrink-0`, it is laid out before the title gets
its remaining width — the title truncates, the marker never does. The
pill shape (`rounded-full`, `text-xs`, tinted background) is the one
`/recurring` already uses for its "ended" marker, in amber rather than
neutral. The row's existing amber tint stays; the pill is what makes the
state readable when a row is scanned rather than read.

## Why the title opens the summary rather than linking to /recurring

`useSummaryModals` is already mounted in all three surfaces that render
the block, and all three already pass `openEntry`; two of them already
pass `openRecurring` for the ledger/report badges. Opening the modal in
place costs one more prop and no navigation, and it matches what the same
title does everywhere else in the app. Navigating to
`/recurring/{id}/edit` instead would take a reader who wanted to know
what a projected row *is* into an edit form, and the summary already
offers Edit for when that is what they wanted.

The title becomes a `<button>` with the ledger's own
`underline-offset-2 hover:underline` treatment, not a link: it opens a
modal, and a `<button>` is what the real entry rows in the same card
already use for the same gesture.

`onOpenRecurring` is a required prop, not optional. Making it optional
would let a fourth surface adopt the block with a dead title, which is
the exact inconsistency this change exists to remove.

## Why the embedded variant is a prop and not a second component

Inside a dashboard card the block's `rounded-lg border p-4` sits inside
the card's identical `rounded-lg border p-4`. Dropping the border and
padding there is two conditional class strings; the heading and the rows
are otherwise byte-for-byte the same, and a second component (or a copy
of the rows in `EntryListCard`) would mean every future row change has to
be made twice. A single `embedded` boolean keeps one row implementation,
which is what makes "the same row everywhere" true rather than aspirational.

Embedded mode adds a bottom rule under the block. Without the box, the
Upcoming rows and the card's real entry rows are the same shape, so
something has to say where the projection ends and the record begins; a
1px rule does it with none of the nesting.

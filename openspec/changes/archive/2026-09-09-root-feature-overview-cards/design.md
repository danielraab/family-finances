## Context

See proposal.md - Why. Relevant existing code this reuses or must stay
consistent with:

- `frontend/src/routes/index.tsx` renders `<Placeholder />` unconditionally
  — no `useAuth`, no branching. That property is preserved; only the
  rendered component changes.
- `frontend/src/components/Placeholder.tsx` is the component being
  replaced; it has no other importers.
- No icon library exists today — every icon is a hand-rolled inline `<svg>`
  (`Sidebar.tsx`'s nav glyphs, `AccountCard.tsx`'s `PlusGlyph`,
  `components/Icon.tsx`'s brand mark). The user has explicitly approved
  installing `lucide-react` for this change rather than hand-drawing four
  more SVGs.
- `frontend/src/lib/amount.ts`'s `amountColorClass` already uses
  `emerald-600`/`emerald-400` as this app's positive/brand-adjacent green,
  close to `components/Icon.tsx`'s `#15803d` tile. Reusing `emerald` for the
  new icon panels keeps the page visually consistent with the rest of the
  app instead of introducing a new accent color.
- Card border/spacing convention from `AccountCard.tsx`:
  `rounded-lg border border-black/10 dark:border-white/10`.

## Goals / Non-Goals

**Goals:**

- Give `/` real, on-brand content that explains the app's four core
  capabilities, identically for anonymous and authenticated visitors.
- Introduce `lucide-react` as a sanctioned dependency for this one use case
  without disturbing the app's existing hand-rolled icon components.

**Non-Goals:**

- No card is a link or has a click handler — purely descriptive (explicit
  decision).
- No change to `/home`'s authenticated dashboard, its `AccountCard`, or any
  auth gating anywhere else.
- No migration of the sidebar/`PlusGlyph`/brand-mark icons to `lucide-react`
  — those stay hand-rolled SVG; this change only touches the four new
  feature icons.
- No data fetching — the page remains fully static markup.

## Decisions

**One new component, `FeatureOverview` (`frontend/src/components/
FeatureOverview.tsx`), replaces `Placeholder` entirely; `Placeholder.tsx`
is deleted.** `index.tsx` becomes a one-line swap. `Placeholder` has no
other callers, so nothing else needs updating.

**The four cards are driven by a single static config array of `{ Icon,
titleKey, descriptionKey }`, mapped over — not four hand-written JSX
blocks.** All four cards share identical structure (panel + text, alternating
side); the only per-card differences are the icon and the two translation
keys, so a `.map()` avoids duplicating the layout markup four times. The
side alternation is `index % 2 === 1` driving a `flex-row-reverse` class.

**Icons: `lucide-react`'s `Wallet`, `ArrowLeftRight`, `Tags`, and
`ListTree`,** matching accounts, transaction/balance-adjustment entries,
categories-and-tags, and the category tree respectively (as agreed in
discovery). Rendered via lucide's `size`/`className` props, sized generously
(e.g. `40`–`48`) and centered inside the panel — not stretched to fill the
panel's rectangular bounds, since distorting a lucide glyph's aspect ratio
would look broken. "Fills the card in height" describes the panel
(`self-stretch` / matching the card's full height), not the icon itself.

**Panel styling: a tinted rounded block, not a plain/transparent
background.** `bg-emerald-600/10 dark:bg-emerald-400/10` for the panel,
`text-emerald-700 dark:text-emerald-400` for the icon stroke — reusing the
app's existing emerald accent (`amountColorClass`) rather than a new color,
per the "colored panel/tile" decision from discovery.

**Layout: each card is `flex flex-col sm:flex-row` (stacked on mobile, side
-by-side at `sm:` and up), alternating `sm:flex-row-reverse` on odd
indices.** The panel is `shrink-0` at a fixed fraction of the row's width on
`sm:`+ (e.g. `sm:w-2/5`) and `self-stretch` so it always spans the full
height of its row; on mobile, where the layout stacks vertically, the panel
gets its own explicit height (e.g. `h-40`) since there's no row height to
stretch to. The text side is vertically centered (`justify-center`) with a
title (`font-medium`) and a description (`text-sm text-zinc-500 dark:text-
zinc-400`, matching `AccountCard`'s institute-line styling). Cards stack in
a `flex flex-col gap-4` list, not a grid — a 2×2 grid was considered and
rejected in discovery in favor of a full-width zigzag list, which reads
better with a large icon panel per card.

**Page keeps a heading and short intro above the cards, reusing the
"Family Finances" title convention** (`AGENTS.md`'s one exception to i18n:
the brand name is a literal, not translated) plus a translated tagline
paragraph — the same two-part header `Placeholder` had, just above the new
card list instead of the old empty-state box.

**i18n: new keys nested under `home.features.<accounts|entries|
categorization|categoryTree>.title` / `.description`**, plus a reused or
lightly reworded `home.tagline`. The old `home.emptyTitle` /
`home.emptyBody` keys are deleted (only `Placeholder.tsx` read them).
German (`de.json`) may lag per `frontend/AGENTS.md`'s i18n-coverage policy.

**No auth branching on `/` is added.** `index.tsx` keeps rendering its
component unconditionally; the feature overview must look identical
whether `useAuth()` would report `anonymous` or `authenticated` (it isn't
even called).

## Risks / Trade-offs

- **`lucide-react` becomes the app's second icon convention.** Accepted:
  it's scoped to these four illustrative feature icons, tree-shaken to just
  the icons imported, and doesn't touch the existing functional glyphs
  elsewhere. The two systems serve different purposes (small functional
  glyphs vs. larger illustrative ones), so visual identity between them was
  never a goal.
- **Zigzag alternation is a small amount of conditional layout logic
  (`index % 2`) inside a shared card renderer.** Minor; contained entirely
  within `FeatureOverview.tsx`.

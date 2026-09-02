## Context

The app shell lives in `frontend/src/routes/__root.tsx` (a flex row of
`<Sidebar />` + `<main>`). `Sidebar.tsx` owns the collapsed state
(`useState` seeded from `localStorage` key `ff:sidebar-collapsed`) and its
footer renders three rows: `<SidebarUser />`, `<ThemeToggle />`, and an inline
collapse `<button>`.

We want the collapse and theme controls promoted into a new top bar, leaving the
sidebar footer with just the account control. The top bar must be able to read
and toggle the collapsed state that `Sidebar` currently owns privately.

## Goals / Non-Goals

**Goals:**

- A thin top bar on every route, above `<Outlet>`, with the collapse control at
  its left edge (flush against the sidebar) and the theme control at its right
  edge.
- Sidebar footer renders `SidebarUser` alone.
- Preserve all existing behaviour: `ff:sidebar-collapsed` persistence and
  first-load-with-no-value correctness, the `ff:theme` three-state cycle, and
  the no-flash pre-paint script.
- No visual regression to the collapsed sidebar (icon + nav glyphs + user
  glyph/avatar).

**Non-Goals:**

- Redesigning the top bar into a full page header (breadcrumbs, page title,
  search). It is chrome-only for now.
- Making the top bar sticky/scroll-aware, or adding a mobile drawer.
- Changing `ThemeToggle`'s internal state machine or `theme.tsx`.
- Touching the backend or the embed/build pipeline.

## Decisions

### Lift collapsed state to the shared shell (`__root.tsx`)

`RootComponent` becomes the owner of the collapsed boolean: it holds the
`useState` seeded from `localStorage`, the `toggle` callback that persists, and
the `mounted` flag that gates the width transition. It passes `collapsed` to
`<Sidebar />` and `{ collapsed, onToggle }` to `<TopBar />`.

- **Why:** the collapse control now lives in a sibling of `Sidebar`, so the
  state must live at their common parent. `__root.tsx` is already the shell
  composition point.
- **Alternative — a context provider (`SidebarStateProvider`):** more ceremony
  than a two-child prop drill warrants; revisit only if a third consumer
  appears.
- **Alternative — keep state in `Sidebar`, lift only a ref/callback:** inverts
  the data flow awkwardly and leaves `Sidebar` re-rendering the parent.

`Sidebar` loses its `useState`/`toggle`/`readCollapsed` and becomes a
presentational component taking `collapsed: boolean`. The `STORAGE_KEY` constant
and `readCollapsed` helper move to `__root.tsx` (or a tiny
`lib/sidebar-state.ts` if that reads cleaner during implementation).

### New `TopBar.tsx` component

A `<header>` that is a flex row: `justify-between`, fixed height matching the
sidebar's `h-14` header (so the sidebar wordmark and the top bar align), bottom
border matching the sidebar's border tokens. Left slot = the collapse button
(the JSX currently inline in `Sidebar`, including `ChevronGlyph`, moved here).
Right slot = `<ThemeToggle />`.

- The collapse button drops its `w-full`/`justify-*` collapsed styling — in the
  top bar it is a fixed-size icon button in both states. Its `aria-label` /
  `aria-pressed` / `title` logic is unchanged.
- `ThemeToggle` is reused as-is. It currently takes `collapsed` to decide
  whether to show the text label; in the top bar we want the icon-only
  presentation, so pass `collapsed` (label hidden) or add an explicit
  `labelled={false}` prop. Decision: **pass `collapsed={true}` semantics via a
  renamed `iconOnly` prop** — cleaner than overloading `collapsed` for a
  sidebar that no longer contains it. Keep the change to `ThemeToggle` minimal
  (rename the prop, same behaviour).

### `ChevronGlyph` ownership

Move `ChevronGlyph` into `TopBar.tsx` (its only consumer after this change).
`HomeGlyph` stays in `Sidebar.tsx`.

### Layout structure in `__root.tsx`

```
<div class="flex min-h-screen">
  <Sidebar collapsed={collapsed} />
  <div class="flex flex-1 flex-col overflow-x-hidden">
    <TopBar collapsed={collapsed} onToggle={toggle} />
    <main class="flex-1"><Outlet /></main>
  </div>
</div>
```

- **Why nest `main` under a flex column:** the top bar and content share the
  reclaimed width when the sidebar collapses; a column keeps the bar above the
  scrollable content. `overflow-x-hidden` moves from `main` to the column
  wrapper.

## Risks / Trade-offs

- **[Existing pages assume `main` is the direct scroll/rows container]** →
  `index.tsx` / `login.tsx` render a self-centring `<section>` with its own
  padding; wrapping `main` in a column with a bar above it does not change their
  layout. Verify visually on `/` and `/login`.
- **[Top bar adds vertical chrome on every route, including `/login`]** → the
  login page keeps the top bar (collapse + theme still make sense there, and the
  sidebar already shows on `/login` today). No conditional rendering — simpler
  and consistent.
- **[`mounted` gate for the width transition now lives in `__root`]** → move the
  `useEffect(() => setMounted(true), [])` up with the state; pass `mounted` to
  `Sidebar` for its `transition-[width]` class. Small prop, avoids a
  first-paint animation.
- **[Prop rename on `ThemeToggle` touches an otherwise-stable file]** → keep it
  to the rename; no logic change, lint + tsc catch call-site drift.

## Migration Plan

Pure frontend refactor, no data or API surface. Ship in one change:

1. Lift state to `__root.tsx`; make `Sidebar` take `collapsed` (+ `mounted`).
2. Add `TopBar.tsx`; move the collapse button + `ChevronGlyph` into it.
3. Rename `ThemeToggle`'s `collapsed` prop to `iconOnly`; render it in `TopBar`.
4. Remove `ThemeToggle` and the collapse button from `Sidebar`'s footer.
5. `pnpm lint && pnpm exec tsc && pnpm build`; visual check `/` and `/login`,
   expanded and collapsed, light/dark/system, and a reload to confirm
   persistence.

Rollback is a straight revert of the frontend commit.

## Open Questions

- Should the top bar be `sticky top-0`? Deferring — not required by the spec and
  no long-scroll pages exist yet. Easy to add later.

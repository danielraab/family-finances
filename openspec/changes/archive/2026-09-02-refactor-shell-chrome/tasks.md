## 1. Lift collapsed state to the shell

- [x] 1.1 In `frontend/src/routes/__root.tsx`, move `STORAGE_KEY`
  (`ff:sidebar-collapsed`), `readCollapsed()`, the `collapsed` `useState`, the
  `toggle` callback (with `localStorage` persistence), and the `mounted`
  `useEffect` up from `Sidebar.tsx` into `RootComponent`.
- [x] 1.2 Change `Sidebar` to a presentational component: accept
  `{ collapsed: boolean; mounted: boolean }` props, delete its own state/effect
  and the `toggle`/`readCollapsed` helpers.
- [x] 1.3 Verify no other file imports the removed helpers or expects `Sidebar`
  to be prop-less.

## 2. Add the top bar

- [x] 2.1 Create `frontend/src/components/TopBar.tsx` as a `<header>` flex row
  (`justify-between`, height matching the sidebar's `h-14` header, bottom border
  matching the sidebar border tokens).
- [x] 2.2 Move the collapse `<button>` and `ChevronGlyph` from `Sidebar.tsx`
  into `TopBar.tsx`, at the bar's left edge. Drop the `w-full`/`justify-*`
  collapsed styling — it is a fixed-size icon button in both states. Keep the
  `aria-label` / `aria-pressed` / `title` logic. `TopBar` takes
  `{ collapsed: boolean; onToggle: () => void }`.
- [x] 2.3 Render `<ThemeToggle iconOnly />` at the bar's right edge.

## 3. Rewire ThemeToggle and the sidebar footer

- [x] 3.1 In `frontend/src/components/ThemeToggle.tsx`, rename the `collapsed`
  prop to `iconOnly` (same behaviour: hide the text label when true). Update the
  `title` gating accordingly.
- [x] 3.2 In `Sidebar.tsx`, remove `<ThemeToggle />` and the collapse button
  from the footer; the footer now renders `<SidebarUser collapsed={collapsed} />`
  alone. Remove the now-unused `ThemeToggle` import.
- [x] 3.3 In `__root.tsx`, compose the shell:
  `<Sidebar collapsed mounted />` beside a `flex flex-1 flex-col
  overflow-x-hidden` wrapper containing `<TopBar collapsed onToggle={toggle} />`
  then `<main class="flex-1"><Outlet /></main>`. Move `overflow-x-hidden` off
  `main`.

## 4. Verify

- [x] 4.1 `cd frontend && pnpm lint` — passes.
- [x] 4.2 `pnpm exec tsc` — no type errors.
- [x] 4.3 `pnpm build` — writes `out/index.html`.
- [x] 4.4 Manual check in `pnpm dev`: on `/` and `/login`, top bar shows the
  collapse control at the left edge (flush against the sidebar) and the theme
  control at the right edge; sidebar footer shows only the user control.
- [x] 4.5 Manual check: collapse the sidebar — icon + nav glyphs + user
  glyph/avatar only, main content widens, top bar controls stay visible and
  working. Reload — collapsed state persists.
- [x] 4.6 Manual check: cycle the theme control through System / Light / Dark;
  reload confirms `ff:theme` persistence and no theme flash.

## 5. Update docs and archive

- [x] 5.1 Update `frontend/AGENTS.md` §"Theming" (theme control is in the top
  bar, not the sidebar footer) and any sidebar-footer description in
  `frontend/README.md` / root `README.md` if present.
- [x] 5.2 Run `openspec validate refactor-shell-chrome` and archive the change
  once implementation is verified.

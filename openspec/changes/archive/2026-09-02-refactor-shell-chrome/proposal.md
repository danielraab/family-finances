## Why

The sidebar footer currently stacks three unrelated controls — the user / sign-in
control, the theme toggle, and the sidebar collapse toggle. Bundling a layout
control (collapse) and a display control (theme) into the navigation footer
crowds it and hides those actions when the sidebar is collapsed. Pulling the
collapse and theme controls into a persistent top bar makes them always visible
and leaves the sidebar footer for the one thing that belongs there: the account.

## What Changes

- Add a thin top bar spanning the main content region, above the route
  `<Outlet>`, present on every route.
- Move the sidebar collapse/expand toggle out of the sidebar footer to the
  **left edge** of the top bar, flush against the sidebar border. It keeps its
  current behaviour (cycles collapsed/expanded, persisted to `localStorage`).
- Move the colour-theme control out of the sidebar footer to the **right edge**
  of the top bar. Its behaviour (three-state cycle, `ff:theme` persistence,
  no-flash) is unchanged.
- The sidebar footer keeps only the user / sign-in control, at the bottom.
- **BREAKING** (spec-level only): the documented "footer holding, in order, a
  user control, a theme control, and a collapse control" layout no longer holds;
  the collapse and theme controls are no longer in the sidebar.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `web-client-shell`: the "Application layout with collapsible sidebar"
  requirement changes — the sidebar footer holds only the user control, and the
  collapse control moves to a new top bar. The "Selectable colour theme"
  requirement changes — the theme control lives in the top bar, not the sidebar
  footer.

## Impact

- `frontend/src/routes/__root.tsx` — app shell gains the top bar around
  `<Outlet>`.
- `frontend/src/components/Sidebar.tsx` — collapse toggle and `ThemeToggle`
  removed from the footer; footer renders `SidebarUser` alone. Collapse state
  ownership moves up so the top bar can drive it.
- New `frontend/src/components/TopBar.tsx` (or equivalent) hosting the collapse
  and theme controls.
- `frontend/src/components/ThemeToggle.tsx` — reused as-is (or with a minor
  style prop) in the new location.
- No backend, API, or dependency changes.

## MODIFIED Requirements

### Requirement: Application layout with collapsible sidebar

The web client SHALL render every route inside a shared layout consisting of a
left sidebar, a top bar spanning the main content region, and a main content
region below the top bar.

The sidebar SHALL contain, from top to bottom: the application icon with a
"Family Finances" wordmark, a navigation list, and a footer holding the user /
sign-in control alone.

The top bar SHALL span the full width of the main content region, above the
active route's content. It SHALL hold the control to collapse or expand the
sidebar pinned to its left edge (visually adjacent to the sidebar), and the
colour-theme control pinned to its right edge. The top bar SHALL be present on
every route.

When collapsed, the sidebar SHALL show only the icon and the navigation item
glyphs (the footer user control rendering as its glyph or avatar alone); the
main content region SHALL expand to use the reclaimed width. The top bar and its
controls SHALL remain visible and functional regardless of the collapsed state.
The collapsed/expanded state SHALL persist across page loads in the browser via
`localStorage`, and the layout MUST render correctly on first load when no
stored value exists.

The layout SHALL be composed with the router's root route and an `<Outlet>` for
the active route; all components render in the browser (there are no
server-rendered components).

#### Scenario: Default expanded layout

- **WHEN** a visitor opens the site in a browser with no stored sidebar state
- **THEN** the sidebar renders expanded, showing the icon, the "Family Finances"
  wordmark, the navigation list with labels, and the user / sign-in control in
  the footer
- **AND** the top bar renders with the collapse control at its left edge and the
  theme control at its right edge

#### Scenario: Collapsing the sidebar

- **WHEN** the visitor activates the collapse control in the top bar
- **THEN** the sidebar narrows to show only the icon and navigation glyphs, with
  the footer user control shown as its glyph or avatar alone
- **AND** the main content region widens to fill the reclaimed space
- **AND** the top bar and its controls remain visible

#### Scenario: State persists across reloads

- **WHEN** the visitor collapses the sidebar and then reloads the page
- **THEN** the sidebar is still collapsed after the reload

#### Scenario: Controls are not in the sidebar footer

- **WHEN** the sidebar footer renders in either the expanded or collapsed state
- **THEN** it contains only the user / sign-in control
- **AND** neither the collapse control nor the theme control appears inside the
  sidebar

### Requirement: Selectable colour theme

The web client SHALL let the visitor choose one of three colour-theme states —
**System**, **Light**, or **Dark** — via a control at the right edge of the top
bar. The control SHALL be a single button that cycles through the three states
in a fixed order on activation, and its glyph SHALL indicate the currently
selected state (including distinguishing **System** from the resolved light or
dark appearance).

The selected state SHALL persist in the browser via `localStorage` under the
key `ff:theme`. When no stored value exists, the state SHALL be **System**.

When the state is **Light** or **Dark**, the page SHALL render in that scheme
regardless of the operating system's `prefers-color-scheme`. When the state is
**System**, the page SHALL follow the operating system's `prefers-color-scheme`,
and SHALL update immediately if that preference changes while the page is open.

The page MUST NOT show a flash of the wrong theme on load: the resolved theme
SHALL be applied before first paint by an inline script in `index.html` that
runs before the app mounts. The theme mechanism SHALL be entirely client-side,
built in-app (no `next-themes` or other framework theme runtime), and MUST NOT
require changes to existing `dark:` styling in individual components (the `dark`
variant keys off a `.dark` class on `<html>`).

#### Scenario: Default follows the OS

- **WHEN** a visitor opens the site with no stored theme value
- **THEN** the page renders in the scheme matching the OS `prefers-color-scheme`
- **AND** the theme control in the top bar indicates the **System** state

#### Scenario: Explicit choice overrides the OS

- **WHEN** the visitor cycles the theme control to **Dark** on a system whose
  `prefers-color-scheme` is light
- **THEN** the page renders in dark immediately
- **AND** after reloading the page it still renders in dark

#### Scenario: System state tracks the OS live

- **WHEN** the theme control is set to **System** and the visitor changes the OS
  colour scheme while the tab is open
- **THEN** the page re-renders in the new scheme without a reload

#### Scenario: Returning to System

- **WHEN** the visitor has an explicit **Light** or **Dark** choice stored and
  cycles the control back to **System**
- **THEN** the page resumes following the OS `prefers-color-scheme`
- **AND** the stored value reflects **System**

#### Scenario: No flash of the wrong theme

- **WHEN** the visitor has **Dark** stored and loads the site
- **THEN** the first painted frame is already dark, with no visible light flash

#### Scenario: Theme control available when the sidebar is collapsed

- **WHEN** the visitor collapses the sidebar
- **THEN** the theme control remains visible and operable at the right edge of
  the top bar

#### Scenario: Static build is preserved

- **WHEN** a developer runs `pnpm build` in `frontend/`
- **THEN** the build succeeds and emits the static site to `frontend/out/` with
  no Node server required to serve it

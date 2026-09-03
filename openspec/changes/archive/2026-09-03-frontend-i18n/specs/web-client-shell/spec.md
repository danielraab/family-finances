## MODIFIED Requirements

### Requirement: Application layout with collapsible sidebar

The web client SHALL render every route inside a shared layout consisting of a
left sidebar, a top bar spanning the main content region, and a main content
region below the top bar.

The sidebar SHALL contain, from top to bottom: the application icon with a
"Family Finances" wordmark, a navigation list, and a footer holding the
colour-theme control and the user / sign-in control.

The top bar SHALL span the full width of the main content region, above the
active route's content. It SHALL hold the control to collapse or expand the
sidebar pinned to its left edge (visually adjacent to the sidebar). The top
bar SHALL be present on every route.

When collapsed, the sidebar SHALL show only the icon and the navigation item
glyphs (the footer theme and user controls rendering as their glyph or
avatar alone); the main content region SHALL expand to use the reclaimed
width. The top bar and its controls SHALL remain visible and functional
regardless of the collapsed state. The collapsed/expanded state SHALL persist
across page loads in the browser via `localStorage`, and the layout MUST
render correctly on first load when no stored value exists.

The layout SHALL be composed with the router's root route and an `<Outlet>` for
the active route; all components render in the browser (there are no
server-rendered components).

All rendered labels in this layout (the navigation item(s), the theme
control's labels, the collapse/expand/open/close controls, and the user /
sign-in control) SHALL be sourced from the client's i18n translation
resources rather than hard-coded literals, per `web-client-i18n`.

#### Scenario: Default expanded layout

- **WHEN** a visitor opens the site in a browser with no stored sidebar state
- **THEN** the sidebar renders expanded, showing the icon, the "Family Finances"
  wordmark, the navigation list with labels, and the theme and user / sign-in
  controls in the footer
- **AND** the top bar renders with the collapse control at its left edge

#### Scenario: Collapsing the sidebar

- **WHEN** the visitor activates the collapse control in the top bar
- **THEN** the sidebar narrows to show only the icon and navigation glyphs, with
  the footer theme and user controls shown as their glyph or avatar alone
- **AND** the main content region widens to fill the reclaimed space
- **AND** the top bar and its controls remain visible

#### Scenario: State persists across reloads

- **WHEN** the visitor collapses the sidebar and then reloads the page
- **THEN** the sidebar is still collapsed after the reload

#### Scenario: Layout labels follow the resolved language

- **WHEN** the client resolves German as the active language
- **THEN** the sidebar navigation label(s), the theme control's labels, and
  the collapse/expand/open/close control labels all render in German

### Requirement: Selectable colour theme

The web client SHALL let the visitor choose one of three colour-theme states —
**System**, **Light**, or **Dark** — via a control in the sidebar footer. The
control SHALL be a single button (icon-only when the sidebar is collapsed) or
a segmented pill (expanded) that lets the visitor pick a state directly, and
its glyph SHALL indicate the currently selected state (including
distinguishing **System** from the resolved light or dark appearance).

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

The control's **System** / **Light** / **Dark** labels, and any `aria-label` /
`title` text derived from them, SHALL be sourced from the client's i18n
translation resources rather than hard-coded literals, per `web-client-i18n`.

#### Scenario: Default follows the OS

- **WHEN** a visitor opens the site with no stored theme value
- **THEN** the page renders in the scheme matching the OS `prefers-color-scheme`
- **AND** the theme control in the sidebar footer indicates the **System** state

#### Scenario: Explicit choice overrides the OS

- **WHEN** the visitor picks **Dark** on a system whose `prefers-color-scheme`
  is light
- **THEN** the page renders in dark immediately
- **AND** after reloading the page it still renders in dark

#### Scenario: System state tracks the OS live

- **WHEN** the theme control is set to **System** and the visitor changes the OS
  colour scheme while the tab is open
- **THEN** the page re-renders in the new scheme without a reload

#### Scenario: Returning to System

- **WHEN** the visitor has an explicit **Light** or **Dark** choice stored and
  picks **System** again
- **THEN** the page resumes following the OS `prefers-color-scheme`
- **AND** the stored value reflects **System**

#### Scenario: No flash of the wrong theme

- **WHEN** the visitor has **Dark** stored and loads the site
- **THEN** the first painted frame is already dark, with no visible light flash

#### Scenario: Theme control available when the sidebar is collapsed

- **WHEN** the visitor collapses the sidebar
- **THEN** the theme control remains visible and operable in the sidebar
  footer

#### Scenario: Static build is preserved

- **WHEN** a developer runs `pnpm build` in `frontend/`
- **THEN** the build succeeds and emits the static site to `frontend/out/` with
  no Node server required to serve it

#### Scenario: Theme labels follow the resolved language

- **WHEN** the client resolves German as the active language
- **THEN** the theme control's **System** / **Light** / **Dark** labels (and
  any `aria-label`/`title` derived from them) render in German

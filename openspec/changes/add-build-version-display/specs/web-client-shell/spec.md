## MODIFIED Requirements

### Requirement: Application layout with collapsible sidebar

The web client SHALL render every route inside a shared layout consisting of a
left sidebar, a top bar spanning the main content region, and a main content
region below the top bar.

The sidebar SHALL contain, from top to bottom: the application icon with a
"Family Finances" wordmark, a navigation list, and a footer holding the
colour-theme control, the user / sign-in control, and — below them — a
small, plain build-version line.

The top bar SHALL span the full width of the main content region, above the
active route's content. It SHALL hold the control to collapse or expand the
sidebar pinned to its left edge (visually adjacent to the sidebar). The top
bar SHALL be present on every route.

When collapsed, the sidebar SHALL show only the icon and the navigation item
glyphs (the footer theme and user controls rendering as their glyph or
avatar alone, and the build-version line rendering as the short commit or
tag alone); the main content region SHALL expand to use the reclaimed
width. The top bar and its controls SHALL remain visible and functional
regardless of the collapsed state. The collapsed/expanded state SHALL persist
across page loads in the browser via `localStorage`, and the layout MUST
render correctly on first load when no stored value exists.

The layout SHALL be composed with the router's root route and an `<Outlet>` for
the active route; all components render in the browser (there are no
server-rendered components).

All rendered labels in this layout (the navigation item(s), the theme
control's labels, the collapse/expand/open/close controls, the user /
sign-in control, and the build-version line's `title`) SHALL be sourced from
the client's i18n translation resources rather than hard-coded literals, per
`web-client-i18n`.

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

### Requirement: Build version is shown in the sidebar

The sidebar footer SHALL show the running backend build's identity as a
small, muted, non-interactive line: the release version (e.g. `v0.4.2`)
when the backend reports one, otherwise the first seven characters of the
commit hash when the backend reports one, otherwise nothing. The full
commit hash SHALL be available as the line's `title` attribute whenever a
commit is shown.

The client SHALL read this from `GET /api/version` once, on mount of the
component that renders it. A failed or pending request SHALL render no
line rather than a placeholder or an error.

The line SHALL remain visible, reduced to just its text (no separate
label), when the sidebar is collapsed, positioned below the theme and user
controls in the footer.

#### Scenario: Release build shows its tag

- **WHEN** the backend reports `{"version": "v0.4.2", "commit": "abc123…"}`
  from `GET /api/version`
- **THEN** the sidebar footer shows `v0.4.2`
- **AND** hovering it shows the full commit hash

#### Scenario: Untagged build shows a short commit

- **WHEN** the backend reports `{"version": "", "commit": "abc1234567…"}`
- **THEN** the sidebar footer shows the first seven characters of the
  commit, e.g. `abc1234`

#### Scenario: Nothing to report shows nothing

- **WHEN** the backend reports `{"version": "", "commit": ""}`, or the
  request fails
- **THEN** the sidebar footer shows no version line

#### Scenario: Visible when collapsed

- **WHEN** the sidebar is collapsed and the backend reports a version or
  commit
- **THEN** the short text is still shown below the user control

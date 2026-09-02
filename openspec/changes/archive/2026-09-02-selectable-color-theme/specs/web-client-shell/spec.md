## MODIFIED Requirements

### Requirement: Application layout with collapsible sidebar

The web client SHALL render every page inside a shared layout consisting of a
left sidebar and a main content region. The sidebar SHALL contain, from top to
bottom: the application icon with a "Family Finances" wordmark, a navigation
list, and a footer holding a colour-theme control and a control to collapse or
expand the sidebar.

When collapsed, the sidebar SHALL show only the icon, the navigation item
glyphs, and the footer control glyphs; the main content region SHALL expand to
use the reclaimed width. The collapsed/expanded state SHALL persist across page
loads in the browser via `localStorage`, and the layout MUST render correctly on
first load when no stored value exists.

The sidebar MAY be a Client Component; all other layout and page components
SHALL remain Server Components rendered at build time.

#### Scenario: Default expanded layout

- **WHEN** a visitor opens the site in a browser with no stored sidebar state
- **THEN** the sidebar renders expanded, showing the icon, the "Family Finances"
  wordmark, the navigation list with labels, the theme control, and the collapse
  control

#### Scenario: Collapsing the sidebar

- **WHEN** the visitor activates the collapse control
- **THEN** the sidebar narrows to show only the icon, navigation glyphs, and the
  theme and collapse control glyphs
- **AND** the main content region widens to fill the reclaimed space

#### Scenario: State persists across reloads

- **WHEN** the visitor collapses the sidebar and then reloads the page
- **THEN** the sidebar is still collapsed after the reload

## ADDED Requirements

### Requirement: Selectable colour theme

The web client SHALL let the visitor choose one of three colour-theme states —
**System**, **Light**, or **Dark** — via a control in the sidebar footer. The
control SHALL be a single button that cycles through the three states in a fixed
order on activation, and its glyph SHALL indicate the currently selected state
(including distinguishing **System** from the resolved light or dark appearance).

The selected state SHALL persist in the browser via `localStorage`. When no
stored value exists, the state SHALL be **System**.

When the state is **Light** or **Dark**, the page SHALL render in that scheme
regardless of the operating system's `prefers-color-scheme`. When the state is
**System**, the page SHALL follow the operating system's `prefers-color-scheme`,
and SHALL update immediately if that preference changes while the page is open.

The page MUST NOT show a flash of the wrong theme on load: the resolved theme
SHALL be applied before first paint. Introducing script-driven theming MUST NOT
require changes to existing `dark:` styling in individual components.

The theme mechanism SHALL be entirely client-side and MUST NOT depend on
server-only Next.js features; the static build output contract is unchanged.

#### Scenario: Default follows the OS

- **WHEN** a visitor opens the site with no stored theme value
- **THEN** the page renders in the scheme matching the OS `prefers-color-scheme`
- **AND** the theme control indicates the **System** state

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

#### Scenario: Static build is preserved

- **WHEN** a developer runs `pnpm build` in `frontend/`
- **THEN** the build succeeds and emits the static site to `frontend/out/` with
  no Node server required to serve it

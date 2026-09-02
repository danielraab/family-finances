## Why

The web client mirrors the OS colour scheme via a `prefers-color-scheme` media
query, with no way for a visitor to override it. People often want the app in a
specific mode regardless of their OS (e.g. dark app on a light desktop). This
adds an explicit, browser-persisted theme choice while keeping "follow the OS"
as the default and as a first-class, re-selectable option.

## What Changes

- Add a three-state theme control — **System / Light / Dark** — to the sidebar
  footer, as a single icon button that cycles through the states (sun → moon →
  monitor). It renders icon-only when the sidebar is collapsed, matching the
  existing collapse control.
- Persist the selection in `localStorage`. On first load with no stored value,
  the theme is **System**.
- When **System** is selected, the page tracks the OS preference live — a
  `matchMedia` change while the tab is open re-themes the page immediately.
- Switch Tailwind v4's `dark:` variant from the `prefers-color-scheme` media
  query to a `.dark` class on `<html>`, so an explicit choice can win over the
  OS. Every existing `dark:` utility keeps working unchanged.
- Add an inline, render-blocking script in `<head>` (via `next-themes`) that
  resolves and applies the theme before first paint, so introducing
  JS-driven theming does not introduce a flash of the wrong theme.
- Add the `next-themes` dependency to `frontend/`.

No breaking changes. The static-export build contract is unaffected —
`next-themes` is entirely client-side.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `web-client-shell`: The "Application layout with collapsible sidebar"
  requirement gains a theme control in the sidebar footer alongside the collapse
  control. A new requirement, "Selectable colour theme", specifies the
  three-state model, `localStorage` persistence, System-as-default, live OS
  tracking while System is selected, and no flash of the wrong theme on load.

## Impact

- **Dependencies**: `frontend/package.json` gains `next-themes` (client-only;
  works under `output: "export"`).
- **Code**:
  - `frontend/app/globals.css` — add
    `@custom-variant dark (&:where(.dark, .dark *));`; convert the
    `@media (prefers-color-scheme: dark)` block to a `.dark` selector.
  - `frontend/app/layout.tsx` — add `suppressHydrationWarning` to `<html>`;
    wrap children in the theme provider.
  - `frontend/app/components/` — new client `Providers` wrapper and
    `ThemeToggle` component; `Sidebar.tsx` footer renders `ThemeToggle`.
- **Build / CI**: no new server features; `pnpm lint` (Biome) and `pnpm build`
  remain the gate. The Docker image embeds the same static bundle, now with the
  extra client JS.
- **Spec**: delta on `openspec/specs/web-client-shell/spec.md`.

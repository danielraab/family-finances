## Context

Today the web client has no theme *selection* — it passively mirrors the OS.
`app/globals.css` flips `--background` / `--foreground` inside
`@media (prefers-color-scheme: dark)`, and Tailwind v4's default `dark:` variant
compiles to that same media query. `Sidebar.tsx` and `Placeholder.tsx` already
carry `dark:` utilities. Because the resolution is pure CSS, it is flash-free by
construction: the browser applies it before first paint.

Constraints that shape this design:

- **Static export, no server.** `next.config.ts` sets `output: "export"`;
  `frontend/AGENTS.md` forbids route handlers, middleware, and request-time
  rendering. Every visitor receives one byte-identical prebuilt HTML file served
  by the Go binary. A server cannot read a cookie and stamp `class="dark"` into
  the HTML. **User preference can only live in the browser and be resolved by
  JavaScript after the HTML loads.**
- **New versions of Next / React.** `next@16.3.4`, `react@19.2.8`. Conventions
  may differ from older Next; `frontend/AGENTS.md` requires reading
  `node_modules/next/dist/docs/` before editing `layout.tsx`.
- **Minimal-dependency posture.** The app has no UI dependencies beyond
  Next / React / Tailwind.

## Goals / Non-Goals

**Goals:**

- A visitor can explicitly choose **System**, **Light**, or **Dark**.
- The choice persists in the browser across reloads.
- With no stored choice, the theme is **System** (current behaviour preserved).
- While **System** is selected, the page follows OS changes live, not only at
  load.
- No flash of the wrong theme on load, despite moving from CSS-media resolution
  to JS-class resolution.
- Every existing `dark:` utility keeps working with no per-file edits.

**Non-Goals:**

- A settings page or any theme surface outside the sidebar.
- Per-route or per-component theming; more than three states; custom accent
  colours or palettes.
- Syncing the preference to the backend or across devices (it is browser-local,
  like `ff:sidebar-collapsed`).
- Changing the actual colour values of either palette.

## Decisions

### Decision: Three-state model (System / Light / Dark), System re-selectable

`localStorage` holds `"system" | "light" | "dark"`. Selecting **System** means
"follow the OS live" — not "snapshot the OS now". This matches GitHub / VS Code
and is what "default to system preference" most naturally implies once a real
selector exists.

*Alternative considered:* two-state (`"light" | "dark"` only, System as an
un-storable first-run default seeded from the OS). Simpler — no live listener —
but there is no way back to "follow the OS" after the first pick without clearing
storage. Rejected: losing the ability to return to System is a surprising
one-way door.

### Decision: Use `next-themes` rather than hand-rolling

`next-themes` provides exactly this three-state model, `localStorage`
persistence, the live `matchMedia` listener for `system`, and — critically — an
inline script injected into `<head>` that resolves and applies the theme class
before first paint. It is client-only and works under `output: "export"` (the
provider and its script render into the static HTML; no server features).

*Alternative considered:* hand-roll (~30–40 lines: inline FOUC script + a small
context/hook + storage key, mirroring the `ff:sidebar-collapsed` pattern in
`Sidebar.tsx`). Keeps the zero-UI-dependency posture. Rejected for this change:
the FOUC-avoidance + hydration-suppression details are fiddly to re-derive
correctly, and `next-themes` is tiny and purpose-built for the static-export
case. The dependency cost is judged acceptable.

### Decision: Repoint Tailwind's `dark:` variant to a class

Add to `app/globals.css`:

```css
@custom-variant dark (&:where(.dark, .dark *));
```

and convert the existing `@media (prefers-color-scheme: dark) { :root { … } }`
block to a `.dark { … }` selector for the CSS variables. `next-themes` is
configured with `attribute="class"`, so it toggles `.dark` on `<html>`. This one
line is what makes every `dark:` utility already in `Sidebar.tsx` /
`Placeholder.tsx` obey the stored choice instead of the OS — those files are not
touched.

### Decision: Cycling icon button in the sidebar footer

A single button in the sidebar footer, styled like the existing collapse control
(`Sidebar.tsx:125-141`): full-width row, `border-t`, icon + label that hides when
collapsed. Clicking cycles **System → Light → Dark → System**. The glyph
reflects the *selected* state (monitor / sun / moon), not the resolved one, so
"System" is visually distinct. New inline SVG glyphs follow the existing
`HomeGlyph` / `ChevronGlyph` pattern in that file.

*Alternative considered:* a 2- or 3-segment segmented control showing all states
at once. Clearer affordance, but heavier and awkward at `w-16` when the sidebar
is collapsed. Rejected in favour of matching the sidebar's existing minimal
icon-button language.

### Decision: `suppressHydrationWarning` on `<html>`

`layout.tsx` renders `<html className="…">`, a React-managed attribute. The
`next-themes` inline script mutates `<html>`'s class before React hydrates, so
server and client markup disagree at hydration. `suppressHydrationWarning` on the
`<html>` element is the standard, documented fix and scopes the suppression to
that one element.

### Decision: `disableTransitionOnChange`

`Sidebar.tsx` puts `transition-colors` on nearly every element. Without
`disableTransitionOnChange` on the provider, switching theme visibly sweeps
colours across the sidebar. The provider briefly disables transitions during the
switch.

## Risks / Trade-offs

- **`next-themes` vs Next 16 / React 19** → After `pnpm add next-themes`,
  confirm the current release mounts cleanly under React 19 and static export
  (build succeeds, no hydration errors in the console). Read
  `node_modules/next/dist/docs/` for any changed `layout` / metadata
  conventions before editing `layout.tsx`.
- **`useTheme()` is undefined on the server render** → `ThemeToggle` must guard
  with a `mounted` flag (as `Sidebar.tsx:58` already does) and render a stable
  placeholder — the monitor glyph — until mounted, so the button does not cause
  its own hydration mismatch.
- **New dependency in a deliberately lean frontend** → Accepted; scoped to
  `frontend/package.json`, client-only, no transitive server code. Revisit only
  if it breaks under a future Next upgrade.
- **Extra client JS in the embedded bundle** → `next-themes` is a few KB; no
  measurable impact on the Go-embedded static bundle or the Docker image.
- **CI has no frontend tests** → The gate stays `pnpm lint` (Biome) +
  `pnpm build`. Scenarios in the spec are verified manually / by build success,
  not an automated suite.

## Migration Plan

1. `cd frontend && pnpm add next-themes`.
2. Edit `globals.css` (`@custom-variant` + `.dark` block), `layout.tsx`
   (`suppressHydrationWarning` + provider wrapper), add `Providers.tsx` and
   `ThemeToggle.tsx`, render `ThemeToggle` in the `Sidebar.tsx` footer.
3. `pnpm lint` and `pnpm build`; preview `out/` with `npx serve out`; verify no
   theme flash, persistence across reload, and live OS tracking while System is
   selected.
4. Update `frontend/AGENTS.md` / `frontend/README.md` if they describe the
   theme behaviour.

Rollback: revert the change and `pnpm remove next-themes`. No data or schema
migration — the only persisted state is a single `localStorage` key that is
simply ignored once the code is gone.

## Open Questions

- Storage key name — proposed `ff:theme`, consistent with `ff:sidebar-collapsed`.
  `next-themes` defaults to `"theme"`; set `storageKey="ff:theme"` to match the
  repo convention.
- Cycle order — **System → Light → Dark** proposed; could start at Light. Minor;
  settled at implementation.

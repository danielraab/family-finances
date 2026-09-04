## Context

`web-client-shell`'s "Application icon and favicon" requirement currently
covers only the browser-tab favicon (`icon.svg` + `favicon.ico`) and the
inline React sidebar logo — there is no web app manifest, so the app has
never been installable with a correct icon. `frontend/public/` is served
as-is by Vite (and, in production, by the Go backend's static handler per
`backend-static-serving`), so any file dropped there is reachable at its
root-relative path with no build step.

Carried-over constraints, per `frontend/AGENTS.md` and the root `AGENTS.md`:

- Static bundle, no server-side generation — icons must be committed static
  assets, not rendered per-request or at build time.
- No new frontend dependency for something this narrow: no
  `vite-plugin-pwa`, no service worker, no offline caching. This change is
  scoped to installability (a manifest + correctly-sized icons), not full
  PWA/offline behaviour — that's a separate, larger change if ever wanted.
- The existing SVG mark (`frontend/public/icon.svg`, house-with-Euro-sign on
  `#15803d`) stays the single source of truth for the brand mark; the new
  PNGs are renders of it, not a redesign.

## Goals / Non-Goals

**Goals:**

- A mobile browser's install/"Add to Home Screen" prompt shows the correct
  house-with-Euro-sign mark at full resolution, not a blurry upscale or a
  generic placeholder.
- Android's adaptive-icon mask (which crops a maskable icon to a circle/
  squircle/rounded-square depending on the launcher) doesn't clip the mark.
- iOS home-screen installs (which read `apple-touch-icon`, not the manifest)
  get the same mark.

**Non-Goals:**

- Offline support, a service worker, or any caching strategy — none is added
  by this change.
- An install-prompt UI (a custom "Install this app" button) — this change
  only makes the browser's own native prompt able to render the icon
  correctly; it doesn't add any new UI.
- Redesigning the mark itself — the PNGs are mechanical renders of the
  existing `icon.svg`.

## Decisions

### Decision: generate PNGs once, commit them as static assets — no build-time rendering

`icon.svg` is small (64x64 viewBox) and vector, so a naive `<img>`/CSS
approach can't produce the fixed-size PNGs a manifest requires (`icons[].src`
must point at real raster files at the declared size for browsers/OSes that
don't rasterize SVG for this purpose). Rendering happens once, offline
(headless Chromium via Playwright, already available in this environment, so
no new project dependency), by loading the SVG in a page sized to each target
resolution and screenshotting it. The output PNGs are committed to
`frontend/public/icons/`, same as `favicon.ico` today — regenerating them
only matters if `icon.svg` itself changes, at which point regenerate by hand
the same way `favicon.ico` already implicitly requires when the mark
changes (no committed regeneration script, matching the precedent that
`favicon.ico` has none either).

### Decision: sizes — 192, 512 (`any`), 512 maskable, and a separate apple-touch-icon

`192x192` and `512x512` are the two sizes the W3C manifest spec and Chrome's
installability criteria actually check for (`any` purpose covers browser
install prompts and Android's non-adaptive fallback). A separate
`512x512` `purpose: "maskable"` entry is needed because Android's adaptive
icons crop *up to* the safe-area circle inscribed in the icon; the `any`
variant is drawn edge-to-edge (matching the existing tile) and would lose
its rounded corners/margin under that mask, so the maskable variant pads the
mark inward (visually, a smaller mark centered on a full-bleed `#15803d`
square) so the safe area survives any launcher's mask shape. `apple-touch-
icon` is `180x180` (Apple's current recommended size) and is linked directly
from `index.html` via `<link rel="apple-touch-icon">`, not the manifest,
because iOS Safari's home-screen add flow ignores the web app manifest for
its icon entirely.

### Decision: `theme_color`/`background_color` reuse the existing tile colour

Both are set to `#15803d` (the icon tile's green), matching the colour
already used for the icon and avoiding introducing a second brand colour
decision. `background_color` is the splash-screen background some browsers
show while a standalone-launched PWA is loading; `theme_color` tints the
OS/browser chrome (status bar, task switcher) around the app — both belong
to the mark's own colour, not the app's light/dark theme (`web-client-shell`'s
existing colour-theme system stays a page-rendering concern, unrelated to
these OS-level chrome colours).

### Decision: `display: "standalone"`, `start_url: "/"`

`standalone` is the conventional choice for an app that isn't a page-content
site (no browser chrome once installed, matching how a home-screen app is
expected to look) and requires no new routing — `start_url: "/"` is the
existing root route, unchanged.

## Risks / Trade-offs

- **Manual regeneration if the mark changes.** No automated pipeline
  regenerates the PNGs from `icon.svg`; a future edit to the SVG must
  manually re-run the same render step. Accepted, matching the existing
  `favicon.ico` precedent (also hand-regenerated, never automated).
- **No service worker means some browsers may not show an install prompt at
  all** (a subset of Chromium's installability criteria historically wanted
  one). This is explicitly a non-goal here; if that becomes a problem later
  it's a separate, larger change (adding `vite-plugin-pwa` or a hand-written
  service worker), not a reason to hold this one back — the manifest and
  icons are correct and useful on their own for every browser that doesn't
  require it (including current Chrome/Edge and Android's "Add to Home
  Screen").

## Migration Plan

1. Render `icon-192.png`, `icon-512.png`, `icon-512-maskable.png`,
   `apple-touch-icon.png` from `frontend/public/icon.svg` into
   `frontend/public/icons/`.
2. Add `frontend/public/manifest.webmanifest`.
3. Link it and the Apple/theme-color meta from `frontend/index.html`.
4. Update `web-client-shell`'s "Application icon and favicon" requirement
   (delta spec) to describe the manifest and its icon set.
5. Manual verification: load the built site, confirm
   `GET /manifest.webmanifest` and each icon resolve; Chrome DevTools'
   Application panel reports the manifest with no installability errors and
   renders the maskable icon preview without clipping the mark.

Rollback: revert the commit — the manifest and icons are additive, static
files with no code depending on them elsewhere.

## Open Questions

None blocking.

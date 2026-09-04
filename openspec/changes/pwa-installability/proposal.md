## Why

The web client has no web app manifest today — `index.html` links only
`icon.svg` and `favicon.ico` for the browser tab (per `web-client-shell`'s
"Application icon and favicon" requirement). Without a manifest, a mobile
browser's "Add to Home Screen" / install prompt has no properly-sized icon to
draw from: it either falls back to a blurry upscale of the small favicon or
skips the icon entirely, so the installed app looks broken next to every
other icon on the home screen.

## What Changes

- Add `frontend/public/manifest.webmanifest` — `name`, `short_name`,
  `start_url`, `display: standalone`, `theme_color`/`background_color`
  matching the existing icon tile colour (`#15803d`), and an `icons` array
  covering the standard installable sizes.
- Generate PNG icons from the existing `frontend/public/icon.svg` mark (the
  house-with-Euro-sign on the green tile) at the resolutions manifest
  consumers actually request: `192x192` and `512x512` (`purpose: "any"`),
  plus a `512x512` `purpose: "maskable"` variant with safe-area padding so
  Android's adaptive-icon mask doesn't crop the mark, and a `180x180`
  `apple-touch-icon` for iOS (which ignores the manifest for its home-screen
  icon). Committed as static PNGs, generated once — the build stays a static
  bundle with no image-processing step at build time.
- Link the manifest and the Apple/`theme-color` meta from `index.html`.
- Extends `web-client-shell`'s existing "Application icon and favicon"
  requirement — no new capability.

## Capabilities

### Modified Capabilities

- `web-client-shell`: the "Application icon and favicon" requirement gains a
  web app manifest with multi-resolution icons, so the client is installable
  with the correct mark instead of only carrying a favicon.

## Impact

- **Code**: `frontend/public/manifest.webmanifest` (new),
  `frontend/public/icons/{icon-192.png,icon-512.png,icon-512-maskable.png,apple-touch-icon.png}`
  (new, generated from `icon.svg`), `frontend/index.html` (manifest link +
  theme-color/apple-touch-icon meta).
- **Spec**: delta on `web-client-shell`.
- No backend, API contract, or i18n changes — the manifest's `name`/
  `short_name` are the fixed "Family Finances" brand name, the one string in
  this codebase that's exempt from `useTranslation()` (per
  `frontend/AGENTS.md`), same as everywhere else it appears verbatim.

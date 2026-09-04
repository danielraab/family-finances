## MODIFIED Requirements

### Requirement: Application icon and favicon

The web client SHALL define its own icon: a house silhouette with a Euro sign
(€) cut out of it, on a deep-green (`#15803d`) rounded tile, provided as an SVG
at `frontend/public/icon.svg` and as a multi-resolution
`frontend/public/favicon.ico`. Browsers SHALL load this icon as the site
favicon via `<link>` elements in `index.html`, and the same mark SHALL be used
as the sidebar logo (an inline React component kept in sync with the SVG).
Because the build is a static bundle, the icon MUST be a static asset, not
generated at request time.

The web client SHALL also ship a web app manifest
(`frontend/public/manifest.webmanifest`, linked via `<link rel="manifest">`
in `index.html`) declaring the same mark at the resolutions installability
requires: `frontend/public/icons/icon-192.png` (192x192) and `icon-512.png`
(512x512), both `purpose: "any"`, plus `icon-512-maskable.png` (512x512,
`purpose: "maskable"`, the mark padded inward so it survives Android's
adaptive-icon safe-area mask). `index.html` SHALL additionally link
`frontend/public/icons/apple-touch-icon.png` (180x180) via `<link
rel="apple-touch-icon">` and set `<meta name="theme-color">`, since iOS reads
neither the manifest's icon list nor its `theme_color` for its home-screen
icon or chrome tinting. The manifest's `theme_color` and `background_color`
SHALL both be `#15803d` (the icon tile's colour), `display` SHALL be
`"standalone"`, and `start_url` SHALL be `"/"`. All of these icons are
static, committed assets rendered from `icon.svg` — no image processing runs
at build or request time.

#### Scenario: Favicon is the custom icon

- **WHEN** the site is loaded in a browser
- **THEN** the browser tab shows the house-with-Euro icon, not a framework
  default

#### Scenario: Sidebar uses the same mark

- **WHEN** the sidebar renders
- **THEN** the icon shown at the top of the sidebar is the same house-with-Euro
  mark used as the favicon

#### Scenario: Manifest is discoverable and installable

- **WHEN** a browser loads the site and inspects it for installability
- **THEN** it finds `manifest.webmanifest` via the `<link rel="manifest">`
  element, with `icons` entries at 192x192 and 512x512 (`purpose: "any"`)
  that resolve to real PNG files at those declared sizes

#### Scenario: Home screen / install icon renders at full resolution

- **WHEN** a visitor installs the app or adds it to their home screen on a
  platform that reads the manifest
- **THEN** the installed icon is the house-with-Euro mark rendered sharply
  at the platform's requested size, not an upscaled favicon or a generic
  placeholder

#### Scenario: Maskable icon survives an adaptive-icon crop

- **WHEN** an Android launcher applies its adaptive-icon mask (circle,
  squircle, or rounded square) to the installed icon
- **THEN** it uses the `purpose: "maskable"` icon, and the house-with-Euro
  mark remains fully visible within the mask's safe area, uncropped

#### Scenario: iOS home-screen install uses the Apple touch icon

- **WHEN** a visitor adds the site to their home screen from iOS Safari
- **THEN** the installed icon is `apple-touch-icon.png` (180x180), linked
  directly in `index.html`, not sourced from the manifest

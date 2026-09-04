## 1. Icons

- [x] 1.1 Render `frontend/public/icons/icon-192.png` (192x192) and
  `icon-512.png` (512x512) from `frontend/public/icon.svg`, edge-to-edge
  (matching the existing tile), `purpose: "any"`.
  → Rendered via headless Chromium (Playwright, already available in this
  environment — no new project dependency); transparent corners outside the
  rounded tile (RGBA, not a white halo).
- [x] 1.2 Render `frontend/public/icons/icon-512-maskable.png` (512x512):
  the mark padded inward on a full-bleed `#15803d` square so it survives
  Android's adaptive-icon safe-area mask.
  → Mark scaled to 60% and centered on an edge-to-edge, opaque `#15803d`
  square, well inside the spec's 80% safe-area circle.
- [x] 1.3 Render `frontend/public/icons/apple-touch-icon.png` (180x180),
  edge-to-edge, for iOS's `<link rel="apple-touch-icon">`.
  → Opaque, un-rounded square (no alpha, no pre-rounded corners) per
  Apple's guidance — iOS applies its own mask/rounding and fills
  transparency with black, so this variant deliberately differs from the
  rounded-and-transparent 192/512 "any" icons.

## 2. Manifest

- [x] 2.1 Add `frontend/public/manifest.webmanifest`: `name: "Family
  Finances"`, `short_name: "Family Finances"`, `start_url: "/"`,
  `display: "standalone"`, `theme_color`/`background_color: "#15803d"`,
  `icons` array listing the three sized-and-purposed PNGs from section 1.

## 3. Wire into index.html

- [x] 3.1 `frontend/index.html`: `<link rel="manifest"
  href="/manifest.webmanifest">`, `<link rel="apple-touch-icon"
  href="/icons/apple-touch-icon.png">`, `<meta name="theme-color"
  content="#15803d">`.

## 4. Spec sync

- [x] 4.1 Apply this change's `specs/web-client-shell` delta onto
  `openspec/specs/web-client-shell/spec.md` (the `openspec` CLI is
  unavailable in this environment, as for prior changes — applied by hand).

## 5. Verify

- [x] 5.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
  → All pass (the one pre-existing `noUselessFragments` info-level finding
  in `src/routes/index.tsx` predates this change and is untouched by it).
- [x] 5.2 Manual pass: serve `frontend/out/`, confirm
  `/manifest.webmanifest` and every icon under `/icons/` resolve with the
  right content-type/size; Chrome DevTools' Application panel shows no
  installability errors and previews the maskable icon without clipping the
  mark.
  → `pnpm preview` + `curl`: `index.html` (200), `manifest.webmanifest`
  (200, `application/manifest+json`), and all four PNGs under `/icons/`
  (200, `image/png`) all resolve. Driven with Playwright/Chromium against
  the running preview: `document.querySelector('link[rel="manifest"]')`
  and `link[rel="apple-touch-icon"]` resolve to the right URLs, `meta[name
  ="theme-color"]` is `#15803d`, and `fetch()`ing the manifest returns the
  expected JSON with all three icon entries. Pixel-sampled each rendered
  PNG via canvas `getImageData`: `icon-192`/`icon-512` have transparent
  (alpha 0) corners outside the rounded tile, `apple-touch-icon` and the
  maskable icon are fully opaque (alpha 255) edge-to-edge, and the
  maskable mark's visible extent sits well inside the 80% safe-area circle.

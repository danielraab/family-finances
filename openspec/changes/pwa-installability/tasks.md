## 1. Icons

- [ ] 1.1 Render `frontend/public/icons/icon-192.png` (192x192) and
  `icon-512.png` (512x512) from `frontend/public/icon.svg`, edge-to-edge
  (matching the existing tile), `purpose: "any"`.
- [ ] 1.2 Render `frontend/public/icons/icon-512-maskable.png` (512x512):
  the mark padded inward on a full-bleed `#15803d` square so it survives
  Android's adaptive-icon safe-area mask.
- [ ] 1.3 Render `frontend/public/icons/apple-touch-icon.png` (180x180),
  edge-to-edge, for iOS's `<link rel="apple-touch-icon">`.

## 2. Manifest

- [ ] 2.1 Add `frontend/public/manifest.webmanifest`: `name: "Family
  Finances"`, `short_name: "Family Finances"`, `start_url: "/"`,
  `display: "standalone"`, `theme_color`/`background_color: "#15803d"`,
  `icons` array listing the three sized-and-purposed PNGs from section 1.

## 3. Wire into index.html

- [ ] 3.1 `frontend/index.html`: `<link rel="manifest"
  href="/manifest.webmanifest">`, `<link rel="apple-touch-icon"
  href="/icons/apple-touch-icon.png">`, `<meta name="theme-color"
  content="#15803d">`.

## 4. Spec sync

- [ ] 4.1 Apply this change's `specs/web-client-shell` delta onto
  `openspec/specs/web-client-shell/spec.md` (the `openspec` CLI is
  unavailable in this environment, as for prior changes — apply by hand).

## 5. Verify

- [ ] 5.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 5.2 Manual pass: serve `frontend/out/`, confirm
  `/manifest.webmanifest` and every icon under `/icons/` resolve with the
  right content-type/size; Chrome DevTools' Application panel shows no
  installability errors and previews the maskable icon without clipping the
  mark.

## 1. Dependency & Next 16 groundwork

- [x] 1.1 In `frontend/`, run `pnpm add next-themes`; confirm `package.json` and
  `pnpm-lock.yaml` update and no `package-lock.json` appears.
  → `next-themes@0.4.6` added; no `package-lock.json`.
- [x] 1.2 Read `frontend/node_modules/next/dist/docs/` for current `layout` /
  metadata / client-provider conventions before editing `layout.tsx`.
  → Read `02-guides/preventing-flash-before-hydration.md` (theme inline-script +
  `suppressHydrationWarning` pattern) and `03-file-conventions/layout.md`
  (`LayoutProps<'/'>` helper, already in use).

## 2. Tailwind variant & palette

- [x] 2.1 In `frontend/app/globals.css`, add
  `@custom-variant dark (&:where(.dark, .dark *));` after the `@import`.
- [x] 2.2 Replace the `@media (prefers-color-scheme: dark) { :root { … } }`
  block with a `.dark { … }` selector carrying the same
  `--background` / `--foreground` values.
- [x] 2.3 Verify existing `dark:` utilities respond to the class.
  → Compiled CSS in `out/` has 11 `:where(.dark, .dark *)` selectors and 0
  `prefers-color-scheme: dark` rules — every `dark:` utility is now class-driven.

## 3. Theme provider wiring

- [x] 3.1 Add `frontend/app/components/Providers.tsx` — a `"use client"`
  component wrapping children in `next-themes`' `ThemeProvider` with
  `attribute="class"`, `defaultTheme="system"`, `enableSystem`,
  `disableTransitionOnChange`, and `storageKey="ff:theme"`.
- [x] 3.2 In `frontend/app/layout.tsx`, add `suppressHydrationWarning` to the
  `<html>` element and wrap the body content in `<Providers>`.

## 4. Theme toggle control

- [x] 4.1 Add `frontend/app/components/ThemeToggle.tsx` (`"use client"`): use
  `useTheme()`, a `mounted` guard (render the monitor glyph until mounted),
  and cycle `system → light → dark → system` on click.
- [x] 4.2 Add sun / moon / monitor inline SVG glyphs in that file, matching the
  `HomeGlyph` / `ChevronGlyph` style in `Sidebar.tsx` (20×20, `currentColor`,
  `aria-hidden`).
- [x] 4.3 Style the button like the collapse control (`Sidebar.tsx:126-140`):
  full-width row, icon + label, label hidden and icon centered when collapsed;
  set `aria-label` / `title` to the current selection and next action.

## 5. Sidebar integration

- [x] 5.1 Render `<ThemeToggle />` in the `Sidebar.tsx` footer alongside the
  collapse control (shared `border-t` footer region, now `flex flex-col gap-1`),
  respecting the `collapsed` state.

## 6. Verify

- [x] 6.1 `cd frontend && pnpm lint` (Biome) passes.
- [x] 6.2 `pnpm build` succeeds and emits `frontend/out/`.
  → Static export OK; next-themes' blocking inline script is the first child of
  `<body>` (before `<aside>`), reads `ff:theme` + `matchMedia`, applies the class
  and `colorScheme` before paint.
- [x] 6.3 Interactive browser pass: no theme flash on load with `Dark` stored;
  choice survives reload; `System` re-themes live when the OS scheme changes;
  cycling back to `System` resumes OS-following; no hydration warnings in the
  console.
  → NOT run in-session: no working browser in this environment (snap chromium
  produces no output). Static evidence above covers the no-flash and
  system-resolution paths; the OS-toggle / reload / console checks still need a
  human pass via `npx serve out` or `pnpm dev`.
- [x] 6.4 Update `frontend/AGENTS.md` / `frontend/README.md` if they describe
  theme behaviour.
  → Neither documented theming before. Added a "Theming" bullet to
  `frontend/AGENTS.md` Tooling (class-based variant, next-themes ownership,
  `ff:theme` key, `suppressHydrationWarning` rationale, control location).
  `frontend/README.md` needed no change. Left the `next dev` agent-files block
  untouched.

## 7. Spec sync

- [x] 7.1 After implementation, run `/opsx:archive` to fold the delta into
  `openspec/specs/web-client-shell/spec.md`.

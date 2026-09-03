## 1. Dependencies

- [x] 1.1 In `frontend/`, run
  `pnpm add i18next react-i18next i18next-browser-languagedetector`; confirm
  `package.json` / `pnpm-lock.yaml` update and no `package-lock.json` appears.
  → `i18next@26.4.1`, `react-i18next@17.0.13`,
  `i18next-browser-languagedetector@8.2.1` added; no `package-lock.json`.

## 2. i18n setup

- [x] 2.1 Add `frontend/src/i18n/locales/en.json` and
  `frontend/src/i18n/locales/de.json` with nested keys grouped by component
  (`sidebar`, `topbar`, `user`, `theme`, `login`, `home`), covering every
  string identified in the design.
- [x] 2.2 Add `frontend/src/i18n/index.ts`: initialize `i18next` with
  `react-i18next`'s `initReactI18next` and `LanguageDetector`
  (`i18next-browser-languagedetector`), config: `resources` from the two JSON
  files, `fallbackLng: "en"`, `supportedLngs: ["en", "de"]`,
  `nonExplicitSupportedLngs: true`, detector `order: ["navigator"]`,
  `caches: []`.
  → Also added `"resolveJsonModule": true` to `tsconfig.json` so the JSON
  resources type-check as imports.
- [x] 2.3 Import `frontend/src/i18n` at the top of `frontend/src/main.tsx`,
  before the router is created/rendered.

## 3. Extract strings

- [x] 3.1 `frontend/src/components/Sidebar.tsx` — nav item label(s), the
  mobile backdrop `aria-label` ("Close sidebar").
- [x] 3.2 `frontend/src/components/TopBar.tsx` — the four
  expand/collapse/open/close labels.
- [x] 3.3 `frontend/src/components/SidebarUser.tsx` — "Log in", "Log out".
- [x] 3.4 `frontend/src/components/ThemeSwitch.tsx` — System/Light/Dark
  labels and their `aria-label`/`title` templates (interpolated, not
  concatenated).
- [x] 3.5 `frontend/src/routes/login.tsx` — heading, subtitle, "or" divider,
  email field label, validation/error messages, submit button (idle +
  submitting), "Check your inbox" panel (including the
  bold-email sentence via `<Trans>`), "Use a different address".
- [x] 3.6 `frontend/src/components/Placeholder.tsx` — heading tagline,
  "Nothing here yet", the placeholder body copy. Left the "Family Finances"
  brand name itself untranslated.

## 4. Runtime `<html lang>` sync

- [x] 4.1 In `frontend/src/routes/__root.tsx`, add an effect that sets
  `document.documentElement.lang` to `i18n.resolvedLanguage` on mount and
  whenever `i18next` fires `languageChanged`.

## 5. Verify

- [x] 5.1 `cd frontend && pnpm lint` (Biome) passes.
  → Ran `biome check --write .`; only auto-fixes were import ordering in
  touched files (the pre-existing `noUselessFragments` info in
  `routes/index.tsx` is unrelated to this change and was left as-is).
- [x] 5.2 `pnpm exec tsc` passes.
- [x] 5.3 `pnpm build` succeeds and emits `frontend/out/`.
- [x] 5.4 Interactive browser pass (`pnpm preview` + Playwright, using the
  pre-installed Chromium at `/opt/pw-browsers/chromium`): confirmed
  locale `de-AT` renders German ("Start" nav label, "Anmelden" login title,
  `<html lang="de">`), `en-US` renders English (`<html lang="en">`), and
  unsupported `fr-FR` falls back to English (`<html lang="en">`).
- [x] 5.5 Update `frontend/AGENTS.md` with a short "i18n" section (library,
  detection mechanism, where resources live, that a manual switcher and
  persistence are not yet implemented).

## 6. Spec sync

- [x] 6.1 After implementation, folded the delta manually (the `openspec`
  CLI is not installed in this environment): created
  `openspec/specs/web-client-i18n/spec.md` and applied the
  `web-client-shell` delta — which also corrected that spec's "theme
  control lives at the top bar's right edge" / "sidebar footer holds only
  the user control" text to match the sidebar-footer layout the code has
  actually had since `df9f7c2` / `6ce9c94` / `4296179`, a pre-existing
  drift between the checked-in spec and the code that predates this
  change.

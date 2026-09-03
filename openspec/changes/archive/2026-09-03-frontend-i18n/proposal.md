## Why

The web client currently renders every user-facing string as a hard-coded
English literal. There is no infrastructure to show the app in another
language. We want a German-speaking visitor to see the app in German without
any manual step, while keeping English as the default for everyone else, and
without blocking on account infrastructure that does not exist yet (there is
no user-settings surface at all today).

## What Changes

- Add client-side i18n via `i18next` + `react-i18next`, with
  `i18next-browser-languagedetector` reading the browser's language
  (`navigator.language`, e.g. `Accept-Language`-derived) to pick the active
  language on load. No persistence, no manual switcher yet — detection runs
  fresh every load, purely from the browser.
- Ship two languages: **English (default/fallback)** and **German**. Any
  `de-*` browser locale (e.g. `de-AT`, `de-CH`) resolves to German; every
  other locale (including untranslated browser languages) falls back to
  English.
- Extract every user-facing string currently hard-coded in `Sidebar.tsx`,
  `TopBar.tsx`, `SidebarUser.tsx`, `ThemeSwitch.tsx`, `routes/login.tsx`, and
  `Placeholder.tsx` into translation resources (`src/i18n/locales/en.json`,
  `src/i18n/locales/de.json`), and replace them with `useTranslation()` /
  `t()` calls. The "Family Finances" wordmark/brand name itself is not
  translated.
- Keep `<html lang="…">` in sync with the resolved language at runtime (it
  starts as the static `lang="en"` in `index.html` before JS runs).
- No account-level language preference yet — that is explicitly deferred to a
  future change once a user-settings surface exists. This change only wires
  up the mechanism and the browser-detected default.

No breaking changes. Purely client-side and additive; the static-export build
contract (`pnpm build` → `frontend/out/`) is unaffected.

## Capabilities

### New Capabilities

- `web-client-i18n`: the client's internationalization mechanism — supported
  languages, browser-based detection, fallback behavior, and the requirement
  that user-facing UI text goes through translation resources rather than
  hard-coded literals.

### Modified Capabilities

- `web-client-shell`: the layout components (`Sidebar`, `TopBar`,
  `SidebarUser`, `ThemeSwitch`) render their labels through the i18n
  mechanism instead of literal English strings.

## Impact

- **Dependencies**: `frontend/package.json` gains `i18next`, `react-i18next`,
  `i18next-browser-languagedetector`.
- **Code**:
  - New `frontend/src/i18n/` — `index.ts` (i18next init/config),
    `locales/en.json`, `locales/de.json`.
  - `frontend/src/main.tsx` — import/initialize `src/i18n` before render.
  - `frontend/src/routes/__root.tsx` — sync `document.documentElement.lang`
    with the resolved language.
  - `frontend/src/components/Sidebar.tsx`, `TopBar.tsx`, `SidebarUser.tsx`,
    `ThemeSwitch.tsx`, `frontend/src/routes/login.tsx`,
    `frontend/src/components/Placeholder.tsx` — replace literal strings with
    `t()` calls.
- **Build / CI**: no change to the build/lint gate; `pnpm lint` and
  `pnpm build` remain sufficient. JSON translation files are plain data, not
  generated/checked artifacts like `schema.d.ts`.
- **Spec**: new `openspec/specs/web-client-i18n/spec.md`; delta on
  `openspec/specs/web-client-shell/spec.md`.

## Context

The web client is a client-only Vite + React 19 SPA (no SSR, no server
features — `frontend/AGENTS.md`). Every string today is a literal in JSX:
nav labels in `Sidebar.tsx`, the collapse/expand/open/close labels in
`TopBar.tsx`, "Log in" / "Log out" in `SidebarUser.tsx`, the System/Light/Dark
labels in `ThemeSwitch.tsx`, the whole sign-in form copy in `routes/login.tsx`,
and the placeholder home page copy in `Placeholder.tsx`. There is no
user-settings surface of any kind yet (no account preferences page), so a
per-user persisted language choice is out of scope for this change — the
`AuthProvider`/`User` model has no such field, and adding one is a backend
change in its own right.

Constraints:

- **Static, client-only build.** No server can inspect `Accept-Language` and
  render localized HTML; resolution must happen in the browser after JS
  loads, same as the existing theme mechanism (`src/lib/theme.tsx`).
- **`index.html` is static.** `<html lang="en">` is baked in at build time;
  it can only be corrected at runtime, mirroring how the pre-paint theme
  script in `index.html` is later kept in sync by `ThemeProvider`.
- Only two languages are needed right now: English and German.

## Goals / Non-Goals

**Goals:**

- On first load, the app renders in German for a browser reporting a
  `de*` language, and in English for everyone else (including
  unsupported/unset browser languages).
- All user-facing strings in the components touched by this change go through
  a translation layer, not literals, so adding a string in English requires
  adding it in German too (enforced by review, not tooling, at this scale).
- The mechanism is easy to extend later with a persisted, account-level
  language preference, without a rewrite.

**Non-Goals:**

- A manual language switcher in the UI. Explicitly deferred — the task is
  browser-detection only "for now."
- Persisting the detected/chosen language anywhere (`localStorage`, cookie,
  or backend). Every load re-detects from the browser.
- A third language, pluralization edge cases beyond what `i18next` gives for
  free, or translating content that doesn't exist yet (there's no real data
  UI beyond the placeholder).
- Translating the backend (error messages from the API, emails) — out of
  scope for this frontend-only change.

## Decisions

### Decision: `i18next` + `react-i18next` + `i18next-browser-languagedetector`

This is the standard React i18n stack: `react-i18next` gives a `useTranslation()`
hook / `t()` function and a `<Trans>` component for strings with embedded
markup (needed for the login page's "sent to **email**" line);
`i18next-browser-languagedetector` implements exactly the "read the browser,
pick a supported language, fall back to default" behavior this change needs,
without hand-rolling `Accept-Language`/`navigator.language` parsing and
locale-matching (e.g. `de-AT` → `de`).

*Alternative considered:* hand-roll a `~20`-line context (a static
`Record<Lang, Record<Key, string>>`, `navigator.language.startsWith("de")`,
a `useT()` hook), matching the repo's general lean-dependency posture. Rejected: even at two languages, `t()` interpolation
(`{{email}}`) and the embedded-markup case in the login page are exactly what
`react-i18next` is for, and reaching for it now avoids re-deriving detection
edge cases (region variants, `Accept-Language` quality values) that a hand
roll would get wrong first.

### Decision: Detection order is browser-only, no caching

`i18next-browser-languagedetector` is configured with
`order: ["navigator"]` and `caches: []`. No `localStorage`/cookie caching:
every page load re-derives the language fresh from `navigator.language` /
`navigator.languages`. This matches the task's "use the information from the
browser for now" — a future per-user setting will own persistence once it
exists (likely via the account/session, not `localStorage`, since language
should follow the person, not the device) and can be added as a higher-priority detector entry ahead of `navigator`
without touching anything else.

*Alternative considered:* cache the detected language in `localStorage`
(`ff:lang`, mirroring `ff:theme`/`ff:sidebar-collapsed`). Rejected for now:
it would let a visitor's language get "stuck" out of sync with a later OS/
browser language change, and a future account-level preference is a better,
single source of truth than reconciling two persisted values later.

### Decision: `supportedLngs: ["en", "de"]`, `nonExplicitSupportedLngs: true`, `fallbackLng: "en"`

`nonExplicitSupportedLngs` makes a browser-reported `de-AT` or `de-CH` match
the `de` resource bundle instead of falling back to English. Anything else
(`fr`, `es`, unset) falls back to `en`.

### Decision: One flat `translation` namespace, JSON per language

At this app's current size (a handful of components), a single
`src/i18n/locales/{en,de}.json` file per language with nested keys
(`sidebar.*`, `topbar.*`, `user.*`, `theme.*`, `login.*`, `home.*`) is enough.
No per-feature namespace splitting yet — that's easy to introduce later if the
resource files grow unwieldy, and premature now.

### Decision: Keep the "Family Finances" brand name untranslated

The wordmark/app name (`Sidebar.tsx`'s "Family Finances", `Icon.tsx`'s
default `title`, `index.html`'s `<title>`) is a proper noun / brand, not
translated content — consistent with how the German UI still says "iPhone"
or "GitHub." Only descriptive copy (taglines, button labels, form text) is
translated.

### Decision: Sync `<html lang>` at runtime, in `__root.tsx`

`index.html` ships static `lang="en"`. Once `i18next` resolves the actual
language, an effect in `__root.tsx` (alongside the existing sidebar/mobile
effects) sets `document.documentElement.lang = i18n.resolvedLanguage`, and
re-runs on `i18next`'s `languageChanged` event. This is the same
"static-shell-corrected-after-JS-runs" pattern already used for the theme
class in `src/lib/theme.tsx`.

## Risks / Trade-offs

- **Translation drift.** Nothing currently fails CI if `de.json` is missing a
  key that `en.json` has (`i18next` just falls back to the key name or
  English at runtime). Accepted for this change's scope — flag as a follow-up
  if the resource files grow; not worth a custom lint rule for two files.
- **New dependencies in a lean frontend.** Same trade-off the project already
  accepted for `next-themes` (per the archived
  `2026-09-02-selectable-color-theme` change): a small, purpose-built library
  beats hand-rolling detection/interpolation/markup-in-translation edge
  cases correctly.
- **No manual override yet.** A visitor whose browser reports German but who
  wants English (or vice versa) has no in-app way to change it until the
  future per-user setting ships. Acceptable and explicit per the task.

## Migration Plan

1. `cd frontend && pnpm add i18next react-i18next i18next-browser-languagedetector`.
2. Add `src/i18n/index.ts` (init) and `src/i18n/locales/{en,de}.json`.
3. Import `./i18n` at the top of `src/main.tsx` (before the router renders).
4. Extract strings from `Sidebar.tsx`, `TopBar.tsx`, `SidebarUser.tsx`,
   `ThemeSwitch.tsx`, `routes/login.tsx`, `Placeholder.tsx` into the JSON
   resources; replace with `useTranslation()`/`t()`/`<Trans>`.
5. Add the `<html lang>` sync effect to `__root.tsx`.
6. `pnpm lint` and `pnpm build`; manually verify in a browser with the
   devtools language override set to `de` / `de-AT` / `en` / `fr`.

Rollback: revert the commit and `pnpm remove` the three packages. No
persisted state to clean up (nothing is cached anywhere).

## Open Questions

None — scope is deliberately narrow (browser detection only, two languages,
no switcher, no persistence).

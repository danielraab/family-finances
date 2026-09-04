## MODIFIED Requirements

### Requirement: Supported languages and browser-based detection

The web client SHALL support two languages — **English** and **German** —
with English as the default and fallback language. On load, the client SHALL
determine the active language from the browser's reported language
(`navigator.language` / `navigator.languages`), with no persisted
browser-side override (no `localStorage` or cookie caching) — detection is
re-derived from the browser on every load.

For an authenticated visitor, an explicit account-level language preference
(set via `/settings`, per `user-settings`) SHALL take priority over browser
detection once it is known: when `GET /api/auth/me`'s `language` field is
non-`null`, the client SHALL apply it, overriding whatever the browser
detector resolved. An authenticated visitor with no preference set
(`language: null`) SHALL still be resolved by browser detection, exactly as
an anonymous visitor is. Because resolving the account preference requires an
async request, a visitor whose browser and account language disagree MAY see
a brief flash of the browser-detected language before the account preference
applies; this is accepted and not solved by pre-paint means (unlike the
colour theme, whose value lives in `localStorage`).

Any browser language beginning with `de` (e.g. `de`, `de-DE`, `de-AT`,
`de-CH`) SHALL resolve to German. Any other browser language, or the absence
of a supported match, SHALL resolve to English.

#### Scenario: German browser language

- **WHEN** a visitor's browser reports a language of `de`, `de-DE`, `de-AT`,
  or `de-CH`
- **THEN** the client renders its UI text in German

#### Scenario: Unsupported or missing browser language falls back to English

- **WHEN** a visitor's browser reports a language other than `de*` (including
  no supported language at all)
- **THEN** the client renders its UI text in English

#### Scenario: No persistence across loads for browser detection

- **WHEN** a visitor's browser language changes between two page loads and
  they are anonymous, or authenticated with no account-level language
  preference set
- **THEN** the client's rendered language reflects the browser language at
  each load independently, with nothing cached from a previous visit

#### Scenario: Account preference overrides browser detection

- **WHEN** an authenticated visitor has an account-level language preference
  of `de` and their browser reports English
- **THEN** the client renders in German

#### Scenario: No account preference falls back to browser detection

- **WHEN** an authenticated visitor has no account-level language preference
  set
- **THEN** the client's rendered language is resolved from the browser, the
  same as for an anonymous visitor

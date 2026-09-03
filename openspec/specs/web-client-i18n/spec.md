# web-client-i18n Specification

## Purpose

The web client's internationalization mechanism: which languages are
supported, how the active language is detected, the fallback behavior, and
the requirement that user-facing text is sourced from translation resources
rather than hard-coded literals.

## Requirements

### Requirement: Supported languages and browser-based detection

The web client SHALL support two languages — **English** and **German** —
with English as the default and fallback language. On load, the client SHALL
determine the active language from the browser's reported language
(`navigator.language` / `navigator.languages`), with no persisted override
(no `localStorage`, cookie, or account setting) — the language is re-derived
from the browser on every load.

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

#### Scenario: No persistence across loads

- **WHEN** a visitor's browser language changes between two page loads
- **THEN** the client's rendered language reflects the browser language at
  each load independently, with nothing cached from a previous visit

### Requirement: User-facing text goes through translation resources

Components under the web client's shell and auth flow (sidebar navigation,
top bar controls, the account/sign-in control, the colour-theme control, the
login page, and the home placeholder) SHALL render their user-facing text
through the i18n translation mechanism rather than hard-coded string
literals, so each string has both an English and a German resource entry.
The application's brand name ("Family Finances") is exempt and MAY remain a
literal, untranslated proper noun.

The document's `lang` attribute SHALL reflect the resolved language at
runtime, even though the static HTML shell ships a fixed default.

#### Scenario: Document language attribute matches resolved language

- **WHEN** the client resolves German as the active language
- **THEN** the root `<html>` element's `lang` attribute is updated to reflect
  German after the app mounts

#### Scenario: Missing translation falls back to English

- **WHEN** a translation key has no German resource entry
- **THEN** the client renders the English resource for that key rather than
  a blank string or a raw key

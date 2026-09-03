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

Any user-facing text rendered by the web client — including but not limited
to sidebar navigation, top bar controls, the account/sign-in control, the
colour-theme control, the login page, and the home placeholder — SHALL be
sourced from the i18n translation resources rather than a hard-coded string
literal. This applies to every new hardcoded label, title, `aria-label`, or
`title` attribute added to the client, not only to the strings that existed
when i18n was introduced. The application's brand name ("Family Finances")
is exempt and MAY remain a literal, untranslated proper noun.

`frontend/src/i18n/locales/en.json` SHALL be the source of truth for
translation keys: a key SHALL exist in `en.json` before or at the same time
it is added to any other locale file. Other locale files MAY lag `en.json`
temporarily (see the "Translation coverage is reported, not enforced"
requirement) but SHALL NOT contain a key absent from `en.json` without it
also existing in `en.json`.

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

#### Scenario: New hardcoded strings are non-compliant

- **WHEN** a new component adds a literal English string for user-facing text
  instead of a translation key
- **THEN** it violates this requirement, regardless of whether the string
  also happens to have no German equivalent yet

#### Scenario: A key is added to English first

- **WHEN** a new translation key is introduced for a new piece of UI text
- **THEN** it is added to `en.json`, with the German (or any other locale's)
  entry added at the same time or in a later change

### Requirement: Translation coverage is reported, not enforced

The project SHALL make translation coverage of each non-English locale file
against `frontend/src/i18n/locales/en.json` visible, without using it as a
merge gate. Coverage for a given locale SHALL be defined as the percentage
of `en.json`'s translation keys (flattened to dot-paths) that also exist in
that locale's file, regardless of whether the corresponding value is
non-empty or otherwise "high quality." Locale keys absent from `en.json`
SHALL also be surfaced (as "extra"/orphaned keys), separately from the
coverage percentage.

This reporting SHALL NOT block merging a pull request or block, delay, or
gate any other continuous-integration check, and SHALL NOT be relied upon as
a required branch-protection check.

#### Scenario: Fully covered locale

- **WHEN** every key in `en.json` also exists in `de.json`
- **THEN** the reported coverage for German is 100%

#### Scenario: Partially covered locale does not block CI

- **WHEN** `de.json` is missing one or more keys present in `en.json`
- **THEN** the reported coverage for German is below 100%
- **AND** this alone does not fail the overall CI run or block merging the
  pull request

#### Scenario: Orphaned keys are surfaced

- **WHEN** `de.json` contains a key that does not exist in `en.json`
- **THEN** the report identifies it as an extra/orphaned key, separately
  from the coverage percentage

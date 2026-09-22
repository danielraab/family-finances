# Proposal

## Why

The `/entries` ledger spreads account, category, and tags across three adjacent
columns, leaving limited width for each entry's title and its supporting
information. Grouping these related attributes creates a clearer row hierarchy
and gives the title more room without changing the ledger's data or actions.

## What Changes

- Replace the separate Account, Category, and Tags ledger columns with one
  localized **Details** column positioned between Date and Title.
- Stack an entry's account, category, and tag pills vertically in that column,
  in that order, while allowing its tags to wrap within their group.
- Keep existing account/category icons, shared-owner indicators, shared-tag
  tooltips, and unavailable-entity fallback treatment intact.
- Omit category and tags from the stack when the entry has none; the account
  remains the leading detail.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `web-client-entries`: The `/entries` ledger gains a Details column that
  groups account, category, and tags between Date and Title.

## Impact

- `frontend/src/routes/entries.index.tsx` — table header order and per-row
  metadata layout.
- `frontend/src/i18n/locales/{en,de}.json` — translated Details column header.
- `openspec/specs/web-client-entries/spec.md` — ledger presentation
  requirement and scenarios.
- No backend, API contract, generated types, dependencies, or stored-data
  changes.

## Why

`frontend-i18n` (archived 2026-09-03) wired up `i18next` with two languages
and extracted every existing string, but left two things informal: nothing
in the repo says a *new* hardcoded label must go through i18n instead of
being a literal, and nothing checks whether `de.json` (or any future
non-English locale file) actually keeps up with `en.json` as the app grows.
Today that's fine at 43 keys reviewed by eye; it won't stay fine. We want the
rule written down, and a lightweight, always-visible signal — not a merge
gate — when a locale file falls behind.

## What Changes

- Strengthen `frontend/AGENTS.md`'s i18n section from a "should" into an
  explicit MUST: every hardcoded user-facing string (labels, titles,
  `aria-label`/`title` text, etc.) SHALL go through the i18n mechanism,
  except the "Family Finances" brand name. `en.json` is the source of truth
  — a key SHALL exist there before/when it exists in any other locale file.
  Documentation only; no new lint tooling enforces this mechanically.
- Add an `i18n-coverage` job to `.github/workflows/ci.yml`: a small Node
  script flattens `en.json` and every other `frontend/src/i18n/locales/*.json`
  file, and for each non-English locale reports how many of `en.json`'s keys
  it has (coverage %) and how many keys it has that `en.json` doesn't
  (orphaned/stale keys).
- The job posts the same table two places: the GitHub Actions job summary
  (every run) and an upserted PR comment (pull-request runs only, identified
  by a hidden marker so re-pushes update one comment instead of piling up
  new ones).
- The job is **non-blocking**: `continue-on-error: true`, no `needs:` from or
  to the existing `frontend` / `contract` / `backend` jobs, so a locale
  falling behind never fails CI or delays the other checks. It is
  intentionally not something we'd add to required branch-protection checks.

No breaking changes. Purely additive: one new CI job plus a documentation
edit.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `web-client-i18n`: the "User-facing text goes through translation
  resources" requirement becomes a hard MUST with `en.json` as the explicit
  source of truth for keys.
- `release-pipeline`: CI gains a non-blocking translation-coverage report
  (job summary + PR comment) alongside the existing lint/test/contract
  checks.

## Impact

- **Code**: `.github/workflows/ci.yml` gains an `i18n-coverage` job; a new
  script (e.g. `frontend/scripts/i18n-coverage.mjs`) computes the report
  (plain Node, no new dependency — reuses the two committed JSON files).
- **Docs**: `frontend/AGENTS.md`'s i18n section is reworded (MUST language,
  source-of-truth rule).
- **CI**: one new job, running in parallel with the existing three; needs
  `permissions: pull-requests: write` scoped to that job only (the workflow's
  top-level `permissions` stays `contents: read`).
- **Spec**: deltas on `openspec/specs/web-client-i18n/spec.md` and
  `openspec/specs/release-pipeline/spec.md`.

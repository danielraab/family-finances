## Context

`frontend-i18n` established `src/i18n/locales/{en,de}.json` as flat,
same-shaped `translation` resources, with `en` as `fallbackLng`. Its design
doc explicitly accepted drift risk: "nothing currently fails CI if `de.json`
is missing a key... not worth a custom lint rule for two files." That
trade-off was reasonable at the time (2 files, 43 keys, one contributor) but
was always a "revisit later" item, not a permanent stance — this change is
that revisit.

`.github/workflows/ci.yml` today has three jobs — `frontend` (Biome lint +
`pnpm build`), `contract` (OpenAPI drift check, fails the build), and
`backend` — none `needs:` each other for anything relevant here, and the
workflow's top-level `permissions: contents: read` is intentionally minimal.

Constraints:

- The user was explicit: doc-only enforcement (no new lint tooling that
  greps JSX for string literals), a non-blocking CI check, and the report
  must land in **both** the job summary and a PR comment.
- No third-party GitHub Actions are used anywhere in this workflow today
  (only `actions/*` and `pnpm/action-setup`) — introducing one is a real
  supply-chain trust decision, not a drop-in.

## Goals / Non-Goals

**Goals:**

- `frontend/AGENTS.md` states plainly that new hardcoded strings are a
  violation, and that `en.json` is the canonical key set.
- Every CI run (push or PR) shows translation coverage per non-English
  locale in the job summary.
- Every PR additionally gets a single, self-updating comment with the same
  table.
- A locale falling behind is visible (the check can show as failed) but
  never blocks merging, and never blocks or delays the `frontend` /
  `contract` / `backend` jobs.
- The mechanism generalizes to a third language later with no script
  changes — it discovers locale files rather than hardcoding `de.json`.

**Non-Goals:**

- Enforcing i18n usage mechanically (no custom Biome rule / no-hardcoded-jsx-text
  lint). Explicitly out of scope per the user.
- Blocking merges on coverage, or adding this job to required branch-protection
  checks (that's a GitHub repo-settings action outside this change's diff
  anyway, but worth naming as a non-goal).
- Value-level translation quality (a key existing with a nonsense or
  machine-translated value still counts as "covered"). Only key
  presence/absence is measured.
- Per-key deep diffing across nested structures beyond simple presence —
  since resources are already a flat single namespace per
  `frontend-i18n`'s design, this is naturally in scope for JSON of arbitrary
  nesting depth (dot-path flattening), not a limitation to design around.

## Decisions

### Decision: Coverage means key *presence*, not value quality

A locale's coverage % = `(keys in en.json also present in that locale) /
(total keys in en.json) * 100`, computed on flattened dot-paths (e.g.
`theme.switchTo`). An empty-string value still counts as "present" — this
mirrors how `i18next` itself behaves (a missing key falls back to English at
runtime; an empty-but-present key does not). Value-quality checking (empty
strings, obviously-untranslated copies of the English string) is a
plausible future refinement, not this change's job.

*Alternative considered:* require non-empty values to count as covered.
Rejected for v1 — it's an easy game to add later if empty-placeholder keys
turn out to be a real problem, but guessing at "is this value good" (e.g. a
value byte-identical to English, which is sometimes correct — "OK" is "OK"
in German too) adds false positives for no evidenced need yet.

### Decision: Also report orphaned keys (present in a locale, absent from `en.json`)

Same script, same flattening — reporting `Extra` alongside `Missing` costs
nothing extra and catches the real, distinct failure mode of a key renamed
or removed in `en.json` and forgotten elsewhere. Shown as an extra column,
not folded into the coverage percentage (coverage is specifically "how much
of English is translated").

### Decision: Locale discovery, not a hardcoded `de.json` reference

The script globs `frontend/src/i18n/locales/*.json`, treats `en.json` as the
baseline, and reports on every other file found. Adding a third language
later (per `frontend-i18n`'s design doc, e.g. once there's demand) means
dropping in `fr.json` — the CI report picks it up with no script edit.

### Decision: `continue-on-error: true` at the job level, real non-zero exit inside it

The report script's process exit code is non-zero when any locale is below
100% coverage (a meaningful, visible "failure"), but the job carries
`continue-on-error: true` so that failure never flips the overall workflow
run to red and — since nothing `needs:` this job and it `needs:` nothing —
never delays or gates `frontend`, `contract`, or `backend`. This matches
"allowed to fail, must not block merging or further execution" precisely:
GitHub shows the check with a ⚠️ neutral/skipped-style outcome rather than a
blocking ❌, and it is simply never added to branch-protection's required
checks.

*Alternative considered:* always exit 0, treat the percentage as pure
information with no pass/fail semantics at all. Rejected: a job that can
never show as failing gives up a real, free signal (an "N/A, always green"
check is easy to stop noticing entirely) for no gain, given
`continue-on-error` already neutralizes the blocking concern.

### Decision: `actions/github-script` for the PR comment, not a third-party sticky-comment Action

`actions/github-script@v7` is first-party, already the same trust tier as
every other Action this workflow uses, and the upsert-by-marker pattern
(list comments, find one containing `<!-- i18n-coverage-report -->`, PATCH
it if found else POST a new one) is ~15 lines of inline script — no need for
`marocchino/sticky-pull-request-comment` or similar. Runs only when
`github.event_name == 'pull_request'` (a plain push to a branch has no PR to
comment on; it still gets the job summary).

*Alternative considered:* a third-party sticky-comment Action. Rejected —
it's the only third-party Action anywhere in this workflow, for a problem
`actions/github-script` already solves in a few lines.

### Decision: Plain `node`, no `pnpm install`

The script only reads two-or-more JSON files already checked into the repo
and does string/object manipulation — no dependency on anything in
`package.json`. Skipping `pnpm install`/`actions/setup-node`'s cache
machinery keeps this job fast and structurally independent of the `frontend`
job (which does need the full install for `pnpm build`). Only
`actions/setup-node` (no cache) is needed to get a `node` binary at all,
unless the runner's default toolcache already includes one recent enough —
confirm the pinned Node major at implementation time.

### Decision: `permissions: pull-requests: write` scoped to the job, not the workflow

The workflow's top-level `permissions: contents: read` stays untouched;
`i18n-coverage` declares its own job-level `permissions:` block adding
`pull-requests: write` (needed to list/create/update a PR comment), which
GitHub Actions honors as a job-level override. No other job's token
capabilities change.

## Risks / Trade-offs

- **Doc-only enforcement can be ignored.** Nothing stops a PR from adding a
  literal string; this relies on review (human or Claude) to catch it,
  same as any other AGENTS.md rule in this repo. Accepted per the user's
  explicit choice.
- **`continue-on-error` hides real failures from the top-level workflow
  status.** A contributor who only glances at the green checkmark won't see
  a coverage regression; they have to open the job or read the PR comment.
  This is the intended trade-off (visible, not blocking) — worth restating
  if it turns out coverage regressions go unnoticed in practice.
- **PR comment noise.** Mitigated by upserting a single marked comment
  rather than posting a new one per push.
- **`pull-requests: write` is a real permission grant**, even if scoped to
  one job. Worth a reviewer's eyes specifically on that line.

## Migration Plan

1. Add `frontend/scripts/i18n-coverage.mjs` (or `.js`) — flattens JSON,
   computes per-locale coverage/missing/extra, prints a Markdown table,
   writes it to `process.env.GITHUB_STEP_SUMMARY`, exits non-zero if any
   locale is below 100%.
2. Add the `i18n-coverage` job to `.github/workflows/ci.yml`: checkout,
   Node setup, run the script with `continue-on-error: true`; a follow-up
   step (guarded on `pull_request`) runs `actions/github-script` to
   upsert the PR comment from the same table (either re-derive it or have
   the first step also emit the table to a file/output the script step can
   pass along).
3. Reword `frontend/AGENTS.md`'s i18n section per the Why/What above.
4. Verify: intentionally remove a key from `de.json` locally, run the
   script, confirm it reports <100% and exits non-zero; run the workflow
   (or `act`/manual dry run if available) to confirm the job shows
   ⚠️ / non-blocking and the summary/comment render correctly; restore the
   key.

Rollback: revert the commit. No persisted state; the removed CI job and doc
wording simply disappear.

## Open Questions

- Exact Markdown table styling / column order for the summary vs. comment —
  minor, settled at implementation (same table both places is the only hard
  requirement).
- Whether to pin a specific `actions/setup-node` Node version or rely on the
  runner default — settled at implementation by checking what's already
  pinned elsewhere in `ci.yml` (currently `node-version: 22` in the
  `frontend`/`contract` jobs) for consistency.

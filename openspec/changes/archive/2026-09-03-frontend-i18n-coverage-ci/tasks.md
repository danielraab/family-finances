## 1. Coverage script

- [x] 1.1 Add `frontend/scripts/i18n-coverage.mjs`: read
  `frontend/src/i18n/locales/en.json` as the baseline; glob every sibling
  `*.json` in that directory (excluding `en.json`) as the locales to report
  on.
- [x] 1.2 Flatten each JSON file to dot-path keys (e.g. `theme.switchTo`).
  For each non-English locale compute: total English keys, keys present
  (coverage count), keys missing (present in `en.json`, absent here), keys
  extra (present here, absent from `en.json`). Coverage % = present /
  English-total * 100, one decimal place.
- [x] 1.3 Render a single Markdown table (language, keys covered / total,
  coverage %, extra-key count) usable both for the job summary and the PR
  comment body (wrap the PR-comment version with a
  `<!-- i18n-coverage-report -->` marker so it can be found/updated later).
  → Also added a `<details>` block per locale listing actual missing/extra
  key names (collapsed, only rendered when non-empty) — the table alone
  gives the count but not what to fix.
- [x] 1.4 Write the table to `process.env.GITHUB_STEP_SUMMARY` when set
  (append, don't overwrite anything already there). Exit with a non-zero
  status if any locale's coverage is below 100%; print the table to stdout
  regardless (so local runs — `node frontend/scripts/i18n-coverage.mjs` —
  are useful without CI env vars).
  → Also writes the same report to `$RUNNER_TEMP/i18n-coverage-report.md`
  (falling back to the OS temp dir when `RUNNER_TEMP` is unset, e.g.
  locally) for the CI job's PR-comment step to read.

## 2. CI job

- [x] 2.1 Add an `i18n-coverage` job to `.github/workflows/ci.yml`,
  independent of `frontend` / `contract` / `backend` (no `needs:` either
  direction), with `continue-on-error: true` and its own job-level
  `permissions:` block.
  → The block is `{ contents: read, pull-requests: write }`, not just
  `pull-requests: write` as originally worded in the design — a job-level
  `permissions:` block replaces the workflow default for that job entirely
  rather than adding to it, so `contents: read` has to be restated
  explicitly for `actions/checkout` to work. Workflow-level `permissions`
  (used by `frontend`/`contract`/`backend`, which have no job-level block)
  is untouched at `contents: read`. `publish`'s `needs: [frontend, backend,
  contract]` was also confirmed unchanged — this job isn't in it.
- [x] 2.2 Steps: `actions/checkout@v4`, `actions/setup-node@v4` (`node-version: 22`,
  matching the other jobs; no `pnpm` install — the script has no
  dependencies), run the coverage script.
- [x] 2.3 Add a step guarded on `github.event_name == 'pull_request'` using
  `actions/github-script@v7` to upsert the PR comment: list existing PR
  comments, find one containing the `<!-- i18n-coverage-report -->` marker,
  update it if found, otherwise create it.
  → Reads the report from `$RUNNER_TEMP/i18n-coverage-report.md` (written
  by the script in task 1.4) rather than re-rendering — simpler than
  plumbing a step output through.

## 3. Documentation

- [x] 3.1 Reword `frontend/AGENTS.md`'s "i18n" section: change the
  string-extraction guidance from a "should" to an explicit MUST (every
  hardcoded user-facing string goes through `t()`/`<Trans>`, "Family
  Finances" excepted), and state that `en.json` is the source of truth —
  other locale files are graded against it by the CI `i18n-coverage` job
  (non-blocking).

## 4. Verify

- [x] 4.1 Run `node frontend/scripts/i18n-coverage.mjs` locally against the
  current, fully-covered `en.json`/`de.json` — confirm 100% coverage, exit
  code 0, and a sane table.
  → 27/27 keys, 100.0%, exit 0.
- [x] 4.2 Temporarily delete one key from a local copy of `de.json`, re-run
  the script, confirm it reports the missing key and exits non-zero; restore
  the file.
  → Removed `login.useDifferentAddress` and added a bogus
  `sidebar.bogusExtraKey`: reported 26/27 (96.3%), listed both the missing
  and the extra key by name in the `<details>` blocks, exit 1. Restored from
  a backup copy; re-ran to confirm back to 100%/exit 0.
- [ ] 4.3 Push the branch / open a PR and confirm in the Actions UI: the
  `i18n-coverage` job runs in parallel with the other three, shows its
  actual pass/fail state without turning the overall run red, the job
  summary shows the table, and the PR gets exactly one coverage comment
  that updates in place on a follow-up push (not a new comment each time).
- [x] 4.4 `cd frontend && pnpm lint` still passes.
  → Passes (exit 0). Correction to the design/task wording: `frontend/scripts/`
  is actually *outside* Biome's `files.includes` scope (only
  `**/src/**/*`, `.vscode`, `index.html`, `vite.config.ts` are covered) —
  same as the pre-existing `generate-api.mjs`/`generate-favicon.mjs`, so
  this is consistent with the rest of `scripts/`, not a gap introduced here.

## 5. Spec sync

- [ ] 5.1 After implementation, fold the deltas into
  `openspec/specs/web-client-i18n/spec.md` and
  `openspec/specs/release-pipeline/spec.md` (`/opsx:archive` or manually, per
  how `frontend-i18n` was archived — the `openspec` CLI is not installed in
  this environment).

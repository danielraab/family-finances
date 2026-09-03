## 1. Coverage script

- [ ] 1.1 Add `frontend/scripts/i18n-coverage.mjs`: read
  `frontend/src/i18n/locales/en.json` as the baseline; glob every sibling
  `*.json` in that directory (excluding `en.json`) as the locales to report
  on.
- [ ] 1.2 Flatten each JSON file to dot-path keys (e.g. `theme.switchTo`).
  For each non-English locale compute: total English keys, keys present
  (coverage count), keys missing (present in `en.json`, absent here), keys
  extra (present here, absent from `en.json`). Coverage % = present /
  English-total * 100, one decimal place.
- [ ] 1.3 Render a single Markdown table (language, keys covered / total,
  coverage %, extra-key count) usable both for the job summary and the PR
  comment body (wrap the PR-comment version with a
  `<!-- i18n-coverage-report -->` marker so it can be found/updated later).
- [ ] 1.4 Write the table to `process.env.GITHUB_STEP_SUMMARY` when set
  (append, don't overwrite anything already there). Exit with a non-zero
  status if any locale's coverage is below 100%; print the table to stdout
  regardless (so local runs — `node frontend/scripts/i18n-coverage.mjs` —
  are useful without CI env vars).

## 2. CI job

- [ ] 2.1 Add an `i18n-coverage` job to `.github/workflows/ci.yml`,
  independent of `frontend` / `contract` / `backend` (no `needs:` either
  direction), with `continue-on-error: true` and its own
  `permissions: pull-requests: write` block (workflow-level `permissions`
  stays `contents: read`).
- [ ] 2.2 Steps: `actions/checkout@v4`, `actions/setup-node@v4` (`node-version: 22`,
  matching the other jobs; no `pnpm` install — the script has no
  dependencies), run the coverage script.
- [ ] 2.3 Add a step guarded on `github.event_name == 'pull_request'` using
  `actions/github-script@v7` to upsert the PR comment: list existing PR
  comments, find one containing the `<!-- i18n-coverage-report -->` marker,
  update it if found, otherwise create it. Reuse the table produced in 1.3
  (pass it via a step output or regenerate by re-running the script's
  table-rendering in the same job — pick whichever keeps the job simplest).

## 3. Documentation

- [ ] 3.1 Reword `frontend/AGENTS.md`'s "i18n" section: change the
  string-extraction guidance from a "should" to an explicit MUST (every
  hardcoded user-facing string goes through `t()`/`<Trans>`, "Family
  Finances" excepted), and state that `en.json` is the source of truth —
  other locale files are graded against it by the CI `i18n-coverage` job
  (non-blocking).

## 4. Verify

- [ ] 4.1 Run `node frontend/scripts/i18n-coverage.mjs` locally against the
  current, fully-covered `en.json`/`de.json` — confirm 100% coverage, exit
  code 0, and a sane table.
- [ ] 4.2 Temporarily delete one key from a local copy of `de.json`, re-run
  the script, confirm it reports the missing key and exits non-zero; restore
  the file.
- [ ] 4.3 Push the branch / open a PR and confirm in the Actions UI: the
  `i18n-coverage` job runs in parallel with the other three, shows its
  actual pass/fail state without turning the overall run red, the job
  summary shows the table, and the PR gets exactly one coverage comment
  that updates in place on a follow-up push (not a new comment each time).
- [ ] 4.4 `cd frontend && pnpm lint` still passes (the new script lives
  under `frontend/scripts/`, in Biome's lint scope).

## 5. Spec sync

- [ ] 5.1 After implementation, fold the deltas into
  `openspec/specs/web-client-i18n/spec.md` and
  `openspec/specs/release-pipeline/spec.md` (`/opsx:archive` or manually, per
  how `frontend-i18n` was archived — the `openspec` CLI is not installed in
  this environment).

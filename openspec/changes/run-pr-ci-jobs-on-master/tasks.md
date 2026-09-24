# Tasks

## 1. Extend CI coverage

- [x] 1.1 Run integration, translation coverage, and image validation for pull requests and pushes to `master`, while keeping ordinary branch pushes excluded; verify the workflow conditions.
- [x] 1.2 Keep translation-coverage PR commenting limited to pull requests; verify master runs do not execute comment steps.

## 2. Validate

- [x] 2.1 Validate the OpenSpec change and workflow YAML with `openspec validate run-pr-ci-jobs-on-master --strict` and a YAML parser.

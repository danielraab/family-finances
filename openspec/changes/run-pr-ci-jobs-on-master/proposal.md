# Proposal

## Why

Post-merge changes on `master` need the same confidence as pull requests.
Currently the integration, translation-coverage, and image-validation jobs run
only for pull requests.

## What Changes

- Run all pull-request-only CI jobs on pushes to `master` as well.
- Keep those jobs disabled for pushes to non-master branches.
- Preserve tag-only image publication.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-pipeline`: CI runs the integration, translation-coverage, and image
  validation jobs for pull requests and `master` pushes.

## Impact

- `.github/workflows/ci.yml` job conditions.

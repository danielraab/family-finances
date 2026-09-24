# Proposal

## Why

Pull requests currently validate the frontend, backend, and API contract, but
they do not validate the combined production container. Dockerfile failures can
therefore reach a release tag undetected.

## What Changes

- Build the combined application image for pull-request CI runs after its
  required checks pass.
- Keep pull-request image builds local to the runner: do not authenticate to a
  registry or push an image.
- Retain tag-triggered GHCR publishing and its version tags.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-pipeline`: CI validates the production image build on pull requests
  without publishing it.

## Impact

- `.github/workflows/ci.yml` gains a pull-request-only image-build job or
  equivalent conditional behavior.
- Pull-request CI consumes runner time for a Docker build but needs no package
  write permission or registry credentials.

# Design

## Context

The release workflow currently has a tag-only `publish` job that builds and
pushes the combined Docker image after the frontend, backend, and contract
jobs pass. Pull requests run those checks but do not exercise the root
`Dockerfile`.

## Goals / Non-Goals

**Goals:**
- Validate the same combined image build on pull requests.
- Preserve the tag publishing flow, including its tags and build metadata.
- Ensure untrusted pull-request code receives no registry credentials or write
  permissions.

**Non-Goals:**
- Load, export, scan, or deploy the pull-request image.
- Change image publication for branch pushes or tags.

## Decisions

### Add a dedicated pull-request build job

Add a job gated to `pull_request`, dependent on successful frontend, backend,
and contract jobs. It checks out the source, configures Buildx, and invokes the
Docker build action with `push: false`.

A separate job keeps the existing publishing permissions and registry login
strictly tag-only. Extending `publish` with event-dependent credentials and
push flags would combine trusted release behavior with untrusted PR behavior,
which is less clear and easier to misconfigure.

### Reuse release build arguments

Pass the pull request commit SHA as `REVISION`; leave `VERSION` empty because
the build is not a release. This validates the production build path without
assigning a release identity.

## Risks / Trade-offs

- [Longer pull-request CI time] → Run the build only after prerequisite checks
  pass, avoiding unnecessary container builds for failing changes.
- [Docker cache is not retained between runs] → Accept the runner-local build
  cost initially; add caching only if workflow timings demonstrate a need.

## Migration Plan

1. Merge the workflow change.
2. Confirm a pull request reports a successful image-build job with no package
   publication.
3. Confirm a tag push still publishes the tagged and `latest` images.

Rollback consists of removing the pull-request image-build job; the existing
tag publishing job remains unchanged.

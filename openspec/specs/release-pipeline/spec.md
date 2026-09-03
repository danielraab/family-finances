# release-pipeline Specification

## Purpose
TBD - created by archiving change go-serves-frontend-docker-ci. Update Purpose after archive.
## Requirements
### Requirement: Single container image combines frontend and backend

A container image build process SHALL produce one image containing the
compiled backend with the frontend's static export embedded, requiring no
additional files or companion containers at runtime.

#### Scenario: Image runs standalone

- **WHEN** the built container image is run with no other services attached
- **THEN** it serves both the frontend site and its `/api/` routes on its
  configured port

### Requirement: CI lints and tests both packages on every push and pull request

Continuous integration SHALL run the frontend's lint and static-build checks,
and the backend's format, vet, and test checks, on every push and every pull
request, independent of whether an image is published for that run.

Continuous integration SHALL also run a contract check on every push and every
pull request: it SHALL lint `openapi/openapi.yaml`, regenerate the frontend's
committed API types from it, re-sync the committed `backend/openapi.yaml` copy
from it, and fail if either generated artifact differs from what is committed.

#### Scenario: Frontend check failure blocks publishing

- **WHEN** the frontend lint check or static build fails
- **THEN** CI reports failure and does not build or push a container image

#### Scenario: Backend check failure blocks publishing

- **WHEN** the backend `gofmt`, `go vet`, or `go test` check fails
- **THEN** CI reports failure and does not build or push a container image

#### Scenario: Contract drift blocks the build

- **WHEN** `openapi/openapi.yaml` changed but the committed
  `frontend/src/api/schema.d.ts` or `backend/openapi.yaml` was not regenerated to
  match
- **THEN** the contract check reports failure and no container image is built or
  pushed

#### Scenario: Malformed OpenAPI document blocks the build

- **WHEN** `openapi/openapi.yaml` does not lint clean
- **THEN** the contract check reports failure

#### Scenario: Pull requests do not publish

- **WHEN** CI runs for a pull request
- **THEN** no container image is built or pushed to the registry

#### Scenario: Ordinary branch pushes do not publish

- **WHEN** CI runs for a push that is not a git tag
- **THEN** no container image is built or pushed to the registry

### Requirement: Pushing a git tag publishes an image tagged with that tag

WHEN all checks pass for a push of a git tag, CI SHALL build the container
image and push it to the GitHub Container Registry tagged with that git
tag's name, plus `latest`.

#### Scenario: Tag push publishes

- **WHEN** a git tag is pushed and the frontend and backend checks pass
- **THEN** CI builds the container image and pushes it to the registry
  tagged with the pushed tag's name and with `latest`

#### Scenario: Failing checks block a tag publish

- **WHEN** a git tag is pushed but the frontend or backend checks fail
- **THEN** CI reports failure and does not build or push a container image

### Requirement: CI reports non-blocking translation coverage

Continuous integration SHALL run an `i18n-coverage` job on every push and
every pull request, independent of (not gated by, and not gating) the
frontend lint/build check, the contract check, and the backend checks. The
job SHALL compute, for each non-English locale file under
`frontend/src/i18n/locales/`, the translation-coverage report defined by the
`web-client-i18n` capability's "Translation coverage is reported, not
enforced" requirement.

The job SHALL publish the report to the GitHub Actions job summary on every
run, and SHALL additionally publish it as a single pull-request comment on
pull-request runs — updating that same comment in place on subsequent pushes
to the pull request rather than posting a new comment each time.

This job's outcome SHALL NOT block the build or push of a container image,
SHALL NOT block or delay any other CI job, and SHALL NOT be required for a
pull request to be merged.

#### Scenario: Coverage report runs alongside other checks

- **WHEN** CI runs for a push or a pull request
- **THEN** the `i18n-coverage` job runs and completes independently of the
  frontend, contract, and backend jobs, with no dependency in either
  direction

#### Scenario: Job summary always shows the report

- **WHEN** the `i18n-coverage` job completes, on a push or a pull request
- **THEN** the GitHub Actions job summary for that run contains the
  translation-coverage table

#### Scenario: Pull requests get a self-updating comment

- **WHEN** the `i18n-coverage` job runs for a pull request, on the first push
  and on a subsequent push
- **THEN** the pull request has exactly one coverage comment, whose content
  reflects the latest run

#### Scenario: A coverage shortfall does not block anything

- **WHEN** a non-English locale is below 100% coverage
- **THEN** the `i18n-coverage` job may report a failed outcome for itself,
  but the pull request remains mergeable, no container image build is
  blocked, and no other CI job is delayed or skipped as a result


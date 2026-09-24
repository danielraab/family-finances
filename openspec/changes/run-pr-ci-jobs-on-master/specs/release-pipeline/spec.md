# Spec Delta

## MODIFIED Requirements

### Requirement: CI lints and tests both packages on every push and pull request

Continuous integration SHALL run the frontend's lint and static-build checks,
and the backend's format, vet, and test checks, on every push to `master` and
every pull request, independent of whether an image is published for that run.

Continuous integration SHALL also run a contract check on every push to
`master` and every pull request: it SHALL lint `openapi/openapi.yaml`,
regenerate the frontend's committed API types from it, re-sync the committed
`backend/openapi.yaml` copy from it, and fail if either generated artifact
differs from what is committed.

Continuous integration SHALL build the combined production container image on
every pull request and every push to `master` after the frontend, backend, and
contract checks succeed. These image builds SHALL NOT authenticate with or push
to a registry.

#### Scenario: Frontend check failure blocks publishing
- **WHEN** the frontend lint check or static build fails
- **THEN** CI reports failure and does not build or push a container image

#### Scenario: Backend check failure blocks publishing
- **WHEN** the backend `gofmt`, `go vet`, or `go test` check fails
- **THEN** CI reports failure and does not build or push a container image

#### Scenario: Contract drift blocks the build
- **WHEN** `openapi/openapi.yaml` changed but the committed generated artifacts do not match
- **THEN** the contract check reports failure and no container image is built or pushed

#### Scenario: Malformed OpenAPI document blocks the build
- **WHEN** `openapi/openapi.yaml` does not lint clean
- **THEN** the contract check reports failure

#### Scenario: Pull requests do not publish
- **WHEN** CI runs for a pull request and required checks pass
- **THEN** CI builds the combined production container image without authenticating with or pushing to a registry

#### Scenario: Master pushes do not publish
- **WHEN** CI runs for a push to `master` and required checks pass
- **THEN** CI builds the combined production container image without authenticating with or pushing to a registry

#### Scenario: Ordinary branch pushes do not publish
- **WHEN** CI runs for a push that is not to `master` or a git tag
- **THEN** no pull-request-only CI job runs and no container image is built or pushed to the registry

### Requirement: CI reports non-blocking translation coverage

Continuous integration SHALL run an `i18n-coverage` job on every pull request
and every push to `master`, independently of the frontend, contract, and
backend checks. The job SHALL compute the translation-coverage report defined
by the `web-client-i18n` capability.

The job SHALL publish the report to the GitHub Actions job summary on every
run, and SHALL additionally publish one self-updating comment on pull requests.
Its outcome SHALL NOT block image builds, other CI jobs, or pull-request merges.

#### Scenario: Coverage report runs alongside other checks
- **WHEN** CI runs for a pull request or a push to `master`
- **THEN** the `i18n-coverage` job runs independently of the frontend, contract, and backend jobs

#### Scenario: Job summary always shows the report
- **WHEN** the `i18n-coverage` job completes
- **THEN** the GitHub Actions job summary contains the translation-coverage table

#### Scenario: Pull requests get a self-updating comment
- **WHEN** the `i18n-coverage` job runs for a pull request
- **THEN** the pull request has exactly one coverage comment reflecting the latest run

#### Scenario: A coverage shortfall does not block anything
- **WHEN** a non-English locale is below 100% coverage
- **THEN** the job may fail without blocking a container image build or other CI job

## Purpose

Building a single deployable container image that combines the frontend and
backend, and running CI checks that gate publishing that image on both
packages passing lint and tests.

## ADDED Requirements

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

#### Scenario: Frontend check failure blocks publishing

- **WHEN** the frontend lint check or static build fails
- **THEN** CI reports failure and does not build or push a container image

#### Scenario: Backend check failure blocks publishing

- **WHEN** the backend `gofmt`, `go vet`, or `go test` check fails
- **THEN** CI reports failure and does not build or push a container image

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

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

### Requirement: Successful builds on the default branch publish a versioned image

WHEN all checks pass for a push to the repository's default branch, CI SHALL
build the container image and push it to the GitHub Container Registry
tagged `YYYYMMDD-N`, where `YYYYMMDD` is the current UTC date and `N` is a
counter starting at `1` for the first image published that day and
incrementing for each subsequent image published the same day. `N` SHALL be
computed from images already present in the registry, not from a build-run
counter, so it stays gap-free and collision-free across reruns.

#### Scenario: First image of the day

- **WHEN** no image tagged with today's date prefix exists in the registry
  yet and CI publishes
- **THEN** the new image is tagged `YYYYMMDD-1`

#### Scenario: Subsequent image the same day

- **WHEN** an image tagged `YYYYMMDD-N` already exists in the registry and CI
  publishes again the same day
- **THEN** the new image is tagged with `N` incremented by one

#### Scenario: Tag computation reflects the registry, not the run count

- **WHEN** CI has run more times than images have actually been published for
  the current date (e.g. an earlier run's checks failed before publishing)
- **THEN** the computed tag's `N` matches the count of images actually present
  in the registry for today, not the number of CI runs

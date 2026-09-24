# Spec Delta

## MODIFIED Requirements

### Requirement: CI lints and tests both packages on every push and pull request

Continuous integration SHALL run the frontend's lint and static-build checks,
and the backend's format, vet, and test checks, on every push and every pull
request, independent of whether an image is published for that run.

Continuous integration SHALL also run a contract check on every push and every
pull request: it SHALL lint `openapi/openapi.yaml`, regenerate the frontend's
committed API types from it, re-sync the committed `backend/openapi.yaml` copy
from it, and fail if either generated artifact differs from what is committed.

Continuous integration SHALL build the combined production container image on
every pull request after the frontend, backend, and contract checks succeed.
The pull-request image build SHALL NOT authenticate with or push to a registry.

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

- **WHEN** CI runs for a pull request and the frontend, backend, and contract
  checks pass
- **THEN** CI builds the combined production container image
- **AND** CI does not authenticate with or push the image to a registry

#### Scenario: Ordinary branch pushes do not publish

- **WHEN** CI runs for a push that is not a git tag
- **THEN** no container image is built or pushed to the registry

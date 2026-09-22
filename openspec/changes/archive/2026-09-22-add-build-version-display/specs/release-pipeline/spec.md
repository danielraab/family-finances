## MODIFIED Requirements

### Requirement: Pushing a git tag publishes an image tagged with that tag

WHEN all checks pass for a push of a git tag, CI SHALL build the container
image and push it to the GitHub Container Registry tagged with that git
tag's name, plus `latest`. The image build SHALL stamp the binary with
that git tag and the commit it was built from (see `build-version`), so
the published image reports its own identity at runtime.

#### Scenario: Tag push publishes

- **WHEN** a git tag is pushed and the frontend and backend checks pass
- **THEN** CI builds the container image and pushes it to the registry
  tagged with the pushed tag's name and with `latest`
- **AND** the image's `GET /api/version` reports that tag as `version`
  and the tagged commit's hash as `commit`

#### Scenario: Failing checks block a tag publish

- **WHEN** a git tag is pushed but the frontend or backend checks fail
- **THEN** CI reports failure and does not build or push a container image

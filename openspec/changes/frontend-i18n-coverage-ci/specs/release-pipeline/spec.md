## ADDED Requirements

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

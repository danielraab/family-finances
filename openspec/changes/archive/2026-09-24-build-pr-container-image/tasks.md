# Tasks

## 1. Pull-request image validation

- [x] 1.1 Add a pull-request-only CI job that builds the root Docker image after the frontend, backend, and contract jobs succeed; verify its workflow configuration uses `push: false` and has no registry login step.
- [x] 1.2 Pass the pull-request commit SHA as the image build revision without assigning a release version; verify the existing tag publishing build arguments and tags remain unchanged.

## 2. Validation

- [x] 2.1 Validate the OpenSpec change with `openspec validate build-pr-container-image --strict`.
- [x] 2.2 Validate `.github/workflows/ci.yml` syntax and review the event conditions to confirm PRs build without publishing while tag pushes still publish.

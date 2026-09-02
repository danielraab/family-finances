# api-contract Specification

## Purpose
TBD - created by archiving change adopt-openapi-contract. Update Purpose after archive.
## Requirements
### Requirement: A single spec-first OpenAPI document is the API's source of truth

The backend's JSON HTTP API SHALL be described by one hand-written OpenAPI 3.0
document at `openapi/openapi.yaml` in the repository root. That file SHALL be the
source of truth: it is authored by hand, not generated from code, and no
server-side handler code SHALL be generated from it. Every JSON request/response
endpoint under `/api/` that the frontend or an external client calls SHALL have a
corresponding operation in the document, including its path, method, request body
schema (where applicable), response status codes, and response body schemas.
Endpoints added in future changes SHALL be added to the same document in the same
change.

#### Scenario: Existing JSON endpoints are all described

- **WHEN** `openapi/openapi.yaml` is inspected after this change
- **THEN** it defines operations for `GET /api/healthz`, `GET /api/auth/me`,
  `POST /api/auth/logout`, `POST /api/auth/email/start`, and
  `POST /api/auth/invites`, each with its response status codes and body schemas

#### Scenario: The document is valid and lints clean

- **WHEN** an OpenAPI linter (`spectral`) runs against `openapi/openapi.yaml`
- **THEN** the document parses as OpenAPI 3.0 and reports no errors

#### Scenario: A new endpoint ships with its spec entry

- **WHEN** a later change adds a new JSON `/api/` endpoint
- **THEN** that change also adds the endpoint's operation to `openapi/openapi.yaml`

### Requirement: Browser-redirect endpoints are documented but excluded from client generation

Endpoints that a browser reaches by full-page navigation rather than `fetch` —
`GET /api/auth/email/callback`, `GET /api/auth/oidc/start`,
`GET /api/auth/oidc/callback`, and `GET /api/auth/invites/accept` — SHALL be
present in `openapi/openapi.yaml` with their query parameters and `302` responses
documented for human readers. Each SHALL carry a marker (a vendor extension or a
dedicated tag) that excludes it from the generated typed frontend client, so no
`fetch`-based helper is emitted for a redirect flow.

#### Scenario: Redirect operations are present and marked

- **WHEN** `openapi/openapi.yaml` is inspected
- **THEN** the four redirect endpoints appear with a documented `302` response
  and a marker indicating they are browser-navigation flows

#### Scenario: Generated client omits redirect flows

- **WHEN** the frontend typed client is generated from the document
- **THEN** it exposes no callable helper for any of the four redirect endpoints

### Requirement: The backend serves the OpenAPI document

The backend SHALL serve the contents of the OpenAPI document at
`GET /api/openapi.yaml` with `Content-Type: application/yaml` and a `200` status,
without requiring authentication. The bytes served SHALL be embedded in the
compiled binary (no external file path at runtime). The route SHALL be a defined
backend route under `/api/`, so an unmatched-`/api/` JSON `404` is never returned
for it and it never falls through to the static site.

#### Scenario: Spec is retrievable at runtime

- **WHEN** a client requests `GET /api/openapi.yaml` from a running backend
- **THEN** the response is `200` with `Content-Type: application/yaml` and a body
  that parses as the OpenAPI document

#### Scenario: Spec is available without a session

- **WHEN** `GET /api/openapi.yaml` is requested with no session cookie or bearer
  token
- **THEN** the response is still `200` with the document

#### Scenario: Spec is embedded, not read from disk

- **WHEN** the backend binary is run from a directory that contains no
  `openapi.yaml` file
- **THEN** `GET /api/openapi.yaml` still returns the document

### Requirement: Generated frontend types are committed and drift-checked

The frontend SHALL generate its API request/response types from
`openapi/openapi.yaml` into a committed file (`frontend/src/api/schema.d.ts`) and
SHALL consume the backend only through a typed client bound to those types. CI
SHALL regenerate that file and fail if the committed copy differs from the
freshly generated output.

#### Scenario: Committed types match the document

- **WHEN** CI regenerates `frontend/src/api/schema.d.ts` from
  `openapi/openapi.yaml`
- **THEN** `git diff --exit-code` reports no change

#### Scenario: Stale generated types fail CI

- **WHEN** `openapi/openapi.yaml` is changed but
  `frontend/src/api/schema.d.ts` is not regenerated in the same change
- **THEN** the CI contract check fails

#### Scenario: Frontend auth user type comes from the schema

- **WHEN** `frontend/src/components/AuthProvider.tsx` is inspected
- **THEN** the authenticated user's type is derived from the generated schema,
  not a separately hand-declared interface

### Requirement: The committed backend copy of the document stays in sync

Because `//go:embed` cannot reference a parent directory, a copy of the document
SHALL be committed at `backend/openapi.yaml` and embedded by `package main`. A
sync mechanism (a `go:generate` directive or a documented `make` target) SHALL
regenerate that copy from `openapi/openapi.yaml`. CI SHALL fail if the committed
copy differs from the source. The production image build SHALL copy
`openapi/openapi.yaml` into `backend/` before compiling, so a stale committed
copy cannot ship.

#### Scenario: Backend copy matches the source

- **WHEN** CI re-syncs `backend/openapi.yaml` from `openapi/openapi.yaml`
- **THEN** `git diff --exit-code` reports no change

#### Scenario: Image build does not depend on the committed copy

- **WHEN** the root `Dockerfile` builds the image
- **THEN** it copies `openapi/openapi.yaml` into `backend/` before `go build`, so
  the embedded document is the current source regardless of the committed copy

### Requirement: Backend responses are validated against the document in tests

The backend's handler tests SHALL assert that the HTTP responses they produce
conform to `openapi/openapi.yaml` (status code defined for the operation, body
matching the response schema). An implementation that diverges from the
documented contract SHALL fail `go test`. The validation library SHALL NOT be
part of the compiled server binary's dependency graph — it is reached only from
test code, optionally via a dedicated test-support package that no non-test file
in the server binary imports.

#### Scenario: Conforming response passes

- **WHEN** a handler test exercises an endpoint and the response matches the
  documented schema and status
- **THEN** the response-validation assertion passes

#### Scenario: Divergent response fails the test

- **WHEN** a handler is changed to return a field or status not described for its
  operation in `openapi/openapi.yaml`
- **THEN** the handler test's response-validation assertion fails

#### Scenario: Validation code is not in the shipped binary

- **WHEN** the transitive dependency graph of the compiled `server` binary
  (`go list -deps` of `package main`) is inspected
- **THEN** it contains no OpenAPI validation library


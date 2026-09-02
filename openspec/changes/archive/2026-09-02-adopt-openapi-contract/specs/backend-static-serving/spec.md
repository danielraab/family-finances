## MODIFIED Requirements

### Requirement: Backend routes reserved under /api/

The backend SHALL reserve the `/api/` path prefix for its own routes,
distinct from static frontend paths, so future API endpoints cannot collide
with frontend routes. The OpenAPI document SHALL be served as one such route at
`GET /api/openapi.yaml`.

#### Scenario: Health check under the reserved namespace

- **WHEN** a client requests `GET /api/healthz`
- **THEN** the backend returns a `200` status

#### Scenario: API namespace does not serve frontend files

- **WHEN** a client requests a path under `/api/` that is not a defined
  backend route
- **THEN** the backend does not return frontend HTML/JS content for it

#### Scenario: OpenAPI document served under the reserved namespace

- **WHEN** a client requests `GET /api/openapi.yaml`
- **THEN** the backend returns `200` with `Content-Type: application/yaml` and
  the OpenAPI document body, not the SPA shell and not a JSON `404`

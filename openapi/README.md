# openapi/

`openapi.yaml` is the **hand-written source of truth** for the family-finances
backend's JSON HTTP API. It is spec-first: no server code is generated from it;
handlers are written by hand in `backend/internal/**/handler.go` and kept honest
by response-validation tests (`backend/internal/openapicheck`) and the CI
contract check.

## What it must cover

Every JSON `/api/...` endpoint the frontend or an external client calls: path,
method, request body schema, every response status code the handler can return,
and each response body schema.

## `browser-flow` operations

`GET /api/auth/email/callback`, `GET /api/auth/oidc/start`,
`GET /api/auth/oidc/callback`, and `GET /api/auth/invites/accept` are reached by
full-page browser navigation, not `fetch`. They are documented here for humans
but carry `x-internal: true` (and the `browser-flow` tag). The frontend type
generator (`frontend/scripts/generate-api.mjs`) drops any operation with
`x-internal: true` before generating, so the typed client exposes no helper for
them.

## After editing `openapi.yaml`

Two generated artifacts must be regenerated and committed in the **same change**:

```bash
# 1. backend embedded copy (//go:embed cannot cross '..')
cd backend && go generate ./...        # syncs ../openapi/openapi.yaml -> backend/openapi.yaml

# 2. frontend types
cd frontend && pnpm generate:api        # -> src/api/schema.d.ts
```

CI re-runs both and fails on any diff. It also lints this file:

```bash
pnpm dlx @stoplight/spectral-cli lint openapi/openapi.yaml --ruleset openapi/.spectral.yaml
```

## Version

OpenAPI **3.0.3** — `getkin/kin-openapi` (the Go response validator) only
reliably handles 3.0, and nothing here needs a 3.1-only feature.

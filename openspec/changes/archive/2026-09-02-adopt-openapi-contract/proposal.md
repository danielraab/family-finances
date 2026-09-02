## Why

The backend's JSON API shape lives only in Go handler code, and the frontend
re-declares matching types by hand (e.g. `type User` in `AuthProvider`). The two
drift silently. Before the first product nouns (accounts, transactions, budgets)
land, we want one machine-readable contract that both toolchains generate from,
so payload types stay in lock-step and every endpoint is documented in one place.
We are staying with REST — this adds a contract, not a new API paradigm.

## What Changes

- Add a hand-written, **spec-first** OpenAPI 3.0 document at `openapi/openapi.yaml`
  (repo root, beside `openspec/`), documenting every JSON `/api/` endpoint that
  exists today: `GET /api/healthz`, `GET /api/auth/me`, `POST /api/auth/logout`,
  `POST /api/auth/email/start`, `POST /api/auth/invites`.
- Redirect / browser-flow endpoints (`GET /api/auth/oidc/start`,
  `GET /api/auth/email/callback`, `GET /api/auth/oidc/callback`,
  `GET /api/auth/invites/accept`) are **documented** in the spec as `302`
  responses but are **not** part of the generated client surface.
- Serve the spec from the backend at `GET /api/openapi.yaml`
  (`Content-Type: application/yaml`). The bytes are embedded by `package main`
  from a committed `backend/openapi.yaml` synced from the root source of truth.
- Frontend generates `src/api/schema.d.ts` from the spec with `openapi-typescript`
  (committed) and calls the backend through `openapi-fetch` (~2 kB typed `fetch`
  wrapper — no query library). `AuthProvider`'s hand-written `User` type is
  replaced by the generated schema type.
- Backend handler tests additionally validate their HTTP responses against the
  spec (`github.com/getkin/kin-openapi`), so an implementation that diverges from
  the contract fails CI.
- CI (`.github/workflows/ci.yml`) gains a contract gate: `spectral` lint of the
  YAML, regeneration of `schema.d.ts` and the `backend/openapi.yaml` copy with
  `git diff --exit-code`.
- Tighten `frontend/tsconfig.json`: `noUncheckedIndexedAccess`,
  `exactOptionalPropertyTypes`, `noPropertyAccessFromIndexSignature`, and fix the
  resulting fallout in `src/`.
- New dependencies (justified in `design.md`): `openapi-typescript` (dev),
  `openapi-fetch` (runtime) in the frontend; `github.com/getkin/kin-openapi`
  (test scope) in the backend.

## Capabilities

### New Capabilities

- `api-contract`: a single spec-first OpenAPI document is the source of truth for
  the backend's JSON HTTP API — where it lives, what it must cover, how it is
  served, how generated artifacts and the running implementation are kept from
  drifting, and how the frontend consumes it with generated types.

### Modified Capabilities

- `backend-static-serving`: `/api/openapi.yaml` is a reserved backend route under
  the `/api/` prefix and never falls through to the static site.
- `backend-package-architecture`: `package main` embeds a committed
  `backend/openapi.yaml` (synced from `openapi/openapi.yaml`, kept identical by
  CI, same rationale as the `static/out` embed) and passes it to
  `internal/httpapi`, which serves it.
- `release-pipeline`: CI runs a contract job — spec lint plus a generated-artifact
  drift check — on every push and pull request.
- `web-client-auth`: the browser auth state's user record is typed from the
  generated OpenAPI schema rather than a hand-declared interface; behaviour is
  unchanged.

## Impact

- **New files**: `openapi/openapi.yaml`, `openapi/README.md`,
  `backend/openapi.yaml` (synced copy), `frontend/src/api/schema.d.ts`
  (generated), a frontend `src/api/client.ts` wrapper.
- **Backend**: `embed.go` (second `//go:embed`), `internal/httpapi` route +
  `Deps` field, handler `_test.go` files gain response validation, `go.mod`
  (+`getkin/kin-openapi`).
- **Frontend**: `package.json` (+2 deps, +generate script), `tsconfig.json`
  (stricter), `AuthProvider.tsx` (generated type), any `src/` code that trips the
  new compiler flags.
- **CI**: `.github/workflows/ci.yml` gains a contract job; the Docker build copies
  `openapi/openapi.yaml` into `backend/` before compiling.
- **Docs**: root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md` describe
  the contract workflow and the new dependencies.
- No runtime behaviour change for end users; no database change; no breaking API
  change.

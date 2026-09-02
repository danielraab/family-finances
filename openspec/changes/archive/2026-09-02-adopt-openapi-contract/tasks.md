## 1. Author the OpenAPI document

- [x] 1.1 Create `openapi/` with `openapi.yaml` (OpenAPI 3.0): `info`, `servers`
  (`/`), and a `components/schemas` section
- [x] 1.2 Define shared schemas: `User` (`id`, `email`, `display_name?`,
  `is_admin`), `Error` (`{ error: string }`), `EmailStartRequest`,
  `InviteRequest`, `Invite`
- [x] 1.3 Document the JSON operations with request/response schemas and every
  status code: `GET /api/healthz`, `GET /api/auth/me` (200/401),
  `POST /api/auth/logout` (204/401), `POST /api/auth/email/start` (200),
  `POST /api/auth/invites` (201/400/401/403)
- [x] 1.4 Document the browser-redirect operations with query params and `302`
  responses: `GET /api/auth/email/callback`, `GET /api/auth/oidc/start`,
  `GET /api/auth/oidc/callback`, `GET /api/auth/invites/accept`; mark each with
  the agreed exclusion marker (`x-internal: true` or a `browser-flow` tag)
- [x] 1.5 Add `openapi/.spectral.yaml` (ruleset) and `openapi/README.md` (how to
  edit, how to regenerate the copies, what the exclusion marker means)
- [x] 1.6 Run `spectral lint openapi/openapi.yaml` locally; resolve all errors

## 2. Backend: embed and serve the document

- [x] 2.1 Add a sync mechanism: a `go:generate` directive (or `make spec` target)
  that copies `openapi/openapi.yaml` to `backend/openapi.yaml`; run it and commit
  `backend/openapi.yaml`
- [x] 2.2 Add a second `//go:embed openapi.yaml` in `package main` (in `embed.go`)
  exposing the bytes
- [x] 2.3 Add `OpenAPISpec []byte` to `httpapi.Deps`; pass the embedded bytes
  from `main.go`
- [x] 2.4 In `internal/httpapi`, register `GET /api/openapi.yaml` returning the
  bytes with `Content-Type: application/yaml` and a long `Cache-Control`; ensure
  it out-ranks the `/api/` catch-all and needs no auth
- [x] 2.5 Add a `httpapi` test: `GET /api/openapi.yaml` returns `200`,
  `application/yaml`, non-empty body, and works with no session

## 3. Backend: response validation against the document

- [x] 3.1 Add `github.com/getkin/kin-openapi` to `go.mod` (test-support only)
- [x] 3.2 Add `internal/openapicheck` — a test-support package (imported only by
  `_test.go` files) that loads/compiles `backend/openapi.yaml` once and validates
  a recorded response for a given method+path via
  `openapi3filter.ValidateResponse`
- [x] 3.3 Call it from the auth handler tests (`me`, `logout`, `email/start`,
  `invites`) and the httpapi healthz test; assert each response conforms
- [x] 3.4 Add a guard test asserting `go list -deps` of `package main` (the
  server binary) contains neither `kin-openapi` nor `internal/openapicheck`
- [x] 3.5 `cd backend && gofmt -l . && go vet ./... && go test ./...` all clean

## 4. Frontend: generated types and typed client

- [x] 4.1 Add dev dep `openapi-typescript` and runtime dep `openapi-fetch` via
  pnpm; add `onlyBuiltDependencies` entries if needed
- [x] 4.2 Add a `generate:api` script:
  `openapi-typescript ../openapi/openapi.yaml -o src/api/schema.d.ts`
- [x] 4.3 Generate and commit `frontend/src/api/schema.d.ts`
- [x] 4.4 Add `src/api/client.ts`: an `openapi-fetch` client with `baseUrl: "/"`
  and `credentials: "same-origin"`, exported for app use
- [x] 4.5 Switch `AuthProvider.tsx` to the generated `User` type and issue
  `GET /api/auth/me` through the typed client; keep the `loading/anonymous/
  authenticated` behaviour identical
- [x] 4.6 Point `logout()` at the typed client too (no behaviour change)

## 5. Frontend: stricter TypeScript

- [x] 5.1 Add to `tsconfig.json`: `noUncheckedIndexedAccess`,
  `exactOptionalPropertyTypes`, `noPropertyAccessFromIndexSignature`
- [x] 5.2 Run `pnpm exec tsc`; fix all resulting errors in `src/`
- [x] 5.3 `pnpm lint` and `pnpm build` clean

## 6. CI and Docker

- [x] 6.1 Add a `contract` job (or steps in existing jobs) to
  `.github/workflows/ci.yml`: `spectral lint openapi/openapi.yaml`; regenerate
  `frontend/src/api/schema.d.ts`; re-sync `backend/openapi.yaml`;
  `git diff --exit-code`
- [x] 6.2 Update the root `Dockerfile` to `COPY openapi/openapi.yaml` into
  `backend/` before `go build`
- [ ] 6.3 Push a branch and confirm the contract job passes on a clean tree and
  fails on an intentional spec/type mismatch, then revert the mismatch

## 7. Documentation

- [x] 7.1 Root `AGENTS.md`: describe `openapi/` as the API source of truth and
  the sync-to-`backend/` rule
- [x] 7.2 `backend/AGENTS.md`: the second `//go:embed`, the `/api/openapi.yaml`
  route, the `kin-openapi` test dependency and its justification
- [x] 7.3 `frontend/AGENTS.md`: `openapi-typescript` + `openapi-fetch`, the
  `generate:api` script, committed `schema.d.ts`, the stricter `tsconfig` flags
- [x] 7.4 Update `backend/.env.example` only if a new var was introduced (none
  expected)

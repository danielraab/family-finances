## Context

The backend is hand-rolled `net/http` (Go 1.22 pattern routing) with a strict
"no web framework, no router library, no ORM" rule; third-party dependencies are
added only through a proposal that justifies them. Domain packages follow a
"package per noun, not per layer" shape (`<noun>.go`, `service.go`, `store.go`,
`handler.go`) and import neither `internal/httpapi` nor storage.

Today the API's JSON shape is implicit in handler code. The frontend hand-writes
matching TypeScript (`type User` in `AuthProvider`). Only `internal/auth`
endpoints and `GET /api/healthz` exist; the product nouns are still unbuilt. This
is the cheapest moment to fix the contract convention — before there is a pile of
endpoints to retrofit.

The team has already decided (in exploration): stay REST; adopt OpenAPI
spec-first; keep the spec outside `backend/`; check drift in CI; document
redirect flows without generating clients for them; serve the spec; tighten the
frontend `tsconfig`.

## Goals / Non-Goals

**Goals:**

- One hand-written OpenAPI 3.1 document as the single source of truth for the
  JSON HTTP API.
- Frontend request/response types generated from that document, committed, and
  drift-checked in CI.
- Backend handler responses validated against the document in existing tests, so
  implementation drift fails CI.
- The spec is discoverable at runtime (`GET /api/openapi.yaml`).
- Establish the convention and the tooling with the endpoints that exist today;
  future nouns extend the same file.

**Non-Goals:**

- No GraphQL, no RPC, no change to the REST style of the API.
- No server-side handler code generation — handlers stay hand-written in the
  existing `handler.go` shape.
- No API behaviour change, no new endpoint (the OIDC-config endpoint is a
  separate downstream change), no database change.
- No React data-fetching / caching library (`react-query` etc.) — out of scope.
- No bundled Swagger UI / Redoc page in this change (possible later follow-up).
- Not documenting non-JSON internals (static file serving, the SPA fallback).

## Decisions

### D1 — Spec-first, hand-written YAML (not code-first, not server codegen)

The document is authored by hand at `openapi/openapi.yaml`. Handlers remain
hand-written; the spec is kept honest by a response-validation test (D5) and CI
drift checks (D6), not by generating server code.

- **Alternatives considered:**
  - *Code-first (annotations → spec)*, e.g. `swaggo/swag`: annotation comments
    are verbose and noisy, and the tool's output is awkward for OpenAPI 3.1.
    Fights the clean `handler.go` aesthetic.
  - *Full server codegen* (`oapi-codegen` `ServerInterface` + std-`net/http`
    router): strongest guarantee, still stdlib routing, but it rewrites how every
    handler is authored (implement a generated interface) and lands generated Go
    in the tree. Too large a departure for the benefit at this scale.
- **Why spec-first wins here:** it matches a codebase whose identity is
  deliberate minimalism and hand-rolled wiring; the drift risk it introduces is
  cheap to close with one test and one CI diff.

### D2 — Source of truth at repo root `openapi/`, synced copy embedded in the backend

`openapi/openapi.yaml` is the source of truth (a directory of its own, leaving
room for `$ref`-split files and a `README.md`). It sits beside `openspec/`
(prose specs) — different artefacts, both standard names.

`//go:embed` cannot reference a parent directory (the same constraint already
documented for `embed.go` and the frontend bundle). So a **committed copy** lives
at `backend/openapi.yaml`, embedded by `package main` (a second `//go:embed`
beside `static/out`), passed into `internal/httpapi` via a `Deps` field, and
served by a route there. The copy is a build artefact kept byte-identical to the
source by CI (D6); a `go:generate` directive / `make` target performs the local
sync.

- **Alternatives considered:**
  - *Docker-copy + committed placeholder* (exactly the `frontend/out/` →
    `backend/static/out/` pattern): consistent, but a local `go run .` then
    serves a stub spec, which is poor DX for the people editing a small,
    hand-maintained file who want to `curl /api/openapi.yaml` to check their
    edits.
  - *Runtime read from `OPENAPI_SPEC_PATH`*: no embed, but the binary is no
    longer self-contained and the Docker image still has to place the file — all
    cost, little gain.
  - *Serve it from the frontend bundle* (`frontend/public/openapi.yaml`): would
    surface at `/openapi.yaml` under the static handler, not `/api/`, and couple
    the spec to the frontend build. Rejected.
- **Trade-off accepted:** one generated/committed file inside `backend/` whose
  freshness depends on a CI check — the same bargain already made for the
  frontend's generated `schema.d.ts` and `routeTree.gen.ts`.

### D3 — Redirect flows are documented, not part of the generated client

The browser-flow endpoints (`/api/auth/oidc/start`, `/api/auth/email/callback`,
`/api/auth/oidc/callback`, `/api/auth/invites/accept`) are described in the spec
with their `302` responses and query parameters, for completeness and human
readers. They are marked so the frontend generator does not emit typed client
helpers for them: the browser reaches these by full-page navigation / links, not
`fetch`. Mechanism: an `x-internal: true` (or equivalent vendor extension /
excluded tag) on those operations, filtered before `openapi-typescript` runs, or
simply not called through the typed client.

### D4 — Frontend: `openapi-typescript` + `openapi-fetch`, types committed

`openapi-typescript` generates `frontend/src/api/schema.d.ts` (types only, no
runtime). `openapi-fetch` (~2 kB) is a thin typed wrapper over `fetch` that binds
paths, params, and responses to those types while preserving
`credentials: "same-origin"` and relative `/api` paths. A small
`src/api/client.ts` instantiates it. `schema.d.ts` is **committed** (visible in
review, mirrors the existing `routeTree.gen.ts` precedent) and regenerated + diff
-checked in CI.

- **Alternatives considered:**
  - *Plain `fetch` + generated types only*: no path/response binding, easy to
    call an endpoint with the wrong shape. `openapi-fetch` closes that for ~2 kB.
  - *`orval` / `openapi-generator`*: heavier, and `orval` pulls in `react-query`,
    which the frontend has deliberately not adopted.
  - *Apollo/urql-style clients*: GraphQL-oriented, irrelevant here.

### D5 — Backend: validate responses against the spec in handler tests

Handler tests already use `net/http/httptest` + the in-memory store. Each adds an
assertion that the response validates against the loaded `openapi/openapi.yaml`,
using `github.com/getkin/kin-openapi` (`routers/gorillamux` +
`openapi3filter.ValidateResponse`). A shared test helper loads and compiles the
spec once. `kin-openapi` is imported only from `_test.go` files.

- **Alternatives considered:**
  - *No library — rely on generated Go types in handlers* (`oapi-codegen
    -generate types`): the compiler would enforce struct shape but not
    HTTP-level facts (status codes, required fields actually present, additional
    properties). Weaker, and still a dependency.
  - *Contract test as a separate suite hitting a running server*: more moving
    parts than folding one assertion into tests that already exist.

### D6 — CI contract gate

A job in `.github/workflows/ci.yml` (new `contract` job, or folded into the two
existing jobs) runs on every push and pull request:

1. `spectral lint openapi/openapi.yaml` (ruleset committed at
   `openapi/.spectral.yaml`).
2. Regenerate `frontend/src/api/schema.d.ts`; `git diff --exit-code`.
3. Re-sync `backend/openapi.yaml` from the source; `git diff --exit-code`.

The backend job additionally runs the existing `go test ./...`, which now
includes response validation (D5). The Docker build (root `Dockerfile`) copies
`openapi/openapi.yaml` into `backend/` before `go build`, so the image build does
not depend on the committed copy being fresh.

### D7 — `tsconfig` strictness

Add to `frontend/tsconfig.json`: `noUncheckedIndexedAccess`,
`exactOptionalPropertyTypes`, `noPropertyAccessFromIndexSignature`. The first two
pair directly with generated OpenAPI types (optional vs nullable, index access on
records). Existing `src/` fallout is fixed as part of this change; the codebase
is young so the surface is small.

### D8 — Route and serving details

`internal/httpapi` gains `Deps.OpenAPISpec []byte` and a
`GET /api/openapi.yaml` route returning the bytes with
`Content-Type: application/yaml` and a long-lived `Cache-Control`. It is more
specific than the `/api/` catch-all, so it is never a JSON 404 and never reaches
the static handler. No auth required — the document is public API surface.

## Risks / Trade-offs

- **Hand-written spec drifts from handler behaviour** → D5 validates every
  handler-test response against the spec; a mismatch fails `go test`.
- **Committed `backend/openapi.yaml` goes stale** → D6 re-syncs and
  `git diff --exit-code`; the Docker build copies from source regardless, so a
  stale copy cannot ship.
- **Three new dependencies against a minimalist policy** → each is narrow:
  `openapi-typescript` is dev-only codegen; `openapi-fetch` is ~2 kB and a
  strict `fetch` superset; `kin-openapi` is test-scope only. None is a framework
  or router. Justified per `backend/AGENTS.md` / `frontend/AGENTS.md` dependency
  rules.
- **Stricter `tsconfig` surfaces unrelated errors** → fixed within this change;
  small because the frontend is new. If fallout is larger than expected,
  `noPropertyAccessFromIndexSignature` can be dropped without losing the
  OpenAPI-related value.
- **OpenAPI 3.1 tooling maturity** → RESOLVED during implementation: the spec is
  authored as **OpenAPI 3.0.3**. `getkin/kin-openapi` (the response-validation
  library) only reliably parses/validates 3.0, and no requirement here needs a
  3.1-only feature. `openapi-typescript` and `spectral` both handle 3.0 fine.
- **Redoc/Swagger UI not shipped** → the raw spec is served and is enough for
  codegen and tooling; a rendered docs page can be added later without a
  contract change.

## Migration Plan

1. Write `openapi/openapi.yaml` covering the five JSON endpoints + the documented
   redirect operations; add `openapi/.spectral.yaml` and `openapi/README.md`.
2. Add the sync mechanism (`go:generate` / `make` target) and commit
   `backend/openapi.yaml`.
3. Backend: `embed.go` second embed, `httpapi` `Deps` field + route, test helper
   + per-handler response validation, `go.mod` update.
4. Frontend: add deps, `generate:api` script, generate + commit `schema.d.ts`,
   add `src/api/client.ts`, switch `AuthProvider` to the generated `User` type,
   apply `tsconfig` flags and fix fallout.
5. CI: add the contract job; update the root `Dockerfile` copy step.
6. Docs: update the three `AGENTS.md` files.

**Rollback:** the change is additive and behaviour-neutral. Reverting the commit
removes the spec, the route, the generated file, the deps, and the CI job with no
data or API-consumer impact.

## Open Questions

- `contract` as a standalone CI job vs. folding the steps into the existing
  `frontend` / `backend` jobs — decide during implementation based on runtime.
- Whether to also expose `GET /api/openapi.json` (a straight YAML→JSON of the
  same document) now or defer until a consumer needs it.
- Exact vendor-extension / tag name used to exclude redirect operations from
  `openapi-typescript` output (`x-internal` vs. an excluded `browser-flow` tag).

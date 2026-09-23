# Tasks

## 1. API contract and entry filtering

- [x] 1.1 Add optional stored-scale integer `amount_from` and `amount_to` query parameters, including their signed inclusive semantics, to `GET /api/entries` in `openapi/openapi.yaml`; regenerate `backend/openapi.yaml` and `frontend/src/api/schema.d.ts`, and verify no generated-artifact drift remains.
- [x] 1.2 Extend the entry filter and shared HTTP parsing with optional signed amount bounds; reject malformed integers and an inverted pair, and verify handler tests cover valid open/closed ranges and `400` responses.
- [x] 1.3 Apply the bounds to account-oriented amounts in both Postgres and memory list filtering, and verify store/service tests cover inclusivity, balance-adjustment deltas, and independently matched self-transfer legs.

## 2. Ledger filter experience

- [x] 2.1 Add `amount_from` and `amount_to` to `/entries` typed search validation, request construction, persisted filter state, active-filter detection, empty-state detection, and clear-all behavior; verify a changed bound resets the first page and an amount-only URL is not treated as bare for restoration.
- [x] 2.2 Add optional full-precision major-unit amount start/end controls to the ledger filter panel, converting through the existing fixed-scale amount helper and surfacing translated labels; verify start-only, end-only, and both-bound requests serialize the expected integer query values.
- [x] 2.3 Add English source and German translations for the new amount-range controls, then verify `pnpm lint` and `pnpm exec tsc` pass from `frontend/`.

## 3. Verification

- [x] 3.1 Run `go generate ./...`, `gofmt -l .`, `go vet ./...`, and `go test ./...` from `backend/`; verify formatting output is empty and all checks pass.
- [x] 3.2 Run `pnpm generate:api`, `pnpm lint`, `pnpm exec tsc`, and `pnpm build` from `frontend/`; verify generated API types are current and the production bundle builds.
- [x] 3.3 Run `openspec validate add-entry-list-amount-filter --strict` from the repository root and verify the completed change passes validation.

# Tasks

## 1. API contract and magnitude filtering

- [x] 1.1 Redefine `amount_from` and `amount_to` in `openapi/openapi.yaml` as non-negative absolute-amount bounds, then run `go generate ./...` from `backend/` and `pnpm generate:api` from `frontend/` to verify both committed generated artifacts are synchronized.
- [x] 1.2 Update entry filter parsing to reject negative, malformed, and inverted bounds, and update handler tests to verify inclusive open/closed magnitude ranges return entries of either sign and produce `400` for invalid bounds.
- [x] 1.3 Apply magnitude predicates to account-oriented legs in both memory and PostgreSQL storage without overflowing at the minimum signed integer; verify self-transfer legs match together and balance adjustments use their computed delta magnitude in backend tests.

## 2. Ledger amount-range component

- [x] 2.1 Create a dedicated controlled `AmountRangeFilter` component that renders the translated paired amount controls, keeps its textual drafts synchronized with stored-scale bounds, and verifies typed/pasted negative values display and serialize as positive magnitudes.
- [x] 2.2 Replace `/entries`' inline amount-from and amount-to controls with `AmountRangeFilter`; verify URL search state, persisted filters, active-filter counting, clearing, and first-page reload behavior continue to use the bound pair as one filter.
- [x] 2.3 Verify the ledger shows both positive and negative entries for a selected magnitude range while each result retains its own signed amount display.

## 3. Verification

- [x] 3.1 From `backend/`, run `gofmt -l .`, `go vet ./...`, and `go test ./...`; verify formatting output is empty and all checks pass.
- [x] 3.2 From `frontend/`, run `pnpm lint`, `pnpm exec tsc --noEmit`, and `pnpm build`; verify all checks pass.
- [x] 3.3 Run `openspec validate absolute-entry-amount-filter --strict` from the repository root and verify the completed change passes validation.

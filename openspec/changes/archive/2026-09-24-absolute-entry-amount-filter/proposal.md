# Proposal

## Why

The entry ledger's amount filter currently treats bounds as signed values, so
a visitor cannot filter a monetary magnitude without separately considering
income and expense signs. Amount-range inputs should accept positive
magnitudes and return qualifying entries regardless of their stored sign.

## What Changes

- Redefine `GET /api/entries`' existing `amount_from` and `amount_to` query
  parameters as non-negative, inclusive absolute-amount bounds rather than
  signed amount bounds. **BREAKING** for API consumers using the unshipped
  signed semantics.
- Match ledger entries by the absolute value of their computed stored `amount`,
  while retaining their original signed values in list results and displays.
- Replace the two inline amount filter controls on `/entries` with a dedicated
  amount-range filter component that owns positive-only amount input behavior.
- Update the API contract and the ledger's URL/persisted filter semantics to
  represent magnitude bounds.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `account-entries`: Entry-list amount query bounds change from signed values
  to non-negative absolute-value bounds.
- `web-client-entries`: The ledger amount range becomes a positive-magnitude
  filter that returns matching entries regardless of sign.

## Impact

- `openapi/openapi.yaml` plus generated backend/frontend API artifacts.
- Backend entry query parsing, in-memory and PostgreSQL filtering, and handler
  and storage tests.
- `frontend/src/routes/entries.index.tsx`, a new reusable amount-range filter
  component, and entry-ledger filter tests if present.
- No schema migration, new dependency, or change to stored entry amounts.

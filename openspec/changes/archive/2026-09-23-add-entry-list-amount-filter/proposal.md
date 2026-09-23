# Proposal

## Why

The entry ledger cannot currently narrow a large result set by transaction value. Visitors need optional lower and upper amount bounds to find entries within a signed amount range without combining date, category, and text filters as a workaround.

## What Changes

- Add optional inclusive `amount_from` and `amount_to` bounds to `GET /api/entries`.
- Apply bounds to each listed entry's signed, account-oriented amount, including the sign-flipped receiving leg of a self-transfer and a balance adjustment's computed delta.
- Reject malformed or inverted amount ranges.
- Add two optional full-precision amount controls to the `/entries` filter panel, with their applied values represented in the URL and persisted alongside the existing ledger filters.
- Treat the two controls as one active filter, clear them through the existing clear-all action, and reset the infinite-scroll result set when either bound changes.
- Keep `GET /api/entries/summary` unchanged; amount bounds are a ledger-only capability.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `account-entries`: Extend the entry-list API with signed amount-range filtering.
- `web-client-entries`: Add persisted URL-backed amount-range controls to the ledger filter panel.

## Impact

- `openapi/openapi.yaml`, plus its committed backend and frontend generated artifacts.
- Backend entry filter parsing, service/store filter data, Postgres SQL, in-memory filtering, and handler/service/store tests.
- `frontend/src/routes/entries.index.tsx`, English/German translations, and frontend tests where present.

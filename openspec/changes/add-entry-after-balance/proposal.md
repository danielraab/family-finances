## Why

The read-only entry summary shows an entry's delta, but not the account balance
that resulted from applying it. Users therefore have to reconstruct the ledger
context to understand the entry's effect.

## What Changes

- Add an `after_balance` value to every Entry API representation.
- Compute it in ledger order and from the listed account's perspective, including
  the account-facing occurrence of a self-transfer.
- Show the value as “Balance after entry” in the read-only entry summary.
- Clarify that the existing balance-adjustment-only `balance` value is the
  recorded absolute reading, not the general running balance.

## Non-goals

- No editable after-balance field; the value is always server-computed.
- No change to account balance or balance-series endpoints.

## Capabilities

### Modified Capabilities

- `account-entries`: entry representations expose the running balance after the
  represented account leg.
- `web-client-entries`: the read-only summary displays that value.

## Impact

- `openapi/openapi.yaml` and both generated contract copies.
- Entry domain and PostgreSQL/in-memory persistence reads.
- Entry summary UI and English/German translations.

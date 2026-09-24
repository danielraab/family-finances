# Proposal

## Why

The paired amount bounds consume two persistent filter-panel slots and expose
fixed storage precision even when it adds no useful information. They should
behave as one compact range control like the date range.

## What Changes

- Place the entry ledger's amount start/end inputs in one small expandable
  overlay with a concise range summary.
- Format entered and summarized magnitudes with only their necessary decimal
  places, rather than trailing fixed-scale zeroes.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `web-client-entries`: The ledger amount range is presented as one compact,
  expandable control with precision-trimmed values.

## Impact

- `frontend/src/components/AmountRangeFilter.tsx` and the entry ledger only.
- No API, persisted URL value, storage, dependency, or backend changes.

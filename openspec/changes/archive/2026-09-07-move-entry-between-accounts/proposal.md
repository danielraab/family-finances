## Why

An entry's `account_id` is currently a hard, immutable field — enforced in the
spec, the backend (`Update` has no field for it), and the UI (a disabled
input). The only way to fix an entry filed against the wrong account is to
delete it and recreate it on the right one, losing its id and history. Because
an account's balance is always computed live from its entries (never stored),
moving an entry between accounts is safe — both accounts' balances
self-correct on the next read — so this restriction can be relaxed.

## What Changes

- `PATCH /api/entries/{id}` accepts `account_id`. `kind` remains immutable;
  only `account_id` becomes editable.
- The target account must be owned by the caller (same rule `POST
  /api/entries` already applies) and must not be disabled (same
  `ErrAccountDisabled` rejection creation already uses) — a disabled account
  can't newly receive an entry either way, whether by creation or by move.
- No currency conversion or validation happens server-side. Moving a
  `transaction` or `balance_adjustment` entry between accounts of different
  currencies is permitted; the amount's stored value is unchanged.
- `/entries/{id}/edit`: the account field stays a disabled display by
  default, with a pencil-icon button beside it. Clicking it unlocks a
  `<select>` of the caller's own non-deleted, non-disabled accounts (the
  entry's current account still renders even if it has since been disabled,
  mirroring the category picker's disabled-current-value handling) plus an
  "X" button that discards the pending selection and re-locks the field to
  the entry's original account.
- Selecting an account whose currency differs from the entry's original
  account shows an inline warning, without blocking the selection.
- Saving behaves exactly as today when `account_id` is unchanged. When it has
  changed, clicking Save opens a confirmation dialog (mirroring the existing
  delete-confirmation dialog) naming the target account and repeating the
  currency-mismatch note if applicable; confirming submits the whole form
  (including any other edited fields), canceling returns to the form
  untouched.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `account-entries`: splits the existing "account_id and kind SHALL NOT be
  changeable after creation" rule — `kind` stays immutable, `account_id`
  becomes editable subject to the same ownership/disabled-account checks
  `POST /api/entries` already applies.
- `web-client-entries`: replaces the "account and kind are not editable"
  scenario on `/entries/{id}/edit` with the unlock/select/cancel interaction,
  the currency-mismatch warning, and the account-change confirmation dialog.

## Impact

- Backend: `backend/internal/entry/entry.go` (`Update` struct),
  `backend/internal/entry/service.go` (`Update` validation), `backend/internal/entry/handler.go`
  (`entryUpdateBody`), `openapi/openapi.yaml` (+ synced
  `backend/openapi.yaml`), and their tests.
- Frontend: `frontend/src/routes/entries.$entryId.edit.tsx`, and
  `frontend/src/api/schema.d.ts` if the OpenAPI update changes the update
  request shape.

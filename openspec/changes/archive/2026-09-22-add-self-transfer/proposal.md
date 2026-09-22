## Why

Moving money between two of a user's own (or shared) accounts today has no
first-class representation: a user has to log two unrelated entries by hand
— an outgoing transaction on one account, an incoming one on the other —
with nothing tying them together, no guarantee the amounts match, and no way
to edit or delete the movement as a single thing. A `self_transfer` entry
kind makes this one operation: one entry, two accounts, always consistent
because there is only one amount and one row to edit or delete.

## What Changes

- Add `self_transfer` as a third `entry.Kind` (alongside `transaction` and
  `balance_adjustment`). A self-transfer entry carries a new `to_account_id`
  field (the receiving account) in addition to the existing `account_id`
  (the sending account); `amount` stays a signed delta, from the sending
  account's perspective, as it already is for `transaction`.
- `category_id` is optional on a `self_transfer` (unlike `transaction`,
  which requires one). A self-transfer's amount **is** included in
  `GET /api/entries/summary` and `GET /api/entries/flow-summary` — unlike
  `balance_adjustment`, which both already exclude.
- `POST /api/entries` with `kind: "self_transfer"` requires at least
  `append` permission on **both** `account_id` and `to_account_id`, and
  both accounts must share the same `currency` (`400` otherwise, same
  `ErrInvalidValue` mapping as today's `account_id` misuse errors) —
  cross-currency transfers are not supported at all, not just unvalidated.
- Editing a `self_transfer` entry additionally requires the caller to still
  hold `append`+ on both accounts at edit time; if permission on either side
  has since been revoked (or the account disabled/deleted), the entry
  becomes read-only. Deleting keeps the existing per-account rule
  (`entry_admin`/`owner` may delete any entry on an account they hold that
  tier on) evaluated against whichever side the caller still has access to
  — it is not blocked by the other side becoming inaccessible.
- `GET /api/entries` renders a self-transfer entry once per account it
  touches that's within the caller's requested/resolved account scope: once
  (sign as stored) when only `account_id` is in scope, once (sign flipped)
  when only `to_account_id` is in scope, and **twice** — once per side —
  when both accounts are in scope at once (e.g. an unfiltered "all
  accounts" ledger, or an explicit filter naming both).
- **BREAKING**: an account's `currency` becomes immutable once the account
  has any entry (of any kind) — `PATCH /api/accounts/{id}` with a changed
  `currency` on an account that already has entries is rejected. This
  closes the gap that would otherwise let a same-currency transfer pair
  drift into a currency mismatch after the fact.
- The entry form (`/entries/new`, `/entries/{id}/edit`) gains a third kind
  option, "Self-transfer," which reveals a "To account" `<select>` (offering
  only append+, same-currency, non-disabled accounts, excluding whichever
  account is already picked as the source) and hides category/counterparty/
  location the way `balance_adjustment` already hides counterparty/
  location. The account edit form (`/accounts/{id}/edit`) disables the
  currency field once the account has any entries.
- A small badge on each leg of a self-transfer entry (ledger, reports table)
  links to the other side, mirroring `RecurringTransactionBadge`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `account-entries`: new `Kind` value `self_transfer`, its `to_account_id`
  field, its own create/edit/delete permission rules (both-sides append+ to
  create or edit, per-accessible-side entry_admin/owner to delete), its
  optional category, and its inclusion in `Sum`/`FlowSummary`. `GET
  /api/entries` gains the once-or-twice rendering rule for a self-transfer
  whose two accounts are both in the caller's resolved account scope.
- `accounts`: `currency` becomes immutable once the account has any entry;
  `PATCH /api/accounts/{id}` rejects a `currency` change otherwise.
- `web-client-entries`: the create/edit form gains the "Self-transfer" kind
  option and its "To account" picker, and the ledger/reports row gains the
  other-leg badge.
- `web-client-accounts`: the account edit form's currency field is disabled
  once the account has any entries.

## Impact

- Backend: `backend/internal/entry/{entry,service,store,handler}.go` (new
  `Kind`, `to_account_id`, permission checks, Sum/FlowSummary inclusion,
  listing rewrite), `backend/internal/account/{account,service}.go` (new
  `EntryLookup` interface + currency-immutability check), `main.go` (wiring
  `account.WithEntryLookup(entrySvc)`), a new migration under
  `backend/internal/storage/postgres/migrations/`, `openapi/openapi.yaml` (+
  synced `backend/openapi.yaml` and `frontend/src/api/schema.d.ts`), and
  their tests.
- Frontend: `frontend/src/routes/entries.new.tsx`,
  `frontend/src/routes/entries.$entryId.edit.tsx`,
  `frontend/src/routes/accounts.$accountId.edit.tsx`, a new
  `SelfTransferBadge.tsx` (or a generalized badge shared with
  `RecurringTransactionBadge.tsx`), and the ledger/reports row components
  that render entries.

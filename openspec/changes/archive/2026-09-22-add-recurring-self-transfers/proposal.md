## Why

A standing move between two of the household's own accounts — the monthly
transfer from Checking into Savings, the fixed top-up of a joint account —
is one of the most regular movements a family has, and it is the one
movement recurring transactions cannot template. `add-self-transfer` gave
the one-off case a first-class shape (`kind: self_transfer`, one row, two
accounts, always consistent); `add-recurring-transactions` gave repeating
movements a template with a next-suggested date, a per-year total, and a
link back to every booking made from it. The two features do not meet: a
recurring transaction has no `kind` at all and always materializes a plain
`transaction`, so a standing transfer has to be re-entered by hand every
month, with nothing recording that it is a standing commitment and nothing
tying its bookings together.

## What Changes

- **A recurring transaction gains a `kind`**, `transaction` (the default,
  what every existing template is) or `self_transfer`, immutable after
  creation exactly as `entry.Kind` is. A `self_transfer` template carries a
  required `to_account_id`; every other kind's is null. This replaces
  `recurring-transactions`' current "a recurring transaction has no `kind`
  field" rule.
- **A `self_transfer` template mirrors a `self_transfer` entry's field
  rules**, so that what it materializes is always valid: `category_id`
  becomes optional for that kind (required for `transaction`, unchanged),
  and `counterparty`/`location` are rejected on it.
- **Creating or editing a `self_transfer` template requires `append`+
  permission on both accounts**, both non-disabled, sharing the same
  `currency`, and `to_account_id` differing from `account_id` — the same
  rule `POST /api/entries` already applies to a self-transfer entry,
  checked at template time so the template cannot be built into something
  that could never be booked.
- **A `recurring_transaction_legs` view** presents each template as it is
  seen from each account it touches — the stored row as-is, plus, only for
  a `self_transfer`, a second row with `account_id`/`to_account_id` swapped
  and `amount` negated — mirroring `entry_legs` exactly.
- **`GET /api/recurring-transactions` and
  `GET /api/recurring-transactions/summary` gain two flags**,
  `include_self_transfer` (default `false`) and `self_transfer_both_legs`
  (default `false`, meaningful only when the first is `true`): off, a
  `self_transfer` template is excluded entirely; on alone, it appears once,
  from its sending account's side; both on, it appears once per account of
  its two that is within the caller's resolved account scope — income and
  outcome. The summary always sums exactly what the list shows under the
  same flags, so the per-year total under the list always matches the rows
  above it; with both legs in scope the two cancel, which is what
  `GET /api/entries/summary` already does for a real self-transfer.
- **`GET /api/recurring-transactions/preview` always includes
  `self_transfer` templates and always emits both legs** (subject to its
  own account scope), with no flag — matching how a real self-transfer
  entry already behaves everywhere the preview is overlaid (the ledger, the
  Upcoming block, the flow summary, the projected balance line). Without
  this a projected balance on the receiving account would silently omit
  money the visitor can see arriving on the sending one.
- **`/recurring` gains a filter control** carrying those two flags: a "Show
  self-transfers" checkbox, and — revealed only once it is checked — a
  "Show both sides (income and outcome)" checkbox. Both live in the URL,
  the way `/entries` keeps its own filter state.
- **`/recurring/new` and `/recurring/{id}/edit` gain a kind selector** and,
  for `self_transfer`, a "To account" picker, hiding counterparty/location
  and dropping the category requirement — the same shape `/entries/new`
  already has. The recurring list row and the recurring summary modal show
  the other account and which side the row is.
- **An account's `currency` becomes immutable once it is named by a
  `self_transfer` template**, not only once it has entries — closing the
  same drift `add-self-transfer` closed for entries, which a template
  reopens (a template pairs two accounts long before either has an entry).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `recurring-transactions`: the new `kind` field and its `to_account_id`,
  replacing the "no `kind` field" rule; per-kind category/counterparty/
  location rules; the both-accounts create/edit permission and
  same-currency rule; the two listing flags and their effect on
  `GET /api/recurring-transactions` and its summary; preview's
  always-both-legs rule.
- `accounts`: `currency` immutability extends to an account named as either
  side of a `self_transfer` recurring transaction.
- `web-client-recurring-transactions`: the `/recurring` filter control and
  its two flags; the kind selector and "To account" picker on the create/
  edit form; the self-transfer row treatment.

## Impact

- Backend: `backend/internal/recurringtransaction/{recurringtransaction,
  service,store,handler}.go` (new `Kind`/`ToAccountID`, per-kind
  validation, both-accounts checks, the two filter flags, decorating the
  to-account fields), `backend/internal/account/{account,service}.go` (a
  second narrow lookup for the currency lock), `main.go` (wiring it), a new
  migration under `backend/internal/storage/postgres/migrations/` plus the
  `recurring_transaction_legs` view, the memory store's mirror
  implementation, `openapi/openapi.yaml` (+ synced `backend/openapi.yaml`
  and `frontend/src/api/schema.d.ts`), and their tests.
- Frontend: `frontend/src/routes/recurring.index.tsx` (filter control, row
  treatment), `frontend/src/components/RecurringTransactionForm.tsx` (kind
  selector, to-account picker, per-kind fields),
  `frontend/src/components/summary/` (the recurring summary modal's
  to-account row), `frontend/src/routes/recurring.new.tsx` and
  `recurring.$id.edit.tsx` where they shape the form's payload, and new
  i18n keys in both locales.

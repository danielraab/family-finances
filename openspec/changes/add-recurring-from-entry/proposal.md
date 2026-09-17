## Why

Creating a recurring transaction today always starts from a blank
`/recurring/new` form. When a visitor notices a transaction they just
entered is actually one they enter regularly, they have no shortcut: they
have to re-type the title, category, counterparty, location, tags, and
amount by hand into a separate form, then (if they want the two connected
at all) separately open the entry again and link it via the "Linked
recurring transaction" dropdown. This change adds a one-click path from an
existing entry straight to a prefilled recurring-transaction template,
auto-linked back to the entry it came from.

## What Changes

- Add a compact, icon-only "create recurring transaction" action to
  `entries.$entryId.edit.tsx`'s button row, visible only when
  `canEdit && entry.kind === "transaction"`.
- The action has two states depending on whether the form has unsaved
  edits:
  - **Clean**: navigates directly to `/recurring/new?from_entry_id=<id>`.
  - **Dirty**: performs the same validation/save as the existing Save
    button (including the account-change confirmation dialog when
    applicable), and on success navigates to
    `/recurring/new?from_entry_id=<id>` instead of Save's normal redirect
    to `/entries`.
- Add a `from_entry_id` search param to `/recurring/new`: when present, it
  fetches that entry and prefills title, description, category,
  counterparty, location, tags, and amount/sign, locks the account picker
  to the entry's account, and defaults `starts_on` to the entry's booking
  date (date portion only).
- On successful creation from this flow, the new recurring transaction is
  always auto-linked back to the originating entry (`PATCH` its
  `recurring_transaction_id`) — no opt-out checkbox — and the visitor lands
  on `/recurring` rather than `/recurring/new`'s normal (none today, since
  this is a new entry point) destination.
- No backend or API contract changes: this flow is entirely composed from
  existing endpoints (`GET /api/entries/{id}`, `POST
  /api/recurring-transactions`, `PATCH /api/entries/{id}`).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `web-client-entries`: the entry-edit form gains the "create recurring
  transaction" action described above, including its dirty-state label
  switch and save-then-navigate behavior.
- `web-client-recurring-transactions`: `/recurring/new` gains the
  `from_entry_id` prefill/account-lock behavior and the auto-link-back
  step on successful creation from that entry point.

## Impact

- `frontend/src/routes/entries.$entryId.edit.tsx` — new button, dirty
  tracking, save-then-navigate variant of `performSubmit`.
- `frontend/src/routes/recurring.new.tsx` — `from_entry_id` search param,
  entry fetch/prefill, post-create auto-link `PATCH`, redirect to
  `/recurring`.
- `frontend/src/components/RecurringTransactionForm.tsx` — reuse of the
  existing `accountLocked` prop; no shape changes expected.
- i18n: new `entries.edit.createRecurring` /
  `entries.edit.saveAndCreateRecurring` keys in `en.json` then `de.json`.
- No backend, database, or `openapi/openapi.yaml` changes.

## Why

The entry ledger (`/entries`) only supports acting on one entry at a time,
via its own edit page. Cleaning up a batch of entries that all need the
same treatment — recategorizing a run of miscategorized transactions,
tagging everything from an import, detaching a stale recurring link, bulk
deleting duplicates — currently means opening each entry individually.
There is no selection mechanism and no batch endpoint anywhere in the
codebase.

## What Changes

- The entry ledger's table gains a checkbox column: a per-row checkbox
  (always enabled, regardless of the visitor's edit permission on that
  entry) and a header checkbox that selects/deselects every currently
  *loaded* row (the infinite-scroll window, not every row matching the
  active filter that hasn't scrolled into view yet).
- A bulk-action toolbar appears above the table whenever at least one row
  is selected, showing the selection count, a "Clear" action, and six
  actions: **Set category**, **Add tags**, **Remove tags**, **Set tags**,
  **Link to recurring transaction**, **Delete**.
  - **Add tags** / **Remove tags** / **Set tags** each open the existing
    `TagInput` multi-select. Add unions each entry's own tags with the
    chosen ones; Remove subtracts the chosen ones (offered only tags
    actually present on at least one selected entry); Set replaces each
    entry's tag set outright.
  - **Link to recurring transaction** is disabled whenever the current
    selection spans more than one account (a recurring transaction only
    ever applies to entries on its own account — see `account-entries`);
    when enabled, it offers the recurring transactions on that one shared
    account, plus an explicit "Not linked" option to unlink.
  - **Delete** soft-deletes every selected entry, one `DELETE` per entry.
- Every action requires an explicit confirmation step — picking a value
  and pressing Apply, or confirming the delete — before anything runs;
  nothing fires on selection alone.
- Confirming any action opens a shared progress modal that runs a
  sequential client-side loop (one `PATCH`/`DELETE` per selected entry,
  the same shape `entries.import.tsx`'s `ImportRunStep` already uses for
  many-entries-one-call-each work), shows a live progress bar, and on
  completion lists how many succeeded and, individually, which entries
  failed and why. The modal always offers **Reload list** (re-fetches the
  ledger from the current filters) and **Close**.
- No entry is pre-excluded from selection by permission. A selected entry
  the visitor cannot actually write to (an `append`-tier visitor acting on
  someone else's entry, or a `view`-tier one) is attempted like any other
  and simply appears in the failed list — the same server-side
  `account-entries` permission check that already gates single-entry
  edit/delete is what rejects it, unchanged.
- Bulk "set title" was considered and dropped: flattening many different
  transactions to one identical title is a much rarer, more destructive
  intent than the other actions and isn't included in this change.

## Capabilities

### Modified Capabilities

- `web-client-entries`: the ledger gains row/header selection, a
  bulk-action toolbar, and a shared bulk-run progress/results modal.

## Impact

- **Frontend only** — no backend or API contract changes. Every action
  reuses `PATCH /api/entries/{id}` (already a partial update: a bare
  `{category_id}`, `{tag_ids}`, or `{recurring_transaction_id}` body is
  already sufficient) or the existing `DELETE /api/entries/{id}`, both
  unchanged. `entries.index.tsx` gains selection state and the toolbar; new
  shared components for the bulk-action pickers and the run/progress modal;
  new i18n keys under `entries.bulk.*`.
- Permission enforcement, recurring-transaction same-account validation,
  and category/tag usability rules are all already enforced server-side
  per entry — nothing about those checks changes.

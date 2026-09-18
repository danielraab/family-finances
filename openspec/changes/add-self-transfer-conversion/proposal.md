## Why

A user sometimes realizes, after logging a plain transaction, that it was
actually a transfer between two of their own (or shared) accounts. Today
there is no way to fix this in place — `add-self-transfer`'s design
deliberately made `kind` and `to_account_id` immutable after creation, so a
transaction can never become a self-transfer through `PATCH
/api/entries/{id}`. The only workaround is deleting the transaction and
manually recreating it as a self-transfer, losing its category, its
recurring-transaction link, and its id in the process anyway — with none of
the atomicity a first-class action could offer. A dedicated conversion
action gives this a proper, one-step, all-or-nothing path, without
loosening `kind`/`to_account_id`/`account_id` immutability for ordinary
edits at all.

## What Changes

- New endpoint `POST /api/entries/{id}/self-transfer`: converts an existing
  `kind: transaction` entry into a `kind: self_transfer` entry, atomically.
  In one database transaction, it soft-deletes the original entry and
  inserts a new `self_transfer` entry (a new `id`), then recomputes the
  balance-adjustment chain for both accounts now involved.
- The caller supplies the counterparty account and which role the
  *original* entry's account plays in the resulting transfer — sender
  (`account_id`, unchanged) or receiver (`to_account_id`) — rather than
  having it inferred from the amount's sign, since a self-transfer's amount
  is valid either sign on `account_id` and inference would be a guess, not
  a fact.
- Authorization is identical to creating a self-transfer directly: `append`+
  permission on both accounts involved (the original entry's account and
  the newly chosen one), both non-disabled, sharing the same `currency`.
  The one addition beyond Create's rule: the caller must be able to see the
  original entry at all (today's ordinary "no permission on either side" →
  `404` rule), since unlike Create there is a pre-existing entry to look up
  first.
- `category_id` always carries over unchanged. `recurring_transaction_id`
  carries over only when the original account keeps the sender role
  (`account_id`) — a recurring-transaction link is only ever validated
  against `account_id`, never `to_account_id`, so if the original account
  becomes the receiver, the link is dropped rather than silently
  re-pointed. `counterparty`/`location` are always dropped — rejected
  outright on any `self_transfer`, same as today.
- A new, clearly distinct frontend route hosts the conversion (not a
  mode-switch on the existing edit form), reached via a "Transfer to
  self-transfer" action on `/entries/{id}/edit`, offered only when the
  entry is a `transaction` and available for editing at all.
- Both this conversion's success redirect and the ordinary edit-save
  redirect now send the visitor to `/entries?last=true` instead of
  reconstructing filters by hand (today's edit-save redirect only restores
  `account_id`, silently dropping every other active filter —
  `category_id`, `tag_id`, `kind`, date range, search text, sort). The
  entries list persists its current filter/search/sort state to
  `localStorage` on every change; `?last=true`, alone, resolves to that
  remembered state (or a bare, unfiltered `/entries` if nothing is stored
  yet); `last` alongside any other filter parameter is ignored, since that
  combination is never expected to occur and an explicit filter should win
  if it somehow does.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `account-entries`: new `POST /api/entries/{id}/self-transfer` endpoint,
  its authorization rule (Create's self-transfer rule plus the ordinary
  entry-visibility gate), and its soft-delete-plus-create, per-field
  carry-over semantics.
- `web-client-entries`: new "Transfer to self-transfer" action and its
  dedicated conversion route; the `?last=true` filter-restoring redirect,
  replacing the current `account_id`-only edit-save redirect.

## Impact

- Backend: `backend/internal/entry/{entry,service,store,handler}.go` (new
  service method + handler route, soft-delete/create-with-carry-over
  logic, recompute for both accounts), `openapi/openapi.yaml` (+ synced
  `backend/openapi.yaml` and `frontend/src/api/schema.d.ts`), and tests
  across `internal/entry` and `internal/storage/{memory,postgres}`.
- Frontend: a new route file for the conversion UI,
  `entries.$entryId.edit.tsx` (new action, updated redirect),
  `entries.index.tsx` (persist filter state, resolve `?last=true`), new
  i18n keys.

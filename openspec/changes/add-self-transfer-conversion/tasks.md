## 1. Backend domain/service (`internal/entry`)

- [ ] 1.1 Design the conversion input shape (e.g. `to_account_id` +
      `original_account_role: "sender" | "receiver"`) and add it to
      `entry.go` alongside `New`/`Update`, documented like they are.
- [ ] 1.2 `Service.ConvertToSelfTransfer(ctx, callerID, id, in)` (naming
      TBD to match the package's existing verbs): fetch the entry the same
      way `Get`/`Update` do (`404` on zero permission); reject `400` if
      `Kind != KindTransaction`; apply `checkAccountCurrency` to both the
      entry's own `AccountID` and the requested counterparty account
      (mirroring `Create`'s `self_transfer` checks exactly — `append`+,
      non-disabled, matching currency); build the new entry's fields
      (`category_id` carried, `recurring_transaction_id` carried only when
      the original account keeps the sender role, `counterparty`/
      `location` dropped, `title`/`booking_timestamp`/`tag_ids` carried
      unchanged); soft-delete the original and create the new entry.
- [ ] 1.3 Add doc comments explaining the new method's relationship to
      `Create`'s self-transfer rule and `Delete`'s soft-delete, per the
      package's existing documentation style.

## 2. Backend storage layer

- [ ] 2.1 `storage/postgres/entry.go`: implement the soft-delete +
      insert as one DB transaction (reuse the existing `SoftDelete` and
      `Create` SQL where possible rather than duplicating it), followed by
      the existing recompute machinery for both accounts (mirrors how
      `self_transfer` Create/Update already recompute both sides — see
      `add-self-transfer`'s tasks 3.6/4.3).
- [ ] 2.2 Mirror the same behavior in `storage/memory/entry.go`.
- [ ] 2.3 `storage/postgres/self_transfer_test.go` (or a new file): cover
      the conversion against a real database — new entry created, original
      soft-deleted, both accounts' balances correct afterward, category
      kept, recurring link kept/dropped per role, counterparty/location
      dropped.

## 3. Backend handler and API contract

- [ ] 3.1 `internal/entry/handler.go`: `POST /api/entries/{id}/self-transfer`
      route, request parsing, `201` response with the new entry.
- [ ] 3.2 `openapi/openapi.yaml`: document the new endpoint (request body,
      `201` response reusing the existing `Entry` schema, `400`/`403`/
      `404`/`422` error cases per the spec's scenarios).
- [ ] 3.3 Run `cd backend && go generate ./...` to sync
      `backend/openapi.yaml`; run `cd frontend && pnpm generate:api` to
      regenerate `frontend/src/api/schema.d.ts`.
- [ ] 3.4 `internal/entry/handler_test.go`: response-conformance coverage
      (`internal/openapicheck.AssertResponse`) plus the authorization/
      validation scenarios from the spec (insufficient permission,
      disabled account, currency mismatch, wrong original kind, caller
      with zero visibility).

## 4. Frontend: conversion route

- [ ] 4.1 New route file for the conversion UI (e.g.
      `entries.$entryId.self-transfer.tsx`): counterparty-account picker
      (append+, non-disabled, same-currency, excluding the entry's own
      account — mirrors `entries.new.tsx`'s `toAccountOptions`), an
      explicit sender/receiver choice with no preselected default, submit
      calling the new endpoint, success navigating to
      `/entries?last=true`.
- [ ] 4.2 `entries.$entryId.edit.tsx`: add the "Transfer to self-transfer"
      action, shown only for a `kind: transaction` entry the visitor can
      edit; update the ordinary save-success redirect from its current
      `account_id`-only `search` to `/entries?last=true`.
- [ ] 4.3 Add the needed i18n keys (action label, conversion route's
      field labels/placeholders, sender/receiver choice copy, validation
      messages) to `src/i18n/locales/en.json`.

## 5. Frontend: entries-list filter persistence and `?last=true`

- [ ] 5.1 `entries.index.tsx`: persist the current filter/search/sort
      search-param state to `localStorage` on every change (wrapped in
      try/catch, consistent with this app's other `localStorage` usage).
- [ ] 5.2 `entries.index.tsx`: on load, when `search` has `last: true` and
      no other filter key set, replace-navigate to the persisted state (or
      a bare `/entries` if nothing is persisted); when `last` is present
      alongside any other filter key, ignore it and proceed with the
      explicit filters as today.
- [ ] 5.3 Extend `EntriesSearch`'s `validateSearch` to accept the `last`
      parameter.

## 6. Verification

- [ ] 6.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`
      (including `internal/storage/postgres` against a real database,
      `DATABASE_URL` set).
- [ ] 6.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 6.3 Manually exercise the golden path: convert a transaction with a
      category and a recurring-transaction link, choosing the entry's own
      account as sender — confirm the link survives; repeat choosing
      receiver — confirm the link is dropped; confirm the original entry
      404s afterward and the new entry's `id` differs; confirm both
      accounts' balances update correctly; confirm the edit-save and
      conversion-success redirects both restore previously active
      `/entries` filters via `?last=true`.

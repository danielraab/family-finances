## 1. Database migration

- [x] 1.1 Add a new migration under `backend/internal/storage/postgres/migrations/` adding a nullable `to_account_id` (FK to `accounts`) to `entries`, a `CHECK` constraint requiring it non-null exactly when `kind = 'self_transfer'` and null otherwise (mirroring the existing `balance_reading` CHECK for `balance_adjustment`), and an index on `to_account_id` (mirroring `entries_recurring_transaction_idx`).
- [x] 1.2 Extend the `entries.kind` CHECK/enum constraint (or equivalent) to allow `self_transfer`.

## 2. Backend domain model (`internal/entry`)

- [x] 2.1 Add `KindSelfTransfer` to `entry.Kind` and its `valid()` method (`entry.go`).
- [x] 2.2 Add `ToAccountID *string` to `Entry`, `New`, and `Update`; add `ToAccountCurrency`/`ToAccountName` (omitempty) to `Entry`, resolved server-side like `AccountCurrency`/`CreatedByName`. (Update deliberately gets no `ToAccountID` field — immutable, see design.md.)
- [x] 2.3 Update `validateNew` (and the mirrored update-time validation): `to_account_id` required and non-empty exactly for `self_transfer`, rejected for `transaction`/`balance_adjustment`; `account_id != to_account_id` for `self_transfer`; category requirement narrows to `KindTransaction` only (optional for both `balance_adjustment` and `self_transfer`); counterparty/location rejected for `self_transfer` too (already rejected for non-`transaction`).
- [x] 2.4 Add doc comments on `Entry`/`New`/`Update`/`Kind` explaining the new value and field, following the existing style.

## 3. Backend service layer (`internal/entry/service.go`)

- [x] 3.1 On `Create` with `kind: self_transfer`: check `append`+ permission and non-disabled state on both `AccountID` and `ToAccountID` (extend/reuse the existing account-check helper), and reject a currency mismatch between the two accounts (`ErrInvalidValue`).
- [x] 3.2 On `Update` of an existing `self_transfer` entry: require the caller to currently hold `append`+ on both accounts (both the stored `AccountID`/`ToAccountID`, since neither is reassignable post-creation for this kind) before permitting any field change; return the existing forbidden-error mapping when either check fails.
- [x] 3.3 On `Delete` of a `self_transfer` entry: apply the existing per-account entry_admin/owner-or-append-and-created-by rule against whichever of the two accounts the caller currently holds permission on (no requirement on the other side).
- [x] 3.4 `Sum`: include `self_transfer` alongside `transaction` in the currency-summed query (still excluding `balance_adjustment`).
- [x] 3.5 `FlowSummary`: confirm/extend bucketing so a `self_transfer` contributes to both its accounts' own per-account rows (already bucketed per `AccountID` internally) — verify the "as stored" / "sign flipped" pairing matches the listing logic from task 4. (Postgres needed the `entry_legs`-based rewrite; memory's `matchingRows` already buckets per leg with no further change.)
- [x] 3.6 Recompute-on-mutation: for a `self_transfer` create/update/delete, run the existing balance-adjustment recompute independently for both `AccountID` and `ToAccountID` (two calls instead of one).

## 4. Backend storage layer

- [x] 4.1 `storage/postgres/entry.go`: rewrite the listing query (`buildWhere`/list) as a `UNION ALL` of two projections — the existing one (`account_id`, `amount` as stored) and a second, active only for `self_transfer` rows, keyed by `to_account_id` with `amount` negated — both passing through the same filters, sort, and keyset cursor. Ensure the cursor/JSON identity disambiguates the two projections of one row (they carry opposite-signed `amount`, per design.md). (Implemented as an `entry_legs` SQL view, migration 0030; added `Cursor.Native` to disambiguate the two legs' identical `(booking_timestamp, id)` tuple under the default sort, which the original design note underestimated.)
- [x] 4.2 `storage/postgres/entry.go`: update `Balance(asOf)` to add `CASE WHEN account_id = $1 THEN amount WHEN to_account_id = $1 THEN -amount END`.
- [x] 4.3 `storage/postgres/entry.go`: update the balance-adjustment recompute helpers (`recomputeFrom`/`findAdjustment`/`setAmount`) to be invoked per-account, called for both sides of a `self_transfer` mutation (task 3.6).
- [x] 4.4 `storage/postgres/entry.go`: resolve `to_account_name`/`to_account_currency` in the same query/pass that already resolves `account_currency`/`created_by_name`.
- [x] 4.5 Mirror all of the above in `storage/memory/entry.go` (Go-loop equivalents), keeping it the fast unit/handler-test store.
- [x] 4.6 `storage/postgres/entry_test.go` (and memory-store tests, if any exist there): cover the union listing (once/twice), live balance, and recompute-on-both-sides behavior with real scenarios. (New `storage/postgres/self_transfer_test.go`, run against a real database — caught and drove the fix for a real bug: `setAmount`/`setAmountLocked`'s "balance strictly before" calculation only checked `account_id`, never `to_account_id`, so a self-transfer's contribution to its receiving account's balance-adjustment recompute was silently dropped in both stores.)

## 5. Backend `internal/entry` handler and API contract

- [x] 5.1 `internal/entry/handler.go`: accept `to_account_id` on `POST`/`PATCH` bodies; include `to_account_id`, `to_account_name`, `to_account_currency` on entry responses. (`PATCH` deliberately gets no `to_account_id` field — immutable.)
- [x] 5.2 `openapi/openapi.yaml`: add `self_transfer` to the `EntryKind` enum, add `to_account_id`/`to_account_name`/`to_account_currency` to the `Entry` schema and `to_account_id` to the create request body; document the new validation errors (currency mismatch, same-account transfer, insufficient permission on either side) and added `403` responses to `PATCH`/`DELETE /api/entries/{id}` (previously undeclared, needed once a self-transfer's edit/delete can reach `ErrForbidden` in a directly testable way). Also documented `accounts`' new currency-immutability `422` on `PATCH /api/accounts/{id}`.
- [x] 5.3 Run `cd backend && go generate ./...` to sync `backend/openapi.yaml`; run `cd frontend && pnpm generate:api` to regenerate `frontend/src/api/schema.d.ts`.
- [x] 5.4 `internal/entry/handler_test.go`: add response-conformance coverage (`internal/openapicheck.AssertResponse`) for `self_transfer` create/get responses.

## 6. Backend `internal/account`: currency immutability

- [x] 6.1 Declare `EntryLookup` interface in `internal/account` (`HasEntries(ctx, accountID) (bool, error)`).
- [x] 6.2 `internal/account/service.go`: add `SetEntryLookup(lookup)` setter (a nil lookup fails open — currency stays editable — matching how the rule never applied before this change); `Service.Update` rejects a `Currency` change when `HasEntries` reports `true` (`ErrInvalidValue`).
- [x] 6.3 `internal/entry/service.go` (or store): add a small `HasEntries` method satisfying the interface structurally — a cheap existence query, not a full scan.
- [x] 6.4 `main.go`: wire `accountSvc.SetEntryLookup(entrySvc)` after both services are constructed, mirroring the existing `account.SetUserLookup`/`recurringSvc.SetEntryLookup` wiring.
- [x] 6.5 `internal/account/service_test.go`: cover the locked-once-entries-exist and still-editable-with-no-entries cases.
- [x] 6.6 `openapi/openapi.yaml`: document the new `422` case on `PATCH /api/accounts/{id}` for a currency change on an account with entries (regenerated synced files per 5.3).

## 7. Frontend: entry form

- [x] 7.1 `frontend/src/routes/entries.new.tsx`: add the `self_transfer` radio option; add "To account" `<select>` state, populated from accounts filtered to append+, non-disabled, same-currency-as-selected-"from", excluding the "from" account; hide category/counterparty/location for `self_transfer` (same treatment as `balance_adjustment`); include `to_account_id` in the `POST` body only for `self_transfer`. (Category is shown with the "No category" placeholder, same as `balance_adjustment` — only counterparty/location are actually hidden, matching the spec's "required unless kind is balance_adjustment or self_transfer" wording.)
- [x] 7.2 `frontend/src/routes/entries.$entryId.edit.tsx`: same kind option and "To account" field, pre-populated and locked the same way `account_id` is; render the whole form read-only when the visitor lacks current append+ on either account of a `self_transfer` (even if they'd otherwise qualify via the entry's own `account_id`); keep the Delete action available under the plain per-account rule regardless of the other account's accessibility. (Fetches the receiving account's own permission via a second `GET /api/accounts/{id}`, since the entry response only carries its name/currency, not the caller's tier on it.)
- [x] 7.3 Add the needed i18n keys (`entries.kind.selfTransfer`, "to account" label/placeholder, currency-lock note, etc.) to `src/i18n/locales/en.json` (`de.json` left lagging, per the repo's stated policy — CI's coverage check is informational only).

## 8. Frontend: self-transfer badge

- [x] 8.1 Add `SelfTransferBadge.tsx` (or extend the existing badge component) mirroring `RecurringTransactionBadge.tsx`: renders for a `kind: self_transfer` entry, links to `/entries/{id}/edit`, renders null otherwise. (New `SelfTransferIcon.tsx` glyph, distinct from `RecurringIcon`'s loop.)
- [x] 8.2 Wire it into the ledger row component and the `/reports` results table, alongside the existing recurring-transaction badge. (Also caught and fixed a real bug this surfaced: `entries.index.tsx`, `reports.tsx`, and `EntryListCard.tsx` all keyed their row lists by `entry.id` alone, which now collides for a self-transfer's two same-id occurrences — changed to `${entry.id}-${entry.account_id}`.)

## 9. Frontend: account edit form

- [x] 9.1 `frontend/src/routes/accounts.$accountId.edit.tsx` (or `AccountForm.tsx`): disable the `currency` field and show an inline note when the account has any entries (use the account's existing entry-count/recent-entries data already fetched for this page, or a cheap dedicated check if not already available). (Added a cheap `GET /api/entries?account_id={id}&limit=1` check; `AccountForm` gained a `currencyLocked` prop, and `CurrencySelect` a `disabled` prop.)
- [x] 9.2 Add the corresponding i18n key for the inline note.

## 10. Verification

- [x] 10.1 `cd backend && gofmt -l . && go vet ./... && go test ./...` (including `internal/storage/postgres` against a real database, `DATABASE_URL` set).
- [x] 10.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 10.3 Manually exercised the golden path against a running backend (curl, real Postgres, magic-link auth via mailpit) rather than the browser — this sandbox has no root access to install headless Chromium's system libraries, so a Playwright screenshot pass wasn't possible; recommend a follow-up manual/browser check before shipping. Verified via the API: self-transfer create with append+ on both accounts; both accounts' balances update correctly (-1000/+1000); listing once when filtered to either account alone, twice when both are in scope; summary nets to zero across both accounts; PATCH updates unrelated fields but rejects an `account_id` change; DELETE succeeds and the entry 404s afterward; the currency-lock rejects a `PATCH .../currency` change on an account with entries (400) while still allowing other fields (e.g. `title`) to update. Also verified, via this exercise, three real bugs the tests below didn't catch until this level: (1) `Create` never actually stored a `self_transfer`'s `amount` in either store (a stale `Kind == KindTransaction` check), (2) `setAmount`/`setAmountLocked`'s balance-adjustment recompute never accounted for `to_account_id`'s contribution, (3) three list/card components keyed their rows by `entry.id` alone, which collides now that one entry can render twice.

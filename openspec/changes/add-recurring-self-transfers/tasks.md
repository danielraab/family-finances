## 1. Database migration

- [ ] 1.1 Add a migration under `backend/internal/storage/postgres/migrations/` adding `kind text NOT NULL DEFAULT 'transaction'` to `recurring_transactions` with a `CHECK (kind IN ('transaction', 'self_transfer'))`, and a nullable `to_account_id uuid REFERENCES accounts(id)`.
- [ ] 1.2 Add `CHECK ((kind = 'self_transfer') = (to_account_id IS NOT NULL))` and `CHECK (to_account_id IS NULL OR to_account_id != account_id)`, copying `entries`' own two constraints from migration 0030.
- [ ] 1.3 Drop `category_id`'s `NOT NULL` and replace it with `CHECK ((kind = 'transaction' AND category_id IS NOT NULL) OR kind = 'self_transfer')`, mirroring `entries_check`.
- [ ] 1.4 Add `recurring_transactions_to_account_idx ON recurring_transactions (to_account_id) WHERE to_account_id IS NOT NULL`.
- [ ] 1.5 Create the `recurring_transaction_legs` view: the stored row with `true AS native`, `UNION ALL` the `self_transfer`-only projection with `account_id`/`to_account_id` swapped, `-amount`, and `false AS native`. Document it with the same comment `entry_legs` carries, pointing at design.md.
- [ ] 1.6 Drop the `DEFAULT 'transaction'` once existing rows are backfilled, so `kind` is always written explicitly from here on (matching how every other required column behaves).

## 2. Backend domain model (`internal/recurringtransaction`)

- [ ] 2.1 Add a `Kind` type with `KindTransaction`/`KindSelfTransfer` and a `valid()` method, and a package-doc note on why `balance_adjustment` is still absent (design.md's Context).
- [ ] 2.2 Add `Kind` and `ToAccountID *string` to `RecurringTransaction` and `New`; add `ToAccountName`/`ToAccountCurrency` (omitempty), resolved server-side like `AccountCurrency`. Add a `Native bool` field for which leg a listed row is, following `entry.Cursor.Native`'s precedent.
- [ ] 2.3 Deliberately add neither `Kind` nor `ToAccountID` to `Update` — both immutable — and confirm the handler's request struct has no field for either, so `DisallowUnknownFields` rejects an attempt.
- [ ] 2.4 Update `validateNew`: `to_account_id` required and non-empty exactly for `self_transfer` and rejected otherwise; `account_id != to_account_id`; the category requirement narrows to `KindTransaction`; `counterparty`/`location` rejected for `self_transfer`. Mirror the counterparty/location half in `validateUpdate` against the stored kind.
- [ ] 2.5 Add `SelfTransfers` to `Filter` (and `PreviewFilter` keeps none — preview is always both legs), as a small three-valued type covering exclude / native only / both legs, so `Store` sees one resolved mode rather than two booleans.

## 3. Backend service layer (`internal/recurringtransaction/service.go`)

- [ ] 3.1 `Create` with `kind: self_transfer`: run the existing `checkAccount` against `ToAccountID` as well, and reject a currency mismatch between the two accounts with `ErrInvalidValue`.
- [ ] 3.2 `Update` of a `self_transfer`: require the caller to currently hold `append`+ on both stored accounts before permitting any field change, returning `ErrForbidden` when either fails — on top of the existing tier/created-by rule.
- [ ] 3.3 `Get`/`authorizeWrite`: accept `view`+ on either account for reading a `self_transfer`; leave `Delete`'s rule evaluated against `AccountID` alone.
- [ ] 3.4 `decorate`: resolve `ToAccountName`/`ToAccountCurrency` for a `self_transfer` unconditionally, the way `AccountCurrency` already resolves.
- [ ] 3.5 `List`/`Summary`: pass the caller's resolved self-transfer mode through to `Store`; leave `PerYearAmount` untouched, since the view already negates the receiving leg's `amount`.
- [ ] 3.6 `Preview`: query with both legs unconditionally, and carry `Kind`/`ToAccountID`/`ToAccountName` onto `PreviewItem`.
- [ ] 3.7 Add a `HasSelfTransferTemplate(ctx, accountID) (bool, error)` method for `internal/account`'s currency lock (task 6), checking both sides, excluding soft-deleted rows.

## 4. Backend storage layer

- [ ] 4.1 `storage/postgres/recurringtransaction.go`: point `List` at `recurring_transaction_legs`, selecting `native`, and build the mode's `WHERE` per design.md (`native AND kind = 'transaction'` / `native` / no extra clause), keeping `account_id = ANY(...)` in every mode.
- [ ] 4.2 Order by `created_at, id, native DESC` so a template's outgoing leg sorts immediately before its incoming one.
- [ ] 4.3 `Create`/`Update`/`Get` keep reading `recurring_transactions` directly, never the view — `GET /api/recurring-transactions/{id}` is always the stored orientation (design.md's materialization decision).
- [ ] 4.4 Implement `HasSelfTransferTemplate` in both stores.
- [ ] 4.5 Mirror all of the above in `storage/memory`, including the leg expansion and the three modes.

## 5. Backend handler and API contract

- [ ] 5.1 Accept and validate `include_self_transfer` and `self_transfer_both_legs` on `GET /api/recurring-transactions` and `GET /api/recurring-transactions/summary`, resolving them to the domain's three-valued mode and ignoring the second when the first is false.
- [ ] 5.2 Accept `kind`/`to_account_id` on `POST /api/recurring-transactions`; reject both on `PATCH`.
- [ ] 5.3 Update `openapi/openapi.yaml`: the `RecurringTransaction` schema's `kind`/`to_account_id`/`to_account_name`/`to_account_currency`, the create body, the two query parameters and their three modes, the non-unique `id` within a both-legs response, the preview item's new fields, and preview's deliberate lack of the parameters.
- [ ] 5.4 Regenerate both committed artifacts: `cd backend && go generate ./...` and `cd frontend && pnpm generate:api`.

## 6. Backend `internal/account`: currency immutability

- [ ] 6.1 Declare a second narrow lookup interface in `internal/account` for "is this account named by a self-transfer recurring transaction", satisfied structurally by `*recurringtransaction.Service`, with a `WithXxx` option mirroring `WithEntryLookup`.
- [ ] 6.2 Wire it in `main.go` after both services exist.
- [ ] 6.3 Extend `Service.Update`'s currency check to reject when either lookup reports true.

## 7. Frontend: the `/recurring` filter

- [ ] 7.1 Add the two flags to `/recurring`'s route search params with `false` defaults, and thread them through both the list and summary fetches.
- [ ] 7.2 Render the filter control above the table: "Show self-transfers", and the second checkbox only while the first is checked. Unchecking the first clears the second from the URL.
- [ ] 7.3 Key rows by `${id}-${native}` rather than `id`, since both legs share an id.

## 8. Frontend: the recurring form

- [ ] 8.1 Add a kind selector to `RecurringTransactionForm`, mirroring `/entries/new`'s: selecting "Self-transfer" reveals a "To account" picker, hides counterparty and location, and drops the category requirement.
- [ ] 8.2 Filter the "To account" options to `append`+, non-disabled, same-currency, excluding the selected source account — the same predicate the entry form already applies.
- [ ] 8.3 On the edit page, render the kind and the to-account as immutable, and make the whole form read-only with an explanation when the visitor lacks `append`+ on both accounts.

## 9. Frontend: rows, summary modal, and materialization

- [ ] 9.1 Show the other account and the row's direction on a listed `self_transfer`, reusing `SelfTransferBadge`/`SelfTransferIcon`; render an empty category cell cleanly.
- [ ] 9.2 Add the to-account to the recurring summary modal.
- [ ] 9.3 Extend `entries.new.tsx`'s recurring prefill to carry `kind` and `to_account_id` from the fetched template, and confirm the existing fetch-by-id keeps a receiving-side row booking against the template's sending account.
- [ ] 9.4 Render a previewed `self_transfer` occurrence in `UpcomingBlock` the way a self-transfer entry renders in the ledger.

## 10. Translations

- [ ] 10.1 Add the new keys (kind selector labels, "To account", the two filter labels, the row's direction wording) to both locales, keeping the i18n coverage job at 100%.

## 11. Verification

- [ ] 11.1 `cd backend && go vet ./... && go test ./...`, including new tests for each of the three list modes, the both-accounts create/edit rule, the currency mismatch, the per-kind field rules, the summary's cancellation, and the currency lock.
- [ ] 11.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build` — run `tsc` locally, since CI's `frontend` job runs only `pnpm lint` and `pnpm build`.
- [ ] 11.3 Drive the app in Chromium against a stubbed `/api`: toggle both flags and confirm the list, the second checkbox's reveal, the URL, and the total all follow; create a self-transfer template and book from both its sides; check `/recurring` at 390px and 1280px in both themes.
- [ ] 11.4 Confirm the `contract` job passes — the spec and both generated artifacts in sync.

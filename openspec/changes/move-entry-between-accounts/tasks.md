## 1. Backend: entry service and store

- [ ] 1.1 Add `AccountID *string` to `entry.Update`
      (`backend/internal/entry/entry.go`), documenting that a non-nil value
      changes the entry's account.
- [ ] 1.2 Extract the account ownership/disabled check shared by
      `Service.Create` and `Service.Update` into a helper (e.g.
      `checkAccount(ctx, ownerID, accountID) error` returning
      `ErrInvalidValue` on ownership mismatch or `ErrAccountDisabled` when
      disabled) and use it from `Service.Create`; verify `go test
      ./internal/entry/...` still passes unchanged.
- [ ] 1.3 In `Service.Update`, when `upd.AccountID != nil`, call the same
      helper against the new account id before persisting; verify a unit
      test moving an entry to another owned, non-disabled account succeeds
      and updates `AccountID`.
- [ ] 1.4 Verify a unit test moving an entry to an account owned by a
      different user is rejected with `ErrInvalidValue` and the entry is
      unchanged.
- [ ] 1.5 Verify a unit test moving an entry to a disabled account owned by
      the caller is rejected with `ErrAccountDisabled` and the entry is
      unchanged.
- [ ] 1.6 Verify a unit test moving a `kind: balance_adjustment` entry
      succeeds the same way as a `transaction` entry.
- [ ] 1.7 Verify a unit test moving an entry between accounts with
      different `currency` values succeeds and leaves `amount` unchanged.
- [ ] 1.8 Update `storage/memory` and `storage/postgres` `Store.Update`
      implementations to persist a changed `account_id` (check both already
      apply `Update`'s fields generically before assuming no change is
      needed); verify `internal/storage/postgres` integration tests pass
      locally against `DATABASE_URL` per `backend/AGENTS.md`.

## 2. Backend: HTTP handler and OpenAPI contract

- [ ] 2.1 Add `account_id` to `entryUpdateBody`
      (`backend/internal/entry/handler.go`) and pass it through to
      `Update.AccountID`.
- [ ] 2.2 Update `openapi/openapi.yaml`'s `EntryUpdate` schema to include
      `account_id` and drop the "account_id and kind are immutable" wording
      on `PATCH /api/entries/{id}` (now only `kind` is immutable); add a
      `422` response for the disabled-account case, alongside the existing
      `400`.
- [ ] 2.3 Run `cd backend && go generate ./...` to sync
      `backend/openapi.yaml` from the root spec; verify no diff remains
      uncommitted.
- [ ] 2.4 Add/extend a handler test in `backend/internal/entry/handler_test.go`
      covering: successful move (`200`, new `account_id` in the response),
      move to another user's account (`400`), move to a disabled account
      (`422`) — each asserted against the OpenAPI contract via
      `internal/openapicheck.AssertResponse`.
- [ ] 2.5 Verify `gofmt -l .`, `go vet ./...`, and `go test ./...` are clean
      from `backend/`.

## 3. Frontend: generated types

- [ ] 3.1 Run `cd frontend && pnpm generate:api` to regenerate
      `src/api/schema.d.ts` from the updated OpenAPI spec; verify the diff
      only adds `account_id` to the entry-update request type.

## 4. Frontend: account field lock/unlock

- [ ] 4.1 In `entries.$entryId.edit.tsx`, replace the disabled account
      `<input>` with a component that renders locked (disabled display +
      pencil icon button) by default, and add local state tracking whether
      it is unlocked and the pending selected `account_id`.
- [ ] 4.2 On unlock, fetch/reuse the visitor's own accounts (mirroring how
      categories are already fetched for this page) and render a `<select>`
      restricted to non-deleted, non-disabled accounts, defaulted to the
      entry's current account; verify manually in the browser that
      unlocking shows the select with the current account pre-selected.
- [ ] 4.3 Ensure the entry's current account still appears in the unlocked
      `<select>` even if it has since become disabled, but is not
      selectable once a different account is chosen — mirroring the
      existing disabled-category handling in the same file; verify
      manually against a disabled account fixture.
- [ ] 4.4 Add the cancel button next to the unlocked field: clicking it
      discards the pending selection, restores `account_id` to the entry's
      original value, and re-locks the field; verify manually.

## 5. Frontend: currency warning and save confirmation

- [ ] 5.1 Compare the currently selected account's `currency` to the
      entry's original account's `currency` and render an inline warning
      when they differ, cleared when they match; verify manually by
      selecting accounts with matching and differing currencies.
- [ ] 5.2 On submit, if the selected `account_id` still equals the entry's
      original value, submit immediately as today (no behavior change);
      verify the existing save flow is unaffected.
- [ ] 5.3 On submit, if the selected `account_id` differs, open a
      confirmation `Dialog` (mirroring the existing delete-confirmation
      dialog in the same file) naming the destination account and
      repeating the currency warning when applicable, instead of submitting
      immediately.
- [ ] 5.4 Wire the dialog's confirm action to submit the `PATCH` (including
      `account_id` and any other edited fields) and its cancel action to
      close the dialog without sending a request, leaving the pending
      selection in place; verify manually for both paths.

## 6. i18n and translations

- [ ] 6.1 Add new `en.json` keys for: the unlock button's accessible label,
      the cancel button, the currency-mismatch warning text, and the
      account-change confirmation dialog's title/body/confirm/cancel
      labels; verify `pnpm lint` passes (no unused/missing keys) — `de.json`
      may lag per the i18n coverage policy in `frontend/AGENTS.md`.

## 7. Final verification

- [ ] 7.1 Run `pnpm lint`, `pnpm exec tsc`, and `pnpm build` from
      `frontend/`; verify all pass.
- [ ] 7.2 Manually exercise the full flow in a running app (`docker compose
      up -d db`, `go run .` in `backend/`, `pnpm dev` in `frontend/`): open
      an entry, unlock the account field, select a different account of the
      same currency, save, confirm the dialog, and verify the entry now
      lists under the new account and its balance updates on both accounts.
- [ ] 7.3 Manually verify the disabled-target rejection surfaces as an
      inline error (not a silent failure) when a disabled account is
      somehow selected as the target (e.g. via direct API call), and that
      the cross-currency warning appears when selecting a different-currency
      account.

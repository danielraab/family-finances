## 1. Filter struct and resolution

- [x] 1.1 Add `AllAccounts bool` to `entry.Filter` in
      `backend/internal/entry/entry.go`, documented as "no account_id
      restriction for this query — set only by `resolveFilter` when the
      caller's own category permission authorizes it."
- [x] 1.2 In `backend/internal/entry/service.go`'s `resolveFilter`,
      always resolve `s.categories.Subtree(ctx, callerID,
      *f.CategoryID)` when `f.CategoryID != nil` (regardless of
      `CategoryMode`), reusing its result for `ModeSubtree`'s
      `f.CategoryIDs` as today; for `ModeExact`, keep building
      `f.CategoryIDs` from the raw id alone, using `Subtree`'s
      non-empty-ness only as the permission signal.
- [x] 1.3 Set `f.AllAccounts = true` when that permission signal is
      non-empty and the caller did not supply an explicit
      `AccountIDs` filter on the same request; otherwise leave the
      existing `visible`-intersection behavior unchanged.
- [x] 1.4 Update `resolveFilter`'s doc comment to describe the new
      widening and its two guards (permission required; suppressed by
      an explicit account filter).

## 2. Storage backends

- [x] 2.1 `backend/internal/storage/postgres/entry.go`: `buildWhere`
      skips the `entries.account_id = ANY(...)` clause when
      `f.AllAccounts` is true.
- [x] 2.2 `backend/internal/storage/postgres/entry.go`: `List` and
      `Sum`'s `if len(f.AccountIDs) == 0 { return ... }` guards become
      `if !f.AllAccounts && len(f.AccountIDs) == 0`.
- [x] 2.3 `backend/internal/storage/memory/entry.go`: `matchingRows`
      skips the `accountSet[e.AccountID]` membership check when
      `f.AllAccounts` is true, and its own empty-`AccountIDs`
      short-circuit gets the same `!f.AllAccounts` guard.
- [x] 2.4 Update the doc comments on `buildWhere` and `matchingRows`
      that currently state "f.AccountIDs is always the sole
      caller-scoping mechanism" — no longer true when `AllAccounts` is
      set.

## 3. Tests

- [x] 3.1 In `backend/internal/entry` (service-level tests, run against
      both storage backends per the package's existing harness): a
      category owner's own entry on an account only a share recipient
      can access is returned when the recipient filters by that shared
      category.
- [x] 3.2 Symmetric case: the category's real owner filtering by their
      own category sees an entry created (by a share recipient with
      `append` permission) against an account the owner has no access
      to.
- [x] 3.3 An unfiltered `List`/`Sum` call from either side of the share
      does not include the other's inaccessible-account entries.
- [x] 3.4 Supplying an explicit `account_id` filter alongside the
      shared `category_id` filter excludes an entry on a different,
      inaccessible account (only entries matching both filters
      return).
- [x] 3.5 Filtering by a category the caller has no permission on at
      all (owner, view, or append) never sets `AllAccounts` and never
      returns another user's entries — covers both `ModeSubtree` and
      `ModeExact`.
- [x] 3.6 `Sum`'s report totals correctly include the widened entry's
      amount under its own account's currency (confirms the
      already-permission-blind `AccountLookup.Access` currency lookup
      continues to work for a cross-account entry, per design.md — no
      production code change expected here, this is a regression
      guard).

## 4. Verification

- [x] 4.1 `cd backend && go build ./...` and `go vet ./...`.
- [x] 4.2 `cd backend && go test ./...` (exercises both storage
      backends via the existing suite).
- [x] 4.3 Manual smoke check against the running dev stack: share a
      category (view tier) from one seeded user to another whose
      accounts are otherwise unshared; as the recipient, filter
      `/entries` and `/reports` by that category and confirm the
      owner's entries appear with the existing "not shared" account
      placeholder; confirm the unfiltered `/entries` list is
      unaffected.

## 5. `account_currency` on Entry (found during 4.3's smoke check)

- [x] 5.1 Add `account_currency` to the `Entry` schema in
      `openapi/openapi.yaml` — optional, not in `required` (mirrors
      `created_by_name`'s existing precedent: always populated in
      production/Postgres, but the memory store used for domain/handler
      tests never sets it), documented per specs/account-entries.
- [x] 5.2 Regenerate `backend/openapi.yaml` (`cd backend && go generate
      ./...`) and `frontend/src/api/schema.d.ts` (`cd frontend && pnpm
      generate:api`); confirm no other drift.
- [x] 5.3 Add `AccountCurrency string` to `entry.Entry`
      (`backend/internal/entry/entry.go`), `json:"account_currency"`.
- [x] 5.4 `backend/internal/storage/postgres/entry.go`: extend
      `entryCols`/`scanEntry` with a correlated subquery into
      `accounts.currency`, mirroring `created_by_name`'s existing
      reach-into-another-domain's-table pattern in the same file.
- [x] 5.5 Leave `backend/internal/storage/memory/entry.go` unpopulated
      (stays `""`), matching `CreatedByName`'s existing precedent there.
- [x] 5.6 `frontend/src/routes/entries.index.tsx` and `reports.tsx`:
      delete the local `accountCurrency(accountId)` helper and its two
      call sites; read `entry.account_currency` directly in both
      `formatAmount`/`formatSignedAmount` calls.
- [x] 5.7 Verification: `cd backend && go build ./... && go vet ./...
      && go test ./...`; `cd frontend && pnpm lint && pnpm exec tsc &&
      pnpm build`; re-run the 4.3 manual smoke check and confirm the
      recipient now sees correctly currency-labeled amounts (not bare
      numbers) for the owner's cross-account entries in both
      `/entries` and `/reports`.

## 1. API contract

- [x] 1.1 Add `RecurringTransaction` schema (account_id, title, description,
      category_id, counterparty, location, tag_ids, amount, interval_unit,
      interval_count, starts_on, ends_on, created_by, created_by_name,
      account_currency, plus computed `per_year_amount`, `ended`,
      `next_suggested_date`) and an `IntervalUnit` enum (`day`/`week`/
      `month`/`year`) to `openapi/openapi.yaml`
- [x] 1.2 Add `GET`/`POST /api/recurring-transactions`,
      `GET`/`PATCH`/`DELETE /api/recurring-transactions/{id}`, and
      `GET /api/recurring-transactions/summary` paths to
      `openapi/openapi.yaml`, mirroring the existing entries paths'
      parameter/response shapes and error responses
- [x] 1.3 Add `recurring_transaction_id` (nullable) to the `Entry` schema in
      `openapi/openapi.yaml`, settable in the create/update request bodies
- [x] 1.4 Run `cd backend && go generate ./...` to sync
      `backend/openapi.yaml` and verify it matches the root spec
- [x] 1.5 Run `cd frontend && pnpm generate:api` and verify
      `frontend/src/api/schema.d.ts` regenerates with no manual edits needed

## 2. Backend: `internal/recurringtransaction` domain package

- [x] 2.1 Create `recurringtransaction.go`: domain type, `Unit` enum
      (`day`/`week`/`month`/`year`) with validation, `New`/`Update` input
      types, and field validation (title required, category required,
      interval_count positive, ends_on >= starts_on) — verify with unit
      tests requiring no store
- [x] 2.2 Implement `PerYearAmount(amount, unit, count) int64` and
      `Ended(endsOn, now) bool` as pure functions — verify with table-driven
      unit tests covering month/year (exact) and week/day (365.25-based)
      per the spec's worked examples
- [x] 2.3 Implement `Advance(t time.Time, unit Unit, count int) time.Time`
      (calendar-month/year stepping with end-of-month clamping; fixed
      day-count for week/day) — verify with unit tests including the
      Jan-31-plus-one-month-clamps-to-Feb-28 scenario from the spec
- [x] 2.4 Declare `AccountLookup`, `CategoryLookup`, `TagLookup` interfaces
      (mirroring `internal/entry`'s shape) and a new `EntryLookup` interface
      (`LatestLinkedBookingTime`, `LinkedCount`) in `store.go`, plus sentinel
      errors (`ErrNotFound`, `ErrInvalidValue`, `ErrForbidden`, `ErrInUse`)
- [x] 2.5 Implement `service.go`: `Create`, `Get`, `Update`, `Delete`
      (rejecting `ErrInUse` when `EntryLookup.LinkedCount > 0`), `List`,
      `Summary` — permission checks mirroring `internal/entry`'s
      `append`/`entry_admin`/`owner` tiers — verify with `storage/memory`-
      backed service tests covering each permission scenario from the spec
- [x] 2.6 Implement `handler.go`: `http.Handler` for
      `/api/recurring-transactions...`, mapping requests/responses per the
      OpenAPI contract, including `NextSuggestedDate` resolution via
      `EntryLookup` — verify with `httptest`-based handler tests asserting
      status codes and JSON shape, including an
      `internal/openapicheck.AssertResponse` conformance check per new
      endpoint

## 3. Backend: storage

- [x] 3.1 Add migration `backend/internal/storage/postgres/migrations/0025_recurring_transactions.sql`
      creating `recurring_transactions` (mirroring `entries`' relevant
      columns minus kind/balance/booking_timestamp, plus interval_unit,
      interval_count, starts_on, ends_on, deleted_at) and adding
      `entries.recurring_transaction_id` (nullable FK) — verify migration
      applies cleanly against a fresh database
- [x] 3.2 Implement `internal/storage/memory`'s `RecurringTransactionStore`
      (in-memory, matching the `Store` interface) — verify existing
      `go test ./...` still passes with it wired into service tests
- [x] 3.3 Implement `internal/storage/postgres`'s `RecurringTransactionStore`,
      including the `LatestLinkedBookingTime`/`LinkedCount` queries against
      `entries` — verify with `internal/storage/postgres` integration tests
      (behind `DATABASE_URL`, per existing convention)
- [x] 3.4 Extend `internal/storage/memory` and `internal/storage/postgres`'s
      existing `EntryStore` implementations to persist/read
      `recurring_transaction_id` on create/update, and to validate it
      references a recurring transaction on the same account — verify with
      entry store tests covering the new field

## 4. Backend: entry package changes

- [x] 4.1 Add `RecurringTransactionID *string` to `entry.Entry`, `entry.New`,
      and `entry.Update` (using the existing `OptionalID` pattern for
      clearability), with `omitempty` JSON tag — verify `go vet`/`go build`
- [x] 4.2 Add a narrow `RecurringTransactionLookup` interface to
      `internal/entry` (`SameAccount(ctx, id, accountID) (bool, error)`),
      satisfied structurally by `*recurringtransaction.Service`, consulted
      only when `recurring_transaction_id` is newly set — verify with a
      service test asserting cross-account linking is rejected (`400`)

## 5. Backend: wiring

- [x] 5.1 Wire `recurringtransaction.Service`/`Handler` in `main.go`
      (`storage/postgres` store, mounted at `/api/recurring-transactions/`),
      and wire `entry.WithRecurringTransactionLookup` /
      `recurringtransaction.WithEntryLookup` for the two cross-package
      lookups — verify `go build ./...` and `go run .` start cleanly
      against a local database

## 6. Backend: validation

- [x] 6.1 Run `gofmt -l .`, `go vet ./...`, `go test ./...` in `backend/`
      and verify all pass with no output from `gofmt -l`

## 7. Frontend: routes and nav

- [ ] 7.1 Add `src/routes/recurring.tsx` (auth-gated layout, redirect
      anonymous visitors to `/login`, mirroring `entries.tsx`) and
      `src/routes/recurring.index.tsx` (list page) — verify `pnpm dev`
      renders `/recurring` for an authenticated user
- [ ] 7.2 Add "Recurring" to `Sidebar.tsx`'s `NAV` array (own glyph, own
      i18n key, positioned alongside "Entries") — verify it renders and
      navigates correctly, both expanded and collapsed
- [ ] 7.3 Add `nav.recurring` and the page's i18n keys to
      `src/i18n/locales/en.json` (and best-effort `de.json`)

## 8. Frontend: list page

- [ ] 8.1 Build the `/recurring` list: fetch
      `GET /api/recurring-transactions` and
      `GET /api/recurring-transactions/summary`, render each row's title,
      category, entered amount, `per_year_amount`, an ended-state visual
      treatment, and the per-currency total row — verify visually via
      `pnpm dev` against seeded data
- [ ] 8.2 Add a "Create transaction" action per row (see task 10.1) and an
      Edit link to `/recurring/{id}/edit`

## 9. Frontend: create/edit forms

- [ ] 9.1 Build the shared recurring-transaction form component (account,
      title, description, category, counterparty, location, tags, signed
      amount — reusing existing entry-form field components where
      possible) plus the recurrence section: preset `<select>` (Weekly,
      Every 2 weeks, Monthly, Every 2 months, Quarterly, Every 6 months,
      Yearly, Custom) mapping to `interval_unit`/`interval_count`, a
      Custom-only raw unit+count input pair, `starts_on`, and optional
      `ends_on` — verify preset-to-field and field-to-preset mapping with a
      unit test or manual check for every preset
- [ ] 9.2 Add `src/routes/recurring.new.tsx` using the form, `POST
      /api/recurring-transactions` on submit — verify a new recurring
      transaction appears in the list after creation
- [ ] 9.3 Add `src/routes/recurring.$id.edit.tsx` using the form
      pre-populated from `GET /api/recurring-transactions/{id}`, `PATCH` on
      submit, and a delete action disabled (with hint) when the fetched
      recurring transaction has any linked entries — verify editing and
      the delete-disabled state against a template with a linked entry

## 10. Frontend: create-transaction prefill flow

- [ ] 10.1 Wire "Create transaction" to navigate to `/entries/new` with the
      template's fields (account, title, description, category,
      counterparty, location, tags, amount) and `next_suggested_date`
      passed through (query params or router state), and
      `recurring_transaction_id` included in the eventual
      `POST /api/entries` body — verify the created entry is linked and
      matches the template's fields, with the date defaulted to
      `next_suggested_date`
- [ ] 10.2 Verify every prefilled field remains editable on `/entries/new`
      before submission (manual check)

## 11. Frontend: linking existing entries and badges

- [ ] 11.1 Add a recurring-transaction link field to
      `entries.$entryId.edit.tsx`, offering only recurring transactions on
      the entry's current account, calling `PATCH /api/entries/{id}` with
      `recurring_transaction_id` (set or `null`) — verify linking and
      unlinking an existing entry
- [ ] 11.2 Add a small badge component (icon + link to
      `/recurring/{id}/edit`) shown wherever an entry with a non-null
      `recurring_transaction_id` is rendered; wire it into
      `entries.index.tsx`'s ledger rows and `reports.tsx`'s results table —
      verify the badge appears only on linked entries in both places

## 12. Frontend: validation

- [ ] 12.1 Run `pnpm lint`, `pnpm exec tsc`, `pnpm build` in `frontend/`
      and verify all three succeed

## 13. Final checks

- [ ] 13.1 Run `openspec validate add-recurring-transactions --strict` and
      resolve any reported issues

## 1. Database

- [ ] 1.1 Add migration `0024_entry_counterparty_location.sql` adding
  nullable `counterparty text` and `location text` columns to `entries`.
  No backfill, no CHECK constraint — both default `NULL`.

## 2. Backend domain model (`backend/internal/entry/`)

- [ ] 2.1 `entry.go`: add `Counterparty *string` and `Location *string` to
  `Entry`, `New`, and `Update` (plain nullable strings, mirroring
  `Description`'s shape — no `OptionalID` needed, see design.md).
- [ ] 2.2 `entry.go`'s `validateNew` (and the equivalent update-side
  validation in `service.go`): reject a non-empty `Counterparty` or
  `Location` when `Kind != KindTransaction`, mirroring the existing
  `category_id` kind-gating.
- [ ] 2.3 `store.go`: add `ListInUseCounterparties(ctx, ownerID) ([]string,
  error)` to the `Store` interface, mirroring `account.Store`'s
  `ListInUseTypes`.
- [ ] 2.4 `service.go`: add `ListInUseCounterparties` passthrough method,
  mirroring `account.Service.ListInUseTypes`.
- [ ] 2.5 `handler.go`: add `GET /api/entries/counterparties` (register
  before/alongside the other literal-segment routes like
  `/api/entries/summary` and `/api/entries/flow-summary`, so Go's mux
  resolves it ahead of `/api/entries/{id}`); wire `counterparty`/
  `location` through the create/update request bodies.

## 3. Backend storage (`backend/internal/storage/postgres/`,
   `backend/internal/storage/memory/`)

- [ ] 3.1 `postgres/entry.go`: add `counterparty`/`location` to
  `entryCols`/`scanEntry`, insert/update statements; implement
  `ListInUseCounterparties` (`SELECT DISTINCT counterparty … WHERE
  created_by = $1 AND counterparty IS NOT NULL AND counterparty != ''
  ORDER BY lower(counterparty), counterparty`, scoped like
  `ListInUseTypes`).
- [ ] 3.2 `postgres/entry.go`: extend the `Query` free-text search
  (currently `title ILIKE %q% OR description ILIKE %q%`, around
  `entry.go:320-322`) to also match `counterparty`.
- [ ] 3.3 `storage/memory`: mirror the same field storage, search
  extension, and `ListInUseCounterparties` for the in-memory test store.

## 4. API contract

- [ ] 4.1 Update `openapi/openapi.yaml`: `Entry`, `EntryCreate`,
  `EntryUpdate` schemas gain `counterparty` and `location` (both nullable
  strings); document that both are accepted only when `kind =
  transaction` (400 otherwise, mirroring `category_id`'s description
  text); add `GET /api/entries/counterparties` (mirrors
  `GET /api/account-types` exactly: `200` with `string[]`, auth required).
- [ ] 4.2 Run `cd backend && go generate ./...` to sync
  `backend/openapi.yaml`.
- [ ] 4.3 Run `cd frontend && pnpm generate:api` to regenerate
  `frontend/src/api/schema.d.ts`.
- [ ] 4.4 Add response-conformance assertions
  (`internal/openapicheck.AssertResponse`) to the new/changed handler
  tests, per `backend/AGENTS.md`.

## 5. Backend tests

- [ ] 5.1 `entry_test.go`/`service_scenarios_test.go`: creating a
  transaction with `counterparty`/`location` succeeds and round-trips;
  creating a `balance_adjustment` with either set is rejected (`400`);
  updating either field on an existing transaction works; clearing either
  via an empty-string update works.
- [ ] 5.2 `service_test.go` (or equivalent): `ListInUseCounterparties` is
  distinct, sorted, owner-scoped, and excludes soft-deleted entries and
  empty/null values — mirror `TestListInUseTypesIsDistinctSortedAndOwnerScoped`.
- [ ] 5.3 `handler_test.go`: `GET /api/entries/counterparties` requires
  auth, returns the caller's distinct values, empty list for a caller with
  none; `q` search matches on `counterparty` alone (no title/description
  match) in a listing test.
- [ ] 5.4 `postgres/entry_test.go`: integration coverage for the new
  columns and `ListInUseCounterparties`, mirroring the account-types
  integration tests' shape.

## 6. Frontend: dependency and shared components

- [ ] 6.1 Add Leaflet (and its CSS) as a `frontend/` dependency via
  `pnpm add`; import its stylesheet once, near the other global style
  imports.
- [ ] 6.2 Create `frontend/src/components/LocationField.tsx`: a text input
  bound to the raw `location` string, plus a "Use GPS" button
  (`navigator.geolocation.getCurrentPosition`, writes `{"lat","lng"}` JSON
  into the field; shows an inline error on failure/denial/timeout) and a
  "Pick on map" button opening `LocationPickerModal`.
- [ ] 6.3 Create `frontend/src/components/LocationPickerModal.tsx`: an
  interactive Leaflet/OSM map with a click-to-place/draggable marker,
  centered on the device's current GPS position if available else a world
  view (no address search), with Confirm/Cancel; Confirm writes
  `{"lat","lng"}` JSON back to the caller.
- [ ] 6.4 Create `frontend/src/components/LocationPreviewModal.tsx`: a
  small, read-only Leaflet/OSM map centered on a given `{lat,lng}`, with a
  static (non-draggable) marker and no editing controls.
- [ ] 6.5 Add a small `frontend/src/lib/location.ts` helper: `parseLocation
  (value: string | null | undefined) => {lat, lng} | null`, the single
  parse-attempt implementation shared by the ledger's globe-icon condition
  and the entry form's own live preview, so the two never disagree (see
  design.md's risk mitigation).

## 7. Frontend: entry form

- [ ] 7.1 `entries.new.tsx`/`entries.$entryId.edit.tsx`: add a
  `counterparty` text input with a `<datalist>` sourced from
  `GET /api/entries/counterparties`, shown only when `kind = transaction`
  (same conditional treatment `category_id` already gets); add a
  `LocationField` bound to `location`, also shown only when `kind =
  transaction`; both are omitted from the submitted body (or sent as
  `undefined`/cleared) when `kind` is `balance_adjustment`.
- [ ] 7.2 Wire both fields into the `POST /api/entries` /
  `PATCH /api/entries/{id}` request bodies via `compact(...)`, consistent
  with how `description`/`category_id` are already handled.

## 8. Frontend: ledger

- [ ] 8.1 `entries.index.tsx`: render `entry.counterparty`, when present,
  as a second line under the title link (same slot/style as the existing
  `created_by_name` line).
- [ ] 8.2 `entries.index.tsx`: when `parseLocation(entry.location)`
  succeeds, render a small globe icon inline next to the title; clicking
  it opens `LocationPreviewModal` centered on that point. No icon when
  `location` is null, empty, or doesn't parse as coordinates.

## 9. i18n

- [ ] 9.1 Add `entries.form.counterparty*`, `entries.form.location*`
  (label, GPS button, map-picker button, error strings),
  `entries.location.*` (preview modal title/close) keys to
  `frontend/src/i18n/locales/en.json`; add German equivalents to
  `de.json` where practical (non-blocking in CI if it lags).

## 10. Verification

- [ ] 10.1 Backend: `gofmt -l .`, `go vet ./...`, `go test ./...` from
  `backend/`.
- [ ] 10.2 Frontend: `pnpm lint`, `pnpm exec tsc`, `pnpm build` from
  `frontend/`.
- [ ] 10.3 Manually exercise the golden path in a browser: create a
  transaction with a typed counterparty (confirm it later appears as a
  suggestion), set its location via "Use GPS" and separately via "Pick on
  map," confirm the ledger shows the globe icon and opens the correct
  point in `LocationPreviewModal`, confirm a plain typed address shows no
  globe icon, confirm `q` search matches on counterparty, confirm a
  balance adjustment offers neither field.

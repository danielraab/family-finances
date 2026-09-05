## 1. Backend: `account_types` + `internal/account`

- [ ] 1.1 Migration `0006_account_types.sql`: `account_types(id uuid PK,
  name text NOT NULL UNIQUE, created_at timestamptz NOT NULL DEFAULT now())`.
- [ ] 1.2 Migration `0007_accounts.sql`: `accounts(id uuid PK, owner_id uuid
  NOT NULL REFERENCES users(id), title text NOT NULL, description text,
  type_id uuid NOT NULL REFERENCES account_types(id), currency char(3) NOT
  NULL, financial_institute text, opening_date date NOT NULL, closing_date
  date, deleted_at timestamptz, created_at timestamptz NOT NULL DEFAULT
  now(), updated_at timestamptz NOT NULL DEFAULT now())`, with
  `CHECK (closing_date IS NULL OR closing_date >= opening_date)` and an
  index on `(owner_id)`.
- [ ] 1.3 Migration `0011_account_disabled.sql`: `ALTER TABLE accounts ADD
  COLUMN disabled boolean NOT NULL DEFAULT false`.
- [ ] 1.4 Create `backend/internal/account/` in the four-file shape:
  domain type + validation (title required, currency via
  `settings.ValidateCurrency`, closing >= opening) in `account.go`; `Store`
  interface + sentinel errors (`ErrNotFound`, `ErrInvalidValue`,
  `ErrTypeInUse`) in `store.go`; use-case logic in `service.go` (including
  the `AccountLookup` interface `internal/entry` will consume, returning
  owner/currency/`disabled`); HTTP handlers in `handler.go` for account
  CRUD, `POST /api/accounts/{id}/disable`, `POST /api/accounts/{id}/enable`,
  and account-type CRUD, all under `auth.UserFromContext`, account-type
  writes gated on `IsAdmin`.
- [ ] 1.5 Implement `Store` in `internal/storage/memory` and
  `internal/storage/postgres` — every account query scoped to
  `owner_id = caller` and `deleted_at IS NULL`; account-type delete checks
  for in-use references before deleting (`409` via `ErrTypeInUse`).
- [ ] 1.6 Wire into `main.go`, mount at `/api/accounts` and
  `/api/account-types`.
- [ ] 1.7 Unit tests (service, memory store) + handler tests, including:
  cross-owner access reads as `404`; closing-date-before-opening rejected;
  invalid currency shape rejected; non-admin blocked from account-type
  writes (`403`); deleting an in-use account type rejected (`409`); soft
  delete excludes from listing; disable/enable round-trip; a disabled
  account still appears in listings and is still readable/editable.

## 2. Backend: `internal/category`

- [ ] 2.1 Migration `0008_categories.sql`: `categories(id uuid PK, parent_id
  uuid REFERENCES categories(id), name text NOT NULL, created_at timestamptz
  NOT NULL DEFAULT now())`, unique `(parent_id, name)`.
- [ ] 2.2 Create `backend/internal/category/` in the four-file shape:
  tree type + cycle-prevention validation in `category.go`; `Store` +
  sentinel errors (`ErrNotFound`, `ErrInUse`, `ErrCycle`) in `store.go`;
  `service.go` (including the `CategoryLookup` interface `internal/entry`
  will consume); `handler.go` for CRUD, read open to any authenticated
  user, writes gated on `IsAdmin`.
- [ ] 2.3 Implement `Store` in `internal/storage/memory` and
  `internal/storage/postgres`; delete rejected (`409`) when the category has
  children or is referenced by a non-deleted entry.
- [ ] 2.4 Wire into `main.go`, mount at `/api/categories`.
- [ ] 2.5 Unit tests + handler tests: non-admin blocked from writes;
  deleting a category with children rejected; deleting an in-use category
  rejected; setting a category's parent to itself or a descendant rejected.

## 3. Backend: `internal/tag`

- [ ] 3.1 Migration `0009_tags.sql`: `tags(id uuid PK, owner_id uuid NOT
  NULL REFERENCES users(id), name text NOT NULL, created_at timestamptz NOT
  NULL DEFAULT now())`, unique `(owner_id, name)`.
- [ ] 3.2 Create `backend/internal/tag/` in the four-file shape, including
  the `TagLookup` interface `internal/entry` will consume; CRUD restricted
  to `owner_id = caller`.
- [ ] 3.3 Implement `Store` in `internal/storage/memory` and
  `internal/storage/postgres`.
- [ ] 3.4 Wire into `main.go`, mount at `/api/tags`.
- [ ] 3.5 Unit tests + handler tests: a user cannot see or reference
  another user's tags; duplicate name for the same owner rejected.

## 4. Backend: `internal/entry`

- [ ] 4.1 Migration `0010_entries.sql`: `entries(id bigserial PK, owner_id
  uuid NOT NULL REFERENCES users(id), account_id uuid NOT NULL REFERENCES
  accounts(id), kind text NOT NULL CHECK (kind IN ('transaction',
  'balance_adjustment')), amount bigint NOT NULL, booking_timestamp
  timestamptz NOT NULL, title text NOT NULL, description text, category_id
  uuid REFERENCES categories(id), deleted_at timestamptz, created_at
  timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL
  DEFAULT now())`, with the `kind`/`category_id` `CHECK` from design.md and
  an index on `(account_id, booking_timestamp, id)`; `entry_tags(entry_id
  REFERENCES entries(id) ON DELETE CASCADE, tag_id REFERENCES tags(id) ON
  DELETE CASCADE, PRIMARY KEY (entry_id, tag_id))`.
- [ ] 4.2 Create `backend/internal/entry/` in the four-file shape:
  domain type + `kind`/`category_id` validation + the `AmountScale = 4`
  constant in `entry.go`; `Store` + sentinel errors (including
  `ErrAccountDisabled`) in `store.go`; `service.go` declaring
  `AccountLookup`, `CategoryLookup`, `TagLookup` and implementing
  create/update/delete, `List` (filters, search, sort, cursor pagination
  per design.md), and `Balance(ctx, accountID, asOf)`; `handler.go` for
  `/api/entries` CRUD + listing and `GET /api/accounts/{id}/balance`.
- [ ] 4.3 `Create` rejects (`422`, `ErrAccountDisabled`) when the target
  account's `disabled` is true.
- [ ] 4.4 Implement `Store` in `internal/storage/memory` and
  `internal/storage/postgres`, including: the balance query; the
  filter/search/sort/keyset-cursor listing query (category filter resolves
  the category's subtree first, then filters `category_id IN (...)`); every
  entry query scoped to `owner_id = caller`, `deleted_at IS NULL`, and
  joined to a non-deleted account.
- [ ] 4.5 Wire into `main.go`: construct `account.Service`,
  `category.Service`, `tag.Service` first, pass them as
  `entry.NewService`'s `AccountLookup` / `CategoryLookup` / `TagLookup`.
- [ ] 4.6 Unit tests + handler tests covering: transaction without a
  category rejected; balance-adjustment with a null category accepted;
  entry for another user's account rejected; entry creation against a
  disabled account rejected; tagging with another user's tag rejected;
  `account_id`/`kind` immutable on update; two entries at the identical
  millisecond tie-break by insertion order in both listing and balance;
  balance with no prior adjustment starts from `0`; soft-deleted account's
  entries excluded from listing and balance; each listing filter
  (`account_id`, `category_id` including descendants, `tag_id`, `kind`,
  `from`/`to`) narrows results correctly; `q` matches title/description;
  `sort`/`dir` order results correctly for both `booking_timestamp` and
  `amount`; `next_cursor` round-trips to the next page with no gap or
  overlap, and is `null` on the last page.

## 5. Backend: `user-settings` displayed decimal places

- [ ] 5.1 Migration `0012_user_settings_displayed_decimal_places.sql`:
  `ALTER TABLE user_settings ADD COLUMN displayed_decimal_places smallint
  CHECK (displayed_decimal_places BETWEEN 0 AND 4)`.
- [ ] 5.2 `internal/settings`: add `DisplayedDecimalPlaces` to the resolved
  `Settings` struct (hardcoded default `2`) and to `Update`; validate the
  `0..4` range in the service before it reaches the database.
- [ ] 5.3 Extend `GET`/`PUT /api/settings` request/response handling and
  tests for the new field, including the invalid-range-rejected case.

## 6. API contract

- [ ] 6.1 `openapi/openapi.yaml`: add `Account`, `AccountCreate`,
  `AccountUpdate`, `AccountType`, `AccountTypeWrite`, `Category`,
  `CategoryWrite`, `Tag`, `TagWrite`, `Entry`, `EntryCreate`, `EntryUpdate`,
  `EntryPage`, `Balance` schemas and the paths under `/api/accounts`
  (including `{id}/disable`, `{id}/enable`), `/api/account-types`,
  `/api/categories`, `/api/tags`, `/api/entries` (with its filter/search/
  sort/cursor query parameters), and `GET /api/accounts/{id}/balance`, each
  with every status code its handler can return. Add
  `displayed_decimal_places` to `UserSettings`/`UserSettingsUpdate`.
- [ ] 6.2 `cd backend && go generate ./...` (sync `backend/openapi.yaml`).
- [ ] 6.3 `cd frontend && pnpm generate:api` (regenerate
  `src/api/schema.d.ts`).
- [ ] 6.4 Add `internal/openapicheck.AssertResponse` assertions to the new
  handler tests; lint with spectral.

## 7. Backend verification

- [ ] 7.1 `gofmt -l .`, `go vet ./...`, `go test ./...` from `backend/`.
- [ ] 7.2 `internal/storage/postgres` integration tests against a real
  Postgres for every new store (accounts incl. disable/enable, account
  types, categories, tags, entries incl. listing/balance, and the
  `user_settings` column).

## 8. Frontend: shell and routing

- [ ] 8.1 `src/components/Sidebar.tsx`: add "Accounts" and "Entries" to
  `NAV` with their own icons and i18n keys (`nav.accounts`, `nav.entries`),
  active-state matching that highlights the section for any nested route
  (`/accounts/*`, `/entries/*`), not just an exact path match.
- [ ] 8.2 New route files: `accounts.tsx` (layout + auth gate, same
  redirect-to-`/login` pattern as `settings.tsx`), `accounts.index.tsx`
  (`/accounts`), `accounts.new.tsx`, `accounts.$accountId.tsx`,
  `accounts.$accountId.edit.tsx`; `entries.tsx` (layout + auth gate),
  `entries.index.tsx` (`/entries`, with `validateSearch` for filter/search/
  sort params), `entries.new.tsx`, `entries.$entryId.edit.tsx`.
- [ ] 8.3 New i18n keys in `src/i18n/locales/{en,de}.json` for every label,
  empty state, and confirmation copy introduced below — `en.json` first,
  per the existing source-of-truth rule.

## 9. Frontend: `web-client-accounts`

- [ ] 9.1 `/accounts` (overview): on mount, `GET /api/accounts` then, per
  account, `GET /api/accounts/{id}/balance` for the live balance; render
  title, type, currency, balance (formatted at the user's
  `displayed_decimal_places`), and a status indicator reflecting
  disabled/closed/open. Empty-state text when there are no accounts, with
  a create action.
- [ ] 9.2 `/accounts/{id}` (details): `GET /api/accounts/{id}` plus the
  most recent entries (`GET /api/entries?account_id={id}&sort=
  booking_timestamp&dir=desc&limit=…`, a small fixed page), each row
  linking to its edit page; a "See all" link to
  `/entries?account_id={id}`.
- [ ] 9.3 `/accounts/new` and `/accounts/{id}/edit`: form for title,
  description, type (`GET /api/account-types` populates the select),
  currency, financial institute, opening/closing date; client-side
  validation mirroring the backend's (currency shape, closing >= opening).
- [ ] 9.4 On the edit page: Disable/Enable action
  (`POST /api/accounts/{id}/disable` / `.../enable`) and a soft-delete
  action (`DELETE /api/accounts/{id}`), each behind a
  `@headlessui/react` `Dialog` confirmation mirroring
  `/settings/users`' pattern — delete's confirmation copy states it cannot
  be undone.
- [ ] 9.5 Manual verification in a running dev instance: create, edit,
  disable (confirm a new entry against it is rejected with a clear
  message), enable, soft-delete (confirm it disappears from the overview)
  an account.

## 10. Frontend: `web-client-entries`

- [ ] 10.1 `/entries` search-param shape: `account_id` (repeatable),
  `category_id`, `tag_id`, `kind`, `from`, `to`, `q`, `sort`, `dir` —
  typed via `validateSearch`, read via `useSearch`, written via `navigate`/
  `Link search={}` so every control (filter chip, sort header, search box)
  updates the URL rather than local-only state.
- [ ] 10.2 Filter UI: account multi-select, category picker (flat indented
  list per design.md), tag picker, kind toggle, date-range inputs, a debounced
  search input for `q`.
- [ ] 10.3 List rendering: sortable column headers for booking timestamp
  and amount (toggle `dir`, only one `sort` active at a time); infinite
  scroll — an intersection observer (or scroll-position check) triggers the
  next `GET /api/entries?...&after=<next_cursor>` once the previous request
  resolves, appending rows; the loaded list resets whenever any filter/
  search/sort param changes.
- [ ] 10.4 Empty state (no entries match the current filters, distinct
  from "no entries at all") and a loading indicator for the initial load
  and each subsequent page fetch.
- [ ] 10.5 `/entries/new`: preselects `account_id` when arriving via
  `?account_id=` (from the account details page's link); form for account
  (if not preselected — only the caller's own, non-deleted accounts),
  kind, amount (input at full precision, independent of the display
  setting), booking timestamp, title, description, category (required
  unless kind is `balance_adjustment`), and a tag input that matches
  existing tags as-you-type and creates (`POST /api/tags`) any typed value
  with no match on submit, then attaches it.
- [ ] 10.6 `/entries/{id}/edit`: same form minus account/kind (rendered
  read-only — immutable per the backend); includes a delete action behind
  a confirmation dialog.
- [ ] 10.7 Manual verification: create a transaction and a balance
  adjustment; edit one; delete one; apply each filter individually and in
  combination; search; sort by both columns in both directions; scroll far
  enough to trigger at least two "load more" fetches; confirm the URL
  reflects the current view and reloading it reproduces the same results;
  create a new tag inline and confirm it's usable immediately after.

## 11. Frontend: displayed decimal places setting

- [ ] 11.1 `settings.index.tsx` (Common tab): add a "Displayed decimal
  places" control (0–4), saving immediately on change via
  `PUT /api/settings`, matching the existing fields' interaction pattern.
- [ ] 11.2 A shared amount-formatting helper (used by the accounts overview/
  details balance and the entries list) that rounds the stored 4-decimal
  integer to the current user's `displayed_decimal_places` for display —
  used nowhere in the create/edit forms, which always work at full
  precision.

## 12. Frontend verification

- [ ] 12.1 `pnpm lint`, `pnpm exec tsc`, `pnpm build` from `frontend/`.

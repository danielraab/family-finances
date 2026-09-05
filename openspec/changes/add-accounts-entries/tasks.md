## 1. Config

- [ ] 1.1 Add `AmountDecimalPlaces int` to `config.Config` (env
  `AMOUNT_DECIMAL_PLACES`, default `2`); add an `intEnv` helper alongside
  `durationEnv`/`boolEnv` and document the variable in `backend/.env.example`.

## 2. Backend: `account_types` + `internal/account`

- [ ] 2.1 Migration `0006_account_types.sql`: `account_types(id uuid PK,
  name text NOT NULL UNIQUE, created_at timestamptz NOT NULL DEFAULT now())`.
- [ ] 2.2 Migration `0007_accounts.sql`: `accounts(id uuid PK, owner_id uuid
  NOT NULL REFERENCES users(id), title text NOT NULL, description text,
  type_id uuid NOT NULL REFERENCES account_types(id), currency char(3) NOT
  NULL, financial_institute text, opening_date date NOT NULL, closing_date
  date, deleted_at timestamptz, created_at timestamptz NOT NULL DEFAULT
  now(), updated_at timestamptz NOT NULL DEFAULT now())`, with
  `CHECK (closing_date IS NULL OR closing_date >= opening_date)` and an
  index on `(owner_id)`.
- [ ] 2.3 Create `backend/internal/account/` in the four-file shape:
  domain type + validation (title required, currency via
  `settings.ValidateCurrency`, closing >= opening) in `account.go`; `Store`
  interface + sentinel errors (`ErrNotFound`, `ErrInvalidValue`,
  `ErrTypeInUse`) in `store.go`; use-case logic in `service.go` (including
  the `AccountLookup` interface `internal/entry` will consume); HTTP
  handlers in `handler.go` for account CRUD + account-type CRUD, all under
  `auth.UserFromContext`, account-type writes gated on `IsAdmin`.
- [ ] 2.4 Implement `Store` in `internal/storage/memory` and
  `internal/storage/postgres` — every account query scoped to
  `owner_id = caller` and `deleted_at IS NULL`; account-type delete checks
  for in-use references before deleting (`409` via `ErrTypeInUse`).
- [ ] 2.5 Wire into `main.go`, mount at `/api/accounts` and
  `/api/account-types`.
- [ ] 2.6 Unit tests (service, memory store) + handler tests, including:
  cross-owner access reads as `404`; closing-date-before-opening rejected;
  invalid currency shape rejected; non-admin blocked from account-type
  writes (`403`); deleting an in-use account type rejected (`409`); soft
  delete excludes from listing.

## 3. Backend: `internal/category`

- [ ] 3.1 Migration `0008_categories.sql`: `categories(id uuid PK, parent_id
  uuid REFERENCES categories(id), name text NOT NULL, created_at timestamptz
  NOT NULL DEFAULT now())`, unique `(parent_id, name)`.
- [ ] 3.2 Create `backend/internal/category/` in the four-file shape:
  tree type + cycle-prevention validation in `category.go`; `Store` +
  sentinel errors (`ErrNotFound`, `ErrInUse`, `ErrCycle`) in `store.go`;
  `service.go` (including the `CategoryLookup` interface `internal/entry`
  will consume); `handler.go` for CRUD, read open to any authenticated
  user, writes gated on `IsAdmin`.
- [ ] 3.3 Implement `Store` in `internal/storage/memory` and
  `internal/storage/postgres`; delete rejected (`409`) when the category has
  children or is referenced by a non-deleted entry.
- [ ] 3.4 Wire into `main.go`, mount at `/api/categories`.
- [ ] 3.5 Unit tests + handler tests: non-admin blocked from writes;
  deleting a category with children rejected; deleting an in-use category
  rejected; setting a category's parent to itself or a descendant rejected.

## 4. Backend: `internal/tag`

- [ ] 4.1 Migration `0009_tags.sql`: `tags(id uuid PK, owner_id uuid NOT
  NULL REFERENCES users(id), name text NOT NULL, created_at timestamptz NOT
  NULL DEFAULT now())`, unique `(owner_id, name)`.
- [ ] 4.2 Create `backend/internal/tag/` in the four-file shape, including
  the `TagLookup` interface `internal/entry` will consume; CRUD restricted
  to `owner_id = caller`.
- [ ] 4.3 Implement `Store` in `internal/storage/memory` and
  `internal/storage/postgres`.
- [ ] 4.4 Wire into `main.go`, mount at `/api/tags`.
- [ ] 4.5 Unit tests + handler tests: a user cannot see or reference
  another user's tags; duplicate name for the same owner rejected.

## 5. Backend: `internal/entry`

- [ ] 5.1 Migration `0010_entries.sql`: `entries(id bigserial PK, owner_id
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
- [ ] 5.2 Create `backend/internal/entry/` in the four-file shape:
  domain type + `kind`/`category_id` validation in `entry.go`; `Store` +
  sentinel errors in `store.go`; `service.go` declaring `AccountLookup`,
  `CategoryLookup`, `TagLookup` and implementing create/update/delete/list
  plus `Balance(ctx, accountID, asOf)` (live computation per design.md);
  `handler.go` for `/api/entries` CRUD and `GET
  /api/accounts/{id}/balance`.
- [ ] 5.3 Implement `Store` in `internal/storage/memory` and
  `internal/storage/postgres`, including the balance query; every entry
  query scoped to `owner_id = caller`, `deleted_at IS NULL`, and joined to a
  non-deleted account.
- [ ] 5.4 Wire into `main.go`: construct `account.Service`, `category.Service`,
  `tag.Service` first, pass them as `entry.NewService`'s `AccountLookup` /
  `CategoryLookup` / `TagLookup`, plus `cfg.AmountDecimalPlaces`.
- [ ] 5.5 Unit tests + handler tests covering: transaction without a
  category rejected; balance-adjustment with a null category accepted;
  entry for another user's account rejected; tagging with another user's
  tag rejected; `account_id`/`kind` immutable on update; two entries at the
  identical millisecond tie-break by insertion order in both listing and
  balance; balance with no prior adjustment starts from `0`; soft-deleted
  account's entries excluded from listing and balance.

## 6. API contract

- [ ] 6.1 `openapi/openapi.yaml`: add `Account`, `AccountCreate`,
  `AccountUpdate`, `AccountType`, `AccountTypeWrite`, `Category`,
  `CategoryWrite`, `Tag`, `TagWrite`, `Entry`, `EntryCreate`, `EntryUpdate`,
  `Balance` schemas and the paths under `/api/accounts`,
  `/api/account-types`, `/api/categories`, `/api/tags`, `/api/entries`, and
  `GET /api/accounts/{id}/balance`, each with every status code its handler
  can return.
- [ ] 6.2 `cd backend && go generate ./...` (sync `backend/openapi.yaml`).
- [ ] 6.3 `cd frontend && pnpm generate:api` (regenerate
  `src/api/schema.d.ts`) — no frontend UI changes, just keeping the
  generated client in sync per `api-contract`.
- [ ] 6.4 Add `internal/openapicheck.AssertResponse` assertions to the new
  handler tests; lint with spectral.

## 7. Verification

- [ ] 7.1 `gofmt -l .`, `go vet ./...`, `go test ./...` from `backend/`.
- [ ] 7.2 `internal/storage/postgres` integration tests against a real
  Postgres for every new store (accounts, account types, categories, tags,
  entries, balance computation).

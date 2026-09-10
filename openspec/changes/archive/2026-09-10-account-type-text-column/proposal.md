## Why

An account's `type` is currently a full relational entity: a per-user
`account_types` lookup table with its own CRUD API (six endpoints), a
Settings management tab, a `disabled` flag, delete-in-use protection, a
new-user seeding hook, and a sharing carve-out that makes `type_id` the one
field a shared `owner`-tier user may not touch. For a family-finances app
where a type is just a label like "Checking" or "Savings", that machinery
costs far more than it returns. Collapsing it to a plain required text
column on `accounts` removes a table, an API surface, a settings screen, a
Store sub-interface, an auth hook, and a special case in the sharing model.

## What Changes

- **BREAKING** `accounts.type_id` (uuid FK) is replaced by `accounts.type`
  (text, `NOT NULL`). Migration `0021` adds the column, backfills it from
  `account_types.title` via the existing FK, drops `type_id`, and drops the
  `account_types` table.
- **BREAKING** The account-type CRUD API is removed:
  `POST/PATCH/DELETE /api/account-types`,
  `POST /api/account-types/{id}/disable`, `.../enable`. The
  `AccountType` and `AccountTypeWrite` schemas are removed.
- `GET /api/account-types` is **repurposed**: instead of returning managed
  `AccountType` objects it returns `string[]` — the caller's own distinct,
  trimmed, in-use `accounts.type` values, sorted, for client autocomplete.
- `Account.type_id` becomes `Account.type` (a required non-empty string,
  only trimmed — no case-folding, no canonical list) on every account
  request and response body.
- The "`type_id` reassignable only by the real owner" sharing rule is
  **removed**: `type` is now an ordinary `owner`-tier-editable field like
  `title`, editable by a shared owner.
- The new-user **seeding** of default account types is removed; the default
  starter labels (Checking, Savings, Cash, Credit Card, Loan, Investment,
  English only) move to a frontend constant used to seed the autocomplete
  suggestions.
- The **Account Types settings tab** and its navigation entry are removed.
- The account create/edit form's type **`<select>`** (with disabled-type
  handling and the `typeLocked` read-only mode) becomes a required text
  input with a `<datalist>` merging the default labels and the caller's
  in-use values.

## Capabilities

### New Capabilities

- (none)

### Modified Capabilities

- `accounts`: `type` is a required free-text field on the account, only
  trimmed; the per-user self-managed `account_types` lookup, the seeded
  default types, and the "`type_id` reassignable only by the real owner"
  rule are removed; a new read-only `GET /api/account-types` returns the
  caller's distinct in-use type strings.
- `account-sharing`: a shared `owner`-tier caller may now change `type`
  like any other account field; the `type_id` carve-out is dropped.
- `web-client-accounts`: the account form selects `type` as free text with
  suggestions (default labels + in-use values) rather than from a managed
  list; disabled-type reselection behavior is removed.
- `web-client-settings`: the Account Types management tab is removed.

## Impact

- **Database**: new migration `0021`; `account_types` table dropped;
  `accounts.type_id` → `accounts.type`.
- **API contract**: `openapi/openapi.yaml` (source of truth) plus the two
  generated artifacts — `backend/openapi.yaml` and
  `frontend/src/api/schema.d.ts` — regenerated. Account-type paths and
  schemas removed; `GET /api/account-types` response type changed;
  `type_id` → `type` on `Account`, `AccountCreate`, `AccountWrite`.
- **Backend** (`internal/account`): `Account` struct field; drop `Type`
  struct, `DefaultTypeTitles`, `resolveAssignableType`, `ErrTypeInUse`,
  `ErrTypeDisabled`; drop the seven `*Type`/`SeedDefaultTypes` methods from
  the `Store` interface and both implementations
  (`internal/storage/memory`, `internal/storage/postgres`); drop the
  account-type routes/handlers; add a distinct-in-use query + endpoint;
  remove the shared-owner `type_id` guard in `Service.Update`; drop the
  two `registerErrStatus` lines in `internal/httpapi`.
- **Backend wiring** (`main.go`): remove `account.Service.SeedDefaults`
  from `auth.WithNewUserHooks`.
- **Backend CLI** (`internal/cli/fixtures.go`): seed `accounts.type` as
  text instead of creating/referencing `account_types` rows.
- **Frontend**: delete `routes/settings.account-types.tsx` and its
  `settings.tsx` nav entry; rework `components/AccountForm.tsx`
  (`typeLocked` prop removed); update `routes/accounts.index.tsx`,
  `routes/accounts.$accountId.edit.tsx`,
  `lib/useAccountsWithBalances.ts`; remove `settings.accountTypes.*` i18n
  keys and adjust `accounts.form.type*` keys across all locales.
- **Tests**: `internal/account` (`handler_test.go`, `service_test.go`,
  `sharing_test.go`, `icon_color_test.go`), `internal/storage/postgres`
  (`account_test.go`, `account_sharing_test.go`, `icon_color_test.go`,
  `reset_test.go`), `internal/cli/fixtures_test.go`,
  `internal/auth/service_test.go`.

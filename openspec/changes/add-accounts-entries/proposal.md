## Why

The backend has no financial domain model yet — only authentication and
per-user settings. This change adds the core bookkeeping primitives
everything else (statements, budgets, reports) will build on: accounts a
user owns, and the entries recorded against them, plus the lookup tables
entries need to be classified (admin-managed account types, admin-managed
category tree) and organized (per-user tags). Business logic only — no
frontend screens.

## What Changes

- New backend domain packages, each in the repo's four-file shape
  (`<noun>.go`, `service.go`, `store.go`, `handler.go`):
  - `internal/account` — accounts, and the admin-managed `account_types`
    lookup table.
  - `internal/category` — a global, tree-structured, admin-managed category
    lookup.
  - `internal/tag` — per-user tags.
  - `internal/entry` — entries recorded against an account (transactions and
    balance adjustments), including live balance computation.
- **Accounts**: `title`, `description` (optional), `type_id` (FK to
  `account_types`), `currency` (ISO-4217 shape), `financial_institute`
  (optional), `opening_date`, `closing_date` (optional). Each account has
  exactly one `owner_id` and is visible only to its owner — no sharing or
  permissions model yet (explicitly deferred to a future change). Soft
  delete via `deleted_at` (one-way, no undelete).
- **Account types**: a flat, instance-wide table, listable by any
  authenticated user, writable only by an admin (`is_admin`, the same gate
  `user-administration` already established).
- **Categories**: a global tree (self-referencing `parent_id`, no depth
  limit), listable by any authenticated user, writable only by an admin.
- **Tags**: per-user (`owner_id`), full CRUD restricted to their owner, never
  visible to another user.
- **Entries**: `account_id` (required, immutable), `owner_id` (required,
  copied from the account's owner at creation — its own column so a future
  per-entry share doesn't require a schema change), `kind` (`transaction` |
  `balance_adjustment`, immutable), `amount` (integer minor units),
  `booking_timestamp` (millisecond precision), `title`, `description`
  (optional), `category_id` (required when `kind = transaction`, optional
  when `kind = balance_adjustment`), zero or more `tags`. Visible only to
  their owner. Soft delete via `deleted_at`.
- **Balance is always computed live**, never cached: for a point in time,
  take the latest `balance_adjustment` at or before it — ties on an
  identical millisecond `booking_timestamp` broken by insertion order — or
  `0` if no adjustment exists yet, then add every `transaction` after that
  point.
- **Monetary amounts** are stored as integers scaled by a fixed number of
  minor-unit decimal places, instance-wide and configurable via a new
  `AMOUNT_DECIMAL_PLACES` environment variable (default `2`, read once at
  startup in `internal/config`). Changing it does not validate or rescale
  already-stored amounts — an accepted, documented limitation, not handled
  by this change.
- `openapi/openapi.yaml` gains the new schemas and paths; `backend/openapi.yaml`
  and `frontend/src/api/schema.d.ts` are regenerated in the same change.

## Capabilities

### New Capabilities

- `accounts`: the `accounts` table, its ownership/visibility/soft-delete
  rules, and the admin-managed `account_types` lookup.
- `entry-categories`: the global, tree-structured, admin-managed `categories`
  lookup.
- `entry-tags`: the per-user `tags` lookup.
- `account-entries`: the `entries` table (transactions and balance
  adjustments), its relationship to accounts/categories/tags, and live
  balance computation.

### Modified Capabilities

None. New endpoints reuse the existing authenticated-request accessor
(`auth.UserFromContext`) and the existing `is_admin` gate the same way
`user-administration` already does — referenced, not changed.

## Impact

- **Dependencies**: none new.
- **Code**: four new `internal/<noun>/` packages (with `storage/memory` and
  `storage/postgres` implementations each); five new migrations
  (`0006`–`0010`); `internal/config` gains `AmountDecimalPlaces` (env
  `AMOUNT_DECIMAL_PLACES`, default `2`).
- **No frontend UI in this change.** `/accounts` and entry-management
  screens are a separate follow-up once this backend surface exists.
- **API contract**: `openapi/openapi.yaml` gains `Account`, `AccountCreate`,
  `AccountUpdate`, `AccountType`, `AccountTypeWrite`, `Category`,
  `CategoryWrite`, `Tag`, `TagWrite`, `Entry`, `EntryCreate`, `EntryUpdate`,
  `Balance` schemas and their paths — regenerate `backend/openapi.yaml` and
  `frontend/src/api/schema.d.ts` in the same change.
- **Spec**: new `accounts`, `entry-categories`, `entry-tags`,
  `account-entries`.

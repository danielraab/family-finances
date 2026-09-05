## Why

The backend has no financial domain model yet — only authentication and
per-user settings. This change adds the core bookkeeping primitives
everything else (statements, budgets, reports) will build on: accounts a
user owns, and the entries recorded against them, plus the lookup tables
entries need to be classified (admin-managed account types, admin-managed
category tree) and organized (per-user tags) — and, this being the first
domain feature with any real UI surface, the frontend pages to actually use
it: an accounts overview/detail/create/edit flow and a filterable,
searchable, sortable, infinitely-scrolling entry ledger.

## What Changes

- New backend domain packages, each in the repo's four-file shape
  (`<noun>.go`, `service.go`, `store.go`, `handler.go`):
  - `internal/account` — accounts, and the admin-managed `account_types`
    lookup table.
  - `internal/category` — a global, tree-structured, admin-managed category
    lookup.
  - `internal/tag` — per-user tags.
  - `internal/entry` — entries recorded against an account (transactions and
    balance adjustments), including live balance computation and a
    filterable/searchable/sortable, cursor-paginated listing.
- **Accounts**: `title`, `description` (optional), `type_id` (FK to
  `account_types`), `currency` (ISO-4217 shape), `financial_institute`
  (optional), `opening_date`, `closing_date` (optional). Each account has
  exactly one `owner_id` and is visible only to its owner — no sharing or
  permissions model yet (explicitly deferred to a future change). A
  reversible `disabled` flag blocks creating new entries against the
  account while leaving it (and its existing entries) fully visible and
  editable — distinct from `closing_date` (informational only) and from
  soft delete via `deleted_at` (one-way, no undelete).
- **Account types**: a flat, instance-wide table, listable by any
  authenticated user, writable only by an admin (`is_admin`, the same gate
  `user-administration` already established).
- **Categories**: a global tree (self-referencing `parent_id`, no depth
  limit), listable by any authenticated user, writable only by an admin.
- **Tags**: per-user (`owner_id`), full CRUD restricted to their owner, never
  visible to another user. No dedicated management UI yet — this change's
  only tag-creation surface is inline, on the fly, from the entry form.
- **Entries**: `account_id` (required, immutable), `owner_id` (required,
  copied from the account's owner at creation — its own column so a future
  per-entry share doesn't require a schema change), `kind` (`transaction` |
  `balance_adjustment`, immutable), `amount` (integer minor units),
  `booking_timestamp` (millisecond precision), `title`, `description`
  (optional), `category_id` (required when `kind = transaction`, optional
  when `kind = balance_adjustment`), zero or more `tags`. Visible only to
  their owner. Soft delete via `deleted_at`. Creating an entry against a
  disabled account is rejected.
- **Balance is always computed live**, never cached: for a point in time,
  take the latest `balance_adjustment` at or before it — ties on an
  identical millisecond `booking_timestamp` broken by insertion order — or
  `0` if no adjustment exists yet, then add every `transaction` after that
  point.
- **Monetary amounts** are stored as integers at a fixed **4** decimal
  places, a Go constant, not environment-configurable (no
  `AMOUNT_DECIMAL_PLACES` — dropped from what was originally proposed).
  Display precision is separate and per-user: `user-settings` gains
  `displayed_decimal_places` (default `2`), which only affects how amounts
  are rounded for *display*; editing an amount always works at the full
  stored precision.
- `openapi/openapi.yaml` gains the new schemas and paths; `backend/openapi.yaml`
  and `frontend/src/api/schema.d.ts` are regenerated in the same change.
- **Frontend**: two new top-level `Sidebar` sections, "Accounts" and
  "Entries":
  - `/accounts` — overview of the caller's accounts (title, type, currency,
    live balance, status).
  - `/accounts/{id}` — account details plus its most recent entries, with a
    link to `/entries?account_id={id}` for the full filtered list.
  - `/accounts/new`, `/accounts/{id}/edit` — create/edit, including
    disable/enable and soft delete, each destructive action behind a
    confirmation (mirroring `/settings/users`' pattern).
  - `/entries` — the full ledger across every account the caller owns:
    filter by account/category/tag/kind/date range, free-text search,
    sort by booking timestamp (default) or amount, infinite-scroll
    pagination — all reflected in the URL via TanStack Router search
    params, so a filtered view is linkable/bookmarkable/shareable across a
    reload.
  - `/entries/new` (optionally preselecting `account_id` from a query
    param), `/entries/{id}/edit` — create/edit/delete an entry, including a
    category picker for the tree and a tag input that creates a new tag
    inline when the typed name doesn't match an existing one.
  - Settings → Common tab gains a "Displayed decimal places" field
    alongside language/timezone/default currency.

## Capabilities

### New Capabilities

- `accounts`: the `accounts` table, its ownership/visibility/soft-delete
  rules, the `disabled` flag, and the admin-managed `account_types` lookup.
- `entry-categories`: the global, tree-structured, admin-managed `categories`
  lookup.
- `entry-tags`: the per-user `tags` lookup.
- `account-entries`: the `entries` table (transactions and balance
  adjustments), its relationship to accounts/categories/tags, live balance
  computation, and the filterable/searchable/sortable/paginated listing.
- `web-client-accounts`: the `/accounts` overview, detail, create, and edit
  pages.
- `web-client-entries`: the `/entries` ledger (filter/search/sort/paginate)
  and the entry create/edit pages, including inline tag creation.

### Modified Capabilities

- `user-settings`: gains `displayed_decimal_places` (default `2`) alongside
  language/timezone/default currency.
- `web-client-settings`: the Common tab gains a "Displayed decimal places"
  field, saved the same way as the other three.

## Impact

- **Dependencies**: none new. Pagination, filtering, and infinite scroll are
  built on TanStack Router's typed search params and plain `fetch`/effects,
  matching the existing no-query-library convention — no client-side data
  or table library added.
- **Code**: four new `internal/<noun>/` packages (with `storage/memory` and
  `storage/postgres` implementations each); six new migrations
  (`0006`–`0011`, see `design.md`); `internal/settings` gains
  `displayed_decimal_places`; new frontend routes under `src/routes/`
  (`accounts.*`, `entries.*`), a new `Sidebar` entries, and supporting
  components (account status badge, category tree picker, tag input,
  infinite-scroll list).
- **API contract**: `openapi/openapi.yaml` gains `Account`, `AccountCreate`,
  `AccountUpdate`, `AccountType`, `AccountTypeWrite`, `Category`,
  `CategoryWrite`, `Tag`, `TagWrite`, `Entry`, `EntryCreate`, `EntryUpdate`,
  `Balance`, `EntryPage` schemas and their paths, plus
  `displayed_decimal_places` on `UserSettings`/`UserSettingsUpdate` —
  regenerate `backend/openapi.yaml` and `frontend/src/api/schema.d.ts` in
  the same change.
- **Spec**: new `accounts`, `entry-categories`, `entry-tags`,
  `account-entries`, `web-client-accounts`, `web-client-entries`; deltas on
  `user-settings`, `web-client-settings`.

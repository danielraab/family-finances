## Context

`GET /api/entries` already supports cursor pagination (`after`/`limit` →
`{ items, next_cursor }`) plus `account_id`/`category_id`/`tag_id`/`kind`/
`from`/`to`/`q` filters, all resolved through `entry.Service.resolveFilter`
which narrows `Filter.AccountIDs` to the caller's own visible (owned or
shared) accounts before `postgres.buildWhere` turns it into SQL — see
`account-entries`'s "Entry listing supports filtering..." requirement and
`buildWhere`'s own doc comment in
`backend/internal/storage/postgres/entry.go`. Every additional filter
`buildWhere` adds is a plain `WHERE` clause layered on top of that
already-resolved `account_id = ANY(...)` scoping (or, for the `AllAccounts`
category-permission case, layered on top of nothing) — so a new filter
column is safe by construction: it can only narrow the resolved result
set further, never widen it past what the caller could already see.

`entries.recurring_transaction_id` is already an indexed column
(migration `0025_recurring_transactions.sql`:
`entries_recurring_transaction_idx`), just not exposed as a filter.

`/recurring/{id}/edit` (`recurring.$id.edit.tsx`) already fetches the
template via `GET /api/recurring-transactions/{id}`, whose response
carries `linked_entry_count` — currently rendered as a bare number next to
the delete-blocked hint.

## Goals / Non-Goals

**Goals:**
- Let a visitor see and reach the entries linked to a recurring
  transaction from its own edit page, paginated so a long-lived template
  with hundreds of occurrences doesn't force one huge fetch.
- Add the new filter the same way every existing `buildWhere` filter is
  added — no new permission logic, since layering on the existing
  visible-accounts scoping is sufficient (see Context).

**Non-Goals:**
- No change to the ledger (`/entries`) itself — filtering the main ledger
  by recurring transaction isn't asked for here (a visitor can already do
  a manual comparison via each entry's badge; this change is about the
  template's own page, not the ledger's filter row).
- No infinite-scroll/IntersectionObserver mechanism for this list — see
  Decisions.
- No change to `linked_entry_count`'s own semantics or to the
  delete-blocked-while-linked behavior.

## Decisions

**"Load more" button, not scroll-triggered infinite scroll.**
`entries.index.tsx`'s ledger uses an `IntersectionObserver` sentinel
because it *is* the primary, page-filling view. This list is a secondary
section partway down a settings-like edit page; auto-fetching more rows
as a visitor scrolls past it while reading the rest of the page would be
surprising, and a manual "Load more" avoids needing a sentinel ref inside
a component that isn't the page's main scroll container. Same
`{ items, next_cursor }` shape either way — only the trigger differs.

**Skip the fetch entirely when `linked_entry_count` is 0.**
The recurring transaction's own response already carries this count
(fetched on mount regardless). Reusing it to skip an empty-result round
trip is free — no separate "is there anything to show" check needed.

**No explicit `account_id` sent alongside `recurring_transaction_id`.**
Every entry linked to a recurring transaction is already guaranteed to be
on that template's own account (server-enforced, `account-entries`'s "An
entry can be linked to a recurring transaction on the same account"), and
`resolveFilter`'s default (no explicit `account_id`) already scopes to
every account the caller can see — which, for a caller who successfully
loaded this template's own edit page at all, already includes its
account. Adding a redundant `account_id` would duplicate that scoping for
no benefit.

**Row content mirrors `UpcomingBlock.tsx`'s row shape (date, title,
category, signed amount) rather than a new visual language**, but links
to `/entries/{id}/edit` instead of a "Create transaction" action — this
list is for *reviewing/reaching* existing entries, not creating new ones.
Not extracted into a shared component with `UpcomingBlock`: that one is
typed to `RecurringTransactionPreviewItem` (a projected, not-yet-created
occurrence) and intentionally non-paginated; forcing a shared component
over two different data shapes and pagination models would cost more than
it saves for two call sites.

## Risks / Trade-offs

- **[Risk]** A very active recurring transaction (hundreds of linked
  entries) makes this page's initial load do one extra request beyond
  today's single `GET /api/recurring-transactions/{id}` call. →
  **Mitigation**: it's skipped when `linked_entry_count` is 0 (the common
  case for a newly created template), and the first page is small
  (same default `limit` the ledger itself uses) either way.

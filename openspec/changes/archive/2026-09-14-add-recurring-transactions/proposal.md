## Why

Recurring costs and income (rent, subscriptions, salary, insurance) are
currently tracked the same way as one-off entries: the user re-types the same
title/amount/category by hand every time it recurs, with no single place to
see what they're committed to per year or to tell which past entries came
from which recurring obligation.

## What Changes

- Add a **recurring transaction** domain object: the same fields as a
  transaction entry (account, title, description, category, counterparty,
  location, tags, signed amount), plus a recurrence rule (`interval_unit`:
  `day`/`week`/`month`/`year`, and `interval_count`), a `starts_on` date, and
  an optional `ends_on` date (e.g. a cancelled subscription).
- A recurring transaction is **transaction-kind only** — there is no
  balance-adjustment equivalent.
- Add a new `/recurring` page (list) plus create/edit forms, and a sidebar
  nav item alongside "Entries" (same level, not nested under it).
- The list shows each recurring transaction's entered amount, its calculated
  per-year amount, and a total per-year sum grouped by currency (mirroring
  how `entries/summary` already groups by currency) — a recurring
  transaction past its `ends_on` is visually marked ended and excluded from
  that total.
- **"Create transaction" is always a manual click, never automatic** — this
  backend has no scheduler/job queue, and the feature deliberately doesn't
  add one. Clicking it opens the existing entry-create form prefilled from
  the template, with the booking date prefilled to a computed "next
  suggested date" (the last linked entry's date advanced by one interval —
  calendar-month/year aware — or `starts_on` if none exist yet); the user
  can still edit anything before submitting.
- Entries gain an optional `recurring_transaction_id`. It can be set at
  entry creation (via the prefilled create flow above) and can be attached
  to or cleared from an **existing** entry only via the entry edit page —
  there is no reverse "pick an entry" flow on the recurring transaction
  page itself.
- Wherever an entry is listed (the `/entries` ledger and the `/reports`
  results table), a linked entry SHALL show a small icon/badge linking to
  its recurring transaction.
- A recurring transaction with one or more linked entries **cannot be
  deleted** (mirrors the existing category in-use block) — it must be
  unlinked from every entry first (via each entry's edit page).

## Capabilities

### New Capabilities
- `recurring-transactions`: backend domain package owning the recurring
  transaction template — CRUD, per-year amount calculation, "next suggested
  date" computation, account-permission gating (mirrors `account-entries`'
  tiers), and the delete-blocked-while-linked rule.
- `web-client-recurring-transactions`: the `/recurring` list/create/edit
  pages, the "Recurring" sidebar nav item (same level as "Entries", per
  the existing per-page "link in the sidebar" pattern each nav-bearing
  capability already follows — see `web-client-entries`/`web-client-
  reports`/etc.), and the "Create transaction" prefill flow into
  `/entries/new`.

### Modified Capabilities
- `account-entries`: an `Entry` gains an optional `recurring_transaction_id`,
  settable on `POST /api/entries` and `PATCH /api/entries/{id}`, validated
  against the same account and owner as the entry itself.
- `web-client-entries`: the entry edit form (`/entries/{id}/edit`) gains a
  field to link/unlink an existing entry to a recurring transaction; the
  ledger (`/entries`) shows the linked-entry badge described above.
- `web-client-reports`: the results table shows the same linked-entry badge.

## Impact

- **Backend**: new `internal/recurringtransaction` package (domain/service/
  store/handler, `storage/memory` + `storage/postgres` implementations, one
  new migration for `recurring_transactions` plus the `entries` FK column);
  `openapi/openapi.yaml` gains the new schema/endpoints and the `Entry`
  schema's new field, regenerated into `backend/openapi.yaml` and
  `frontend/src/api/schema.d.ts`.
- **Frontend**: new routes under `src/routes/recurring*.tsx`; `Sidebar.tsx`'s
  `NAV`; a small badge component reused by `entries.index.tsx` and
  `reports.tsx`; the entry edit form.
- No changes to authentication, sharing tiers, or existing entry semantics
  beyond the new optional field.

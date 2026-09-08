## Why

The account details page (`/accounts/{id}`) shows the live balance and the
five most recent entries, but nothing about how an account's money has moved
over time. There is no chart anywhere in the app. Separately, a
`balance_adjustment` entry (used to set a known balance without
reconstructing full history) only ever displays as an absolute reading —
there is no visible sense of how much it actually added or removed, in the
recent-entries list or anywhere else.

Both gaps share one missing piece of data: a `balance_adjustment`'s *delta*
(the change from the balance immediately before it). Once that delta is a
real, stored value on the entry, it can be shown next to the reading
wherever an entry is rendered, and it makes a per-period income/outcome
aggregate a simple sum instead of a special-cased computation. This change
adds that stored delta, a new backend endpoint that buckets entries into
per-month or per-day income/outcome totals, and a bar chart on the account
details page (with a year switcher) built from it.

## What Changes

- **`entries.amount` becomes a uniform signed delta.** For a `transaction`
  it is unchanged (the amount the user enters). For a `balance_adjustment`
  it becomes the *computed* change from the balance immediately before that
  entry, instead of the absolute reading. The absolute reading itself moves
  to a new field, `balance` (`balance_reading` in storage), which is what
  the client now sends when creating or editing a `balance_adjustment` (the
  UI keeps asking for "what does your statement say," unchanged) — the
  server computes and stores the delta.
- **`Balance()` simplifies to a single running sum** (`SUM(amount)` up to a
  point in time, no more kind-based branching), because each
  `balance_adjustment`'s stored delta is, by construction, whatever value
  makes the running sum land exactly on its `balance_reading` — the same
  "ignore everything before the latest adjustment" behavior as today, just
  computed once at write time instead of on every read.
- **A `balance_adjustment`'s delta is recomputed synchronously**, inside the
  same database transaction, whenever any entry (of either kind) is
  created, updated, or deleted in a way that could change it — never
  asynchronously, and never more than the one or two affected adjustment
  rows.
- **New endpoint**, `GET /api/entries/flow-summary`, buckets the caller's
  matching entries by month or by day and sums each bucket's positive and
  negative contributions separately (`income` / `outcome`, per currency).
  Accepts repeatable `account_id`, matching the existing filter shape.
- **Account details page** gains a bar chart above "Recent entries": two
  bars (income, outcome) per month of a selected year, with previous/next
  year controls (no bound on how far forward it can go). Built from a new,
  reusable, hand-rolled (no charting library) bar-chart component, so a
  future per-day-of-month chart or another page's chart can reuse it.
- **Recent-entries list (account details) and the full `/entries` list**
  both gain a small, gray, non-color-coded annotation next to a
  `balance_adjustment` row showing its signed delta, alongside the
  unchanged main reading display.

## Capabilities

### Modified Capabilities

- `account-entries`: `kind: balance_adjustment`'s `amount` semantics
  change to a computed delta; adds the `balance` field and its recompute
  rule; simplifies the live-balance computation's description; adds the
  `GET /api/entries/flow-summary` endpoint.
- `web-client-accounts`: adds the income/outcome bar chart and year
  switcher to the account details page; extends the sign-coloring
  requirement with the balance-adjustment delta annotation.
- `web-client-entries`: extends the sign-coloring requirement with the same
  delta annotation on the full entries list; the create/edit form sends
  `balance` instead of `amount` for a `balance_adjustment`.

## Impact

- Backend: new migration adding `entries.balance_reading` (nullable,
  required exactly when `kind = 'balance_adjustment'`) and backfilling
  existing balance-adjustment rows; `internal/entry` gains the delta-recompute
  logic (both storage backends), the `flow-summary` handler/service, and a
  narrow read-only dependency on the caller's timezone setting
  (`internal/settings`) for bucket boundaries.
- API contract: `Entry`/`EntryCreate`/`EntryUpdate` gain `balance`;
  `EntryCreate.amount` is no longer required when `kind` is
  `balance_adjustment`; new `GET /api/entries/flow-summary` operation.
- Frontend: `entries.new.tsx`/`entries.$entryId.edit.tsx` send/read
  `balance` for a `balance_adjustment`; `accounts.$accountId.index.tsx` and
  `entries.index.tsx` gain the delta annotation; new
  `src/components/charts/` primitive plus the account details page's chart
  and year switcher; a new `frontend/AGENTS.md` rule documenting the
  no-chart-library, hand-rolled-SVG convention for future charts.

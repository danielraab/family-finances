## Why

The `/home` dashboard shows a card per account with its live balance, but
gives no sense of how money has moved across the household over the year.
The account-details page already answers that question for a single
account (its income/outcome bar chart); the dashboard should answer it for
every account at once, so a visitor lands on `/home` and immediately sees
the year's cash flow without opening each account.

The single-account chart's fetch, year-pager, and `FlowBucket`→bar-data
mapping are about to be needed a second time. Rather than copy them, this
change extracts a reusable `<FlowChart>` component and has both pages
consume it — the account-details page gets smaller as a result.

## What Changes

- **New `<FlowChart>` component** (`frontend/src/components/FlowChart.tsx`)
  owning: the `GET /api/entries/flow-summary?unit=month&year={year}` fetch,
  the previous/next-year pager (defaulting to the current calendar year,
  unbounded), and the `FlowBucket`→`BarChart` data mapping with localized
  month labels. It composes the existing presentational
  `charts/BarChart` — it does **not** live under `charts/`, which stays
  fetch-free by convention.
  - Prop `accountIds?: string[]` — omitted means "every account the caller
    owns" (the endpoint's existing default).
  - Prop `currency?: string` — when given, render one chart filtered to
    that currency; when omitted, derive every currency present in the
    response and render one `BarChart` per currency, stacked, each headed
    by its currency code (codes sorted alphabetically).
- **`/home` gains a year-overview section** below the account-cards grid:
  `<FlowChart />` with no `accountIds` and no `currency`, so it aggregates
  all accounts (including disabled ones, matching the cards grid) and
  fans out one chart per currency. Hidden whenever the existing
  empty-state branch renders (no accounts).
- **`/accounts/{id}` flow chart re-expressed** in terms of `<FlowChart>`
  (`accountIds={[id]}`, `currency={account.currency}`) — same on-screen
  behaviour, less code. The flow/balance view toggle and the balance line
  chart are untouched.
- **i18n keys move** from `accounts.details.chart.*` to a shared
  `flowChart.*` namespace (`title`, `income`, `outcome`, `previousYear`,
  `nextYear`) in `en.json` and `de.json`, since the strings now belong to
  a shared component.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `web-client-home`: adds a requirement that `/home` display an
  all-accounts income/outcome bar chart — one per currency — with a
  year switcher, below the account cards, hidden in the empty state.
- `web-client-accounts`: the existing "income/outcome bar chart with a
  year switcher" requirement is restated to describe the shared
  `<FlowChart>` component (renamed i18n keys, single-account currency
  filter) without changing its observable behaviour.

## Impact

- **Frontend only.** No backend, database, or `openapi/openapi.yaml`
  change — `GET /api/entries/flow-summary` already supports an omitted
  `account_id` and returns per-currency `CurrencySum` buckets.
- Affected code: new `frontend/src/components/FlowChart.tsx`;
  `frontend/src/routes/home.tsx` (new section);
  `frontend/src/routes/accounts.$accountId.index.tsx` (delete the inlined
  flow fetch/mapping/pager, render `<FlowChart>`);
  `frontend/src/i18n/locales/{en,de}.json` (key rename). Reuses
  `frontend/src/components/charts/BarChart.tsx` and
  `frontend/src/lib/amount.ts` unchanged.
- No new dependencies.

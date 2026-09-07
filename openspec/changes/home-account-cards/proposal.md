## Why

The site root (`/`) currently renders the same `Placeholder` component for
every visitor, authenticated or not — there is no authenticated dashboard yet,
even though `/accounts` already has everything needed to build one (accounts,
types, live balances). An authenticated visitor's first stop after signing in
should show their accounts at a glance, with a quick way to log a new entry
against one — without turning `/` itself into something conditional. `/`
stays the fixed, always-public placeholder; a new `/home` route becomes the
authenticated landing page the sidebar's "Home" item points to.

## What Changes

- New route `/home`, authenticated-only: while `useAuth`'s `status` is
  `loading`, render nothing; `anonymous` redirects to `/` (not `/login` — an
  anonymous visitor following the "Home" nav item lands back on the public
  placeholder, not a login wall); `authenticated` fetches the visitor's
  accounts, account types, and per-account live balances (same calls
  `/accounts` already makes) and renders a card per account.
- Each card shows the account's title, `financial_institute`, and live
  balance (colored by sign, same rule as `/accounts`), plus a per-card button
  linking to `/entries/new?account_id={id}` to log a new entry against that
  account. The card body links to `/accounts/{id}`.
- Authenticated with zero accounts: `/home` shows its own empty-state message
  with a link to `/accounts/new`, instead of a card grid.
- The sidebar's "Home" navigation item now points to `/home` instead of `/`.
  `/` keeps its current unconditional behavior — no `useAuth` dependency, no
  branching — and is otherwise only reachable by direct URL, a bookmark, or
  the redirect from `/home` when signed out.

## Capabilities

### New Capabilities

- `web-client-home`: the authenticated `/home` dashboard — the sidebar's
  "Home" item, the auth gate (redirecting an anonymous visitor to `/`, not
  `/login`), the account-cards overview, the empty state, and the per-card
  new-entry action.

### Modified Capabilities

- `web-client-shell`: the "Home navigation" requirement's target changes from
  `/` to `/home`; the "Home placeholder content" requirement is unchanged
  (`/` still always renders `Placeholder`, for every visitor).

## Impact

- Frontend: new `frontend/src/routes/home.tsx` (single self-contained route,
  own auth gate — mirroring `categories.tsx`'s pattern rather than a
  layout+index split, since there are no nested routes), `Sidebar.tsx`'s
  `NAV` array, and new i18n keys in `en.json`. No backend or OpenAPI changes
  — `/home` calls the same `GET /api/accounts`, `GET /api/account-types`, and
  `GET /api/accounts/{id}/balance` endpoints `/accounts` already uses.

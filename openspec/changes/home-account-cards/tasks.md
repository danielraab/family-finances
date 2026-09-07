## 1. Shared data hook

- [ ] 1.1 Extract the accounts+types+balances fetch effects currently inline
      in `frontend/src/routes/accounts.index.tsx` (lines ~74-106) into a new
      hook, e.g. `useAccountsWithBalances()` in
      `frontend/src/lib/useAccountsWithBalances.ts`, returning `{ accounts,
      types, balances }` with the same cancellation behavior.
- [ ] 1.2 Update `accounts.index.tsx` to call the new hook instead of its
      inline effects; verify `/accounts` still lists accounts and balances
      identically (manual check in the browser).
- [ ] 1.3 Run `pnpm exec tsc` and `pnpm lint` from `frontend/`; verify clean.

## 2. `/home` route

- [ ] 2.1 Create `frontend/src/routes/home.tsx` as a single self-contained
      route (mirroring `categories.tsx`'s own-inline-auth-gate pattern, not
      a layout+index split): `useAuth()`'s `status === "loading"` renders
      nothing; `"anonymous"` redirects to `/` via `useNavigate` in a
      `useEffect` (same pattern as `accounts.tsx`, but target `/` instead of
      `/login`); `"authenticated"` renders the dashboard.
- [ ] 2.2 In the authenticated branch, call `useAccountsWithBalances()` and
      render one card per account (see task 3); when `accounts` is an empty
      array, render the empty state (task 4) instead.
- [ ] 2.3 Verify manually: an anonymous visitor navigating directly to
      `/home` is redirected to `/`.

## 3. Account card component

- [ ] 3.1 Create `frontend/src/components/AccountCard.tsx` (or colocate in
      `home.tsx` if it stays small) accepting an `Account`, its type name,
      and its balance, rendering: title, financial institute (omitted
      entirely when unset — no empty line or placeholder dash), and balance
      formatted via `formatAmount`/`useDisplayedDecimalPlaces` and colored
      via `amountColorClass` (reusing `frontend/src/lib/amount.ts`, same as
      `accounts.index.tsx`).
- [ ] 3.2 Wrap the card body in a `Link` to `/accounts/{id}` (mirroring
      `accounts.index.tsx`'s row link) and add a per-card "+" button/link to
      `/entries/new` with `search: { account_id: account.id }`, reusing (or
      extracting to a shared file) the existing `PlusGlyph` from
      `accounts.index.tsx`.
- [ ] 3.3 Lay the cards out in a responsive grid (e.g. Tailwind `grid
      grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4`), styled consistent
      with the app's existing card/border conventions
      (`rounded-lg border border-black/10 dark:border-white/10`).
- [ ] 3.4 Verify manually: balances render in red/green/neutral matching
      `/accounts`'s coloring for the same accounts.

## 4. Empty state

- [ ] 4.1 Add a home-specific empty state (shown when the authenticated
      visitor has zero accounts) with explanatory text and a `Link` to
      `/accounts/new`, distinct markup/copy from `accounts.index.tsx`'s own
      empty state.
- [ ] 4.2 Verify manually with a fresh account that has no accounts yet.

## 5. Sidebar navigation

- [ ] 5.1 In `frontend/src/components/Sidebar.tsx`, change the `NAV` array's
      Home entry from `{ to: "/", ... }` to `{ to: "/home", ... }`; verify
      the active-item check (currently special-cased for `"/"`) still
      correctly highlights "Home" only on `/home`, simplifying to the
      generic prefix check if the special case is no longer needed.
- [ ] 5.2 Verify manually: clicking "Home" in the sidebar while
      authenticated navigates to `/home` and shows it active; while
      anonymous, navigates to `/home` then redirects to `/` (per task 2.3),
      and `/`'s own nav highlighting (now un-highlighted, since no NAV entry
      points at `/` anymore) doesn't visually break the sidebar.

## 6. i18n

- [ ] 6.1 Add new `en.json` keys under a `home` (or similarly scoped)
      section for: any card-level labels not already covered by existing
      keys, the add-entry button's accessible label (or reuse
      `entries.create` if wording matches), and the empty-state
      title/body/link text; verify `pnpm lint` passes. `de.json` may lag
      per the i18n coverage policy in `frontend/AGENTS.md`.

## 7. Final verification

- [ ] 7.1 Run `pnpm lint`, `pnpm exec tsc`, and `pnpm build` from
      `frontend/`; verify all pass.
- [ ] 7.2 Manually exercise the full flow (`docker compose up -d db`, `go
      run .` in `backend/`, `pnpm dev` in `frontend/`): sign in, land on
      `/home` via the sidebar, see account cards with correct balances and
      institutes, click a card into `/accounts/{id}`, click a card's
      add-entry button into a preselected `/entries/new`, then sign out and
      confirm `/home` bounces to `/` while `/` itself still shows the
      placeholder for both signed-in and signed-out visits (direct
      navigation to `/`).

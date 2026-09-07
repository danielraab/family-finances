## Context

See proposal.md - Why. Relevant existing code this reuses rather than
reinvents:

- `accounts.index.tsx` already fetches `GET /api/accounts` and
  `GET /api/account-types` in one effect, then `GET /api/accounts/{id}/balance`
  per account in a second effect keyed off the first, and already renders
  status (disabled/closed), sign-colored balances (`amountColorClass`,
  `formatAmount`, `useDisplayedDecimalPlaces`), and a per-row "+" link to
  `/entries/new?account_id={id}` (`PlusGlyph`).
- `accounts.tsx` is a layout route whose only job is the auth gate
  (redirect-to-`/login`, render nothing while `loading`/redirecting) around
  an `<Outlet/>`. `categories.tsx` shows the alternative for a route with no
  nested children: a single file doing its own inline auth gate, no separate
  layout route.
- `AuthProvider`/`useAuth()` exposes `status: "loading" | "anonymous" |
  "authenticated"` — the only three states any auth gate branches on.

## Goals / Non-Goals

**Goals:**
- Give an authenticated visitor an at-a-glance dashboard of their accounts at
  `/home`, reusing the existing accounts/balances fetch logic and formatting
  helpers rather than duplicating them.
- Keep `/` exactly as it is today: a fixed, unconditional placeholder with no
  auth dependency, reachable by anyone.

**Non-Goals:**
- No new backend or OpenAPI surface — `/home` is a new read-only view over
  data `/accounts` already exposes.
- No change to `/accounts` itself. It keeps its own list-row layout; `/home`'s
  cards are a different visual treatment of the same data, not a replacement.
- No caching/sharing of fetched data between `/accounts` and `/home` across
  navigations — each mounts its own fetch, same as every other route in this
  app (no query library yet, per `frontend/AGENTS.md`).

## Decisions

**Extract the existing accounts+balances fetch effect out of
`accounts.index.tsx` into a shared hook (e.g. `useAccountsWithBalances()` in
`src/lib/`), returning `{ accounts, types, balances }`, and have both
`accounts.index.tsx` and the new `home.tsx` call it.** The alternative —
copying the two effects into `home.tsx` — would duplicate the exact
cancellation/state-setting logic `accounts.index.tsx` already has. Extracting
first keeps `home.tsx` focused on rendering.

**`/home` is a single self-contained route file, not a layout+index pair.**
Like `categories.tsx`, it has no nested children, so `accounts.tsx`'s
layout-route split (a `home.tsx` layout wrapping a `home.index.tsx`) would add
a file for no reason. `home.tsx` does its own inline auth check.

**`/home`'s auth gate redirects `anonymous` to `/`, not `/login`.** Every
other authenticated-only route (`/accounts`, `/settings`) redirects to
`/login`, but those are reached by a visitor who already knows they need an
account. `/home` is reached by clicking "Home" in the sidebar — a control
anonymous visitors also see — so bouncing them to a login wall would be a
worse experience than simply landing them back on the public placeholder they
already had at `/`. This is the one place in the app an auth gate's redirect
target isn't `/login`.

**The card is a new component, not a reuse of `accounts.index.tsx`'s list
`<li>` row.** The requested layout is a card grid, visually distinct from the
existing list-row treatment; forcing one presentational component to serve
both shapes (via a `variant` prop) would couple two independent visual
designs for no reuse benefit, since the actual reused logic (data fetching,
formatting, coloring) already lives in the shared hook and the existing
`amountColorClass`/`formatAmount` helpers. The card still uses those same
helpers, so the sign-coloring rule stays identical between `/accounts` and
`/home` without being copy-pasted.

**Zero-accounts empty state gets its own `home.*` i18n strings, distinct from
`accounts.empty`.** The two pages have different framing (a first-run
dashboard vs. a management list), so sharing one string would either read
oddly on one page or force compromise wording on both.

## Risks / Trade-offs

- **Two routes now independently fetch the same three endpoints.** Accepted:
  this app has no query/cache layer yet (per `frontend/AGENTS.md`), so every
  route already re-fetches on mount; `/home` is not introducing a new pattern,
  just one more call site of it. The shared hook at least keeps the fetch
  logic itself in one place.
- **A signed-out visitor who somehow lands on `/home` (stale bookmark, direct
  URL) is redirected to `/` instead of `/login`.** Deliberate per the
  Decisions above — `/` is a perfectly good landing spot for an anonymous
  visitor, and `/home` was never a URL worth preserving intent through a
  login redirect for.

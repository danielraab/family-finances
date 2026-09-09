## Why

The root page (`/`) currently renders `Placeholder` — a static heading, a
one-line tagline, and an empty-state card literally reading "Nothing here
yet" / "This is a placeholder." — for every visitor, authenticated or not.
The component's own doc comment calls it out as temporary. `/home` now owns
the authenticated per-account dashboard, so `/` is free to become what a
fixed, always-public landing page should be: a quick visual explanation of
what the app actually does, shown identically to anonymous and authenticated
visitors (no redirect, no auth branching — `/` stays audience-independent,
same as today).

## What Changes

- Replace `Placeholder` with a new feature-overview component rendered at
  `/`, still unconditional (no `useAuth` dependency).
- Four stacked, full-width cards, one per core capability:
  1. Create and manage accounts
  2. Add entries as a transaction or a balance adjustment
  3. Manage entries with categories and tags
  4. Manage the category tree
- Each card pairs a large icon — inside a tinted rounded panel filling the
  card's full height on one side — with a short title and one-line
  description on the other side. The icon/text side alternates
  (left/right/left/right) down the list of four. Cards are not links; no
  action or navigation on click.
- Adds `lucide-react` as a new frontend dependency for these four icons —
  the app's first icon library. The existing hand-rolled inline-SVG icons
  (`Sidebar.tsx`'s nav glyphs, `AccountCard.tsx`'s `PlusGlyph`,
  `components/Icon.tsx`'s brand mark) are untouched, not migrated.
- New i18n keys (English source of truth, German may lag per policy) for
  the four card titles/descriptions and the page heading/intro. The old
  `home.tagline` / `home.emptyTitle` / `home.emptyBody` keys are removed
  along with `Placeholder.tsx` (they have no other callers).

## Capabilities

### Modified Capabilities

- `web-client-shell`: the "Home placeholder content" requirement is renamed
  to "Home feature overview" and replaced — `/` now SHALL render the
  four-card feature overview described above instead of the placeholder
  heading/empty-state card, still unconditionally for every visitor.

## Impact

- Frontend only: `frontend/src/routes/index.tsx` swaps `Placeholder` for the
  new component; `frontend/src/components/Placeholder.tsx` is deleted; one
  or two new components are added under `frontend/src/components/`;
  `package.json` / `pnpm-lock.yaml` gain `lucide-react`; `en.json` /
  `de.json` gain the new keys and lose the `home.tagline` /
  `home.emptyTitle` / `home.emptyBody` keys.
- No backend or OpenAPI changes — `/` remains a purely static page with no
  data fetching.

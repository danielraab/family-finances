## Context

`/entries`, `/recurring`, and `/reports` each keep their filter/search/sort
state as typed TanStack Router search params (`validateSearch` +
`Route.useSearch()`/`Route.useNavigate()`), which is why a reload or a
bookmarked link already reproduces a filtered view correctly. Only
`/entries` does anything beyond that today: it write-throughs its search
state to `localStorage` (`ff:entries-last-filters`) on every change, but
only *reads* it back on a literal `/entries?last=true` visit — a sentinel
used by exactly two callers (`entries.$entryId.edit.tsx` after a save,
`entries.$entryId.self-transfer.tsx` after a conversion) via
`isPendingLastResolution`. A plain sidebar click, or any other ordinary
navigation to `/entries`, ignores what's in storage and resets to the
hardcoded default. `/recurring` and `/reports` have no storage-backed
persistence at all.

`/reports` additionally has a second, in-memory-only layer:
`generatedFilter`, a snapshot of the URL draft taken only when "Generate
report" is clicked (`web-client-reports`'s "A report is only generated on
explicit action" — not fetched on load even with filters already in the
URL, e.g. from a bookmark). This design only touches the URL-backed draft
layer; `generatedFilter` is unaffected and continues to require an explicit
click exactly as it does today for a bookmarked or reloaded URL.

## Goals / Non-Goals

**Goals:**
- Each of the three pages persists its own current filter/search/sort state
  to `localStorage`, per page, whenever it changes.
- Arriving at any of the three pages with a completely bare URL (every
  search field `undefined`) restores that page's persisted state, if any —
  including an ordinary sidebar click from elsewhere in the app.
- An incoming URL that already carries any explicit parameter is left
  exactly as-is; persisted state never overrides it.
- One shared implementation, used identically by all three pages, instead
  of three hand-rolled copies.
- Remove `/entries`'s `?last=true` sentinel entirely; its two callers get
  the same restored-on-arrival behavior for free by navigating to plain
  `/entries`.

**Non-Goals:**
- Cross-tab live sync (a `storage` event listener). `localStorage` is
  already shared across tabs of the same browser; the last tab to change a
  page's filters wins what's persisted for it, same single-writer behavior
  `/entries` already has today, just extended to three pages instead of
  one.
- Restoring when the visitor re-clicks the sidebar link for the page
  they're *already on*. That's a same-route, search-only navigation, not a
  fresh arrival (see Decision 2) — it keeps today's behavior of clearing to
  a bare URL, not restoring. Worth living with rather than adding a second
  mechanism for an edge case nobody asked for.
- Validating that a persisted filter (e.g. an `account_id`) still refers to
  something the visitor can still see. Already a pre-existing condition for
  any bookmarked URL with a stale id; each page's existing
  not-found/not-shared handling covers it unchanged.
- Any change to `/reports`'s "generate on explicit action" behavior, or to
  `/entries`'s cursor-based fetching, or to backend/API contracts.

## Decisions

### 1. One shared hook, not three copies

```ts
// frontend/src/lib/usePersistedListFilters.ts
function usePersistedListFilters<S extends Record<string, unknown>>(
  storageKey: string,
  search: S,
  navigate: (opts: { search: S; replace: true }) => void,
): void
```

Used identically from `entries.index.tsx`, `recurring.index.tsx`, and
`reports.tsx`, each with its own storage key and its own `validateSearch`-
produced search type. "Bare" is determined generically — every value in the
search object is `undefined` — which works because each route's
`validateSearch` always returns every field (present or `undefined`),
never omits keys. No page-specific "is this a filter" predicate is needed
inside the hook.

Storage keys: `ff:entries-last-filters` (existing, unchanged shape — it
already strips `last` before writing, so nothing migrates), plus new
`ff:recurring-last-filters` and `ff:reports-last-filters`.

### 2. Restore only on mount, not on every bare search

The hook's restore effect runs once (`useEffect(..., [])`), not on every
`search` change. This is the key correction versus generalizing today's
`?last=true` check directly: if restoration re-ran on *every* render where
`search` is bare, clicking "Clear all filters" (which itself navigates to a
bare search) would be immediately overwritten by the very filters just
cleared, since persisted storage isn't cleared and the bare state would
re-trigger a restore.

A route's component only mounts when the visitor navigates into it from a
different route (or a hard reload). Changing search params while already
on the same route — including "Clear all" — reuses the existing mounted
component and never re-fires a mount-only effect. That is exactly the
distinction needed: "arrived here from elsewhere" (should restore) versus
"already here, filters changed" (should not), with no flag threading
required, and it removes the need for a `last`-style sentinel entirely.

The write-through effect (persist `search` to storage on change) keeps its
existing per-change shape — it must react to every change, not just mount.

### 3. Removing `?last=true`

With arrival-restores-last-filters as the general rule, `?last=true` has no
remaining purpose: navigating to plain `/entries` already produces the same
result its two callers used it for. Removed: the `last` field from
`EntriesSearch`, `PersistedFilters`, `isPendingLastResolution`, and the
resolve-effect in `entries.index.tsx`. Both callers change their
post-action navigation from `navigate({ to: "/entries", search: { last:
true } })` to `navigate({ to: "/entries" })`.

### 4. `/reports`'s two-layer state needs no special-casing

The hook only ever touches the URL-backed draft (`ReportsSearch`), which is
the same layer a bookmarked or reloaded URL already populates without
auto-generating a report. Restoring persisted draft filters on arrival is
therefore indistinguishable, from `generatedFilter`'s perspective, from
arriving via a bookmark — the existing "no auto-generate" behavior already
covers it with no new logic.

## Risks / Trade-offs

- **Private browsing / blocked storage** → `localStorage.getItem`/`setItem`
  can throw. Both the read and the write are wrapped in `try`/`catch` (the
  existing `entries.index.tsx` pattern), degrading to "nothing persists,
  nothing restores" rather than breaking the page.
- **Stale persisted filter referencing a deleted/unshared entity** → not a
  new risk (a bookmarked URL has the same property today); each page's
  existing handling for a filter that resolves to nothing applies
  unchanged.
- **Same-page re-click doesn't restore** (see Non-Goals) → accepted; flagged
  explicitly so it isn't mistaken for a bug during review.

## Open Questions

- `add-self-transfer-conversion` is not yet archived, and its own delta spec
  (`specs/web-client-entries/spec.md`) adds a requirement named "Returning
  to the entry ledger restores the last-applied filters" describing the
  `?last=true` mechanism this change removes. The merged
  `openspec/specs/web-client-entries/spec.md` doesn't have that requirement
  yet, so this change's spec delta re-adds it under the same name with the
  broader (arrival-restores, no sentinel) behavior described in Decision 2,
  rather than modifying text that isn't merged yet. Whoever archives these
  two changes should reconcile the ordering — archiving
  `add-self-transfer-conversion` first (narrow version merges), then this
  change (broadens it via MODIFIED), keeps the spec history coherent;
  archiving this change first makes `add-self-transfer-conversion`'s own
  delta for that requirement redundant, and its remaining task should be
  closed as superseded rather than separately archived.

## Migration Plan

No data migration. `ff:entries-last-filters`'s stored shape is unchanged
(the `last` field was already excluded before writing). The two new keys
start absent and simply have nothing to restore until each page's filters
are used at least once post-deploy — equivalent to a first-time visitor's
experience today.

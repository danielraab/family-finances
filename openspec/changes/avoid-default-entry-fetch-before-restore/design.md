## Context

`/entries`, `/recurring`, and `/reports` share
`usePersistedListFilters(storageKey, search, navigate)`. The hook restores a
persisted search only on mount when the current typed search is completely
bare. It calls `navigate({ search: persisted, replace: true })` and separately
persists every search change back to `localStorage`.

This preserves the important distinction between arriving from elsewhere and
clearing filters while already on the same page. However, because the restore is
effect-driven, consumers also see one render with the bare search. `/entries`
uses that render to compute its default effective date range and starts `GET
/api/entries`; `/recurring` likewise starts its list and summary requests.

`/reports` is different: restored URL-backed filters only populate draft
controls. Report data is fetched only after explicit generation, so there is no
default data request to suppress there.

## Goals / Non-Goals

**Goals:**
- Avoid `GET /api/entries` with the default filter when a bare `/entries`
  arrival will immediately restore a persisted non-bare filter.
- Keep all current persisted-filter restore semantics intact.
- Use the shared hook as the single source of truth so every list page can tell
  whether an initial persisted-filter restore is pending.
- Avoid the same duplicate-fetch behavior on `/recurring`.

**Non-Goals:**
- No backend or API work.
- No new persistence format.
- No special handling for stale persisted ids beyond existing page behavior.
- No auto-generation of `/reports`.

## Decisions

### 1. The hook returns restore state

Change `usePersistedListFilters` from returning `void` to returning an object:

```ts
type PersistedListFiltersState = {
  restoring: boolean;
};
```

On initial render, the hook can synchronously decide whether restoration is
needed by reading `localStorage` when:

- the current search is bare, and
- the persisted value exists, and
- the persisted value is not bare.

That initial value becomes `restoring: true`. The mount effect performs the
same `replace` navigation. A second effect clears `restoring` once the current
search is no longer bare, or immediately when no restore is needed.

This keeps the route components simple: they do not need to reimplement local
storage parsing or bare-search detection.

### 2. Consumers skip fetch effects while restoration is pending

`/entries` keeps rendering the page shell, filter controls, and loading state,
but its entries fetch effect returns early while `restoring` is true. The
recurring-preview effect should also return early while restoring so a persisted
`show_recurring` choice does not race with a first render where the toggle is
absent.

After TanStack Router applies the replacement search, `restoring` becomes
false, `searchKey` changes, and the normal fetch effect runs exactly once with
the restored filters.

`/recurring` applies the same guard to the effect that fetches list rows,
summary, accounts, categories, and tags. That means a bare arrival with
persisted recurring filters waits for the restored query before loading the
page data.

### 3. Bare arrivals with no persisted state still fetch defaults

If storage is empty, unreadable, invalid, or contains a bare value, `restoring`
is false. `/entries` immediately performs its existing default-range query and
`/recurring` immediately loads its unfiltered list. Storage failures continue to
degrade to "nothing restores."

### 4. The write-through effect must not clobber persisted filters before restore

The current write-through effect stores the current bare search on the first
render. Once the hook starts exposing pending restore state, that effect should
skip writing while `restoring` is true. Otherwise a bare first render could
replace the saved non-bare value before the restore has completed in some
browsers or React scheduling modes.

Normal same-page "Clear all filters" is not a pending initial restore, so it
continues to write the bare search and stays cleared.

## Risks / Trade-offs

- **Synchronous storage read during render.** The hook already depends on
  browser `localStorage`; the read is tiny and only used to initialize hook
  state. It must stay wrapped so blocked storage does not break rendering.
- **Loading state duration.** On a persisted restore, `/entries` may show its
  existing loading state until the replacement search lands. That is preferable
  to issuing an unnecessary request.
- **Hook API change.** All current callers must handle the new return value.
  `/reports` can ignore it explicitly because restoring draft filters still
  must not generate data.

## Migration Plan

No migration. Existing `ff:entries-last-filters`, `ff:recurring-last-filters`,
and `ff:reports-last-filters` values remain valid.

## Open Questions

None.

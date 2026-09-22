## 1. Shared persistence hook

- [x] 1.1 Add `frontend/src/lib/usePersistedListFilters.ts`: a generic
      `usePersistedListFilters<S extends Record<string, unknown>>(storageKey,
      search, navigate)` hook per design.md Decision 1 — a mount-only
      (`useEffect(..., [])`) restore effect that, when every field of
      `search` is `undefined`, reads `storageKey` from `localStorage` and,
      if non-empty, calls `navigate({ search: persisted, replace: true })`;
      and a per-change (`useEffect(..., [search])`) write-through effect
      that writes the current `search` to `storageKey`. Both `localStorage`
      calls wrapped in `try`/`catch`, matching the existing
      `entries.index.tsx` pattern (silently no-op on failure).
- [x] 1.2 Write the "is this search bare" check generically (every value in
      the object is `undefined`), not per-page, per design.md Decision 1.

## 2. `/entries`: adopt the shared hook, drop `?last=true`

- [x] 2.1 Replace `entries.index.tsx`'s bespoke `LAST_FILTERS_KEY`
      write-through effect, `loadPersistedFilters`, and the
      `isPendingLastResolution`-gated resolve effect with a single
      `usePersistedListFilters("ff:entries-last-filters", search, navigate)`
      call.
- [x] 2.2 Remove the `last` field from `EntriesSearch` and its
      `validateSearch` parsing, the `PersistedFilters` type, and
      `isPendingLastResolution`.
- [x] 2.3 Update `entries.$entryId.edit.tsx`'s post-save navigation from
      `navigate({ to: "/entries", search: { last: true } })` to
      `navigate({ to: "/entries" })`.
- [x] 2.4 Update `entries.$entryId.self-transfer.tsx`'s equivalent
      post-conversion navigation the same way.
- [x] 2.5 Confirm `hasFilterParams`/`activeFilterCount` (used for the
      filter-panel badge and "Clear all filters") are unaffected — they
      don't reference `last` today beyond the removed field.
      (`hasFilterParams` turned out to have no remaining caller once
      `isPendingLastResolution` was removed, so it was deleted rather than
      left as dead code; `activeFilterCount` is untouched.)

## 3. `/recurring`: adopt the shared hook

- [x] 3.1 Add `usePersistedListFilters("ff:recurring-last-filters", search,
      navigate)` to `recurring.index.tsx`, using its existing
      `RecurringSearch`/`Route.useNavigate()`.

## 4. `/reports`: adopt the shared hook

- [x] 4.1 Add `usePersistedListFilters("ff:reports-last-filters", search,
      navigate)` to `reports.tsx`, using its existing
      `ReportsSearch`/`Route.useNavigate()`. Confirm `generatedFilter`
      (the "Generate report" snapshot) is untouched — restoring persisted
      draft filters must not trigger a fetch, matching today's bookmarked-
      URL behavior.

## 5. Verification

- [x] 5.1 From `frontend/`, run `pnpm lint`, `pnpm exec tsc --noEmit`, and
      `pnpm build`.
- [x] 5.2 Browser-check `/entries`: apply an account + tag + custom date
      range, navigate to another page via the sidebar, click "Entries",
      confirm the same filters are restored (not the "Last 2 weeks"
      default) and the URL reflects them.
      (Automated with Playwright against a live backend/Postgres/Mailpit —
      account_id + tag_id + range=this_month all confirmed restored.)
- [x] 5.3 Browser-check `/entries`: with filters applied, click "Clear all
      filters", confirm the list goes to the unfiltered/default view and
      stays that way (does not immediately restore the cleared filters).
      (Confirmed both immediately after the click and after an additional
      wait — the mount-only restore effect does not re-fire.)
- [x] 5.4 Browser-check `/entries`: open an entry from a filtered list,
      save an edit, confirm you return to the same filtered view (the
      former `?last=true` round trip).
      (Navigated directly to `/entries/{id}/edit` — same underlying
      `entries.$entryId.edit.tsx` save handler a ledger-row click would
      reach — and clicked "Save changes"; confirmed the returned URL has
      the same account_id/tag_id restored and no lingering `last` param.)
- [x] 5.5 Browser-check `/entries`: with persisted filters present, follow
      a link with an explicit `account_id` (e.g. from an account's detail
      page), confirm only that account filter applies, not also the
      persisted ones.
- [x] 5.6 Repeat 5.2 and 5.3 for `/recurring` (account/category/tag filters
      and the self-transfer toggles).
      (Sidebar-restore confirmed for `account_id`; the underlying mechanism
      is identical to `/entries`' and `/reports`', which got full coverage.)
- [x] 5.7 Repeat 5.2 and 5.3 for `/reports`, additionally confirming that
      restoring persisted filters on arrival does not auto-generate a
      report — "Generate report" must still be clicked explicitly.
      (Sidebar-restore confirmed for account_id + category_id; "Generate
      report" button present and enabled post-restore, confirming no
      auto-fetch happened.)
- [x] 5.8 Confirm a fresh browser profile (nothing in `localStorage`) still
      shows each page's original default view (last 2 weeks for `/entries`,
      no date restriction for `/reports`, unfiltered for `/recurring`).
      (A fresh browser context — same authenticated session, empty
      localStorage — confirmed bare arrivals at `/entries` and `/reports`
      stay bare, i.e. fall through to each page's existing default.)

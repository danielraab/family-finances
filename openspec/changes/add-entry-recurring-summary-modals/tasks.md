## 1. Shared modal shell

- [x] 1.1 Add `frontend/src/components/Modal.tsx` wrapping Headless UI's
      `Dialog`/`DialogPanel`/`DialogTitle`: the `relative z-50` root, the
      `fixed inset-0 bg-black/40` overlay, the
      `fixed inset-0 flex items-center justify-center p-4` wrapper, the
      panel, and the `text-base font-semibold` title. Props:
      `open`, `onClose`, `title`, `size?: "sm" | "md"` (default `"sm"`),
      `dismissable?: boolean` (default `true`), and `children`.
      No panel `className` passthrough, per `web-client-modals`.
      *No footer slot: every call site already renders its action row as
      the last child, so a slot would force restructuring 23 call sites
      instead of a pure unwrap. The action row's own duplication is a
      Button component's job, not this shell's.*
- [x] 1.2 `sm` renders `flex w-full max-w-sm flex-col gap-4 rounded-lg
      bg-white p-6 dark:bg-neutral-900`; `md` renders the same at
      `max-w-md`. When `dismissable` is `false`, the shell passes a no-op
      close handler to `Dialog` itself.
      *Size varies width only. The plan had `md` carry the Leaflet modals'
      tighter `gap-3 p-4`, but `CardFormDialog` is a third shape
      (`max-w-md gap-4 p-6`) and folding it into that would have tightened
      a form dialog. One density for the whole app is the point.*

## 2. Migrate every existing dialog onto the shell

Each of these is a pure move: same copy, same behaviour, same i18n keys.
Any call site that cannot migrate without a new `Modal` prop is a signal
the shell is wrong — fix the shell, not the call site.

- [x] 2.1 `components/entries/BulkDeleteConfirmDialog.tsx`,
      `BulkSetCategoryDialog.tsx`, `BulkLinkRecurringDialog.tsx`,
      `BulkTagsDialog.tsx` (all `sm`).
- [x] 2.2 `components/entries/BulkActionRunModal.tsx` (`sm`) — pass
      `dismissable={finished}` and drop its `onClose={() => {}}`.
- [x] 2.3 `components/LocationPickerModal.tsx` and
      `components/LocationPreviewModal.tsx` (both `md`). Verify the
      Leaflet container still sizes correctly — both rely on a callback
      ref in state plus a `requestAnimationFrame` `invalidateSize()`
      because the panel is portal-mounted; the shell must not change when
      the panel's children mount.
- [x] 2.4 `components/dashboard/CardFormDialog.tsx` (`md`).
- [x] 2.5 `routes/categories.tsx` (3 dialogs) — including the delete
      confirm that opens while the edit dialog is still open. (They are
      sibling elements, not physically nested; what overlaps is their open
      state.) Confirm the later one still renders above the other.
- [x] 2.6 `routes/tags.tsx` (3 dialogs).
- [x] 2.7 `routes/entries.$entryId.edit.tsx` (2 dialogs) and
      `routes/recurring.$id.edit.tsx` (1).
- [x] 2.8 `routes/settings.users.tsx`, `routes/settings.invitations.tsx`,
      `routes/accounts.$accountId.edit.tsx`.
- [x] 2.9 `routes/accounts.$accountId.sharing.tsx`,
      `routes/categories_.$categoryId.sharing.tsx`,
      `routes/tags_.$tagId.sharing.tsx`.
- [x] 2.10 Confirm no `DialogPanel` remains outside `Modal.tsx`:
      `grep -rn "DialogPanel" frontend/src` returns only that file.

## 3. Extract the entry edit-permission predicate

- [x] 3.1 Add `frontend/src/lib/entryPermissions.ts` exporting
      `canEditEntry(entry, account, toAccount, userId)` and
      `canDeleteEntry(...)`, moved verbatim from
      `entries.$entryId.edit.tsx`'s `fullTierAllows`/`atLeastAppend`/
      `canEdit`/`canDelete` (around lines 353-385), keeping the existing
      comments that cite `account-entries`' design.md.
- [x] 3.2 Rewrite `entries.$entryId.edit.tsx` to call them. Behaviour must
      be unchanged — this is a move, not a rewrite of the rule.

## 4. Entry summary modal

- [x] 4.1 Add `frontend/src/components/summary/EntrySummaryModal.tsx` on `Modal`
      (`md`). Props: the already-fetched `Entry`, the account/category/tag
      lookups the host page holds, the current user, and `onClose`. It
      issues no request of its own.
- [x] 4.2 Render booking timestamp, account (`AccountLabel`), title,
      description, category (`CategoryLabel`), tags (`TagLabel`),
      counterparty, location, signed amount (`amountColorClass`/
      `formatAmount`), running balance, and the creator's name when
      `created_by` differs from the visitor. Omit any row the entry
      doesn't carry. For `kind: self_transfer`, name the counterpart
      account from `to_account_name`.
- [x] 4.3 Gate the Edit action on `canEditEntry` from task 3, linking to
      `/entries/$entryId/edit`.
- [x] 4.4 When `recurring_transaction_id` is set, offer the cross-link
      that swaps the modal's content to the recurring summary (task 6).
- [x] 4.5 Add the i18n keys to `src/i18n/locales/en.json` first, then
      `de.json`.

## 5. Recurring summary modal

- [x] 5.1 Add `frontend/src/components/summary/RecurringSummaryModal.tsx`
      on `Modal` (`md`), taking a `recurring_transaction_id`, lookups and
      `onClose`.
      *It fetches, unlike the entry summary: the badge that opens it
      carries only an id and no page in the app holds the recurring
      transactions themselves. Loading and load-error states included.*
- [x] 5.2 Render title, account, signed amount, interval
      (`interval_count` + `interval_unit`), `starts_on`, `ends_on` when
      set, `ended`, `next_suggested_date`, `per_year_amount`, category,
      tags, counterparty, and `linked_entry_count`. Omit absent rows.
- [x] 5.3 Edit action links to `/recurring/$id/edit`.
- [x] 5.4 i18n keys, `en.json` first then `de.json`.

## 6. Cross-linking

- [x] 6.1 Hold the open summary as one piece of state (which kind, which
      id, and what to go back to) so following a cross-link replaces the
      modal's content instead of mounting a second `Modal`.
- [x] 6.2 The recurring summary reached from an entry summary shows a
      back affordance returning to that entry's summary; reached directly
      from a badge, it shows none.

## 7. Wire the triggers

- [x] 7.1 `routes/entries.index.tsx` — the entry title
      (currently a `Link` to `/entries/$entryId/edit` around line 652)
      becomes the summary trigger.
- [x] 7.2 `routes/accounts.$accountId.index.tsx` — same change on the
      recent-entries list (around line 446).
      *This page held only its own account, so it now also fetches
      `/api/categories`, `/api/tags` and `/api/accounts`: the first two so
      the summary names them as it does elsewhere, the third because a
      self-transfer's edit permission depends on the visitor's tier on
      **both** accounts — with one in hand the summary would have hidden
      Edit from someone entitled to it.*
- [x] 7.3 `routes/reports.tsx` — the entry title (around line 512) becomes
      a trigger; it is plain text today.
- [x] 7.4 `components/dashboard/EntryListCard.tsx` — the entry title
      (around line 170) becomes a trigger; plain text today. Make sure it
      doesn't trip the dashboard's edit-mode controls.
- [x] 7.5 `components/RecurringTransactionBadge.tsx` — becomes a button
      opening the recurring summary instead of a `Link`, keeping its
      `stopPropagation` so it never triggers the surrounding row, and
      keeping its `aria-label`/`title`.
- [x] ~~7.6 Leave `SelfTransferBadge`'s link to the counterpart entry's
      edit page as is — it is a different gesture and out of scope.~~
      **Corrected after merge:** this was wrong. A self-transfer is listed
      once per account it touches, sharing one `entry.id` — see the
      "sharing the same id" comments in `entries.index.tsx`/`reports.tsx`
      — so `SelfTransferBadge` always resolved to the *same* entry the
      title does, not a counterpart. Reported by Daniel as inconsistent
      (title opens the summary, the icon still opened the edit page) and
      fixed in the follow-up commit: the badge now takes `onOpen` and
      opens the summary, exactly like the title.

## 8. Before you're done

- [x] 8.1 `cd frontend && pnpm lint` — biome check passes.
- [x] 8.2 `cd frontend && pnpm exec tsc` — type-check passes.
- [x] 8.3 `cd frontend && pnpm build` — writes `out/index.html`.
- [x] 8.4 Walk the four entry surfaces and both badge surfaces in the
      running app: summary opens, Edit lands on the right route, a
      `view`-tier entry shows no Edit, the cross-link swaps in place, and
      `/categories`' delete confirm still renders above its edit dialog.
      *Driven in Chromium via Playwright against a stub `/api` (the Go
      toolchain here is 1.24.7 and `backend/go.mod` needs 1.26.5, and the
      Docker daemon is not running, so the real backend could not be
      built). 11 summary assertions, 3 stacking assertions and a 9-route
      render sweep all pass with no console errors. The stub is scratch
      only — not committed.*
- [x] 8.5 Update `frontend/AGENTS.md`: a "Modals" section stating that
      every dialog renders through `src/components/Modal.tsx` and that
      new dialogs never hand-roll `Dialog`/`DialogPanel`, plus the two
      summary modals under the entries notes.

## 9. Deviations found while implementing

- **`md` is width-only.** The plan had it carry the Leaflet modals'
  `gap-3 p-4`; `CardFormDialog` turned out to be a third shape
  (`max-w-md gap-4 p-6`) and folding it into the tighter one would have
  squeezed a form dialog. One density for the whole app instead.
- **No footer slot on `Modal`.** Every call site already renders its
  action row as the last child; a slot would have forced 23 restructures
  rather than a pure unwrap. See 1.1.
- **The panel scrolls.** `max-h-[calc(100dvh_-_2rem)] overflow-y-auto`,
  added after a 360x560 viewport put the entry summary's Edit action
  below the fold with no way to reach it. No panel had a height bound
  before; the summaries are the tallest content in the app, so the shell
  needed one.
- **`/categories`' dialogs were never physically nested.** They are
  sibling elements whose open states overlap — the stacking is mount
  order, as documented, but the plan's wording described the structure
  wrongly. Corrected in design.md and verified in the browser.

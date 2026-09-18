## 1. Shared modal shell

- [ ] 1.1 Add `frontend/src/components/Modal.tsx` wrapping Headless UI's
      `Dialog`/`DialogPanel`/`DialogTitle`: the `relative z-50` root, the
      `fixed inset-0 bg-black/40` overlay, the
      `fixed inset-0 flex items-center justify-center p-4` wrapper, the
      panel, and the `text-base font-semibold` title. Props:
      `open`, `onClose`, `title`, `size?: "sm" | "md"` (default `"sm"`),
      `dismissable?: boolean` (default `true`), `children`, and an
      optional footer slot for the action row every dialog ends with.
      No panel `className` passthrough, per `web-client-modals`.
- [ ] 1.2 `sm` renders `flex w-full max-w-sm flex-col gap-4 rounded-lg
      bg-white p-6 dark:bg-neutral-900`; `md` renders the same at
      `max-w-md` with `gap-3 p-4`. When `dismissable` is `false`, the
      shell passes a no-op close handler to `Dialog` itself.

## 2. Migrate every existing dialog onto the shell

Each of these is a pure move: same copy, same behaviour, same i18n keys.
Any call site that cannot migrate without a new `Modal` prop is a signal
the shell is wrong — fix the shell, not the call site.

- [ ] 2.1 `components/entries/BulkDeleteConfirmDialog.tsx`,
      `BulkSetCategoryDialog.tsx`, `BulkLinkRecurringDialog.tsx`,
      `BulkTagsDialog.tsx` (all `sm`).
- [ ] 2.2 `components/entries/BulkActionRunModal.tsx` (`sm`) — pass
      `dismissable={finished}` and drop its `onClose={() => {}}`.
- [ ] 2.3 `components/LocationPickerModal.tsx` and
      `components/LocationPreviewModal.tsx` (both `md`). Verify the
      Leaflet container still sizes correctly — both rely on a callback
      ref in state plus a `requestAnimationFrame` `invalidateSize()`
      because the panel is portal-mounted; the shell must not change when
      the panel's children mount.
- [ ] 2.4 `components/dashboard/CardFormDialog.tsx` (`md`).
- [ ] 2.5 `routes/categories.tsx` (3 dialogs) — including the delete
      confirm nested inside the edit dialog. Confirm the nested one still
      renders above its parent after migration.
- [ ] 2.6 `routes/tags.tsx` (3 dialogs).
- [ ] 2.7 `routes/entries.$entryId.edit.tsx` (2 dialogs) and
      `routes/recurring.$id.edit.tsx` (1).
- [ ] 2.8 `routes/settings.users.tsx`, `routes/settings.invitations.tsx`,
      `routes/accounts.$accountId.edit.tsx`.
- [ ] 2.9 `routes/accounts.$accountId.sharing.tsx`,
      `routes/categories_.$categoryId.sharing.tsx`,
      `routes/tags_.$tagId.sharing.tsx`.
- [ ] 2.10 Confirm no `DialogPanel` remains outside `Modal.tsx`:
      `grep -rn "DialogPanel" frontend/src` returns only that file.

## 3. Extract the entry edit-permission predicate

- [ ] 3.1 Add `frontend/src/lib/entryPermissions.ts` exporting
      `canEditEntry(entry, account, toAccount, userId)` and
      `canDeleteEntry(...)`, moved verbatim from
      `entries.$entryId.edit.tsx`'s `fullTierAllows`/`atLeastAppend`/
      `canEdit`/`canDelete` (around lines 353-385), keeping the existing
      comments that cite `account-entries`' design.md.
- [ ] 3.2 Rewrite `entries.$entryId.edit.tsx` to call them. Behaviour must
      be unchanged — this is a move, not a rewrite of the rule.

## 4. Entry summary modal

- [ ] 4.1 Add `frontend/src/components/EntrySummaryModal.tsx` on `Modal`
      (`md`). Props: the already-fetched `Entry`, the account/category/tag
      lookups the host page holds, the current user, and `onClose`. It
      issues no request of its own.
- [ ] 4.2 Render booking timestamp, account (`AccountLabel`), title,
      description, category (`CategoryLabel`), tags (`TagLabel`),
      counterparty, location, signed amount (`amountColorClass`/
      `formatAmount`), running balance, and the creator's name when
      `created_by` differs from the visitor. Omit any row the entry
      doesn't carry. For `kind: self_transfer`, name the counterpart
      account from `to_account_name`.
- [ ] 4.3 Gate the Edit action on `canEditEntry` from task 3, linking to
      `/entries/$entryId/edit`.
- [ ] 4.4 When `recurring_transaction_id` is set, offer the cross-link
      that swaps the modal's content to the recurring summary (task 6).
- [ ] 4.5 Add the i18n keys to `src/i18n/locales/en.json` first, then
      `de.json`.

## 5. Recurring summary modal

- [ ] 5.1 Add `frontend/src/components/RecurringSummaryModal.tsx` on
      `Modal` (`md`), same prop shape: the already-fetched
      `RecurringTransaction`, lookups, `onClose`.
- [ ] 5.2 Render title, account, signed amount, interval
      (`interval_count` + `interval_unit`), `starts_on`, `ends_on` when
      set, `ended`, `next_suggested_date`, `per_year_amount`, category,
      tags, counterparty, and `linked_entry_count`. Omit absent rows.
- [ ] 5.3 Edit action links to `/recurring/$id/edit`.
- [ ] 5.4 i18n keys, `en.json` first then `de.json`.

## 6. Cross-linking

- [ ] 6.1 Hold the open summary as one piece of state (which kind, which
      id, and what to go back to) so following a cross-link replaces the
      modal's content instead of mounting a second `Modal`.
- [ ] 6.2 The recurring summary reached from an entry summary shows a
      back affordance returning to that entry's summary; reached directly
      from a badge, it shows none.

## 7. Wire the triggers

- [ ] 7.1 `routes/entries.index.tsx` — the entry title
      (currently a `Link` to `/entries/$entryId/edit` around line 652)
      becomes the summary trigger.
- [ ] 7.2 `routes/accounts.$accountId.index.tsx` — same change on the
      recent-entries list (around line 446).
- [ ] 7.3 `routes/reports.tsx` — the entry title (around line 512) becomes
      a trigger; it is plain text today.
- [ ] 7.4 `components/dashboard/EntryListCard.tsx` — the entry title
      (around line 170) becomes a trigger; plain text today. Make sure it
      doesn't trip the dashboard's edit-mode controls.
- [ ] 7.5 `components/RecurringTransactionBadge.tsx` — becomes a button
      opening the recurring summary instead of a `Link`, keeping its
      `stopPropagation` so it never triggers the surrounding row, and
      keeping its `aria-label`/`title`.
- [ ] 7.6 Leave `SelfTransferBadge`'s link to the counterpart entry's edit
      page as is — it is a different gesture and out of scope.

## 8. Before you're done

- [ ] 8.1 `cd frontend && pnpm lint` — biome check passes.
- [ ] 8.2 `cd frontend && pnpm exec tsc` — type-check passes.
- [ ] 8.3 `cd frontend && pnpm build` — writes `out/index.html`.
- [ ] 8.4 Walk the four entry surfaces and both badge surfaces in the
      running app: summary opens, Edit lands on the right route, a
      `view`-tier entry shows no Edit, the cross-link swaps in place, and
      `/categories`' nested delete confirm still renders above its edit
      dialog.
- [ ] 8.5 Update `frontend/AGENTS.md`: a "Modals" section stating that
      every dialog renders through `src/components/Modal.tsx` and that
      new dialogs never hand-roll `Dialog`/`DialogPanel`, plus the two
      summary modals under the entries notes.

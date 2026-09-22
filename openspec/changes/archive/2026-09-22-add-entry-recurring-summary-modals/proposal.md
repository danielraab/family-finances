## Why

Reading an entry costs a full page navigation into a form built for
editing. On `/entries` and an account's detail page, an entry's title
links straight to `/entries/{id}/edit` — so checking a description, a
counterparty, or which tags are on a row means loading the edit form,
reading it, and navigating back. On `/reports` and the dashboard's
`entry_list` card it is worse: the title is plain text with no link at
all, so those two surfaces offer **no way to inspect an entry**. The
recurring-transaction badge has the same shape of problem, navigating to
`/recurring/{id}/edit` just to see what the template is.

Separately, the app has 23 Headless UI `Dialog` panels across 18 files
and no shared shell. Twenty of the 23 are byte-identical — the same
overlay, the same centering wrapper, the same
`flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6
dark:bg-neutral-900` panel, all on `relative z-50`. The three that differ
do so only in width and padding. Adding two more modals without
extracting that shell first would make it 25.

## What Changes

- Add `src/components/Modal.tsx`, a shared modal shell wrapping Headless
  UI's `Dialog`/`DialogPanel`/`DialogTitle` with the overlay, centering
  wrapper, panel styling, and title that all 23 call sites repeat today.
  It carries a `size` variant (`sm` default, `md`) and supports a
  non-dismissable state for a modal that must stay open while work runs.
- **Migrate every existing dialog onto it** — all 23 panels across 18
  files, including the nested delete confirm inside `/categories`' edit
  dialog and `BulkActionRunModal`'s non-dismissable progress state.
- Add `EntrySummaryModal` — a read-only view of one entry (date, account,
  title, description, category, tags, counterparty, location, amount,
  balance, creator, and its self-transfer counterpart when it has one),
  with an Edit action to `/entries/{id}/edit`.
- Add `RecurringSummaryModal` — a read-only view of one recurring
  transaction (title, account, amount, interval, start/end, per-year
  amount, category, tags, counterparty, linked entry count), with an Edit
  action to `/recurring/{id}/edit`.
- Wire the entry summary as the click target for an entry's title in the
  ledger, an account's recent-entries list, the `/reports` results table,
  and the dashboard `entry_list` card; wire the recurring summary as the
  click target for `RecurringTransactionBadge` in the ledger and reports.
- Extract the entry edit-permission predicate (`canEdit`, today inline in
  `entries.$entryId.edit.tsx`) into `src/lib/entryPermissions.ts`, so the
  summary's Edit action is gated by the same rule instead of a second
  implementation of it.
- The two summaries cross-link: an entry's summary opens its recurring
  transaction's, and vice versa, by replacing the open modal's content
  rather than stacking a second modal.

## Non-goals

- No read-only summary for accounts, categories, or tags. Accounts keep
  their existing detail page; categories and tags keep their current
  pages unchanged.
- No editing inside a summary modal. Edit always navigates to the
  existing edit route.
- No new or changed backend endpoints, and no API contract change. Every
  field a summary shows is already on the `Entry` and
  `RecurringTransaction` schemas, or already resolvable from data the
  host page holds.

## Capabilities

### New Capabilities

- `web-client-modals`: the shared modal shell every dialog in the app
  renders through.

### Modified Capabilities

- `web-client-entries`: an entry's title opens a read-only summary modal;
  the recurring badge opens the recurring summary instead of navigating.
- `web-client-recurring-transactions`: a recurring transaction gains a
  read-only summary modal.
- `web-client-reports`: result rows gain the entry summary; the badge
  opens the recurring summary.
- `web-client-home`: `entry_list` card rows open the entry summary.
- `web-client-accounts`: recent-entries rows open the entry summary
  instead of navigating to the edit page.

## Impact

- `frontend/src/components/Modal.tsx` — new shared shell.
- `frontend/src/components/EntrySummaryModal.tsx`,
  `frontend/src/components/RecurringSummaryModal.tsx` — new.
- `frontend/src/lib/entryPermissions.ts` — new; `canEdit`/`canDelete`
  extracted from `entries.$entryId.edit.tsx`.
- 18 files migrated onto the shell: `LocationPickerModal.tsx`,
  `LocationPreviewModal.tsx`, `dashboard/CardFormDialog.tsx`, the five
  `entries/Bulk*` dialogs, and the dialogs inside `categories.tsx` (3),
  `tags.tsx` (3), `entries.$entryId.edit.tsx` (2), `recurring.$id.edit.tsx`,
  `settings.users.tsx`, `settings.invitations.tsx`,
  `accounts.$accountId.edit.tsx`, `accounts.$accountId.sharing.tsx`,
  `categories_.$categoryId.sharing.tsx`, `tags_.$tagId.sharing.tsx`.
- `frontend/src/routes/entries.index.tsx`,
  `frontend/src/routes/reports.tsx`,
  `frontend/src/routes/accounts.$accountId.index.tsx`,
  `frontend/src/components/dashboard/EntryListCard.tsx`,
  `frontend/src/components/RecurringTransactionBadge.tsx` — summary
  triggers.
- `frontend/src/i18n/locales/{en,de}.json` — summary field labels and
  actions.

## Sequencing

`entry-list-bulk-actions` adds a checkbox column to the same ledger rows,
and `add-recurring-linked-entries` adds a linked-entries list to
`/recurring/{id}/edit` whose rows are a further natural home for the entry
summary. Neither blocks this change, but landing this after both avoids
reworking the same rows twice.

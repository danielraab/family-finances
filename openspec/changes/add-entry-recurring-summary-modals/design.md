## Context

Two things land together here because doing either alone makes the other
worse: two new modals, and the shared shell they should be built on.

The shell is not speculative abstraction. Counting today's panels:

| Panel shape | Count |
|---|---|
| `max-w-sm … gap-4 … p-6` | 20 |
| `max-w-md … gap-3 … p-4` (both Leaflet modals) | 2 |
| `max-w-md … gap-4 … p-6` (`CardFormDialog`) | 1 |

Every one of the 23 uses `relative z-50` on the `Dialog`, the same
`fixed inset-0 bg-black/40` overlay, the same
`fixed inset-0 flex items-center justify-center p-4` wrapper, and the same
`text-base font-semibold` `DialogTitle`. The variation is two axes wide.

## Decisions

### The shell takes a size variant, not a className passthrough

`Modal` exposes `size: "sm" | "md"` (default `"sm"`) and owns the panel
classes outright. A `panelClassName` escape hatch would let the 23 shapes
drift apart again, which is the thing being fixed. The two Leaflet modals'
tighter `gap-3 p-4` is folded into the `md` size rather than kept as a
third axis — they are the only users of it and the difference is not
load-bearing.

### Nested dialogs keep the same z-index

`categories.tsx` nests its delete confirm inside its edit dialog, both at
`z-50`. That works because Headless UI portals stack in DOM order, and the
later-opened dialog mounts after. The shell keeps `relative z-50` fixed
for every modal rather than introducing a stacking prop — raising the
nested one would be a change in behaviour with nothing asking for it.

### Non-dismissable is an explicit prop

`BulkActionRunModal` passes `onClose={() => {}}` while a bulk run is in
flight, so Escape and a backdrop click do nothing until it finishes. The
shell expresses that as `dismissable?: boolean` (default `true`) and
supplies the no-op itself, rather than every caller open-coding an empty
function.

### Edit navigates; it never renders a form in the modal

The summary's Edit action is a router link to the existing edit route.
Rendering the edit form inside the modal would mean extracting
`entries.$entryId.edit.tsx`'s form (852 lines, with account-unlock
confirmation, currency warnings, self-transfer handling, and delete) —
a much larger change with its own risks, and it would put two save paths
in the app. The summary's job is to make opening the form rarely
necessary, not to replace it.

### The permission predicate is extracted, not duplicated

`canEdit` is not a one-liner: `entry_admin`/`owner` may edit any entry on
the account, `append` only entries they created, and a **self-transfer is
stricter still** — `append`+ on *both* accounts with no `created_by`
exemption — while its delete rule is looser. It lives inline in
`entries.$entryId.edit.tsx` today. The summary's Edit action must honour
exactly the same rule, and a second copy of it would drift. So it moves
to `src/lib/entryPermissions.ts` and both call sites import it. This is
the one piece of the change that touches existing logic rather than
adding to it, and it is a pure move: same rule, same behaviour.

### Summaries render from data the host page already holds

A summary takes the already-fetched `Entry`/`RecurringTransaction` object
as a prop, plus the account/category/tag lookup maps the host page keeps
for its labels. No fetch on open, so no spinner and no new failure mode,
and the modal shows exactly what the row behind it shows. This is the
right trade because the ledger, reports, the dashboard card, and the
account detail page all already hold every field a summary needs.

The one consequence worth naming: a summary is as fresh as the list that
opened it. Since nothing in a summary is editable, a stale read is the
worst case, and the Edit route refetches on open.

### Cross-linking replaces content instead of stacking

An entry summary shows a link to its recurring transaction's summary, and
a recurring summary lists its linked entry count. Opening one from the
other swaps the modal's content and shows a back affordance, rather than
opening a second modal on top. Stacked modals are bad on a phone, and the
shell deliberately has no stacking story (see the z-index decision).

### An entry's title becomes the summary trigger everywhere

Four surfaces render an entry title. Two link to the edit page
(`entries.index.tsx`, `accounts.$accountId.index.tsx`) and two render
plain text (`reports.tsx`, `dashboard/EntryListCard.tsx`). All four become
summary triggers, so the gesture means one thing app-wide.

This changes what an existing click does on the two that navigate today.
That is deliberate: reading is the far more common intent, and the Edit
action inside the summary is one click from where the old behaviour
landed you.

`RecurringTransactionBadge` changes the same way, and it needs one extra
care — it currently calls `e.stopPropagation()` because it sits inside a
row that is itself clickable. Opening a modal instead of navigating keeps
needing that.

### The trigger is opt-in at the call site, never inside a label component

`AccountLabel` sits *inside* a router link on `/accounts` and in
`AccountCard`. Baking a button into a label component would nest a button
in an anchor — invalid markup with two competing click targets. Since
accounts are out of scope this does not bite today, but the same rule is
why the entry summary's trigger is wired per call site rather than into a
shared label.

## Risks

- **23 migrations in one change.** Mechanical, but it touches 18 files
  and every dialog in the app. Mitigated by the shapes being identical:
  the diff should be deletion-heavy, and any panel that needs a new prop
  to survive migration is a signal the shell is wrong, not the call site.
- **Changing what an entry-title click does.** Two surfaces navigate
  today. If this reads wrong in use, the cheapest correction is a second
  affordance on the row, not a revert of the summary.
- **Overlap with in-flight work.** `entry-list-bulk-actions` is adding a
  checkbox column to the ledger rows this change re-wires, and
  `add-recurring-linked-entries` adds rows that will want the entry
  summary. See the proposal's Sequencing note.

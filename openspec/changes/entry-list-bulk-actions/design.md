## Context

`entries.index.tsx` already fetches `accounts`, `categories`, and `tags` up
front, and holds the currently-loaded page window in `items: Entry[]`
(cursor-paginated, appended to on scroll — see `web-client-entries`'s
"cursor-based infinite scroll" requirement). `entries.$entryId.edit.tsx`
already establishes the per-entry write-permission rule this change relies
on rather than reinventing:

```
canEdit = account.permission ∈ {entry_admin, owner}
          OR (account.permission === append AND entry.created_by === me)
```

— and the backend enforces the same rule server-side on every
`PATCH`/`DELETE /api/entries/{id}` (`backend/AGENTS.md`'s Entries section),
independently of anything the client does. `EntryUpdate` is already a
partial update — every field optional, `tag_ids` replaces the full set,
`category_id`/`recurring_transaction_id` use the explicit-null-clears
convention. Nothing here needs a new backend field or endpoint.

There's also a direct precedent for "many entries, one API call each,
track failures, let the visitor cancel": `ImportRunStep.tsx` already loops
sequentially over `api.POST("/api/entries")` for a batch of rows, updating
a progress bar and collecting a `RunFailure[]` list. This change reuses
that same shape for `PATCH`/`DELETE` instead of `POST`.

## Goals / Non-Goals

**Goals:**
- Let a visitor act on many entries at once for six operations: set
  category, add tags, remove tags, set tags, link to recurring
  transaction, delete.
- Never silently skip an entry the visitor can't act on — attempt it and
  report the failure, rather than guessing client-side who's allowed to
  write.
- Keep every action's mechanics a thin wrapper over the existing
  single-entry `PATCH`/`DELETE` endpoints — no bulk endpoint, no new
  backend surface.

**Non-Goals:**
- No "select all entries matching the filter" beyond what's currently
  loaded in the ledger's infinite-scroll window — selecting further pages
  requires scrolling to load them first, same as today's list itself.
- No bulk "set title" (dropped — see proposal.md).
- No atomicity — a bulk action is N independent writes; a failure partway
  through never rolls back the ones that already succeeded, mirroring
  `ImportRunStep`'s existing behavior for creation.
- No backend bulk endpoint in this change (see Decisions below for why).

## Decisions

### Client-side sequential loop, not a backend bulk endpoint

Permission is enforced **per entry** (`append` may only touch entries they
created; `entry_admin`/`owner` may touch any entry on the account), so even
a dedicated bulk endpoint would have to report a per-id result — it buys no
atomicity. Given the ledger already caps a practical selection to whatever
fits in the loaded infinite-scroll window (see the next decision), the
performance case for a single round trip is weak too. The client loop
reuses `ImportRunStep.tsx`'s exact shape (sequential `await`, progress
count, per-item failure collection) rather than introducing a second
pattern for the same kind of problem.

### "Select all" means all *loaded* rows, not every filter match

The header checkbox toggles every row currently in `items`. It does not
trigger a fetch of the rest of the filtered set, and does not track
"selected minus explicitly deselected" against an unbounded remote count.
This keeps selection state a plain `Set<string>` of entry ids with no
separate "select-all-matching" mode to reconcile against filter changes,
pagination, or concurrent edits. A visitor who wants more of the filtered
set selected scrolls to load more, same as any other use of this ledger.

### Row checkboxes are always enabled; permission failures surface after the fact

Rather than precomputing `canEdit` per row to greyed-out checkboxes (which
would require resolving `account.permission` for every account among the
loaded rows, including ones the visitor has no account-level access to at
all — see `account-entries`'s category-permission-only visibility case),
every row stays selectable and every bulk action simply attempts every
selected entry. The backend's existing per-entry permission check is the
real gate, unchanged; the run modal's failure list is where a rejected
entry becomes visible, with the entry's title and the failure reason
(`403`/`422`/etc., translated to a short message the same way the entry
edit form already turns a failed save into `entries.form.saveError`-style
text).

### Link to recurring transaction is disabled across a multi-account selection

`EntryUpdate.recurring_transaction_id` must name a recurring transaction on
the *entry's own* `account_id` (`account-entries`) — there is no
"recurring transaction usable from any account" concept to fall back to.
Rather than let the visitor pick one anyway and watch every entry outside
its account fail, the action itself is disabled (with a hint) whenever the
current selection's entries don't all share one `account_id`. When enabled,
its picker fetches `GET /api/recurring-transactions?account_id=<the one
shared account>`, the same scoped fetch `entries.$entryId.edit.tsx` already
does, plus the same "Not linked" clearing option that form offers.

### Add/Remove/Set tags all reuse `TagInput`, differing only in the offered list and the per-entry merge

All three read as "pick some tags," so all three render the same
`TagInput` widget the entry form already uses (creatable, existing-tag
suggestions) rather than three different pickers:

```
Action       existingTags passed to TagInput          per-entry result
──────────────────────────────────────────────────────────────────────
Add tags     normal creatable list (not disabled,      tag_ids ∪ chosen
             not view-tier) — same filter the entry
             edit form already applies
Remove tags  only tags present on ≥1 selected entry     tag_ids − chosen
             (the union of every selected entry's own
             tag_ids, intersected against known tags)
Set tags     same normal creatable list as Add           tag_ids = chosen
             (full replace)
```

Restricting Remove's list to tags actually in use across the selection is
what "only possible to select tags which are used in the selected
entries" means concretely — it's computed client-side from the
already-loaded `items`, no extra fetch.

For all three, a per-entry `PATCH {tag_ids: …}` is only sent when the
computed set actually differs from that entry's current `tag_ids` — an
entry that already has every tag being "added," or none of the tags being
"removed," counts as an immediate success in the modal with no network
call, rather than a wasted round trip. "Set tags" always sends (its intent
is an unconditional replace, and a same-value PATCH is still meaningfully
"did this get applied," not a bug to optimize around — though it's a cheap
no-op write either way).

### Set category / Link to recurring: same skip-if-unchanged treatment

Mirrors the tags decision above: if an entry's `category_id` (or
`recurring_transaction_id`) already equals the value being applied, no
`PATCH` is sent for that entry and it's counted as succeeded immediately.
This matters more than it sounds — re-running a bulk action after fixing
some failures, or applying it to a selection that already partially has
the target value, shouldn't spam identical writes.

### Every action requires an explicit confirm click; none fire on value-pick alone

Picking a category, a set of tags, or a recurring transaction opens that
action's own small `Dialog` (mirroring the edit page's existing
delete-confirm and account-change-confirm dialogs) with the picker plus an
Apply button; nothing is sent to the backend until Apply is pressed. Delete
keeps its existing destructive-confirm wording instead of a value picker.
Pressing Apply/Confirm closes that dialog and opens the shared run/progress
modal, which is where the actual `PATCH`/`DELETE` loop executes.

### The run modal is one shared component across all six actions

`BulkActionRunModal` (name indicative) takes a list of already-decided
per-entry tasks (id → the specific `PATCH` body or "delete") and runs them
sequentially, showing:

```
┌─────────────────────────────────┐
│  Applying… ▓▓▓▓▓▓▓░░░ 7/12       │  ← mid-run
└─────────────────────────────────┘
┌─────────────────────────────────┐
│  9 succeeded, 3 failed:          │  ← done
│   • "Groceries run" — forbidden  │
│   • "Rent" — forbidden           │
│   • "Coffee" — category invalid  │
│                                  │
│           [Reload list] [Close]  │
└─────────────────────────────────┘
```

"Reload list" re-runs the ledger's existing filtered fetch from scratch
(same effect the `searchKey` `useEffect` already performs on a filter
change: reset `items`, `nextCursor`, re-fetch page one) and clears the
selection; "Close" just dismisses the modal, leaving stale rows in place
until the visitor reloads or navigates. Nothing auto-refreshes — the
visitor decides when to see the ledger reflect the just-applied changes,
consistent with "every action needs an explicit step" running through this
whole feature.

### A transaction rejecting a cleared category is a normal failure, not a special case

`EntryUpdate.category_id: null` only clears a `balance_adjustment`'s
category (a `transaction` requires one) — so "Set category → No category"
against a selection that includes transaction-kind entries will reject
those specific ones. This isn't special-cased: it's exactly the same
"attempt it, let the server's existing validation decide, report the
failure" posture as the permission case above, and needs no bulk-specific
handling.

## Risks / Trade-offs

- **[Risk]** A large selection (many loaded rows, header-checkbox
  "select all") still means one HTTP round trip per entry, sequentially —
  slower than a hypothetical bulk endpoint for a very large batch.
  **Mitigation**: accepted, consistent with `ImportRunStep`'s existing
  behavior for entry creation; the ledger's own infinite-scroll page size
  already bounds a realistic "select all loaded" batch, and the skip-if-
  unchanged optimizations above cut real network calls further in the
  common re-run case.
- **[Risk]** Because checkboxes are never permission-gated, a visitor with
  only `view` access on some shared account can select and "attempt" a
  bulk action against entries they were never going to be allowed to
  touch, learning via the failure list that they lack permission on that
  account. **Mitigation**: this discloses nothing beyond what's already
  visible — a `view`-tier visitor already sees those entries in the ledger
  and already gets a `403` attempting to edit one individually; the bulk
  path doesn't expose anything new, just in a batched response.
- **[Risk]** A canceled/interrupted run (navigating away mid-loop) leaves
  the selection partially applied with no rollback. **Mitigation**: same
  accepted behavior `ImportRunStep` already has; the modal's tally is
  always an honest account of exactly what happened before it closed.

## Migration Plan

Purely additive frontend UI — no migration, no data change, no API
contract change. Ships and rolls back as a single frontend deploy.

## Open Questions

None outstanding — selection scope, permission-gating strategy, the
recurring-transaction single-account constraint, tag-action widget reuse,
confirm-before-run, and the shared run-modal shape were all resolved
during exploration.

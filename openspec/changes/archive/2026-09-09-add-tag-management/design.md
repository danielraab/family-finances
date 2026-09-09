## Context

See proposal.md - Why. Relevant existing code this reuses or must stay
consistent with:

- `internal/tag` (`tag.go`/`store.go`/`service.go`/`handler.go`) already
  implements everything except `disabled` and `entry_count`: `Tag{ID, Name,
  CreatedAt, OwnerID}`, a `Store` interface with `List/Get/ByName/Create/
  Update/Delete/OwnedBy`, both storage backends, and a handler mounted at
  `GET/POST /api/tags` + `PATCH/DELETE /api/tags/{id}`. `Delete` is
  unconditional today (no in-use block) — that stays exactly as-is; this
  change adds a second, independent flag next to it.
- `internal/category`'s `disabled` flag is the direct precedent: a
  reversible `SetDisabled(ctx, ownerID, id, disabled) (Category, error)` on
  `Store`, `Disable`/`Enable` on `Service`, and `entry.CategoryLookup.
  Usable(ctx, ownerID, categoryID) (bool, error)` — exists, owned, not
  disabled — consulted by `entry.Service` only when `category_id` is
  explicitly present in the request (create, or an update that supplies
  it). An update that omits `category_id` never re-checks the entry's
  existing, possibly since-disabled category.
- `internal/storage/memory/category.go`'s doc comment: "It has no
  visibility into entries, so unlike the real backend it cannot reject
  deleting a category that is still referenced by one." The Postgres
  implementation reaches into the `entries` table directly by SQL for that
  check (`internal/storage/postgres/category.go`'s `Delete`), with no Go
  import of `internal/entry` — domain packages don't import each other,
  but their Postgres stores may still name each other's tables in raw SQL.
- `internal/entry/service.go`: `TagIDs` is a **full replacement array**,
  not a delta — `Update` only touches it when `upd.TagIDs != nil`, and when
  it does, the entire submitted slice becomes the entry's new tag set (see
  `s.store.Update(ctx, ownerID, id, upd)` and `entry.Update`'s doc comment).
  This is the key difference from `category_id`: an entry that already
  carries a tag and is having an unrelated field edited will, if the
  frontend always resends the full tag list (it does — see
  `web-client-entries`), resubmit that tag's id every time, not just when
  it's actually changing.
- `settings.account-types.tsx` is the closest UI precedent: table with
  per-row Edit/Disable-or-Enable/Delete, a create form above it, a
  `Dialog`-based confirm for every state-changing action. `/categories`
  differs in one respect this change follows instead: disable/enable fire
  directly, with delete the only one behind a confirmation dialog — settled
  in discovery for this change.

## Goals / Non-Goals

**Goals:**

- A user can see, at a glance, how many of their own entries use each tag.
- A user can rename a tag, or take it out of future circulation (disable)
  without touching any entry that already has it.
- Deleting a tag stays exactly as forgiving as it is today — always
  allowed, detaches everywhere — just reachable from a real management
  page.
- The "old ones are untouched" guarantee categories established extends to
  tags despite `tag_ids` being a list resubmitted in full on every update.

**Non-Goals:**

- No in-use block on delete (unlike categories/account types) — deleting a
  tag remains unconditional, per the existing `entry-tags` spec.
- No confirmation dialog for disable/enable — settled explicitly for this
  change (categories' pattern, not account-types').
- No change to how tags are created inline from the entry form
  (`GetOrCreate`-shaped name resolution in `entries.new.tsx`/`entries.
  $entryId.edit.tsx`) beyond filtering disabled tags out of the
  *suggestion* list — the name-to-id resolution map is untouched, so an
  entry that already carries a disabled tag still round-trips correctly.
- No accurate `entry_count` from the in-memory store — it stays `0`
  always, the same accepted gap `category`'s in-memory store has for its
  in-use check.
- No sort order, hierarchy, or grouping for tags — flat list, unchanged.

## Decisions

### Decision: `disabled` is a plain reversible flag, same shape as `categories.disabled`

`tags` gains `disabled boolean NOT NULL DEFAULT false`
(`0017_tag_disabled.sql`), toggled via `POST /api/tags/{id}/disable` /
`/enable`, no admin gate (every tag operation is already caller-scoped,
same as categories). `ListTags`/`GetTag` keep returning disabled tags —
the settings tab needs to display and re-enable them, and any entry
already carrying one still needs its name.

### Decision: `entry.TagLookup` gains `Usable`, alongside the existing `OwnedBy`, and `Update` only calls it on the newly-added subset

```go
// internal/entry/store.go
type TagLookup interface {
    // OwnedBy reports whether every id in tagIDs exists and belongs to
    // owner — checked unconditionally against the whole resubmitted
    // tag_ids array, same as today.
    OwnedBy(ctx context.Context, owner string, tagIDs []string) (bool, error)
    // Usable reports whether every id in tagIDs exists, belongs to owner,
    // and is not disabled — checked only against ids that are newly
    // appearing on the entry, never ones it already carried.
    Usable(ctx context.Context, owner string, tagIDs []string) (bool, error)
}
```

`internal/tag.Service.Usable` satisfies this structurally, delegating to a
new `Store.Usable` that mirrors `OwnedBy`'s existing shape with one extra
predicate (`AND disabled = false` in Postgres; the same loop plus a
`!t.Disabled` check in the in-memory store — no entries visibility needed
for this one, since disabled-ness lives on the tag row itself).

`entry.Service`:

```go
// Create: every tag id is new by definition.
ok, err := s.tags.Usable(ctx, ownerID, in.TagIDs)   // was OwnedBy

// Update, when upd.TagIDs != nil && len(*upd.TagIDs) > 0:
ok, err := s.tags.OwnedBy(ctx, ownerID, *upd.TagIDs)        // unchanged:
                                                              // existence/
                                                              // ownership
                                                              // of the
                                                              // whole array
...
newlyAdded := diff(*upd.TagIDs, current.TagIDs)              // set
                                                              // difference
if len(newlyAdded) > 0 {
    ok, err := s.tags.Usable(ctx, ownerID, newlyAdded)        // disabled
                                                              // check only
                                                              // on what's
                                                              // actually
                                                              // new
}
```

`current` is already fetched at the top of `Update` (`s.store.Get`), and
`Entry` already carries `TagIDs`, so `current.TagIDs` is free — no extra
query. `diff` is a small local helper (elements of the new slice not
present in the current one); it lives in `internal/entry`, not `tag`, since
it's specific to this one call site.

Consequence: creating an entry with a disabled tag is rejected outright
(`ErrInvalidValue`, `400` — same status the analogous disabled-category
check already uses, no new sentinel); editing an entry that already
carries a since-disabled tag, whether or not the edit also touches
`tag_ids`, succeeds as long as no *new* tag id in the resubmitted array
resolves to a disabled tag.

### Decision: `entry_count` is a correlated subquery column, not a `JOIN … GROUP BY`

```sql
SELECT id::text, name, disabled, created_at,
       (SELECT count(*) FROM entry_tags et
          JOIN entries e ON e.id = et.entry_id AND e.deleted_at IS NULL
         WHERE et.tag_id = tags.id) AS entry_count
  FROM tags WHERE …
```

Folded into `tagCols` so `List`, `Get`, `Create`, `Update`, and the new
`SetDisabled` all return an accurate, live `entry_count` with no separate
endpoint and no row-duplication risk a `JOIN … GROUP BY tags.id` would
otherwise need `DISTINCT`/aggregation to avoid. `internal/storage/memory`
has no `entries` visibility (mirroring `category`'s in-memory store, see
Context) — `EntryCount` there always reads `0`; acceptable because
`storage/memory` is test/local-dev infrastructure, never what ships.

### Decision: disable/enable fire directly from the settings tab; delete stays behind a confirmation dialog

Settled explicitly for this change: unlike `settings.account-types.tsx`
(which confirms disable/enable/delete uniformly), the new Tags tab follows
`/categories`'s pattern — disable/enable are cheap and fully reversible, so
they call `POST /api/tags/{id}/disable`/`/enable` immediately on click; only
Delete (irreversible, detaches everywhere) goes through the
`@headlessui/react` `Dialog` confirmation already used throughout
`/settings`.

### Decision: `TagInput.tsx` filters disabled tags from *suggestions* only, at the call site, not inside name resolution

`entries.new.tsx` / `entries.$entryId.edit.tsx` pass
`existingTags={tags.filter((t) => !t.disabled)}` to `TagInput` — the
suggestion dropdown never offers a disabled tag for a new attachment.
`resolveTagIds()` keeps resolving against the **unfiltered** `tags` state
fetched from `GET /api/tags`, so an entry loaded for edit with a
since-disabled tag still shows that tag's name in `TagInput`'s `value`
array (unaffected — `TagInput` never filters `value`, only `existingTags`)
and, on save, resolves that unchanged name back to the same tag id,
landing in the "already carried, not newly added" branch of the backend
rule above. A visitor could still defeat the suggestion filter by typing a
disabled tag's exact name for an entry that didn't have it — that hits the
backend's `Usable` check on the newly-added subset and surfaces as the
entry form's existing generic save-error handling; not special-cased
further, since it's a narrow, self-inflicted edge case.

## Migration Plan

1. `backend/internal/storage/postgres/migrations/0017_tag_disabled.sql`:
   `ALTER TABLE tags ADD COLUMN disabled boolean NOT NULL DEFAULT false`.
2. `internal/tag`: `tag.go` (`Tag` gains `Disabled`, `EntryCount`);
   `store.go` (`Store` gains `SetDisabled`, `Usable`); `service.go`
   (`Disable`/`Enable`/`Usable`); `handler.go` (two new routes).
3. `internal/storage/memory` and `internal/storage/postgres`: implement
   `SetDisabled`, `Usable`; extend the row-scanning helper for `disabled` +
   `entry_count` (Postgres only — memory hardcodes `EntryCount: 0`).
4. `internal/entry`: `store.go` (`TagLookup` gains `Usable`); `service.go`
   (`Create` swaps `OwnedBy` → `Usable`; `Update` adds the newly-added-set
   diff + `Usable` check); its test fakes (`newStubTags()` in
   `handler_test.go`/`service_scenarios_test.go`) implement the new method.
5. `openapi/openapi.yaml`, then `go generate ./...` and `pnpm generate:api`.
6. Frontend: `settings.tags.tsx` + tab entry in `settings.tsx` +
   `TagInput.tsx` call-site filtering + i18n.

## Verification

- Backend: `internal/tag` unit + handler tests for `Disable`/`Enable` and
  `Usable` (owned + not-disabled, foreign id, disabled id, mixed);
  `internal/entry` service tests — creating with a disabled tag rejected;
  updating an entry that already carries a since-disabled tag, touching
  only an unrelated field, succeeds and keeps the tag; the same entry
  resubmitting its full existing `tag_ids` (including the disabled one)
  unchanged succeeds; adding a *new* disabled tag id alongside the
  untouched existing one is rejected and neither tag list nor the rest of
  the update is applied; `internal/storage/postgres` integration tests for
  the migration, `SetDisabled`, `Usable`, and `entry_count` reflecting
  real attached/soft-deleted entries.
- Frontend: `pnpm lint && pnpm exec tsc && pnpm build`; manual pass —
  create/rename/disable/enable/delete a tag on the new tab; confirm entry
  counts match; confirm the entry form's tag suggestions exclude disabled
  tags while an already-tagged entry still shows and saves its disabled
  tag.

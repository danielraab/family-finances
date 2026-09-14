## Context

Tags (`backend/internal/tag/`) and categories (`backend/internal/category/`)
are structurally similar per-user entities attached to entries, but tags
predate categories' sharing feature and never grew the machinery categories
needed for it: no `Permission`/`Shared`/`OwnerName` fields, no
`UserLookup`/`Mailer` wiring, no single-fetch `GET /api/tags/{id}`, no
`GetForCaller` distinct from an owner-scoped `Get`. Category sharing
(`category_shares`, migration `0022`) is the direct, working precedent this
change ports to tags — see `backend/AGENTS.md`'s "Categories and sharing"
section for the pattern being replicated.

Two structural differences from categories matter for this design:

- **Tags are flat**, not a tree — no `parent_id`, no cascade-to-descendants
  question, no reparenting.
- **Tags delete unconditionally today** (`DELETE ... CASCADE` through
  `entry_tags`, no in-use guard), whereas categories soft-delete and block
  on in-use-by-entry or has-children. This change does not adopt categories'
  in-use guard for tags — entries keep referencing a tag freely up to
  deletion, same as today. It adds exactly one new guard: **blocked while
  shared**, addressed in Decisions below.

## Goals / Non-Goals

**Goals:**
- Let a tag be shared with other registered users at `view`/`append` tiers,
  structurally identical to category sharing (same share-row shape, same
  invite/revoke/self-leave semantics).
- Give tags a first-class `/tags` page and sidebar entry, removing them from
  `/settings`.
- Keep the delete UX safe for a share recipient: a tag can't be silently
  pulled out from under someone using it via a share.

**Non-Goals:**
- No in-use-by-entry delete guard for tags (unlike categories) — an
  unshared tag stays exactly as freely, unconditionally deletable as today.
- No household/group sharing model — sharing stays per-recipient email
  invite, same as accounts and categories. (Confirmed: no such concept
  exists anywhere in the codebase.)
- No change to the `UNIQUE (owner_id, name)` constraint on tags — sharing
  doesn't collide with it since it's scoped per owner already; disambiguation
  for a same-named shared tag is a display concern (see TagLabel decision),
  not a schema one.
- No tree/hierarchy semantics added to tags.

## Decisions

### Delete guard is "currently shared," not "in use by an entry"

Categories block delete on two independent conditions (has children, or
referenced by a non-deleted entry). Tags adopt neither of those — instead,
`DELETE /api/tags/{id}` is rejected (`409`) if and only if the tag currently
has at least one row in `tag_shares`. An unshared tag keeps today's
behavior exactly: unconditional hard delete, cascading through
`entry_tags`, regardless of how many of the owner's own entries carry it.

**Why this and not categories' in-use guard**: the risk sharing introduces
is specifically that deleting a tag you've shared silently un-tags entries
belonging to *someone else*, without their knowledge — categories' in-use
guard exists for a related but distinct reason (protecting the tree/report
structure). Reusing the in-use guard for tags would be a bigger behavior
change than asked for (today's tags are freely deletable while in use by
the owner's own entries, and that stays true). The narrower "blocked only
while shared" guard directly closes the cross-user gap without touching
existing single-user delete semantics.

**Alternative considered**: soft-delete (`deleted_at`) for tags, mirroring
categories exactly. Rejected — no schema change needed to satisfy the
actual requirement (protect share recipients), and hard-delete-cascade for
unshared tags is preserved behavior, not something to migrate away from.

### `entry_count` becomes caller-scoped, mirroring categories exactly

`Tag.EntryCount` today is unconditionally the owner's own count (there was
never a non-owner viewer). Once a tag can be viewed by a share recipient,
the same correlated-subquery pattern categories use
(`categoryCols`/`categoryViewCols`'s `callerParam`, see
`backend/internal/storage/postgres/category.go:21-45`) is ported to tags:
the count reflects the *viewing caller's own* entries carrying the tag, not
the owner's. This is a backend field-semantics change, not a new field —
`json:"entry_count"` stays required, always an integer, never omitted at
the API layer.

The frontend then chooses not to render it on shared rows at all (see next
decision) — the field being caller-scoped in the API doesn't imply the UI
must show it everywhere; it only has to be correct if shown.

### Shared rows never render `entry_count`; owned rows always do

Matches confirmed category behavior exactly: `categories.tsx`'s
`renderSharedNode` never reads `entry_count`, `renderNode` (owned) always
does. `tags.tsx` follows the same split — the shared-tags section's row
component omits the count entirely, rather than computing/showing a
caller-scoped number that would read confusingly next to someone else's
tag name.

### TagLabel is a new component, not a CategoryLabel variant

`CategoryLabel` renders a shared badge plus the owner's name as persistent,
always-visible text. Tags get different treatment by explicit design
decision: a small shared-indicator icon always next to the tag name, with
the owner's name surfacing only on hover (a title/tooltip), not as
standing text. Because the always-visible vs. hover-only behavior is a
real UI difference (not just re-skinning), this is a distinct
`frontend/src/components/TagLabel.tsx`, not a shared component
generalized over both entity types.

### Self-leave and revoke mirror categories exactly, unmodified

Verified against the actual `category.Service.RevokeShare` implementation
(`backend/internal/category/service.go:361-383`) rather than assumed: only
one `(entity_id, user_id)` row is ever touched by a leave/revoke, nothing
else about the tag changes, and any non-owner can revoke their own row at
any tier with no elevated permission required — only revoking *someone
else's* share requires being the real owner. `tag.Service.RevokeShare`
ports this logic verbatim (owner/target/caller checks identical to
category's).

### Route naming: avoid the dot-nesting trap from the start

The category sharing page shipped initially as `categories.$categoryId.
sharing.tsx` (dot-nesting), which TanStack Router's file-based generator
registered as a child of `categories.tsx` — since `categories.tsx` has no
`<Outlet/>`, the Share button changed the URL but rendered nothing, fixed
by renaming to `categories_.$categoryId.sharing.tsx` (trailing-underscore
= non-nested). `tags_.$tagId.sharing.tsx` is created with the underscore
from the start, sidestepping the same bug rather than needing the same
fix-up commit.

### Backend wiring mirrors category's construction-order pattern

`tag.Service` gains `UserLookup`/`Mailer` interfaces and a
`WithMailer`/`WithBaseURL` option set, structurally identical to
`category.Service`. `main.go`'s `buildTag` grows the same
`(pool, mail, baseURL)` signature as `buildCategory`, and `tagSvc.
SetUserLookup(authSvc)` is called post-`buildAuth`, same construction-order
workaround `category.Service` already needs (domain package needs
`auth.Service` as a lookup, `auth.Service` needs the domain service as a
`NewUserHook`).

## Risks / Trade-offs

- **[Risk]** A user might expect "delete" to always work on their own tag,
  and be surprised by a `409` the first time they share one → **Mitigation**:
  the delete confirmation dialog states the reason inline (tag is
  currently shared with N people), same treatment `web-client-categories`
  already gives its in-use `409`.
- **[Risk]** Divergent delete semantics between categories (in-use guard)
  and tags (shared-only guard) is one more asymmetry a future reader has to
  hold in their head → **Mitigation**: documented explicitly in this
  design and in the `entry-tags` delta spec's requirement text, not left
  implicit.
- **[Risk]** Porting `categoryViewCols`'s caller-scoped `entry_count`
  pattern to tags touches the hot path of `GET /api/tags` → **Mitigation**:
  it's the identical correlated-subquery shape already proven at
  category's scale; no new index expected beyond what `tag_shares_user_id_
  idx` (mirroring `category_shares_user_id_idx`) already provides.

## Migration Plan

1. Additive migration: `tag_shares` table, structurally identical to
   `category_shares` (`id`, `tag_id`, `user_id`, `permission` CHECK
   `('view','append')`, `granted_by`, `created_at`, `updated_at`, UNIQUE
   `(tag_id, user_id)`, index on `user_id`). No change to the existing
   `tags` table.
2. Backend: extend `tag.go`/`service.go`/`store.go`/`handler.go` with the
   sharing surface; add `GET /api/tags/{id}`; add the delete guard; update
   `openapi/openapi.yaml` (new schemas/paths) and regenerate
   `backend/openapi.yaml` + `frontend/src/api/schema.d.ts`.
3. Frontend: add `tags.tsx` + `tags_.$tagId.sharing.tsx` + `TagLabel.tsx`;
   remove `settings.tags.tsx`; update `settings.tsx`'s tab list and
   `Sidebar.tsx`'s nav.
4. No data backfill needed — `tag_shares` starts empty, every existing tag
   behaves exactly as before until someone shares it.
5. Rollback: dropping the `tag_shares` table and reverting the code is
   sufficient at any point pre-release; nothing else depends on its
   existence.

## Open Questions

None outstanding — permission tiers, delete guard condition, entry_count
scoping, shared-row display, and leave/revoke semantics were all resolved
during exploration against the verified category-sharing implementation.

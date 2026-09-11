## Why

Tags are currently private and buried in a Settings tab, which forces
duplicate tags across a household and gives tags no path to become a
shared vocabulary the way categories already are. Categories solved both
problems — a first-class nav item plus per-user, per-tier sharing — and
tags should get the same treatment, mirroring that proven pattern rather
than inventing a new one.

## What Changes

- Add tag sharing: a new `tag_shares` table and `view`/`append` permission
  tiers, structurally identical to `category_shares` — `view` makes a
  shared tag resolve on filters/entries but not selectable when tagging,
  `append` additionally makes it selectable. Only the real owner may
  rename, disable/enable, or delete a tag, or manage its shares.
- Add `GET /api/tags/{id}` (single-fetch, needed for the sharing page
  header — tags have no equivalent today) and the share-management
  endpoints: `GET`/`POST /api/tags/{id}/shares`,
  `PATCH`/`DELETE /api/tags/{id}/shares/{userId}`.
- **BREAKING** (behavioral, not API-shape): a tag can no longer be deleted
  while it has any active share — the owner must first ensure no one else
  holds a share on it (or the caller revokes them) before `DELETE
  /api/tags/{id}` succeeds. An unshared tag keeps today's unconditional
  hard-delete-with-cascade.
- `Tag` gains `permission`, `shared`, `owner_name` fields (mirroring
  `Category`); `entry_count` becomes scoped to the viewing caller rather
  than unconditionally the owner's count (only observable once a tag can
  be viewed by someone other than its owner).
- Move the Tags list out of Settings onto its own top-level page: a new
  `/tags` route (sidebar nav item, own auth gate, no longer nested under
  `/settings`) replacing `settings.tags.tsx`, structured like
  `/categories` — the caller's own tags plus a separate "Shared with me"
  section below it (not interleaved), which omits `entry_count` on shared
  rows. A new `/tags/{id}/sharing` page mirrors
  `/categories/{id}/sharing` (invite-by-email, per-row permission
  editor/revoke for the owner, self-leave for any non-owner recipient).
  The Settings page's Tags tab is removed.
- A shared tag always shows a small shared-indicator icon next to its
  name; hovering it reveals the owner's name (a tooltip, not
  always-visible text — deliberately lighter-weight than
  `CategoryLabel`'s persistent owner badge).

## Capabilities

### New Capabilities

- `tag-sharing`: sharing a tag with other registered users — the `view`/
  `append` tiers, the `tag_shares` model, share-management endpoints, the
  email-invite flow, revocation/self-leave, and the delete-blocked-while-
  shared rule.
- `web-client-tags`: the authenticated `/tags` page — sidebar link, auth
  gate, the caller's own tags (create/rename/disable/enable/delete), the
  "Shared with me" section, the shared-tag icon/hover-owner treatment,
  and the Share entry point into `web-client-tag-sharing`.
- `web-client-tag-sharing`: the `/tags/{id}/sharing` page — permission-
  gated read-only vs. management modes, invite-by-email form, per-row
  permission/revoke/leave controls.

### Modified Capabilities

- `entry-tags`: the delete requirement changes from "always deletable by
  its owner regardless of use" to "deletable by its owner unless the tag
  currently has an active share"; the tag response gains sharing-aware
  fields (`permission`, `shared`, `owner_name`) and caller-scoped
  `entry_count`; a `GET /api/tags/{id}` single-fetch endpoint is added.
- `web-client-settings`: the Tags tab is removed from `/settings` (its
  tab list and the `entry-tags` cross-reference); Tags management moves
  to `web-client-tags`.

## Impact

- **Backend**: `backend/internal/tag/` (`tag.go`, `service.go`,
  `store.go`, `handler.go`) gains the sharing surface mirroring
  `backend/internal/category/`; a new migration adds `tag_shares`;
  `backend/internal/mailer/mailer.go` gains `SendTagShare`;
  `backend/main.go` wiring adds `SetUserLookup` for `tag.Service` the same
  way `category.Service` needed it. `openapi/openapi.yaml` gains the new
  paths/schemas, synced to `backend/openapi.yaml`.
- **Frontend**: new `frontend/src/routes/tags.tsx` and
  `frontend/src/routes/tags_.$tagId.sharing.tsx` (underscore-nested from
  the start), a new `frontend/src/components/TagLabel.tsx`,
  `Sidebar.tsx` gains a Tags nav item and glyph, `settings.tsx` loses its
  Tags tab, `settings.tags.tsx` is removed, `frontend/src/api/schema.d.ts`
  regenerated from the updated spec, i18n strings added to `en.json` (and
  `de.json` where practical).
- **No new dependencies, no schema changes to existing tables** beyond the
  additive `tag_shares` table.

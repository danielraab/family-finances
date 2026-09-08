## Why

`internal/tag` already exists as a full four-file domain package (per-user
tags, `List`/`Get`/`ByName`/`Create`/`Update`/`Delete`/`OwnedBy`, both
storage backends, a mounted handler), but the only surface that touches it
is `TagInput.tsx` on the entry form — tags can only be created inline,
never renamed from a list, never disabled, and there's no way to see how
many entries a tag is actually attached to. Every other per-user lookup
(`account_types`, `categories`) already has a dedicated settings-tab
management surface with a reversible `disabled` flag; tags are the
odd one out. This change gives tags the same tab, plus the `disabled` flag
and a visible per-tag entry count, without touching the existing inline
creation flow on the entry form.

## What Changes

- `tags` gains `disabled` (boolean, default `false`), toggled via
  `POST /api/tags/{id}/disable` / `/enable` — reversible, no confirmation
  needed on either side, mirroring `categories.disabled`. A disabled tag
  can no longer be **newly** attached to an entry; any entry that already
  carries it keeps working, and an edit that doesn't touch that tag is
  never blocked by it — same "old ones are untouched" rule `categories`
  already established, adapted for tags being a list rather than a
  single field: only tag ids **newly added** to an entry's `tag_ids` on an
  update are checked against `disabled`, not ones already present before
  that update.
- `GET /api/tags` (and every other endpoint returning a `Tag`) gains
  `entry_count` — the number of the caller's own non-deleted entries
  currently carrying that tag.
- New **Tags** settings tab (`/settings/tags`), open to every authenticated
  visitor like Account Types and Categories — no admin gate. Lists the
  caller's own tags with their entry count and Active/Disabled status;
  supports create, rename, disable/enable (direct, no confirmation dialog),
  and delete (confirmed via the same `@headlessui/react` `Dialog` pattern
  used elsewhere). Deleting a tag is always allowed and detaches it from
  every entry, unchanged from today's behavior — just now reachable from a
  management page instead of only implicitly via the entry form.
- The entry form's tag autocomplete (`TagInput.tsx`) stops suggesting
  disabled tags for a new attachment; an entry that already carries a
  since-disabled tag keeps showing and resubmitting it normally.

## Capabilities

### Modified Capabilities

- `entry-tags`: adds the `disabled` flag, `POST /api/tags/{id}/disable` /
  `/enable`, and `entry_count` on every `Tag` response.
- `account-entries`: adds the "a disabled tag cannot be newly attached to
  an entry" rule, parallel to the existing disabled-category rule but
  scoped to only the ids newly added to `tag_ids`.
- `web-client-settings`: adds the Tags tab.
- `web-client-entries`: the entry form's tag autocomplete excludes
  disabled tags from suggestions; the "no dedicated management UI" caveat
  in the existing inline-tag-creation requirement is removed now that one
  exists.

## Impact

- Backend: new migration adding `tags.disabled`; `internal/tag` gains
  `Disable`/`Enable`, a `Usable` lookup (exists + owned + not disabled) for
  a set of ids, and `entry_count` computed per tag (via a SQL subquery
  against `entry_tags`/`entries` in the Postgres store — mirroring how
  `internal/category`'s in-use check already reaches into `entries` by SQL
  alone, no Go import; the in-memory store has no visibility into entries,
  same precedent as `category`'s in-memory store, so it always reports
  `entry_count: 0`); `internal/entry`'s `TagLookup` interface gains
  `Usable`, consulted only for newly-added tag ids on create/update.
- API contract: `Tag` gains `disabled`/`entry_count`; two new zero-body
  operations `POST /api/tags/{id}/disable` / `/enable`.
- Frontend: new route `settings.tags.tsx`; a new tab entry in
  `settings.tsx`; `TagInput.tsx`'s suggestion list (not its name-resolution
  map) filters out disabled tags; new i18n keys.

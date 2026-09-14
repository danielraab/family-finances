## REMOVED Requirements

### Requirement: Tags tab lists, creates, renames, disables/enables, and deletes tags

**Reason**: Tags management moves to its own top-level `/tags` page and
sidebar entry — see `web-client-tags` — rather than living as a Settings
tab, since a shared tag needs a "Shared with me" section and a sharing
entry point that don't fit the Settings tab layout.

**Migration**: none for data — this is a UI relocation. `settings.tags.tsx`
is removed; every capability it offered (list, create, rename,
disable/enable, delete) is offered identically by the new `/tags` page,
with delete additionally blocked while the tag is shared (see the
modified `entry-tags` delete requirement).

### Requirement: Disabling and enabling a tag apply immediately, without a confirmation dialog

**Reason**: this behavior moves unchanged to `web-client-tags`, along
with the rest of tag management.

**Migration**: none — the same immediate, unconfirmed toggle is offered
on `/tags`.

### Requirement: Deleting a tag requires confirmation

**Reason**: this behavior moves to `web-client-tags`, and is additionally
modified there: delete is now rejected (`409`) while the tag is shared,
surfaced as an inline error rather than always succeeding.

**Migration**: none for existing unshared tags — deletion continues to
require confirmation and succeeds exactly as before; only a tag that has
been shared gains the new blocked-while-shared behavior.

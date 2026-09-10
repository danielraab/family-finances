## Context

`frontend/src/routes/categories.tsx` renders each tree node with a
seven-control wrapping button row and drives three separate
`@headlessui/react` `Dialog`s off local state:

- `editing: Category | null` — name + `IconColorPicker`, one PATCH on Save.
- `moving: Category | null` — a parent `<select>` filtered by `subtreeIds`,
  its own PATCH with just `parent_id`, then a full `refresh()`.
- `confirming: { kind: "disable" | "enable" | "delete", target } | null` —
  one dialog reused for all three, copy keyed by `kind`.

The backend `PATCH /api/categories/{id}` already accepts `name`, `icon`,
`color`, and `parent_id` in one body (`categories.tsx` just never sends them
together today). `move-up`/`move-down`, `disable`/`enable`, and `DELETE` are
unchanged by this work.

## Goals / Non-Goals

**Goals:**

- Inline node row = ▲, ▼, Edit only.
- Edit dialog owns reparent (as a Save-applied field), disable/enable
  (immediate), and delete (confirmed).
- Remove the now-dead `moving` state/dialog and the disable/enable branch of
  the `confirming` dialog.
- No backend, `openapi/`, or generated-file changes.

**Non-Goals:**

- Changing reorder (▲/▼ stay on the row).
- Changing the create form.
- Changing delete's server-side in-use semantics or the `409` handling.
- Adding a subtree/rolled-up anything — unrelated to this change.

## Decisions

### Reparent = a field on the edit form, applied on Save

The edit dialog gains a parent `<select>` built from
`flattenCategoryTree(categories)` filtered through the existing `subtreeIds`
helper against `editing.id`. `onSaveEdit` sends
`{ name, icon, color, parent_id }` in one `PATCH`, where `parent_id` is
`editParentId || null`. After a successful save the page does a full
`refresh()` (the current Move flow already does this) so sort order under
the new parent is correct without local patching.

`editParentId` state is initialised in `openEdit` from
`cat.parent_id ?? ""`. A `422` (`ErrCycle`) sets the existing `editError`
string, shown in the same place as the name/save error.

Alternative considered: keep reparent as a separate immediate-PATCH button
inside the modal. Rejected per the resolved design question — a single Save
that applies everything is less surprising than one field that
auto-commits while the others wait for Save.

### Disable/enable = immediate, inside the modal

Two buttons (one shown at a time) call `performDisableToggle(editing)`
directly — no `confirming` round-trip. Reuse the existing service call from
`performConfirmed`'s disable/enable branch; on success update
`categories` state in place and keep the dialog open (so the user sees the
status flip) — the dialog's status text/pill updates from the refreshed
`editing`. `frontend/AGENTS.md` already states this is the intended
behaviour ("no confirmation dialog needed — cheaply reversible").

The `categories.confirm.disableTitle/disableBody/enableTitle/enableBody`
i18n keys become unused and are deleted from `en.json` and `de.json`. The
`ConfirmKind` type narrows to just `"delete"`.

### Delete = confirmed, inside the modal

A Delete button in the edit dialog, `disabled` when
`editing`'s node has children (look the node up in the built tree, or carry
`children.length` — simplest is to compute "has children" from
`categories.some(c => c.parent_id === editing.id)`). Clicking opens the
existing confirmation `Dialog` (`confirming = { kind: "delete", target }`).
`performConfirmed` already handles the `DELETE` + `409` → `actionError`
path; on success it also now closes the edit dialog (`setEditing(null)`).

Stacked dialogs (confirm over edit) are acceptable here and already the
pattern the `IconColorPicker` popover composes into.

### Inline row

Drop the Move to…, Edit-adjacent Disable/Enable, and Delete buttons.
Keep ▲, ▼, and Edit. The row's `flex-wrap` can stay; it will rarely wrap
now.

## Risks / Trade-offs

- **Reparent no longer one-click** → it's one extra Save click, on a
  low-frequency action; the payoff is a coherent single-apply edit dialog.
- **Deleting from within the edit dialog means two dialogs on screen** →
  accepted; the confirm dialog is small and modal, and this mirrors
  existing nested-overlay usage.
- **Removing the disable/enable confirmation is a behaviour change beyond
  "just move it"** → explicitly chosen; aligns the page with the Tags tab
  and the existing AGENTS.md note, and avoids a confirm-over-modal stack for
  a reversible action.
- **i18n key deletion** → `i18n-coverage` is informational only; removing
  keys from both locales keeps them in step.

## Migration Plan

Pure frontend. Ship `categories.tsx` + the two locale files together. No
data, no contract, no rollback concerns beyond a plain revert.

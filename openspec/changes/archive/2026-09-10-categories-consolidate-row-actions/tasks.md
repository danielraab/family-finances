## 1. Edit dialog: reparent field

- [x] 1.1 In `frontend/src/routes/categories.tsx`, add `editParentId` state;
  set it in `openEdit` from `cat.parent_id ?? ""`.
- [x] 1.2 Add a parent `<select>` to the edit `<form>`, options from
  `flattenCategoryTree(categories ?? [])` filtered to exclude
  `subtreeIds(categories ?? [], editing.id)`, plus a "make root" option
  (`value=""`). Reuse the create form's parent-picker markup/labels.
- [x] 1.3 Change `onSaveEdit` to send
  `{ name, icon, color, parent_id: editParentId || null }` in the one
  `PATCH /api/categories/{id}`; on success call `refresh()` (not just local
  state patch) and `setEditing(null)`.
- [x] 1.4 On a non-ok save, keep showing `editError`; a `422` reads the same
  as any other save failure (or a dedicated cycle message if a key exists).

## 2. Edit dialog: disable/enable + delete

- [x] 2.1 Add a Disable button (or Enable when `editing.disabled`) to the
  edit dialog that calls the disable/enable endpoint directly — no
  `confirming` dialog — updates `categories` state and the open `editing`
  object in place, and leaves the dialog open.
- [x] 2.2 Add a Delete button to the edit dialog, `disabled` when the
  category has children (`categories.some(c => c.parent_id === editing.id)`),
  with the existing `deleteDisabledHint` title. Clicking sets
  `confirming = { kind: "delete", target: editing }`.
- [x] 2.3 In `performConfirmed`, on a successful delete also `setEditing(null)`
  so the edit dialog closes with the confirm dialog.

## 3. Remove the old surfaces

- [x] 3.1 Delete the `moving` / `moveParentId` / `movingSubmitting` /
  `moveError` state, `openMove`, `onConfirmMove`, `moveOptions`, and the
  entire "Move to…" `<Dialog>`.
- [x] 3.2 Remove the "Move to…", Disable/Enable, and Delete buttons from
  `renderNode`'s inline row; keep ▲, ▼, and Edit.
- [x] 3.3 Narrow `ConfirmKind` to `"delete"`; drop the disable/enable arms
  of `performConfirmed` and the `confirm.${kind}Title/Body` lookups now that
  only delete remains (inline the delete copy).

## 4. i18n

- [x] 4.1 In `frontend/src/i18n/locales/en.json`, remove
  `categories.confirm.disableTitle`, `disableBody`, `enableTitle`,
  `enableBody`; remove `categories.actions.moveTo` if no longer referenced;
  keep/adjust `categories.actions.disable` / `enable` / `delete` /
  `deleteDisabledHint` and `categories.move.*` keys still used by the edit
  dialog's parent field (or migrate them under `categories.edit.*`).
- [x] 4.2 Mirror every `en.json` change in `de.json`.
- [x] 4.3 Grep `categories.tsx` for every `t("categories.…")` key and
  confirm each still resolves in `en.json`.

## 5. Verification

- [x] 5.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 5.2 Manual: open `/categories`, confirm the row shows only ▲/▼/Edit;
  in the edit dialog reparent + Save moves the node, Disable/Enable flips
  status with no prompt, Delete is greyed for a parent and confirms+closes
  for a leaf; a `409` on delete shows an inline error and keeps the node.
- [x] 5.3 Update `frontend/AGENTS.md`'s "Categories" section to describe the
  edit dialog owning reparent/disable/enable/delete and the row keeping only
  reorder + Edit.

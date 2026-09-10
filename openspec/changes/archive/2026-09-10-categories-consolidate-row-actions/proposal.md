## Why

Each `/categories` tree node currently carries seven controls in a wrapping
button row (▲, ▼, Move to…, Edit, Disable/Enable, Delete). On a phone the
row wraps to three lines and the destructive Delete sits inches from the
reorder arrows. Folding the low-frequency, category-scoped actions into the
Edit modal — which the user has already deliberately opened for that one
category — leaves the row short and scannable and puts "everything about
this category" in one place.

## What Changes

- The inline node row keeps only **▲ / ▼** (reorder) and **Edit**.
- **Move to…** stops being a standalone dialog. The edit modal gains a
  parent-category picker field next to name/icon/colour; **Save** sends one
  `PATCH /api/categories/{id}` with `name`, `icon`, `color`, and `parent_id`
  together. The cycle-exclusion filtering (`subtreeIds`) and the inline
  `422` surfacing move onto that field.
- **Disable / Enable** becomes a button inside the edit modal that fires
  `POST /api/categories/{id}/disable` / `/enable` **immediately, with no
  confirmation** (cheaply reversible — matches the Tags settings tab and the
  note already in `frontend/AGENTS.md`). The `confirm.disable*` /
  `confirm.enable*` dialog copy is removed.
- **Delete** becomes a button inside the edit modal, still behind its
  confirmation `Dialog`, still disabled (with the existing hint) when the
  node has children, still surfacing a `409` as an inline error. On success
  the modal closes and the node leaves the tree.
- i18n: no new user-facing strings beyond relabelled/relocated keys; the
  removed disable/enable confirmation keys are deleted from `en.json` and
  `de.json`.

No backend, API-contract, or database change — every endpoint already
accepts the combined `PATCH` body and the disable/enable/delete calls are
unchanged.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `web-client-categories`: the "Move to…", disable/enable, and delete
  actions are offered from the category **edit dialog** rather than the
  inline node row; reparenting is applied as part of the edit dialog's Save,
  and disable/enable no longer has a confirmation step.

## Impact

- **Frontend:** `frontend/src/routes/categories.tsx` (remove the `moving`
  and disable/enable `confirming` state and their dialogs; extend the edit
  form with a parent picker and the disable/enable + delete buttons; trim
  the inline row), `frontend/src/i18n/locales/{en,de}.json` (relocate keys,
  drop `categories.confirm.disable*` / `enable*`).
- **Specs:** `openspec/specs/web-client-categories/spec.md` — three
  requirements reworded.
- No backend, `openapi/`, or generated-artifact changes.

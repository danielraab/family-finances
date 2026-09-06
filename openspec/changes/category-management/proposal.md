## Why

Categories are currently a single global tree, writable only by an admin
(`internal/category`, migration `0008_categories.sql`) — every user picks
from the same admin-curated list when recording a transaction, with no
self-service way to add, rename, reorganize, retire, or remove one. That
doesn't fit how a family actually budgets: each user wants their own
categories, shaped how they think about their own spending, without asking
an admin to do it for them. This change turns categories into a per-user
resource, adds a management page of their own (categories can now only be
created or edited from that page, never inline elsewhere), and adds the two
lifecycle controls a self-service tree needs that the admin-managed one
never did: a reversible "disable" for retiring a category from new use
without touching history, and manual sibling ordering.

## What Changes

- **Ownership**: `categories` gains `owner_id`, scoped exactly like
  `entry-tags` — every list/get/create/update/disable/enable/delete is
  scoped to `owner_id = <authenticated caller>`; a category belonging to a
  different user behaves as if it doesn't exist (`404`). The `is_admin` gate
  on category writes is removed entirely — any authenticated user fully
  manages their own tree, no admin involvement.
- **Disable becomes reversible, delete becomes soft, and they're
  independent of each other**:
  - `disabled` (new bool, default `false`): toggled via
    `POST /api/categories/{id}/disable` / `/enable`, the same shape as
    `account_types.disabled`. Blocks the category from being selected on a
    *new or edited* entry; every entry and child category already
    referencing it is untouched, and it still appears in the tree so it can
    be reorganized or re-enabled.
  - `DELETE /api/categories/{id}` becomes a soft delete (`deleted_at`,
    no undelete) instead of today's hard delete, but keeps the same guard:
    rejected (`409`) while it has any non-deleted child category, or is
    referenced by any non-deleted entry.
- **Manual sibling ordering**: a new `sort_order` column, scoped per
  `(owner_id, parent_id)` — it only orders siblings under the same parent.
  Two new button-driven endpoints, `POST /api/categories/{id}/move-up` and
  `/move-down`, each swap `sort_order` with the adjacent sibling; at either
  end of the sibling list the call is a no-op (`200`, order unchanged).
  Reparenting (`PATCH .../{id}` with `parent_id`, unchanged mechanism, same
  cycle guard) appends the category to the end of its new parent's sibling
  order.
- **No naming-uniqueness constraint** — today's `UNIQUE(parent_id, name)`
  is dropped rather than reworked into an owner-scoped one; duplicate
  sibling names are allowed.
- **Frontend**: a new "Categories" sidebar entry, next to Accounts and
  Entries, linking to a new `/categories` page — the only place a category
  can be created or edited. It renders the caller's tree with
  mobile-friendly, button-driven controls per node (rename, disable/enable,
  ▲/▼ reorder among siblings, a "Move to…" picker for reparenting instead
  of drag-and-drop, and delete, disabled client-side whenever the category
  currently has a child or an entry reference). The entry form's category
  picker (`entries.new.tsx` / `entries.$entryId.edit.tsx`) keeps its
  existing flattened-tree `<select>` for choosing a category on an entry —
  it never offered inline creation and still won't — but now excludes
  disabled categories from new selections, rendering an already-selected,
  since-disabled category as a non-selectable extra option so the form
  doesn't look like it silently lost data (mirroring the account-type
  dropdown's existing pattern in `AccountForm.tsx`).
- `openapi/openapi.yaml` gains the new `Category`/`CategoryWrite` fields and
  paths; `backend/openapi.yaml` and `frontend/src/api/schema.d.ts` are
  regenerated in the same change.
- This is a greenfield change to the `categories` table — no production data
  to preserve, so the migration adds the new columns directly with no
  backfill logic.

## Capabilities

### New Capabilities

- `web-client-categories`: the `/categories` sidebar page — tree view,
  create/rename, disable/enable, move-up/move-down reordering, the "Move
  to…" reparent picker, and delete.

### Modified Capabilities

- `entry-categories`: from a global, admin-managed tree to a per-user tree
  the owner fully self-serves — ownership/visibility scoping, disable,
  soft delete (replacing hard delete), and sibling ordering.
- `account-entries`: an entry's `category_id` must now belong to the same
  owner as the entry (`422` otherwise), mirroring the existing tag-ownership
  rule.
- `web-client-entries`: the category picker on the entry form excludes
  disabled categories from new selections, same treatment as the account
  type dropdown.

## Impact

- **Dependencies**: none new.
- **Code**: `backend/internal/category/*.go` rewritten for ownership,
  disable, soft delete, and sort order; `internal/storage/memory` and
  `internal/storage/postgres`'s `CategoryStore` updated to match; one new
  migration; `internal/entry`'s category-ownership check added the same way
  its tag-ownership check already exists. Frontend: new
  `src/routes/categories*.tsx`, a category-tree component, a new `Sidebar`
  entry, `AccountForm.tsx`-style filtering added to the entry form's
  category picker.
- **API contract**: `Category` gains `disabled`, `sort_order`; adds
  `POST /api/categories/{id}/disable`, `/enable`, `/move-up`, `/move-down`;
  removes the `is_admin` requirement noted in the existing endpoint
  summaries/descriptions. Regenerate `backend/openapi.yaml` and
  `frontend/src/api/schema.d.ts` in the same change.
- **Spec**: new `web-client-categories`; deltas on `entry-categories`,
  `account-entries`, `web-client-entries`.

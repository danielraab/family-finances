## Context

`internal/category` (from `add-accounts-entries`) is a small, four-file
domain package: `category.go` (the `Category`/`New`/`Update` types plus the
`OptionalID` tri-state helper for explicit reparenting-to-root),
`store.go` (the `Store` interface + sentinels `ErrNotFound`,
`ErrInvalidValue`, `ErrInUse`, `ErrCycle`), `service.go` (thin use-case
logic — cycle detection via `Subtree` before an update), `handler.go` (HTTP
wiring, writes gated on `is_admin`). `storage/postgres/category.go` and
`storage/memory` implement `Store`; the Postgres `Delete` hard-deletes
inside one query that rejects (`0` rows affected → `ErrInUse`) when a child
or a non-deleted entry references the row. `internal/entry` consumes
`category.Service` only through a narrow `CategoryLookup` interface
(`Exists`, `Subtree`) it declares itself — it never imports `internal/category`
directly, matching the one-way-dependency rule in `backend/AGENTS.md`.

Two existing features are the templates for everything this change adds:

- `internal/tag` (`entry-tags`): `owner_id`, every `Store` method scoped to
  it, cross-owner access reads as `404`. `internal/entry` already enforces
  "a tag on an entry must belong to the entry's owner" — the same check,
  swapped to categories, is what `account-entries` needs added.
- `internal/account`'s `account_types.disabled` (`account-types-settings-tab`):
  a reversible bool, `POST .../disable` / `/enable`, that blocks new
  assignment without touching anything already using it, and stays visible
  in listings so it can be managed and re-enabled.

Neither precedent covers soft delete on a tree, or manual ordering — those
are new to this codebase, though `deleted_at`-with-no-undelete is an
established shape (`auth.users`, `auth.invites`, `accounts`, `entries`).

## Goals / Non-Goals

**Goals:**

- Every category operation is scoped to its owner; the `is_admin` gate is
  gone from `internal/category` entirely.
- `disabled` and soft delete are independent controls: disabling never
  affects the delete gate (children/entries in use), and deleting never
  requires having disabled first.
- A category's siblings (rows sharing the same `owner_id` + `parent_id`)
  have a stable, user-adjustable order via `sort_order`.
- A `/categories` page is the only place a category can be created or
  renamed; it's mobile-friendly, driven entirely by buttons — no drag
  gestures for either reordering or reparenting.

**Non-Goals:**

- Any household/shared-ownership concept — categories are private to one
  `user`, exactly like tags, not shared across a family (decided during
  exploration; introducing a household model is out of scope here).
- Preserving existing category/entry data through the ownership change —
  this is a greenfield migration; no backfill strategy is needed or
  attempted (confirmed: no production data to preserve).
- Re-adding any naming-uniqueness constraint. Today's
  `UNIQUE(parent_id, name)` didn't even work as intended (Postgres treats
  `NULL parent_id` as distinct per row, so two roots could already collide
  or not by accident) — this change drops it outright rather than fixing it
  into an owner-scoped version. Duplicate sibling names are allowed,
  deliberately.
- Bulk actions (multi-move, multi-disable) on the new page.
- Any change to `account_types` or `accounts` — this touches only
  `internal/category` and `internal/entry`'s category-ownership check.
- Inline category creation from the entry form — it never offered that,
  and still won't; the `/categories` page is the only creation surface.

## Decisions

### Decision: owner-scoping mirrors `entry-tags`, not `account_types`

Categories move from "global, admin-managed" to "private, owner-managed" —
the opposite direction from `account_types`, which stays admin-managed.
Every `Store` method gains an explicit `ownerID` parameter (not an implicit
context filter), matching `account.Store`'s existing shape rather than
`tag.Store`'s (tags don't currently take an explicit `ownerID` param on
every method since `internal/tag` is smaller and newer; `internal/category`
already has more surface — `Subtree`, `Exists` — used by `internal/entry`,
so being explicit about whose tree is in scope avoids an easy mistake in a
recursive query). A category belonging to a different owner is `ErrNotFound`
(`404`), never `403` — there is no scenario where a user should learn
another user's category exists.

### Decision: `disabled` and soft delete are orthogonal, not staged

The task explicitly separates "disable" (new-use gate) from "delete"
(existence gate on children/entries) — this change implements them as two
independent flags with two independent gates, deliberately **not** requiring
`disabled = true` before delete is even attempted. In practice a user will
usually disable before deleting (to confirm nothing new gets added to it
while cleaning up its remaining references), but the backend doesn't
enforce that ordering — a category with zero children and zero entry
references can be deleted directly, disabled or not, the same day it was
created.

`Delete` changes from a hard `DELETE FROM categories ... RETURNING` to an
`UPDATE categories SET deleted_at = now() WHERE ... AND deleted_at IS NULL
AND <same two NOT EXISTS guards as today>` — the guards themselves are
unchanged except now scoped to the owner and to `deleted_at IS NULL` on the
child-category check (a soft-deleted child no longer blocks its
already-soft-deleted parent from ever being deleted, though that shouldn't
actually arise given delete already required zero live children). No
undelete endpoint — matches `auth.users`/`auth.invites`'s existing
soft-delete shape. `DeletedAt *time.Time` stays `json:"-"`, same as
`account.Account.DeletedAt` — a deleted row simply stops appearing from
`GET /api/categories`, the frontend never needs to render its own deleted
state.

### Decision: reordering is two dedicated swap endpoints, not a generic "set sort_order" field

`POST /api/categories/{id}/move-up` and `/move-down` each swap the target's
`sort_order` with its immediate previous/next sibling (same `owner_id` +
`parent_id`, ordered by `sort_order` then `id` for a stable tiebreak). This
maps directly onto the mobile UI's ▲/▼ buttons with no client-side sort-order
arithmetic, and it's atomic and always valid — there's no invalid
`sort_order` value a client could send. At either end of the sibling list
(no previous/next sibling to swap with) the endpoint is a no-op: `200` with
the category unchanged, not an error — consistent with how this codebase
treats an edge-of-range action as success-but-nothing-happened rather than
a client error (e.g. `invite-revocation`'s idempotent re-revoke).

New categories append to the end of their sibling group:
`sort_order = COALESCE(MAX(sort_order) + 1, 0) WHERE owner_id = $1 AND
parent_id IS NOT DISTINCT FROM $2`. Reparenting (`PATCH` with `parent_id`)
does the same append-to-end-of-new-siblings computation as part of the
update — it does not preserve the category's old `sort_order` value into
its new sibling group, since that number was only ever meaningful among its
old siblings.

### Decision: reparenting stays on `PATCH`, not a new `/move-to` endpoint

The "Move to…" picker is frontend UX, not a new backend shape — it's the
same `PATCH /api/categories/{id}` with `parent_id` (via the existing
`OptionalID` tri-state: absent = untouched, `null` = make root, a value =
that parent) that reparenting already uses today, now owner-scoped (the
`Subtree`/cycle-detection query only walks the caller's own tree) and
landing at the end of the new parent's sibling order per the decision
above. No new sentinel or endpoint needed — `ErrCycle` (`422`) still covers
"reparent onto self or a descendant."

### Decision: no naming-uniqueness constraint, dropped rather than migrated

Explicitly decided during exploration: don't replace
`UNIQUE(parent_id, name)` with an owner-scoped equivalent. The migration
drops the constraint; `category.New`/`Update` validation stays exactly
`validateName` (non-empty after trim) — no duplicate-name check anywhere,
client or server.

## Migration Plan

1. `backend/internal/storage/postgres/migrations/0014_category_ownership.sql`:
   ```sql
   ALTER TABLE categories DROP CONSTRAINT categories_parent_id_name_key;
   ALTER TABLE categories
     ADD COLUMN owner_id   uuid NOT NULL REFERENCES users(id),
     ADD COLUMN sort_order integer NOT NULL DEFAULT 0,
     ADD COLUMN disabled   boolean NOT NULL DEFAULT false,
     ADD COLUMN deleted_at timestamptz;
   CREATE INDEX ON categories (owner_id, parent_id);
   ```
   Greenfield: `owner_id NOT NULL` with no default is fine because there's
   no real data to preserve; a dev database with seeded rows may need those
   rows cleared as part of applying this migration locally (not a data
   migration this change needs to write — just a note for anyone with local
   dev data predating it).
2. `internal/category`: `category.go` — `Category` gains `OwnerID string`
   (`json:"-"`, same as `Account.OwnerID`), `SortOrder int`,
   `Disabled bool`, `DeletedAt *time.Time` (`json:"-"`); `New`/`Update`
   unchanged in shape (still just `ParentID`/`Name` — `owner_id` is always
   the caller, never client-supplied; `sort_order`/`disabled` are never
   client-settable directly, only through the dedicated endpoints).
   `store.go` — every `Store` method gains an `ownerID` parameter; add
   `SetDisabled(ctx, ownerID, id string, disabled bool) (Category, error)`,
   `MoveUp`/`MoveDown(ctx, ownerID, id string) (Category, error)`.
   `service.go` — thread `ownerID` through; `Create` computes the
   append-to-end `sort_order`; `Update`'s reparent path recomputes it for
   the new parent. `handler.go` — drop `requireAdmin`; every handler reads
   `ownerID` from `auth.UserFromContext`; add the four new routes.
3. `internal/storage/memory` and `internal/storage/postgres`: implement the
   updated `Store` signatures; `Delete` becomes the soft-delete `UPDATE`
   described above; `MoveUp`/`MoveDown` as a single swap query (find the
   adjacent sibling by `sort_order`, swap both rows' values in one
   statement or a short transaction).
4. `internal/entry`: extend the existing tag-ownership check
   (`CategoryLookup`/`TagLookup`-style interface, see how the tag version
   works today) so a `category_id` on a create/update must belong to the
   entry's owner — same `422` treatment as the equivalent tag rule.
5. `openapi/openapi.yaml`: `Category` gains `disabled` (required, bool),
   `sort_order` (required, integer); add
   `POST /api/categories/{id}/disable`, `/enable`, `/move-up`, `/move-down`
   (`200` → `Category`, `401`, `404`); update the `POST`/`PATCH`/`DELETE
   /api/categories...` summaries/descriptions to drop every "(admin only)"
   note and drop their now-unreachable `403` responses. Then
   `cd backend && go generate ./...` and
   `cd frontend && pnpm generate:api`.
6. Frontend: new `Sidebar` entry + route + tree component (see `tasks.md`
   for the full breakdown); `AccountForm.tsx`-style disabled-option
   handling added to the entry form's category picker.

## Verification

- Backend: unit + handler tests — cross-owner category access reads as
  `404` on every verb; non-admin (now: any user) can create/update/disable/
  enable/delete their own categories with no `is_admin` check anywhere;
  disabling a category in use leaves referencing entries and child
  categories untouched and still functional; deleting a category with a
  live child or a live entry reference is rejected (`409`) whether or not
  it's disabled; deleting a category with neither succeeds and it
  disappears from `GET /api/categories`; move-up/move-down swap order
  correctly and no-op at either end of the sibling list; reparenting lands
  at the end of the new parent's sibling order and still rejects a
  self/descendant cycle (`422`); creating/updating an entry with a
  `category_id` owned by a different user is rejected (`422`);
  `internal/storage/postgres` integration tests for the migration and the
  new store methods.
- Frontend: `pnpm lint && pnpm exec tsc && pnpm build`; manual pass —
  create a small tree on `/categories`, reorder siblings with ▲/▼, reparent
  via "Move to…" (including onto a descendant, confirming it's rejected),
  disable a category and confirm it's still visible on the page but
  excluded from the entry form's picker for a new selection, confirm an
  entry already on that category still renders it read-only, confirm
  Delete is disabled/greyed on any category with a child or an entry
  reference and enabled once neither is true, confirm the whole flow on a
  narrow (mobile) viewport with no drag gestures needed anywhere.

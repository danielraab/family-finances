# entry-categories Specification

## Purpose

A per-user, tree-structured category lookup entries are classified by —
each user fully self-serves their own tree, no admin involvement. See
`account-entries` for how a category is required for a transaction and
optional for a balance adjustment, and for the owner-match and
not-disabled rules an entry's `category_id` must satisfy.

## Requirements

### Requirement: Categories form a per-user tree the owner fully self-serves

`categories` SHALL be private to the user who owns them, via a required
`owner_id` set to the authenticated caller at creation — not a single
global tree shared by the whole instance. Each category tree is
self-referencing (`parent_id`, no depth limit) within its owner's own
categories. `GET /api/categories` SHALL return the full tree belonging to
the authenticated caller. Creating, updating, disabling, enabling, and
deleting a category SHALL require only authentication — there is no
admin-only gate on any category operation. A category belonging to a
different owner SHALL behave as if it does not exist (`404`) for every
operation, not `403`.

#### Scenario: An authenticated user lists their own category tree

- **WHEN** an authenticated user calls `GET /api/categories`
- **THEN** the response is `200` with every non-deleted category they own,
  including each one's `parent_id`

#### Scenario: A user cannot see another user's category

- **WHEN** an authenticated user calls an operation on a category owned by
  a different user
- **THEN** the response is `404`

#### Scenario: Any authenticated user manages their own categories

- **WHEN** an authenticated user calls `POST`, `PATCH`, `DELETE`, or the
  `/disable`/`/enable` actions on a category they own
- **THEN** the request is not rejected for lack of admin privileges

### Requirement: Category deletion is a soft delete, rejected when it would break the tree or orphan an entry

Deleting a category SHALL set `deleted_at` rather than removing the row,
and SHALL be irreversible — there is no undelete operation. Deleting a
category that has one or more non-deleted child categories SHALL be
rejected (`409`). Deleting a category referenced by at least one
non-deleted entry SHALL also be rejected (`409`). Neither case cascades or
reparents automatically. A soft-deleted category no longer appears in
`GET /api/categories` and cannot be selected on any entry.

#### Scenario: Deleting a category with children is rejected

- **WHEN** a user attempts to delete a category that has at least one
  non-deleted child category
- **THEN** the response is `409` and the category is not deleted

#### Scenario: Deleting an in-use category is rejected

- **WHEN** a user attempts to delete a category referenced by a non-deleted
  entry
- **THEN** the response is `409` and the category is not deleted

#### Scenario: Deleting an unused, childless category succeeds

- **WHEN** a user deletes a category that has no child categories and no
  entry referencing it
- **THEN** the response is `204`, `deleted_at` is set, and the category no
  longer appears in `GET /api/categories`

### Requirement: A category cannot become its own ancestor

Setting a category's `parent_id` to itself, or to any of its own
descendants, SHALL be rejected (`422`) rather than creating a cycle. Cycle
detection is scoped to the caller's own tree.

#### Scenario: Self-parenting rejected

- **WHEN** a user updates their own category to set its own id as
  `parent_id`
- **THEN** the response is `422` and the category's `parent_id` is
  unchanged

#### Scenario: Cycle through a descendant rejected

- **WHEN** a user updates category A, whose child is B, to set A's
  `parent_id` to B
- **THEN** the response is `422` and no category's `parent_id` is changed

### Requirement: Disabling a category blocks new use without touching history

Each category SHALL carry `disabled` (boolean, `false` by default),
toggled via `POST /api/categories/{id}/disable` and reversed via
`POST /api/categories/{id}/enable`. Disabling a category SHALL NOT change,
hide, or otherwise affect any entry or child category already referencing
it, and SHALL NOT remove it from `GET /api/categories` — its only effect is
that it can no longer be newly selected on an entry (see `account-entries`).
Disabling and soft delete are independent: a category need not be disabled
before it can be deleted, and disabling a category never affects whether it
can subsequently be deleted.

#### Scenario: Disabling does not affect existing entries or children

- **WHEN** a user disables a category that one or more of their entries or
  child categories currently reference
- **THEN** those entries and child categories are unchanged, and the
  category itself still appears in `GET /api/categories` with
  `disabled: true`

#### Scenario: Enabling reverses it

- **WHEN** a user enables a previously disabled category
- **THEN** it becomes selectable again for new or edited entries

#### Scenario: A disabled category with no relations can still be deleted directly

- **WHEN** a user deletes a disabled category that has no children and no
  entry referencing it
- **THEN** the response is `204`, the same as for a non-disabled category
  in the same state

### Requirement: Sibling categories have a user-adjustable order

Each category SHALL carry `sort_order` (integer), meaningful only among its
siblings — the other categories sharing its `owner_id` and `parent_id`. A
newly created category SHALL be appended to the end of its sibling group.
`POST /api/categories/{id}/move-up` and `/move-down` SHALL swap the
category's `sort_order` with its immediate previous or next sibling,
respectively. Calling `move-up` on the first sibling, or `move-down` on the
last, SHALL be a no-op: `200` with the category's order unchanged, not an
error. Reparenting a category (via `PATCH .../{id}` with `parent_id`)
SHALL append it to the end of its new parent's sibling order.

#### Scenario: Moving a category up swaps it with the previous sibling

- **WHEN** a user calls `POST /api/categories/{id}/move-up` on a category
  that is not already first among its siblings
- **THEN** its `sort_order` and its previous sibling's `sort_order` are
  swapped

#### Scenario: Moving the first sibling up is a no-op

- **WHEN** a user calls `POST /api/categories/{id}/move-up` on a category
  that is already first among its siblings
- **THEN** the response is `200` and no `sort_order` changes

#### Scenario: A new category is appended to the end of its siblings

- **WHEN** a user creates a category under a parent that already has
  existing children
- **THEN** the new category's `sort_order` places it after every existing
  sibling

#### Scenario: Reparenting appends to the new parent's sibling order

- **WHEN** a user reparents a category onto a different parent that already
  has children
- **THEN** the reparented category's `sort_order` places it after that
  parent's existing children

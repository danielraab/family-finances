# entry-categories Specification

## Purpose

A per-user, tree-structured category lookup entries are classified by —
each user fully self-serves their own tree, no admin involvement, though a
category MAY also be shared with other users (see `category-sharing` for
the tiers and share-management endpoints). See `account-entries` for how a
category is required for a transaction and optional for a balance
adjustment, and for the owner-match and not-disabled rules an entry's
`category_id` must satisfy.

## Requirements

### Requirement: Categories form a per-user tree the owner fully self-serves

`categories` SHALL be private to the user who owns them, via a required
`owner_id` set to the authenticated caller at creation — not a single
global tree shared by the whole instance. Each category tree is
self-referencing (`parent_id`, no depth limit) within its owner's own
categories; reparenting is scoped strictly to the owner's own tree,
unaffected by any share (see `category-sharing`). `GET /api/categories`
SHALL return the full tree belonging to the authenticated caller, plus
every category currently shared with them (each a flat row, not nested,
regardless of its position in its real owner's tree), each of the
latter carrying `permission` (`view` or `append`), `shared: true`, and
`owner_name`. A category the caller owns SHALL carry `permission: owner`
and no `shared`/`owner_name`. Creating, updating, disabling, enabling,
and deleting a category SHALL require only authentication and, for any
of those five operations, real ownership — a share never grants any of
them, regardless of tier. A category the caller neither owns nor has any
share on SHALL behave as if it does not exist (`404`) for every
operation, not `403`.

#### Scenario: An authenticated user lists their own category tree

- **WHEN** an authenticated user calls `GET /api/categories`
- **THEN** the response is `200` with every non-deleted category they
  own, each carrying `permission: owner`, including each one's
  `parent_id`

#### Scenario: Listing includes categories shared with the caller

- **WHEN** an authenticated user who owns one category and has a share on
  another calls `GET /api/categories`
- **THEN** both appear, the shared one carrying its `permission`
  (`view`/`append`), `shared: true`, and the real owner's `owner_name`

#### Scenario: A user cannot see another user's unshared category

- **WHEN** an authenticated user calls an operation on a category owned
  by a different user, with no share granting them access
- **THEN** the response is `404`

#### Scenario: Only the real owner manages a category's own metadata and lifecycle

- **WHEN** an authenticated user calls `POST`, `PATCH`, `DELETE`, or the
  `/disable`/`/enable` actions on a category they do not really own —
  including one shared with them at `append`
- **THEN** the request is rejected, exactly as if they had no permission
  on it at all

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

### Requirement: A new user starts with a seeded set of default categories

Every new user account SHALL be seeded, at creation, with a fixed starter
set of root categories owned by them (Salary, Groceries, Rent, Utilities,
Transportation, Entertainment, Health, Other), in English, ordered by
`sort_order` in that same order, regardless of any language preference. Each
seeded category SHALL also be given a fixed default `icon` and `color` from
the client's curated icon set and palette — the same assignment for every
user, independent of language. A user MAY freely rename, reparent, disable,
delete, or change the icon/colour of any seeded category exactly as if they
had created it themselves; seeding SHALL NOT recur or top up a user's tree
after account creation, and SHALL NOT run for a user who already has at
least one category.

#### Scenario: A freshly created user already has categories to choose from

- **WHEN** a new user account is created (via either sign-in method)
- **THEN** `GET /api/categories` for that user immediately returns the full
  starter set, before they have created any category themselves

#### Scenario: Seeded categories arrive with an icon and colour

- **WHEN** a new user account is created and `GET /api/categories` is called
  for that user
- **THEN** every seeded category carries a non-empty `icon` and `color`

#### Scenario: Seeded categories are ordinary, fully editable categories

- **WHEN** a user renames, reparents, disables, deletes, or changes the
  icon or colour of a seeded category
- **THEN** the operation succeeds exactly as it would for a category they
  created themselves

### Requirement: A category carries an optional icon and colour

A category SHALL carry two optional fields, `icon` and `color`, each a short
opaque string the backend stores and returns but does not interpret. Each
field, when present, SHALL match `^[a-z0-9-]{1,40}$`; a value failing that
shape SHALL be rejected (`400`) and nothing changed. The two fields are
independent: a category MAY have neither, only `icon`, only `color`, or
both. Both SHALL be accepted on `POST /api/categories` and
`PATCH /api/categories/{id}`, and SHALL appear on every category returned by
`GET /api/categories` and the other category responses. On update, supplying
a field as an empty string SHALL clear it; omitting the field SHALL leave it
unchanged. The backend SHALL NOT validate `icon` against any icon set nor
`color` against any palette.

#### Scenario: Creating a category with an icon and colour

- **WHEN** an authenticated user calls `POST /api/categories` with a `name`
  and a valid `icon` and `color`
- **THEN** the response is `201` and the created category carries the given
  `icon` and `color` verbatim

#### Scenario: Icon and colour are optional

- **WHEN** an authenticated user calls `POST /api/categories` with only a
  `name`
- **THEN** the category is created with neither `icon` nor `color` set

#### Scenario: A malformed icon or colour value is rejected

- **WHEN** `POST /api/categories` or `PATCH /api/categories/{id}` sends an
  `icon` or `color` containing characters outside `[a-z0-9-]` or longer than
  40 characters
- **THEN** the request is rejected (`400`) and the category is not created
  or changed

#### Scenario: Clearing the colour on update

- **WHEN** the owner calls `PATCH /api/categories/{id}` with `color` set to
  `""` (empty string)
- **THEN** the update succeeds and the category's `color` is thereafter
  unset, while its `icon` is unchanged

#### Scenario: Renaming a category leaves its icon and colour intact

- **WHEN** the owner calls `PATCH /api/categories/{id}` changing only
  `name`, with no `icon` or `color` in the body
- **THEN** the category keeps whatever `icon` and `color` it already had

### Requirement: Every category response includes the caller's entry count

Every `Category` returned by the API — from `GET /api/categories` and from
every other category endpoint that returns a category
(`POST /api/categories`, `PATCH /api/categories/{id}`, the `/disable`,
`/enable`, `/move-up`, and `/move-down` actions) — SHALL include
`entry_count`: the number of the *caller's own* non-deleted entries whose
`category_id` is that category, regardless of whether the caller is that
category's real owner or a share recipient. The count SHALL reflect
direct references only — an entry categorized under a child category
SHALL NOT be counted toward that child's ancestors. A soft-deleted entry
SHALL NOT be counted. The count SHALL be computed per request from
current data, not stored on the category row.

#### Scenario: A freshly created category has zero entries

- **WHEN** a user creates a new category
- **THEN** the response's `entry_count` is `0`

#### Scenario: Entry count reflects current categorization

- **WHEN** a user categorizes two of their entries under a category and
  then recategorizes one of them under a different category
- **THEN** `GET /api/categories` subsequently reports the first category's
  `entry_count` as `1`

#### Scenario: A deleted entry no longer counts toward its category

- **WHEN** an entry categorized under a category is (soft-)deleted
- **THEN** that category's `entry_count` no longer includes it

#### Scenario: A child category's entries do not count toward its parent

- **WHEN** a user has a parent category with one child category, and three
  of their entries are categorized under the child and none directly under
  the parent
- **THEN** `GET /api/categories` reports the parent's `entry_count` as `0`
  and the child's as `3`

#### Scenario: Another user's entries do not count

- **WHEN** two users each categorize entries under categories of their own
- **THEN** each category's `entry_count` counts only entries owned by that
  category's owner

#### Scenario: A shared category's entry count reflects only the viewer's own entries

- **WHEN** a category's real owner has five of their own entries under
  it, and a user it's shared with at `append` has two of their own
  entries under the same category
- **THEN** the real owner's `GET /api/categories` reports that category's
  `entry_count` as `5`, and the recipient's own `GET /api/categories`
  reports it as `2`

### Requirement: A category shared at append tier is usable on an entry, exactly like an owned one

A category SHALL be selectable as an entry's `category_id` (on create, or
on an update that explicitly supplies it) when the acting user either
owns it or holds an `append`-tier share on it, and it is not disabled —
in every other respect identical to the existing owned-category rule. A
category shared only at `view` tier, or shared at `append` but currently
disabled, SHALL NOT be newly selectable, exactly as an unshared disabled
category already isn't. As with disabling, this check applies only when
a category is being *newly* set — an update that leaves `category_id`
untouched never re-validates it, so a revoked or downgraded share never
retroactively invalidates an entry already categorized under it.

#### Scenario: An append-tier recipient can categorize an entry with it

- **WHEN** a user with an `append`-tier share on a category creates or
  updates an entry setting that `category_id`
- **THEN** the entry is saved with that category

#### Scenario: A view-tier recipient cannot select the category on an entry

- **WHEN** a user with only a `view`-tier share on a category attempts to
  create or update an entry setting that `category_id`
- **THEN** the request is rejected (`400`), identically to selecting a
  category the user has no access to at all

#### Scenario: A disabled shared category cannot be newly selected

- **WHEN** a user with an `append`-tier share on a category that has
  since been disabled attempts to newly select it on an entry
- **THEN** the request is rejected (`400`), identically to an owned
  disabled category

#### Scenario: Revoking a share does not affect entries already categorized under it

- **WHEN** a user's `append`-tier share on a category is revoked after
  they categorized entries under it
- **THEN** those entries keep that `category_id` and continue to resolve
  and display it normally, for every user who can otherwise see them

### Requirement: An entry-list or report category filter resolves a category shared with the caller

Resolving a `category_id` filter (for `GET /api/entries` and its summary/
report variants) to "this category or its descendants" (the default
`CategoryMode: subtree`) SHALL succeed for a category the caller has at
least `view` permission on, whether owned or shared. For a category the
caller owns, this resolves the category and its full descendant subtree
within the caller's own tree, unchanged from before this capability
existed. For a category visible to the caller only via a share, it
resolves to that category alone — a share never cascades to descendants,
so there is nothing further to include. A category the caller has no
permission on at all SHALL be rejected, unchanged from before.

#### Scenario: Filtering by an owned category still includes its descendants

- **WHEN** a caller filters entries by a category they own that has
  children
- **THEN** matching entries include ones categorized under that category
  or any of its descendants

#### Scenario: Filtering by a shared category resolves to itself alone

- **WHEN** a caller filters entries by a category shared with them (view
  or append)
- **THEN** matching entries include only ones categorized directly under
  that category, not under any of its (unshared) children

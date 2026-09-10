## MODIFIED Requirements

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

### Requirement: Every category response includes the caller's entry count

Every `Category` returned by the API — from `GET /api/categories` and
from every other category endpoint that returns a category
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
- **THEN** `GET /api/categories` subsequently reports the first
  category's `entry_count` as `1`

#### Scenario: A deleted entry no longer counts toward its category

- **WHEN** an entry categorized under a category is (soft-)deleted
- **THEN** that category's `entry_count` no longer includes it

#### Scenario: A child category's entries do not count toward its parent

- **WHEN** a user has a parent category with one child category, and
  three of their entries are categorized under the child and none
  directly under the parent
- **THEN** `GET /api/categories` reports the parent's `entry_count` as
  `0` and the child's as `3`

#### Scenario: A shared category's entry count reflects only the viewer's own entries

- **WHEN** a category's real owner has five of their own entries under
  it, and a user it's shared with at `append` has two of their own
  entries under the same category
- **THEN** the real owner's `GET /api/categories` reports that category's
  `entry_count` as `5`, and the recipient's own `GET /api/categories`
  reports it as `2`

## ADDED Requirements

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

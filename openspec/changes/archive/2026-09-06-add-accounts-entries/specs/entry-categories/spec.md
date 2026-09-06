## ADDED Requirements

### Requirement: Categories form a global, admin-managed tree

`categories` SHALL be a single tree shared by the whole instance (not
per-user), via a nullable `parent_id` self-reference with no depth limit.
`GET /api/categories` SHALL be available to any authenticated user, returning
the full tree. Creating, updating, or deleting a category SHALL require
`is_admin` (`403` otherwise, `401` unauthenticated).

#### Scenario: Any authenticated user can list the category tree

- **WHEN** an authenticated non-admin calls `GET /api/categories`
- **THEN** the response is `200` with every category, including its
  `parent_id`

#### Scenario: Non-admin cannot manage categories

- **WHEN** an authenticated non-admin calls `POST`, `PUT`, or `DELETE` on
  `/api/categories`
- **THEN** the response is `403`

### Requirement: Category deletion is rejected when it would break the tree or orphan an entry

Deleting a category that has one or more child categories SHALL be rejected
(`409`). Deleting a category referenced by at least one non-soft-deleted
entry SHALL also be rejected (`409`). Neither case cascades or reparents
automatically.

#### Scenario: Deleting a category with children is rejected

- **WHEN** an admin attempts to delete a category that has at least one
  child category
- **THEN** the response is `409` and the category is not deleted

#### Scenario: Deleting an in-use category is rejected

- **WHEN** an admin attempts to delete a category referenced by a
  non-deleted entry
- **THEN** the response is `409` and the category is not deleted

### Requirement: A category cannot become its own ancestor

Setting a category's `parent_id` to itself, or to any of its own
descendants, SHALL be rejected (`422`) rather than creating a cycle.

#### Scenario: Self-parenting rejected

- **WHEN** an admin updates a category to set its own id as `parent_id`
- **THEN** the response is `422` and the category's `parent_id` is
  unchanged

#### Scenario: Cycle through a descendant rejected

- **WHEN** an admin updates category A, whose child is B, to set A's
  `parent_id` to B
- **THEN** the response is `422` and no category's `parent_id` is changed

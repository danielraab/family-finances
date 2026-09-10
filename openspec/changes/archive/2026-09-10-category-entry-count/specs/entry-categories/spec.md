## ADDED Requirements

### Requirement: Every category response includes the caller's entry count

Every `Category` returned by the API — from `GET /api/categories` and from
every other category endpoint that returns a category
(`POST /api/categories`, `PATCH /api/categories/{id}`, the `/disable`,
`/enable`, `/move-up`, and `/move-down` actions) — SHALL include
`entry_count`: the number of the caller's own non-deleted entries whose
`category_id` is that category. The count SHALL reflect direct references
only — an entry categorized under a child category SHALL NOT be counted
toward that child's ancestors. A soft-deleted entry SHALL NOT be counted.
The count SHALL be computed per request from current data, not stored on
the category row.

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

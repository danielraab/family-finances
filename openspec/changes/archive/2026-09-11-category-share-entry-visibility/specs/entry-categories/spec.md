## MODIFIED Requirements

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

When the caller holds any permission on the filtered category (real
ownership, or a `view`/`append` share) and did not also supply an
explicit `account_id` filter on the same request, matching entries
SHALL be returned regardless of whether the caller has any account-level
access to each entry's account — the caller's permission on the category
itself is what authorizes seeing its entries, independent of
account-sharing. This applies only to that specific, explicit
`category_id`-filtered request: it SHALL NOT change what an unfiltered
`GET /api/entries` or report shows, and SHALL NOT apply when the caller
supplies an explicit `account_id` filter alongside `category_id` (that
combination stays scoped to the caller's own visible accounts, as
today). A category the caller has no permission on at all continues to
resolve nothing, exactly as before — this widening is strictly narrower
than "any caller-supplied category_id value."

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

#### Scenario: Filtering by a shared category surfaces its entries across accounts

- **WHEN** a caller with a `view` or `append` share on a category filters
  entries (or a report) by that category, and the category's entries
  include one on an account the caller has no account-level access to
- **THEN** that entry is included in the results, exactly as if it were
  on an account the caller can access

#### Scenario: Filtering by an owned category surfaces its entries across accounts

- **WHEN** a category's real owner filters entries (or a report) by a
  category they own, and one of its entries was created against an
  account the owner has no account-level access to (created by a share
  recipient with `append` permission on the category and independent
  access to that account)
- **THEN** that entry is included in the results

#### Scenario: An explicit account filter alongside a category filter stays account-scoped

- **WHEN** a caller filters entries by both a specific `account_id` and a
  category they hold permission on
- **THEN** only entries matching both the named account and the category
  are returned — an entry under that category on a different,
  inaccessible account is not included

#### Scenario: An unfiltered listing is unaffected

- **WHEN** a caller with a share on a category lists entries or views a
  report with no `category_id` filter applied
- **THEN** the results include only entries on accounts the caller has
  account-level access to, exactly as before this capability existed

#### Scenario: A category with no permission at all never widens anything

- **WHEN** a caller supplies a `category_id` filter for a category they
  have no ownership or share on
- **THEN** the request is rejected (or resolves to no matching
  categories, per existing behavior) and no entry outside the caller's
  own visible accounts is ever returned

## ADDED Requirements

### Requirement: A new user starts with a seeded set of default categories

Every new user account SHALL be seeded, at creation, with a fixed starter
set of root categories owned by them (Salary, Groceries, Rent, Utilities,
Transportation, Entertainment, Health, Other), in English, ordered by
`sort_order` in that same order, regardless of any language preference. A
user MAY freely rename, reparent, disable, or delete any seeded category
exactly as if they had created it themselves; seeding SHALL NOT recur or
top up a user's tree after account creation, and SHALL NOT run for a user
who already has at least one category.

#### Scenario: A freshly created user already has categories to choose from

- **WHEN** a new user account is created (via either sign-in method)
- **THEN** `GET /api/categories` for that user immediately returns the full
  starter set, before they have created any category themselves

#### Scenario: Seeded categories are ordinary, fully editable categories

- **WHEN** a user renames, reparents, disables, or deletes a seeded
  category
- **THEN** the operation succeeds exactly as it would for a category they
  created themselves

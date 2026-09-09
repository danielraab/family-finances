## ADDED Requirements

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

## MODIFIED Requirements

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

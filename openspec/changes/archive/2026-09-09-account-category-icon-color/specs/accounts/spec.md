## ADDED Requirements

### Requirement: An account carries an optional icon and colour

An account SHALL carry two optional fields, `icon` and `color`, each a short
opaque string the backend stores and returns but does not interpret. Each
field, when present, SHALL match `^[a-z0-9-]{1,40}$`; a value failing that
shape SHALL be rejected (`400`) and nothing changed. The two fields are
independent: an account MAY have neither, only `icon`, only `color`, or
both. Both SHALL be accepted on `POST /api/accounts` and
`PATCH /api/accounts/{id}`, and SHALL appear on every response that carries
an `Account`. On update, supplying a field as an empty string SHALL clear
it; omitting the field SHALL leave it unchanged. The backend SHALL NOT
validate `icon` against any icon set nor `color` against any palette — their
meaning is entirely a client concern.

#### Scenario: Creating an account with an icon and colour

- **WHEN** an authenticated user calls `POST /api/accounts` with a valid
  `icon` and `color` alongside the required fields
- **THEN** the response is `201` and the created account carries the given
  `icon` and `color` verbatim

#### Scenario: Icon and colour are optional

- **WHEN** an authenticated user calls `POST /api/accounts` with no `icon`
  and no `color`
- **THEN** the response is `201` and the account is created with neither
  field set

#### Scenario: A malformed icon or colour value is rejected

- **WHEN** `POST /api/accounts` or `PATCH /api/accounts/{id}` sends an
  `icon` or `color` containing characters outside `[a-z0-9-]` or longer than
  40 characters
- **THEN** the request is rejected (`400`) and the account is not created or
  changed

#### Scenario: Clearing the icon or colour on update

- **WHEN** the owner calls `PATCH /api/accounts/{id}` with `icon` set to
  `""` (empty string)
- **THEN** the update succeeds and the account's `icon` is thereafter unset,
  while its `color` is unchanged

#### Scenario: An unrelated update leaves icon and colour intact

- **WHEN** the owner calls `PATCH /api/accounts/{id}` changing only `title`,
  with no `icon` or `color` in the body
- **THEN** the account keeps whatever `icon` and `color` it already had

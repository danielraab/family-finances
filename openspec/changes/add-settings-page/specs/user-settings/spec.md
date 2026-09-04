## ADDED Requirements

### Requirement: Per-user settings storage with hardcoded defaults

The backend SHALL store per-user preferences — display language, timezone,
and default currency — in a `user_settings` table with one row per user keyed
by `user_id`, each preference column nullable. A missing row and a row whose
column is `NULL` SHALL be treated identically: both resolve to a hardcoded
application default (`language: "en"`, `timezone: "UTC"`,
`default_currency: "EUR"`). No row SHALL be created automatically at account
creation; a row SHALL be created (or updated) only when a preference is
explicitly set.

Resolution (substituting the hardcoded default for a `NULL` or absent value)
SHALL happen once, in the settings service, so every consumer — the HTTP
endpoint below and any future backend feature — receives fully-populated
values, never `NULL`.

#### Scenario: No preferences set

- **WHEN** a user has no `user_settings` row
- **THEN** the resolved settings are `{ language: "en", timezone: "UTC",
  default_currency: "EUR" }`

#### Scenario: Partial preferences set

- **WHEN** a user's `user_settings` row has `language = "de"` and `timezone`
  and `default_currency` both `NULL`
- **THEN** the resolved settings are `{ language: "de", timezone: "UTC",
  default_currency: "EUR" }`

### Requirement: Settings validation

`language` SHALL be accepted only as `en` or `de`. `timezone` SHALL be
accepted only as a value `time.LoadLocation` (or equivalent IANA-tzdata
lookup) resolves successfully. `default_currency` SHALL be accepted only as
three uppercase ASCII letters (ISO-4217 shape); it is not checked against a
canonical currency list. An invalid value for any field SHALL be rejected
without changing any other field's stored value.

#### Scenario: Invalid language rejected

- **WHEN** a settings update is submitted with `language: "fr"`
- **THEN** the request is rejected and no field is changed

#### Scenario: Invalid timezone rejected

- **WHEN** a settings update is submitted with `timezone: "Not/AZone"`
- **THEN** the request is rejected and no field is changed

#### Scenario: Malformed currency rejected

- **WHEN** a settings update is submitted with `default_currency: "usd"` (not
  three uppercase letters) or `default_currency: "US"`
- **THEN** the request is rejected and no field is changed

### Requirement: Settings endpoints

`GET /api/settings` SHALL require authentication and SHALL return the
authenticated user's resolved settings (all three fields always populated).
`PUT /api/settings` SHALL require authentication and SHALL accept a partial
body containing any subset of `language`, `timezone`, `default_currency`;
only the fields present SHALL be changed, and the response SHALL be the
resulting resolved settings.

#### Scenario: Reading resolved settings

- **WHEN** an authenticated user with no stored preferences calls
  `GET /api/settings`
- **THEN** the response is `200` with the hardcoded defaults

#### Scenario: Updating one field leaves the others untouched

- **WHEN** an authenticated user with `language = "de"` already stored calls
  `PUT /api/settings` with `{ "timezone": "Europe/Vienna" }`
- **THEN** the response shows `language: "de"` and `timezone:
  "Europe/Vienna"`, and a subsequent `GET /api/settings` returns the same

#### Scenario: Unauthenticated access is rejected

- **WHEN** `GET /api/settings` or `PUT /api/settings` is called with no valid
  session
- **THEN** the response is `401`

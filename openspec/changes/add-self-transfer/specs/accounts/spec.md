## MODIFIED Requirements

### Requirement: Account fields and creation

An account SHALL carry: `title` (required, non-empty), `description`
(optional), `type` (required, a free-text label the backend only trims of
leading/trailing whitespace before storing — non-empty after trimming, not
case-folded, not checked against any canonical list or lookup table),
`currency` (required, ISO-4217 shape — three uppercase letters, not checked
against a canonical list, validated the same way as `user-settings`'
`default_currency`), `financial_institute` (optional), `opening_date`
(required), and `closing_date` (optional). When `closing_date` is present it
SHALL NOT be before `opening_date`. `type` SHALL be accepted on
`POST /api/accounts` and `PATCH /api/accounts/{id}` and SHALL appear on
every response that carries an `Account`.

`currency` SHALL become immutable once the account has any entry (of any
`kind`) recorded against it, regardless of whether the new value would
differ from the current one. `PATCH /api/accounts/{id}` supplying a
`currency` on such an account SHALL be rejected (`400` — the same
`ErrInvalidValue` mapping as a blank `title`/`type`); every other field
remains editable. An account with no entries yet keeps `currency` freely
editable, unchanged from before this rule existed.

#### Scenario: Creating an account with valid fields

- **WHEN** an authenticated user calls `POST /api/accounts` with a title, a
  non-empty `type`, a three-letter currency, and an opening date
- **THEN** the response is `201` with the created account, owned by the
  caller, its `type` stored with surrounding whitespace trimmed and its
  casing preserved verbatim

#### Scenario: A blank type is rejected

- **WHEN** `POST /api/accounts` or `PATCH /api/accounts/{id}` sends a
  `type` that is empty or only whitespace
- **THEN** the request is rejected (`400` — the same `ErrInvalidValue`
  mapping as a blank `title`) and nothing is created or changed

#### Scenario: Closing date before opening date rejected

- **WHEN** `POST /api/accounts` or an update sets `closing_date` earlier
  than `opening_date`
- **THEN** the request is rejected (`422`) and nothing is changed

#### Scenario: Invalid currency shape rejected

- **WHEN** an account is created or updated with a `currency` that is not
  three uppercase letters
- **THEN** the request is rejected (`422`)

#### Scenario: Currency is locked once an account has entries

- **WHEN** an authenticated user calls `PATCH /api/accounts/{id}` with a
  `currency` different from the account's current one, and the account has
  at least one entry recorded against it
- **THEN** the request is rejected (`400`) and the account's `currency` is
  unchanged

#### Scenario: Currency remains editable on an account with no entries

- **WHEN** an authenticated user calls `PATCH /api/accounts/{id}` with a
  new `currency` on an account that has no entries yet
- **THEN** the response is `200` and the account's `currency` is updated

#### Scenario: Other fields remain editable once currency is locked

- **WHEN** an authenticated user calls `PATCH /api/accounts/{id}` on an
  account with existing entries, changing `title` (or any field other than
  `currency`)
- **THEN** the update succeeds

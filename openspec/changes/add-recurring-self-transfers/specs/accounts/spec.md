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
`kind`) recorded against it, **or is named as either side of any
non-deleted `self_transfer` recurring transaction**, regardless of whether
the new value would differ from the current one. `PATCH /api/accounts/{id}`
supplying a `currency` on such an account SHALL be rejected (`400` — the
same `ErrInvalidValue` mapping as a blank `title`/`type`); every other
field remains editable. An account with neither entries nor a
`self_transfer` recurring transaction naming it keeps `currency` freely
editable, unchanged from before this rule existed. The recurring-transaction
half of this rule closes the same drift the entry half closes: a
`self_transfer` recurring transaction requires its two accounts to share a
currency when it is written, and pairs them long before either necessarily
has an entry, so without it a later currency change would silently make
every future booking from that template invalid.

#### Scenario: Currency locked once the account has entries

- **WHEN** `PATCH /api/accounts/{id}` supplies a `currency` for an account
  that has at least one entry of any kind
- **THEN** the request is rejected (`400`) and the account is unchanged

#### Scenario: Currency locked once a self-transfer template names the account

- **WHEN** `PATCH /api/accounts/{id}` supplies a `currency` for an account
  that has no entries but is the `account_id` or the `to_account_id` of a
  non-deleted `self_transfer` recurring transaction
- **THEN** the request is rejected (`400`) and the account is unchanged

#### Scenario: Currency still editable on an untouched account

- **WHEN** `PATCH /api/accounts/{id}` supplies a `currency` for an account
  with no entries and no `self_transfer` recurring transaction naming
  either side
- **THEN** the request succeeds and the account's currency is changed

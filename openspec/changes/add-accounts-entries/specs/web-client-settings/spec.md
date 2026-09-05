## MODIFIED Requirements

### Requirement: Common tab

The settings page SHALL default to a **Common** tab, visible to every
authenticated visitor, containing four controls: display language
(English/German), timezone (populated from the browser's supported IANA
zones), default currency (a three-letter code, validated client-side to
that shape), and displayed decimal places (an integer from 0 to 4). Each
control SHALL save on change, calling `PUT /api/settings` with only that
field, with no separate save action. Changing the language control SHALL
also switch the running app's language immediately, without a reload.

#### Scenario: Changing language applies immediately

- **WHEN** an authenticated visitor on the Common tab selects German
- **THEN** `PUT /api/settings` is called with `{ "language": "de" }`
- **AND** the app's UI text switches to German without a page reload

#### Scenario: Changing timezone does not affect other fields

- **WHEN** an authenticated visitor changes only the timezone control
- **THEN** the request updates only `timezone`, leaving language, default
  currency, and displayed decimal places as they were

#### Scenario: Changing displayed decimal places

- **WHEN** an authenticated visitor on the Common tab changes the
  displayed-decimal-places control to `0`
- **THEN** `PUT /api/settings` is called with
  `{ "displayed_decimal_places": 0 }`, and amounts shown elsewhere in the
  client (account balances, entry lists) subsequently round to whole
  numbers

## RENAMED Requirements

- FROM: `### Requirement: Common tab`
- TO: `### Requirement: Profile tab`

## MODIFIED Requirements

### Requirement: Profile tab

The settings page SHALL default to a **Profile** tab, visible to every
authenticated visitor, containing five controls: the visitor's display name (a
free-text field), display language (English/German), timezone (populated from
the browser's supported IANA zones), default currency (a three-letter code,
validated client-side to that shape), and displayed decimal places (an integer
from 0 to 4). The tab's label SHALL come from the `settings.tabs.profile` i18n
key and its field strings from the `settings.profile.*` namespace, in both the
`en` and `de` locale files; the route SHALL remain the `/settings` index.

The name control SHALL save on blur, calling `PATCH /api/auth/me` with
`{ "display_name": <value> }`, with no separate save action. On a successful
save the client SHALL update the shared `useAuth` user so the sidebar user
control reflects the new name without a page reload. On a failed save (for
example a `400` from a value the backend rejects) the field SHALL revert to the
last saved value and surface an error, matching the default-currency field's
revert-on-error behaviour. The other four controls SHALL each save on change,
calling `PUT /api/settings` with only that field, with no separate save action.
Changing the language control SHALL also switch the running app's language
immediately, without a reload.

#### Scenario: Changing the name updates the sidebar

- **WHEN** an authenticated visitor on the Profile tab edits the name field to
  "Jane Doe" and blurs it
- **THEN** `PATCH /api/auth/me` is called with `{ "display_name": "Jane Doe" }`
- **AND** on success the sidebar user control shows "Jane Doe" without a page
  reload

#### Scenario: A rejected name reverts

- **WHEN** the name field is edited to a value the backend rejects and the
  `PATCH /api/auth/me` call returns `400`
- **THEN** the field reverts to the last saved value and an error is shown, and
  the other Profile controls are unaffected

#### Scenario: Changing language applies immediately

- **WHEN** an authenticated visitor on the Profile tab selects German
- **THEN** `PUT /api/settings` is called with `{ "language": "de" }`
- **AND** the app's UI text switches to German without a page reload

#### Scenario: Changing timezone does not affect other fields

- **WHEN** an authenticated visitor changes only the timezone control
- **THEN** the request updates only `timezone`, leaving the name, language,
  default currency, and displayed decimal places as they were

#### Scenario: Changing displayed decimal places

- **WHEN** an authenticated visitor on the Profile tab changes the
  displayed-decimal-places control to `0`
- **THEN** `PUT /api/settings` is called with
  `{ "displayed_decimal_places": 0 }`, and amounts shown elsewhere in the
  client (account balances, entry lists) subsequently round to whole
  numbers

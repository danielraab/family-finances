## MODIFIED Requirements

### Requirement: Profile tab

The settings page SHALL default to a **Profile** tab, visible to every
authenticated visitor, containing seven controls: the visitor's display
name (a free-text field), display language (English/German), timezone
(populated from the browser's supported IANA zones), default currency (a
three-letter code, validated client-side to that shape), displayed decimal
places (an integer from 0 to 4), week start (Monday or Sunday), and the
recurring preview horizon (a named preset: 1 month from now, 2 months from
now, 3 months from now, end of this month, end of next month, or end of
this year). The tab's label SHALL come from the `settings.tabs.profile`
i18n key and its field strings from the `settings.profile.*` namespace, in
both the `en` and `de` locale files; the route SHALL remain the
`/settings` index.

The name control SHALL save on blur, calling `PATCH /api/auth/me` with
`{ "display_name": <value> }`, with no separate save action. On a successful
save the client SHALL update the shared `useAuth` user so the sidebar user
control reflects the new name without a page reload. On a failed save (for
example a `400` from a value the backend rejects) the field SHALL revert to the
last saved value and surface an error, matching the default-currency field's
revert-on-error behaviour. The other six controls SHALL each save on
change, calling `PUT /api/settings` with only that field, with no separate
save action. Changing the language control SHALL also switch the running
app's language immediately, without a reload.

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
  default currency, displayed decimal places, week start, and recurring
  preview horizon as they were

#### Scenario: Changing displayed decimal places

- **WHEN** an authenticated visitor on the Profile tab changes the
  displayed-decimal-places control to `0`
- **THEN** `PUT /api/settings` is called with
  `{ "displayed_decimal_places": 0 }`, and amounts shown elsewhere in the
  client (account balances, entry lists) subsequently round to whole
  numbers

#### Scenario: Changing week start

- **WHEN** an authenticated visitor on the Profile tab changes the week
  start control to "Sunday"
- **THEN** `PUT /api/settings` is called with `{ "week_start": "sunday" }`,
  and week-anchored date-range presets on `/entries` and `/reports`
  subsequently use Sunday as the start of the week

#### Scenario: Changing the recurring preview horizon

- **WHEN** an authenticated visitor on the Profile tab selects "End of next
  month" for the recurring preview horizon
- **THEN** `PUT /api/settings` is called with
  `{ "recurring_preview_horizon": "end_of_next_month" }`, and any preview
  toggle a visitor later enables on `/entries`, `/reports`, or a dashboard
  card resolves its cutoff using that horizon

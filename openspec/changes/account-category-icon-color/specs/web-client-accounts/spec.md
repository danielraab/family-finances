## ADDED Requirements

### Requirement: The account form offers an icon and colour picker

The account create form and the account edit form SHALL include the shared
`IconColorPicker`, letting the user set, change, or clear the account's
`icon` and `color` independently. On submit, the chosen values SHALL be sent
on the `POST /api/accounts` / `PATCH /api/accounts/{id}` body; an unset icon
or colour SHALL be sent such that the field is created without it, and
clearing a previously set field on edit SHALL send the empty string so the
backend clears it. The picker's presence SHALL NOT make either field
required — an account can still be created and saved with neither set.

#### Scenario: Setting an icon and colour while creating an account

- **WHEN** a visitor fills the create form, picks an icon and a colour, and
  submits
- **THEN** `POST /api/accounts` is called with that `icon` and `color`, and
  the new account carries them

#### Scenario: Clearing an account's colour from the edit form

- **WHEN** a visitor opens the edit form for an account that has a `color`,
  clears the colour in the picker, and submits
- **THEN** `PATCH /api/accounts/{id}` is called with `color` as `""` and the
  account's colour becomes unset

#### Scenario: Creating an account without touching the picker

- **WHEN** a visitor completes the create form leaving the icon/colour
  picker untouched and submits
- **THEN** the account is created with no `icon` and no `color`

### Requirement: Account surfaces render the account's icon and colour before its title

The accounts overview, the account detail page, and the home account cards
SHALL render each account's `EntityIcon` badge immediately before its title,
via the shared `AccountLabel` component, when the account has an `icon`
and/or `color` set. An account with neither set SHALL render exactly as
before.

#### Scenario: The account detail header shows the badge

- **WHEN** an authenticated visitor opens the detail page of an account that
  has an `icon` and a `color`
- **THEN** the header shows the icon/colour badge immediately before the
  account title

#### Scenario: A home account card shows the badge

- **WHEN** the home page renders an account card for an account that has an
  `icon`
- **THEN** the card shows that icon immediately before the account title

#### Scenario: An account with no icon or colour is unchanged

- **WHEN** the accounts overview lists an account with neither `icon` nor
  `color`
- **THEN** that row shows just the title, with no badge and no layout shift

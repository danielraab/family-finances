## ADDED Requirements

### Requirement: Home dashboard requires authentication, redirecting to the root placeholder

`/home` SHALL be accessible only to an authenticated visitor. While the
visitor's auth status is resolving, `/home` SHALL render nothing. An
anonymous visitor navigating to `/home` SHALL be redirected to `/` — not
`/login` — since `/home` is reached from the sidebar's "Home" item, which
anonymous visitors also see.

#### Scenario: Anonymous visitor is redirected to the placeholder

- **WHEN** an anonymous visitor navigates to `/home`
- **THEN** the client redirects them to `/`

#### Scenario: No flash while auth status resolves

- **WHEN** a visitor opens `/home` and their auth status has not yet
  resolved
- **THEN** the page renders nothing until the status resolves to
  `anonymous` or `authenticated`

### Requirement: Home dashboard shows a card per account with its balance

`/home` SHALL, for an authenticated visitor, fetch and display a card for
every account they own, each card showing at least its title, its
`financial_institute` (when set), and its live balance (`GET
/api/accounts/{id}/balance`, formatted at the visitor's
`displayed_decimal_places`). Activating a card SHALL navigate to that
account's details page (`/accounts/{id}`).

#### Scenario: Accounts render as cards with their balances

- **WHEN** an authenticated visitor with two accounts opens `/home`
- **THEN** both accounts render as cards, each showing its title,
  financial institute, and current live balance

#### Scenario: Activating a card opens the account's details

- **WHEN** an authenticated visitor activates an account card on `/home`
- **THEN** the client navigates to `/accounts/{id}` for that account

#### Scenario: Missing financial institute is handled gracefully

- **WHEN** an authenticated visitor opens `/home` and one of their accounts
  has no `financial_institute` set
- **THEN** that account's card renders without an institute line, rather
  than showing an empty or broken value

### Requirement: Home dashboard balances are colored by sign

Each card's balance on `/home` SHALL be rendered in a color reflecting
sign, matching the rule already applied on `/accounts`: red when negative,
the default/neutral text color when exactly zero, and green when positive.

#### Scenario: Negative balance is red

- **WHEN** `/home` renders a card whose account balance is negative
- **THEN** the balance is shown in red

#### Scenario: Positive balance is green

- **WHEN** `/home` renders a card whose account balance is positive
- **THEN** the balance is shown in green

### Requirement: Each card offers a button to add a new entry for that account

Each account card on `/home` SHALL include a button that navigates to
`/entries/new?account_id={id}`, preselecting that account for a new entry,
mirroring the equivalent per-row action already on `/accounts`.

#### Scenario: Adding an entry from a card

- **WHEN** an authenticated visitor activates the add-entry button on one
  of their account cards
- **THEN** the client navigates to `/entries/new?account_id={id}` for that
  account, with the account preselected

### Requirement: Home dashboard empty state

When an authenticated visitor has no accounts, `/home` SHALL show
explanatory empty-state text with a link to `/accounts/new`, instead of an
empty card grid.

#### Scenario: Empty state for a visitor with no accounts

- **WHEN** an authenticated visitor with no accounts opens `/home`
- **THEN** the page shows text explaining there are none, with a link to
  `/accounts/new`

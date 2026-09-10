## MODIFIED Requirements

### Requirement: Accounts overview lists every account with its balance and status

`/accounts` SHALL, on mount, fetch and display every account the visitor
owns or has any permission on (`GET /api/accounts`), each row showing at
least its title, type, currency, live balance (`GET /api/accounts/{id}/
balance`, formatted at the visitor's `displayed_decimal_places`), and a
status indicator reflecting whether it is disabled and/or closed. A row for
an account the visitor does not really own SHALL additionally show a shared
indicator and the real owner's name, per "Account surfaces render a shared
badge and the real owner's name" below. When the visitor has no accounts at
all (owned or shared), the page SHALL show explanatory empty-state text
with a way to create one instead of an empty list.

#### Scenario: Accounts and balances are listed

- **WHEN** an authenticated visitor with two accounts opens `/accounts`
- **THEN** both accounts are shown, each with its current live balance

#### Scenario: A shared account appears in the overview

- **WHEN** an authenticated visitor has a permission (any tier) on an
  account they do not really own
- **THEN** that account appears in `/accounts` alongside their own,
  showing the shared indicator and the real owner's name

#### Scenario: Disabled account is visually distinguished

- **WHEN** one of the visitor's accounts is disabled
- **THEN** its row indicates the disabled state

#### Scenario: Empty state

- **WHEN** an authenticated visitor with no owned or shared accounts opens
  `/accounts`
- **THEN** the page shows text explaining there are none, with a way to
  create one

### Requirement: Account details page shows account fields and recent entries

`/accounts/{id}` SHALL fetch and display the account's full details
(`GET /api/accounts/{id}`) and its most recent entries
(`GET /api/entries` filtered to that account, sorted by booking timestamp
descending, limited to a small fixed page), each linking to that entry's
edit page. The page SHALL offer a link to `/entries?account_id={id}` for
the account's complete, filterable entry list. Each recent-entries row
SHALL show its creator's name (per `web-client-entries`) whenever it
differs from the visitor.

#### Scenario: Recent entries link to the full filtered list

- **WHEN** an authenticated visitor opens an account's details page
- **THEN** they see its most recent entries and a link that navigates to
  `/entries?account_id={id}`

#### Scenario: A caller with no permission is not found

- **WHEN** an authenticated visitor navigates to `/accounts/{id}` for an
  account they have no permission on — neither real ownership nor a share
- **THEN** the page reflects the backend's `404` (not found), not the
  account's details

#### Scenario: A shared account's detail page shows who logged an entry

- **WHEN** an authenticated visitor with permission on a shared account
  views its recent entries and one was created by a different user
- **THEN** that entry's row shows the creator's name

## ADDED Requirements

### Requirement: Account surfaces render a shared badge and the real owner's name

Wherever an account is shown by name to a visitor who is not its real
owner (the accounts overview, the account detail header, and any other
`AccountLabel` call site), it SHALL additionally render a shared indicator
and the real owner's display name, immediately alongside the account's own
icon/badge and title. An account the visitor really owns SHALL render
exactly as before this capability, with no shared indicator, even when it
also carries shares out to other users.

#### Scenario: A shared account shows its real owner

- **WHEN** a visitor who is not an account's real owner views it on the
  accounts overview or its detail page
- **THEN** a shared indicator and the real owner's name render next to the
  account's title

#### Scenario: An owned account never shows a shared badge to its owner

- **WHEN** the real owner of an account views it, even if they have shared
  it with other users
- **THEN** no shared indicator renders for them

### Requirement: A Share button opens the account's sharing page

The accounts overview's rows and the account detail page SHALL each offer a
Share action, visible whenever the visitor holds at least `view` permission
on the account, navigating to `/accounts/{id}/sharing` (see
`web-client-account-sharing`).

#### Scenario: Opening the sharing page from the accounts overview

- **WHEN** an authenticated visitor activates Share on an account row in
  `/accounts`
- **THEN** the client navigates to `/accounts/{id}/sharing`

#### Scenario: Opening the sharing page from the account detail page

- **WHEN** an authenticated visitor activates Share on an account's detail
  page
- **THEN** the client navigates to `/accounts/{id}/sharing`

### Requirement: Account management affordances are gated by the visitor's permission tier

`/accounts/{id}/edit`, and the account detail page's link to it, SHALL be
offered only to a visitor with `owner`-tier permission on the account (the
real owner or a shared owner) — every other tier's detail page SHALL omit
the edit link, the disable/enable action, and the delete action entirely. A
visitor with a lower tier who navigates directly to `/accounts/{id}/edit`
SHALL be redirected to the account's detail page. Within the edit form, the
`type_id` field SHALL render read-only whenever the visitor is a shared
owner rather than the real owner, per `accounts`' `type_id` restriction;
every other field SHALL remain editable.

#### Scenario: A view or append visitor sees no edit affordance

- **WHEN** a visitor with `view`, `append`, or `entry_admin` permission
  opens an account's detail page
- **THEN** no edit link, disable/enable action, or delete action is shown

#### Scenario: Direct navigation to edit is redirected for a non-owner tier

- **WHEN** a visitor without `owner`-tier permission navigates directly to
  `/accounts/{id}/edit`
- **THEN** the client redirects them to the account's detail page

#### Scenario: A shared owner's edit form locks the type field

- **WHEN** a shared `owner`-tier visitor opens `/accounts/{id}/edit`
- **THEN** every field is editable except `type_id`, which renders
  read-only

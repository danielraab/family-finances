## MODIFIED Requirements

### Requirement: Account details page shows account fields and recent entries

`/accounts/{id}` SHALL fetch and display the account's full details
(`GET /api/accounts/{id}`) and its most recent entries
(`GET /api/entries` filtered to that account, sorted by booking timestamp
descending, limited to a small fixed page). Activating a recent entry's
title SHALL open that entry's read-only summary modal (per
`web-client-entries`) rather than navigating to its edit page; the
summary's own Edit action is the route to the edit page. The page SHALL
offer a link to `/entries?account_id={id}` for the account's complete,
filterable entry list. Each recent-entries row SHALL show its creator's
name (per `web-client-entries`) whenever it differs from the visitor.

#### Scenario: Recent entries link to the full filtered list

- **WHEN** an authenticated visitor opens an account's details page
- **THEN** they see its most recent entries and a link that navigates to
  `/entries?account_id={id}`

#### Scenario: A recent entry's title opens the entry summary

- **WHEN** an authenticated visitor activates a recent entry's title
- **THEN** that entry's read-only summary modal opens and the browser
  stays on the account details page

#### Scenario: A caller with no permission is not found

- **WHEN** an authenticated visitor navigates to `/accounts/{id}` for an
  account they have no permission on — neither real ownership nor a share
- **THEN** the page reflects the backend's `404` (not found), not the
  account's details

#### Scenario: A shared account's detail page shows who logged an entry

- **WHEN** an authenticated visitor with permission on a shared account
  views its recent entries and one was created by a different user
- **THEN** that entry's row shows the creator's name

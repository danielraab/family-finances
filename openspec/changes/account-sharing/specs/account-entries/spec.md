## RENAMED Requirements

- FROM: `### Requirement: An entry belongs to exactly one account and one owner`
- TO: `### Requirement: An entry belongs to exactly one account and records who created it`

## MODIFIED Requirements

### Requirement: An entry belongs to exactly one account and records who created it

Every entry SHALL carry a required `account_id`, and a required, immutable
`created_by` set to the authenticated caller at creation — the user who
logged it, which is not necessarily the account's real owner once
`account-sharing` is in effect. Reading an entry (directly, in a listing, or
in balance/summary computation) SHALL be permitted for any caller who holds
at least `view` permission on the entry's parent account, per
`account-sharing`; an entry on an account the caller has no permission on,
or whose parent account is soft-deleted, SHALL behave as if it does not
exist (`404`).

#### Scenario: Creating an entry against an account the caller has append+ permission on

- **WHEN** an authenticated user with at least `append` permission on an
  account (real ownership, or a share) calls `POST /api/entries` with that
  `account_id`
- **THEN** the response is `201` with the created entry, `created_by` set
  to the caller

#### Scenario: Creating an entry against an account with only view permission is rejected

- **WHEN** a user with only `view` permission on an account calls
  `POST /api/entries` with that `account_id`
- **THEN** the request is rejected (`422`, `account_id` not usable by the
  caller) and no entry is created

#### Scenario: A shared user's entry is created_by them, not the account's real owner

- **WHEN** a user with `append` permission (via a share, not real
  ownership) creates an entry on the account
- **THEN** the entry's `created_by` is that user, not the account's real
  owner

#### Scenario: A view-tier user reads entries they did not create

- **WHEN** a user with `view` permission on a shared account calls
  `GET /api/entries?account_id={id}` and some matching entries were
  created by a different user
- **THEN** those entries are included in the response

#### Scenario: Entry on an account with no permission is not accessible

- **WHEN** an authenticated user with no permission on an account calls
  `GET /api/entries?account_id={id}`
- **THEN** the response is `200` with an empty `items` list

#### Scenario: Entry on a soft-deleted account is not accessible

- **WHEN** an account has been soft-deleted and a user who previously had
  permission on it calls `GET /api/entries?account_id={id}`
- **THEN** the response is `200` with an empty `items` list — the account's
  entries are no longer reachable through it

### Requirement: Editing or deleting an entry is gated by permission tier and, for append, by who created it

`PATCH /api/entries/{id}` and `DELETE /api/entries/{id}` SHALL require the
caller to hold at least `append` permission on the entry's parent account.
A caller whose permission is exactly `append` (not `entry_admin` or
`owner`) SHALL be permitted to edit or delete only an entry whose
`created_by` matches them; the same request against an entry created by a
different user SHALL be rejected (`403`). A caller with `entry_admin` or
`owner` permission SHALL be permitted to edit or delete any entry on the
account, regardless of who created it.

#### Scenario: append can edit their own entry

- **WHEN** a user with `append` permission calls `PATCH /api/entries/{id}`
  on an entry they created
- **THEN** the update succeeds

#### Scenario: append cannot edit another user's entry

- **WHEN** a user with `append` permission calls `PATCH /api/entries/{id}`
  or `DELETE /api/entries/{id}` on an entry created by a different user on
  the same account
- **THEN** the request is rejected (`403`) and the entry is unchanged

#### Scenario: entry_admin can edit any entry on the account

- **WHEN** a user with `entry_admin` permission calls
  `PATCH /api/entries/{id}` or `DELETE /api/entries/{id}` on an entry
  created by a different user
- **THEN** the request succeeds

#### Scenario: view cannot edit or delete at all

- **WHEN** a user with only `view` permission calls
  `PATCH /api/entries/{id}` or `DELETE /api/entries/{id}` on any entry on
  the account
- **THEN** the request is rejected (`403`) and the entry is unchanged

### Requirement: A revoked or departed user loses all access to entries they created, immediately

Once a user's permission on an account is revoked or they leave it (per
`account-sharing`), every entry they previously created on that account
SHALL behave as not found (`404`) for them, the same as every other entry
on that account — not merely uneditable. The entries themselves SHALL be
unaffected for every user who retains a permission on the account: still
listed, still attributed to the departed user via `created_by`, still
editable per the remaining users' own tiers.

#### Scenario: A revoked user cannot read their own past entries

- **WHEN** a user's share on an account is revoked, and they had
  previously created entries on it
- **THEN** `GET /api/entries/{id}` for any of those entries now returns
  `404` for that user

#### Scenario: Remaining users keep full access to the departed user's entries

- **WHEN** a user's share on an account is revoked or they leave it
- **THEN** every remaining permission holder still sees that user's
  entries, still attributed to them, and can act on them per their own
  tier

### Requirement: An entry lists filters/sum/flow-summary/balance-series default to every account the caller has any permission on

`GET /api/entries`, `GET /api/entries/summary`, `GET /api/entries/flow-
summary`, and `GET /api/entries/balance-series`'s `account_id` filter,
when omitted, SHALL default to every non-deleted account the caller has
any permission on (real ownership, or a share of any tier) — not only
accounts they really own. An explicitly supplied `account_id` SHALL be
honored only when it names an account the caller has at least `view`
permission on; one the caller has no permission on SHALL be treated as
matching nothing, the same as an unknown id.

#### Scenario: Omitting account_id includes shared accounts

- **WHEN** a user who owns one account and has a share on another calls
  `GET /api/entries` with no `account_id`
- **THEN** entries from both accounts are included

#### Scenario: Filtering by a shared account the caller can view

- **WHEN** a user with `view` permission on an account calls
  `GET /api/entries?account_id={id}` for that account
- **THEN** only entries on that account are returned

#### Scenario: Filtering by an account the caller has no permission on returns nothing

- **WHEN** a user calls `GET /api/entries?account_id={id}` for an account
  they have no permission on
- **THEN** the response is `200` with an empty `items` list

### Requirement: An entry response carries its creator's identity

Every response carrying an `Entry` SHALL include `created_by` (the
creating user's id) and `created_by_name` (that user's display name, or
email when no display name is set), resolved server-side so the client can
show who logged an entry without a separate lookup.

#### Scenario: An entry created by someone else shows who created it

- **WHEN** a user with permission on a shared account fetches an entry
  created by a different user
- **THEN** the response includes that user's `created_by` id and
  `created_by_name`

#### Scenario: An entry the caller created shows their own identity the same way

- **WHEN** a user fetches an entry they created themselves
- **THEN** `created_by` and `created_by_name` identify the caller,
  consistent with every other entry's shape

## ADDED Requirements

### Requirement: An entry's title opens a read-only summary modal

Wherever an entry is listed by title — the ledger (`/entries`), an
account's recent-entries list, the `/reports` results table, and the
dashboard's `entry_list` card — activating that title SHALL open a
read-only summary modal for that entry rather than navigating. The
summary SHALL show the entry's booking timestamp, account, title,
description, category, tags, counterparty, location, signed amount, and
running balance, each omitted when the entry does not carry it, plus the
creator's name whenever `created_by` differs from the visitor. For a
`self_transfer` entry it SHALL additionally name the counterpart account.
Nothing in the summary SHALL be editable.

The summary SHALL render from the entry data the listing already holds,
issuing no further request when opened.

#### Scenario: Activating an entry title opens the summary

- **WHEN** an authenticated visitor activates an entry's title in the
  ledger
- **THEN** a read-only summary modal for that entry opens, and the browser
  does not navigate away from the ledger

#### Scenario: The summary opens without a request

- **WHEN** the summary modal opens for an entry already present in the
  listing
- **THEN** no additional entry request is issued

#### Scenario: A self-transfer summary names both accounts

- **WHEN** the summary opens for an entry whose `kind` is `self_transfer`
- **THEN** it names both the entry's account and its counterpart account

#### Scenario: Absent fields are omitted

- **WHEN** the summary opens for an entry with no description, no
  counterparty, and no location
- **THEN** those rows are omitted rather than shown empty

### Requirement: The entry summary's Edit action follows the entry edit permission rule

The entry summary modal SHALL offer an Edit action navigating to
`/entries/{id}/edit`, shown only to a visitor permitted to edit that
entry under the same rule the edit page applies: `entry_admin` or `owner`
permission on the entry's account, or `append` permission with
`created_by` matching the visitor; and for a `self_transfer`, at least
`append` on *both* accounts with no `created_by` exemption. That rule
SHALL be evaluated by one shared implementation used by both the summary
and the edit page.

#### Scenario: A permitted visitor sees Edit

- **WHEN** a visitor with `owner` permission on the entry's account opens
  the summary
- **THEN** the summary offers an Edit action to that entry's edit page

#### Scenario: A view-tier visitor sees no Edit action

- **WHEN** a visitor with only `view` permission on a shared account opens
  the summary for one of its entries
- **THEN** the summary shows the entry's details with no Edit action

#### Scenario: An append-tier visitor sees no Edit action on another user's entry

- **WHEN** a visitor with `append` permission opens the summary for an
  entry on the same account that a different user created
- **THEN** the summary shows no Edit action

#### Scenario: A self-transfer needs append on both accounts

- **WHEN** a visitor holds `append` on a self-transfer's own account but
  only `view` on its counterpart account
- **THEN** the summary shows no Edit action

## MODIFIED Requirements

### Requirement: A linked entry shows a badge to its recurring transaction in the ledger

`/entries` SHALL show a small icon/badge on any entry whose
`recurring_transaction_id` is set, distinct from the row's other content,
that opens that recurring transaction's read-only summary modal when
activated. An entry with no `recurring_transaction_id` SHALL show no such
badge. Activating the badge SHALL NOT also trigger the surrounding row's
own click behaviour.

#### Scenario: Linked entry shows the badge

- **WHEN** the ledger lists an entry with a non-null
  `recurring_transaction_id`
- **THEN** its row shows the recurring-transaction badge

#### Scenario: Badge opens the recurring transaction's summary

- **WHEN** the visitor activates a linked entry's badge
- **THEN** that recurring transaction's read-only summary modal opens and
  the browser does not navigate

#### Scenario: Activating the badge does not trigger the row

- **WHEN** the visitor activates the badge on a row that itself responds
  to clicks
- **THEN** only the recurring summary opens; the row's own behaviour does
  not fire

#### Scenario: Unlinked entry shows no badge

- **WHEN** the ledger lists an entry with a null `recurring_transaction_id`
- **THEN** no recurring-transaction badge is shown on its row

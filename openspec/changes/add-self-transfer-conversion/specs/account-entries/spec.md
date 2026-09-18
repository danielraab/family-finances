## ADDED Requirements

### Requirement: An existing transaction can be converted into a self-transfer

`POST /api/entries/{id}/self-transfer` SHALL convert an existing `kind:
transaction` entry into a new `kind: self_transfer` entry, atomically: the
original entry SHALL be soft-deleted and a new entry SHALL be created,
both within a single database transaction, with the balance-adjustment
chain recomputed for every account touched (the original entry's account,
and the newly chosen counterparty account). The response SHALL be `201`
with the newly created `self_transfer` entry; the original entry's `id`
SHALL NOT be reused.

The request SHALL specify the counterparty account and which role the
original entry's account plays in the resulting transfer — sender
(`account_id`, unchanged from the original entry) or receiver
(`to_account_id`) — never inferred from the original entry's `amount`
sign. `category_id` SHALL carry over unchanged from the original entry.
`recurring_transaction_id` SHALL carry over unchanged when the original
account keeps the sender role, and SHALL be dropped (null on the new
entry) when the original account becomes the receiver — a recurring
transaction link is only ever valid against `account_id`, never
`to_account_id`. `counterparty` and `location` SHALL NOT carry over — a
`self_transfer` rejects both, the same as it does on ordinary creation.

Authorization SHALL be `404` when the caller holds no permission at all on
the original entry's account (the same "behaves as not found" rule
`GET`/`PATCH /api/entries/{id}` already apply), and otherwise SHALL apply
exactly the same rule `POST /api/entries` with `kind: self_transfer`
already applies to its `account_id`/`to_account_id` pair: `append`+
permission and non-disabled state required on both the original entry's
account and the newly chosen counterparty account (`400` for insufficient
permission, `422` for a disabled account), and both accounts SHALL share
the same `currency` (`400` otherwise).

Converting an entry whose `kind` is not `transaction` (already
`balance_adjustment` or `self_transfer`) SHALL be rejected (`400`) and
SHALL make no change.

#### Scenario: Converting a transaction with append+ on both accounts

- **WHEN** a caller with `append`+ permission on both a `kind: transaction`
  entry's own account and a chosen, same-currency counterparty account
  calls `POST /api/entries/{id}/self-transfer` naming that counterparty
  and a sender/receiver role
- **THEN** the response is `201` with a new `kind: self_transfer` entry, a
  different `id` from the original, and the original entry is
  soft-deleted (no longer returned by `GET /api/entries/{id}`)

#### Scenario: The new entry's balance-adjustment chain reflects both accounts

- **WHEN** a conversion succeeds
- **THEN** both the original entry's account and the newly chosen
  counterparty account's live balances reflect the new self-transfer entry
  exactly as if it had been created directly via `POST /api/entries`

#### Scenario: Converting keeps the category

- **WHEN** a `kind: transaction` entry with a non-null `category_id` is
  converted
- **THEN** the resulting `self_transfer` entry has the same `category_id`

#### Scenario: Converting keeps the recurring-transaction link when the original account stays the sender

- **WHEN** a `kind: transaction` entry with a non-null
  `recurring_transaction_id` is converted, choosing the original account
  as the sender (`account_id`)
- **THEN** the resulting `self_transfer` entry has the same
  `recurring_transaction_id`

#### Scenario: Converting drops the recurring-transaction link when the original account becomes the receiver

- **WHEN** a `kind: transaction` entry with a non-null
  `recurring_transaction_id` is converted, choosing the original account
  as the receiver (`to_account_id`)
- **THEN** the resulting `self_transfer` entry has a null
  `recurring_transaction_id`

#### Scenario: Converting drops counterparty and location

- **WHEN** a `kind: transaction` entry with a non-empty `counterparty` and
  `location` is converted
- **THEN** the resulting `self_transfer` entry has neither field set

#### Scenario: Converting without permission on the chosen counterparty account is rejected

- **WHEN** a caller with `append`+ permission on the entry's own account
  but no permission at all on the chosen counterparty account calls
  `POST /api/entries/{id}/self-transfer`
- **THEN** the request is rejected (`400`), and neither the original entry
  nor any new entry is changed or created

#### Scenario: Converting to a disabled counterparty account is rejected

- **WHEN** a caller calls `POST /api/entries/{id}/self-transfer` naming a
  disabled counterparty account
- **THEN** the request is rejected (`422`), and neither the original entry
  nor any new entry is changed or created

#### Scenario: Converting across different currencies is rejected

- **WHEN** a caller calls `POST /api/entries/{id}/self-transfer` naming a
  counterparty account whose `currency` differs from the entry's own
  account
- **THEN** the request is rejected (`400`), and neither the original entry
  nor any new entry is changed or created

#### Scenario: Converting an entry the caller cannot see at all is not found

- **WHEN** a caller with no permission at all on a `kind: transaction`
  entry's account calls `POST /api/entries/{id}/self-transfer`
- **THEN** the response is `404`

#### Scenario: Converting an already-converted or non-transaction entry is rejected

- **WHEN** a caller calls `POST /api/entries/{id}/self-transfer` naming an
  entry whose `kind` is `balance_adjustment` or already `self_transfer`
- **THEN** the request is rejected (`400`) and no change is made

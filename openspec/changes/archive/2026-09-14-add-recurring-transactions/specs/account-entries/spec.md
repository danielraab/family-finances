## ADDED Requirements

### Requirement: An entry can be linked to a recurring transaction on the same account

Every entry MAY reference at most one recurring transaction via an optional
`recurring_transaction_id`, settable on `POST /api/entries` and
`PATCH /api/entries/{id}`. Setting it SHALL require the referenced recurring
transaction to belong to the same `account_id` as the entry and to not be
soft-deleted; referencing a recurring transaction on a different account, or
a nonexistent/deleted one, SHALL be rejected. `PATCH /api/entries/{id}` MAY
also clear an existing link by supplying `recurring_transaction_id: null`.
Linking or unlinking an entry follows the same edit-permission rule as any
other entry field (see "Editing or deleting an entry is gated by permission
tier and, for append, by who created it").

#### Scenario: Creating an entry linked to a recurring transaction on the same account

- **WHEN** `POST /api/entries` is called with `account_id` and a
  `recurring_transaction_id` naming a non-deleted recurring transaction on
  that same account
- **THEN** the response is `201` and the entry's `recurring_transaction_id`
  is set

#### Scenario: Linking to a recurring transaction on a different account is rejected

- **WHEN** `POST /api/entries` or `PATCH /api/entries/{id}` supplies a
  `recurring_transaction_id` naming a recurring transaction whose
  `account_id` differs from the entry's own `account_id`
- **THEN** the request is rejected (`400`) and the entry is not linked

#### Scenario: Linking an existing entry via update

- **WHEN** an authenticated user with edit permission on an entry calls
  `PATCH /api/entries/{id}` with a `recurring_transaction_id` naming a valid
  recurring transaction on the entry's account
- **THEN** the response is `200` and the entry is now linked

#### Scenario: Clearing a link

- **WHEN** `PATCH /api/entries/{id}` is called with
  `recurring_transaction_id: null` on a currently-linked entry
- **THEN** the response is `200` and the entry's `recurring_transaction_id`
  is now null

#### Scenario: Every entry response carries its link, when present

- **WHEN** a client fetches an entry (directly or in a listing) that has a
  `recurring_transaction_id` set
- **THEN** the response includes that id, so the client can render a link
  to the recurring transaction without a separate lookup

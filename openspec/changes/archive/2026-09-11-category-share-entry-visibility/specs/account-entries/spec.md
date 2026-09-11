## ADDED Requirements

### Requirement: An entry response includes its account's currency, resolved server-side

Every `Entry` returned by `GET /api/entries`, `GET /api/entries/{id}`, and
any other endpoint that returns an `Entry` object SHALL include
`account_currency`, the currency of that entry's `account_id`, resolved
server-side regardless of whether the caller currently holds any
account-level access to that account — the same resolve-it-server-side,
don't-make-the-client-re-derive-it precedent `created_by_name` already
establishes. This lets a client always render `amount` with the correct
currency without a separate account fetch, including for an entry the
caller can see only because they hold permission on its category (see
`category-sharing`, `entry-categories`) and have no account-level access
to it at all — `account_currency` is the one piece of that account's data
carried onto the entry; every other account detail (name, balance, other
entries) remains unexposed.

#### Scenario: An entry's amount can be rendered without a separate account fetch

- **WHEN** a client receives an entry from `GET /api/entries` or
  `GET /api/entries/{id}`
- **THEN** the response includes `account_currency`, correct for that
  entry's account, whether or not the caller also has account-level
  access to that account

#### Scenario: A category-permitted, account-inaccessible entry still carries its real currency

- **WHEN** a caller filters entries by a category they hold permission on
  (real ownership or a share), surfacing an entry on an account they have
  no account-level access to
- **THEN** that entry's `account_currency` is the real currency of its
  account — distinct from the account's name, balance, or any other
  detail, which stay unexposed

# Spec Delta

## MODIFIED Requirements

### Requirement: The entry ledger's filter, search, and sort state lives in the URL

`/entries` SHALL represent its current account, category, tag, kind, date range, amount-magnitude range, and free-text search filters, and its sort field and direction, as typed URL search parameters, readable and writable through TanStack Router's search-param APIs. The amount range SHALL use optional non-negative `amount_from` and `amount_to` stored-scale integer values and SHALL match entries regardless of whether their displayed signed amount is positive or negative. Reloading a URL with search parameters SHALL reproduce the same filtered/sorted view. Arriving at `/entries` with an `account_id` parameter already set (for example, via the link from an account's details page) SHALL apply that filter immediately on load. Every change to this state SHALL also be written to browser-local storage, keyed per visitor's browser (not synced to the account or the backend) — see "Returning to the entry ledger restores the last-applied filters" for when that stored state is read back.

#### Scenario: A filtered view survives a reload

- **WHEN** an authenticated visitor applies a category filter and a sort order, then reloads the page
- **THEN** the same category filter and sort order are applied after the reload

#### Scenario: An amount range survives a reload

- **WHEN** an authenticated visitor applies an amount start and/or end bound, then reloads the page
- **THEN** the same non-negative amount bounds are present in the URL and applied to the ledger regardless of result signs

#### Scenario: A negative amount input becomes a positive magnitude

- **WHEN** an authenticated visitor types or pastes a negative value into either amount-range input
- **THEN** that control retains the value as a positive magnitude and writes the corresponding non-negative stored-scale bound to the URL

#### Scenario: Arriving with a preset account filter

- **WHEN** an authenticated visitor follows a link to `/entries?account_id={id}`
- **THEN** the entry list is immediately filtered to that account

#### Scenario: Changing a filter updates the URL

- **WHEN** an authenticated visitor changes the search text or a filter control
- **THEN** the corresponding URL search parameter changes to match, and the same state is written to browser-local storage

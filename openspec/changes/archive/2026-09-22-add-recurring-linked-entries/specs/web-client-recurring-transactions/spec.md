## ADDED Requirements

### Requirement: A recurring transaction's edit page lists its linked entries, cursor-paginated

`/recurring/{id}/edit` SHALL show a "Linked transactions" section listing
the entries whose `recurring_transaction_id` is that template's id
(`GET /api/entries?recurring_transaction_id={id}`), newest booking
timestamp first, each row showing at least the entry's booking date,
title, category, and signed amount, and linking to that entry's own
`/entries/{entryId}/edit` page. When the template's `linked_entry_count`
is `0`, the section SHALL show an empty state without issuing the list
request. When more results remain (`next_cursor` non-null), the section
SHALL offer a "Load more" action that appends the next page; it SHALL NOT
fetch further pages automatically on scroll.

#### Scenario: No linked entries shows an empty state, no request

- **WHEN** the visitor opens the edit page for a recurring transaction
  whose `linked_entry_count` is `0`
- **THEN** an empty state is shown and `GET /api/entries` is not called
  with `recurring_transaction_id` for it

#### Scenario: Linked entries list, newest first

- **WHEN** the visitor opens the edit page for a recurring transaction
  with linked entries
- **THEN** the section lists them ordered by booking timestamp, newest
  first

#### Scenario: A row links to its entry's edit page

- **WHEN** the visitor activates a row in the linked-transactions list
- **THEN** the browser navigates to that entry's `/entries/{entryId}/edit`
  page

#### Scenario: Load more appends the next page

- **WHEN** the visitor activates "Load more" and a `next_cursor` was
  present
- **THEN** the next page's entries are appended to the visible list
  without replacing it, using that cursor as `after`

#### Scenario: No load-more action on the last page

- **WHEN** the loaded page's `next_cursor` is `null`
- **THEN** no "Load more" action is shown

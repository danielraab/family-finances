## ADDED Requirements

### Requirement: The entry ledger supports selecting loaded entries for a bulk action

The entry ledger (`/entries`) SHALL show a checkbox on every row, always
enabled regardless of the visitor's write permission on that entry, plus a
header checkbox that selects or deselects every entry currently loaded in
the list (the infinite-scroll window fetched so far — not every entry
matching the active filter). Selection SHALL persist as more entries load
via scrolling, and SHALL be cleared whenever a filter/search/sort change
resets the loaded list or the visitor explicitly clears it.

#### Scenario: Selecting individual rows

- **WHEN** an authenticated visitor checks two rows in the ledger
- **THEN** both entries are selected and a bulk-action toolbar appears
  showing a count of 2

#### Scenario: Header checkbox selects only loaded rows

- **WHEN** a visitor activates the header checkbox with 30 entries loaded
  and more available via scrolling
- **THEN** exactly the 30 loaded entries become selected; entries not yet
  scrolled into view are not

#### Scenario: Changing a filter clears the selection

- **WHEN** a visitor has entries selected and then changes any filter,
  search term, or sort
- **THEN** the selection is cleared along with the loaded list resetting

#### Scenario: A row the visitor cannot edit is still selectable

- **WHEN** a visitor with only `view` permission on a shared account, or
  `append` permission on an entry created by someone else, views that
  entry in the ledger
- **THEN** its row checkbox is enabled and selectable like any other row

### Requirement: Every bulk action requires an explicit confirmation before it runs

Selecting a bulk-action toolbar action SHALL open a dialog to choose the
action's value (a category, one or more tags, a recurring transaction, or
— for delete — a destructive-confirmation prompt with no value to choose).
No request SHALL be sent to the backend until the visitor confirms within
that dialog.

#### Scenario: Picking a value does not run the action

- **WHEN** a visitor opens the "Set category" dialog and selects a
  category, without pressing Apply
- **THEN** no request has been sent and no entry has been changed

#### Scenario: Confirming starts the run

- **WHEN** a visitor presses Apply (or, for delete, confirms the
  destructive prompt)
- **THEN** the bulk-run modal opens and begins applying the action to
  every selected entry

### Requirement: A bulk action applies sequentially to every selected entry and reports per-entry results

Confirming a bulk action SHALL run one request per selected entry,
sequentially, showing live progress. On completion it SHALL report how
many entries succeeded and SHALL list, individually, every entry that
failed with a short reason (e.g. forbidden, not found, invalid value). An
entry whose current state already matches the action's chosen value SHALL
be counted as succeeded without a request being sent for it. The
completion view SHALL offer a "Reload list" action, which re-fetches the
ledger under the current filters and clears the selection, and a "Close"
action, which dismisses the modal without refetching.

#### Scenario: A forbidden entry is reported, not silently skipped

- **WHEN** a bulk action's selection includes an entry the visitor lacks
  write permission on
- **THEN** the run modal completes with that entry listed as failed
  (forbidden), and every other selected entry the visitor could act on is
  still applied

#### Scenario: An already-matching entry counts as succeeded with no request sent

- **WHEN** a bulk "Set category" action is applied to a selection where
  one entry already has the chosen category
- **THEN** that entry is counted among the succeeded entries and no update
  request is sent for it

#### Scenario: Reload list refreshes the ledger

- **WHEN** a visitor presses "Reload list" on the run modal's completion
  view
- **THEN** the ledger re-fetches its current filtered view from the start
  and the selection is cleared

#### Scenario: Close leaves the ledger as-is

- **WHEN** a visitor presses "Close" on the run modal's completion view
- **THEN** the modal dismisses and the ledger's currently loaded rows are
  left unchanged until the visitor reloads or navigates

### Requirement: Bulk "Set category" applies one category to every selected entry

The "Set category" action SHALL offer the same category choices as the
entry edit form (the visitor's own non-disabled categories plus any
shared to them at `append`+ permission), including an option to clear the
category. Confirming SHALL update each selected entry's `category_id` to
the chosen value.

#### Scenario: Setting a category across a mixed selection

- **WHEN** a visitor selects entries with different current categories and
  applies "Set category" with a chosen category
- **THEN** every selected entry the visitor can edit ends up with that
  category

#### Scenario: Clearing the category is rejected for a transaction entry

- **WHEN** a visitor applies "Set category" with "No category" chosen to a
  selection that includes a `transaction`-kind entry
- **THEN** that entry appears in the run modal's failure list (a
  transaction requires a category), while any `balance_adjustment`-kind
  entries in the selection are cleared successfully

### Requirement: Bulk "Add tags" and "Set tags" reuse the entry form's tag input

"Add tags" and "Set tags" SHALL each open the same tag-input control the
entry create/edit form uses, offering the visitor's existing non-disabled,
non-view-tier tags as suggestions and allowing a new tag name to be typed.
"Add tags" SHALL add the chosen tags to each selected entry's existing tags
without removing any tag already present. "Set tags" SHALL replace each
selected entry's tag set with exactly the chosen tags.

#### Scenario: Adding tags preserves existing tags

- **WHEN** a visitor applies "Add tags" with tag "Reviewed" chosen to an
  entry that already carries tag "Groceries"
- **THEN** that entry ends up carrying both "Groceries" and "Reviewed"

#### Scenario: Setting tags replaces the existing set

- **WHEN** a visitor applies "Set tags" with only tag "Reviewed" chosen to
  an entry that currently carries "Groceries" and "Business"
- **THEN** that entry ends up carrying only "Reviewed"

### Requirement: Bulk "Remove tags" only offers tags present on the selected entries

The "Remove tags" action SHALL offer, as choosable tags, only those
currently present on at least one selected entry — not the visitor's full
tag list. Confirming SHALL remove each chosen tag from every selected
entry that carries it, leaving entries that don't carry a chosen tag
unaffected by it.

#### Scenario: Only in-use tags are offered

- **WHEN** a visitor opens "Remove tags" for a selection where the only
  tags present across those entries are "Groceries" and "Business"
- **THEN** the tag picker offers only "Groceries" and "Business", even if
  the visitor has other tags defined elsewhere

#### Scenario: Removing a tag only affects entries that carry it

- **WHEN** a visitor applies "Remove tags" with "Groceries" chosen to a
  selection where only some entries carry that tag
- **THEN** entries carrying "Groceries" have it removed, and entries that
  never carried it are counted as succeeded with no change

### Requirement: Bulk "Link to recurring transaction" is restricted to a single-account selection

The "Link to recurring transaction" action SHALL be disabled, with an
explanatory hint, whenever the current selection's entries do not all
share the same account. When enabled, it SHALL offer the recurring
transactions defined on that one shared account, plus an option to unlink.
Confirming SHALL set (or clear) each selected entry's linked recurring
transaction accordingly.

#### Scenario: Action is disabled across accounts

- **WHEN** the current selection includes entries from two different
  accounts
- **THEN** "Link to recurring transaction" appears disabled with a hint
  explaining why

#### Scenario: Action is enabled and scoped for a single-account selection

- **WHEN** every entry in the current selection belongs to the same
  account
- **THEN** "Link to recurring transaction" is enabled and its picker
  offers only that account's recurring transactions

#### Scenario: Unlinking clears the link on every selected entry

- **WHEN** a visitor applies "Link to recurring transaction" with "Not
  linked" chosen
- **THEN** every selected entry's recurring-transaction link is cleared

### Requirement: Bulk "Delete" soft-deletes every selected entry after a destructive confirmation

The "Delete" action SHALL show a destructive-confirmation prompt (no value
to choose) before running. Confirming SHALL soft-delete every selected
entry the visitor has permission to delete, one at a time, reporting the
same succeeded/failed breakdown as any other bulk action.

#### Scenario: Deleting a selection requires confirmation

- **WHEN** a visitor activates "Delete" on a bulk selection
- **THEN** a confirmation prompt appears and no entry is deleted until it
  is confirmed

#### Scenario: Deleted entries no longer appear after reloading

- **WHEN** a visitor confirms bulk delete and then presses "Reload list" on
  the run modal's completion view
- **THEN** the successfully deleted entries no longer appear in the ledger

## MODIFIED Requirements

### Requirement: Four permission tiers, each a strict superset of the one before it

An account MAY be shared with any number of other registered users, each at
exactly one of four permission tiers: `view`, `append`, `entry_admin`, or
`owner`. A caller's tier SHALL be the sole determinant of what they may do
on the account, and each tier SHALL grant every capability of the tiers
below it. `view` grants reading the account's entries and balance. `append`
additionally grants creating entries and editing/deleting only entries the
same user created. `entry_admin` additionally grants editing/deleting any
entry on the account (not only ones that user created). `owner` additionally
grants editing every one of the account's own metadata fields (including
`type`, which is now plain text on the account rather than a reference into
a per-real-owner lookup), disabling/enabling/soft-deleting the account, and
managing shares (inviting, changing a permission, revoking). A shared
`owner`-tier grant carries every one of these rights identically to the
account's real owner (`accounts.owner_id`), which this capability never
reassigns — the real owner is always a distinct, always-knowable identity
from any `owner`-tier share.

#### Scenario: view grants reading only

- **WHEN** a user with `view` permission on an account requests its entries
  or balance
- **THEN** the response succeeds; a request to create, edit, or delete an
  entry, or to edit the account, is rejected

#### Scenario: append grants creating and editing only what was created

- **WHEN** a user with `append` permission creates an entry, then attempts
  to edit an entry created by a different user on the same account
- **THEN** the create succeeds and the edit of the other user's entry is
  rejected

#### Scenario: entry_admin grants editing any entry

- **WHEN** a user with `entry_admin` permission edits or deletes an entry
  created by a different user on the same account
- **THEN** the request succeeds

#### Scenario: owner grants full account management

- **WHEN** a user with a shared `owner`-tier permission edits the account's
  title, edits the account's `type`, disables the account, or invites
  another user to it
- **THEN** each request succeeds, identically to the real owner performing
  it

## Why

Every account today has exactly one owner and is visible to no one else —
`openspec/specs/accounts/spec.md`'s "An account has exactly one owner and is
visible only to them" is explicit that sharing is out of scope. For a family
finances app that's the wrong default: a couple sharing a joint checking
account, or a parent letting a teenager log their own spending against a
shared account, currently requires giving away the whole account (there is no
such mechanism either) or keeping two disconnected copies of the truth.

This change adds account sharing — four permission tiers (view, append,
entry_admin, owner) a real account owner can grant to another registered
user by email. Category and tag sharing are explicitly deferred; each family
member keeps categorizing and tagging shared-account entries from their own
category/tag lists until a future change addresses that.

## What Changes

- New `account_shares` table: one row per (account, user, permission).
  Permission ∈ `view | append | entry_admin | owner`. The account's
  `owner_id` column keeps meaning "real owner" — a share at the `owner`
  tier grants the same rights as the real owner (including disabling and
  soft-deleting the account, and managing other shares) without ever
  becoming `owner_id`.
- **`entries.owner_id` is renamed to `created_by`.** It already meant "who
  logged this entry" in practice (only an owner could create one); sharing
  makes that literal — any permitted user can now log an entry, and
  `created_by` records who. This is the mechanism that lets an `append`-tier
  user edit/delete only their own entries while an `entry_admin`/`owner`-tier
  user edits/deletes any entry on the account, and it's why category/tag
  validation on an entry continues to check the *creator's* own lists
  unchanged.
- Every account/entry read and write path currently scoped to "owned by the
  caller" widens to "the caller holds a permission on the account, real
  ownership or a share" — `GET /api/accounts`, entry listing, `/summary`,
  `/flow-summary`, `/balance-series`, and entry create/update/delete, each
  gated to the tier the operation actually needs.
- New `/api/accounts/{id}/shares` endpoints: list (any tier, including
  `view`, sees the full list — everyone with access to an account can see
  who else has access), create/invite by email, change a permission, revoke
  a share (`owner` tier only for the latter three), and self-leave (any
  non-real-owner tier, for their own row).
- Sharing by email deliberately does **not** hide whether the address is
  registered from the (already-authenticated, already-permissioned) inviter:
  an unmatched email is reported back synchronously, with whether the
  instance currently allows sending an app invite, so the inviter can invite
  that person first and share with them afterward. A matched email gets a
  notification email naming the account, the granter, and the permission.
- Revoking (or leaving) a share removes that user's access to the account
  entirely — read, edit, and delete, including of entries they created
  themselves — while leaving those entries on the account, attributed to
  them, for everyone who still has access.
- Frontend: a Share button on account cards (home and the accounts overview)
  and the account detail page opens a new `/accounts/{id}/sharing` page —
  read-only below `owner` tier, full management at it. A shared badge plus
  the real owner's name renders next to an account's name everywhere an
  account is shown to a non-real-owner. Shared accounts appear in the
  accounts overview, the home dashboard (cards and the all-accounts chart),
  entry filters, and reports, acting like an owned account modulo the
  caller's permission tier. Any entry not created by the current viewer
  shows who did.

## Capabilities

### Added Capabilities

- `account-sharing`: the `account_shares` model, the four permission tiers
  and what each grants, the share-management endpoints, the email-invite
  flow (including the unregistered-email/app-invite nudge), and revocation/
  self-leave semantics.
- `web-client-account-sharing`: the `/accounts/{id}/sharing` page — the
  entry point (Share button), its read-only vs. management modes, the
  invite-by-email form (including the "not registered, send an invite?"
  affordance), and per-row permission/revoke/leave controls.

### Modified Capabilities

- `accounts`: visibility and lifecycle (disable/enable/soft-delete/update)
  extend from "owned by the caller" to "the caller holds a permission,
  real ownership or a share"; a shared `owner`-tier grant has the same
  rights as the real owner except reassigning `type_id`, which stays
  restricted to the real owner (account types are still a per-owner lookup,
  unextended by this change).
- `account-entries`: `owner_id` renamed to `created_by`; entry read scope
  widens to every account the caller has any permission on; entry
  create/update/delete are gated by tier (`append`+ to create, `append`
  limited to entries the caller created, `entry_admin`/`owner` to any entry
  on the account); an entry response carries its creator's identity.
- `web-client-accounts`: shared badge + real-owner name on account surfaces;
  a Share button; permission-gated edit/disable/delete affordances; the
  accounts overview includes shared accounts.
- `web-client-home`: shared badge on account cards; the cards grid and the
  all-accounts chart include shared accounts; the add-entry shortcut is
  gated to `append`+.
- `web-client-entries`: an entry not created by the current visitor shows
  its creator; edit/delete controls are gated by tier and, at `append`,
  by whether the visitor created that entry; the account picker (new entry,
  filters) includes shared accounts.
- `web-client-reports`: the account filter includes shared accounts.

## Impact

- **Code**:
  - `backend/internal/storage/postgres/migrations/0020_account_sharing.sql`
    — new `account_shares` table; `ALTER TABLE entries RENAME COLUMN
    owner_id TO created_by`.
  - `backend/internal/account/`: new `AccountShare` type + share
    CRUD/list/revoke/leave on `Store`/`Service`; `Owner` replaced by an
    `Access` lookup entry.Service can use for tier checks; `List`/
    `VisibleIDs` widen to owned-or-shared; new handler routes under
    `/api/accounts/{id}/shares`; a new narrow interface to
    `internal/auth` for the email lookup + invite-eligibility check, and to
    `internal/mailer` for the share-notification email.
  - `backend/internal/entry/`: `owner_id` → `created_by` throughout
    (`entry.go`, `service.go`, `store.go`, both storage backends);
    `AccountLookup` interface's `Owner` method replaced with an
    `Access`-returning one; every Store method scoped by owner instead
    scoped by the visible-account-id set already resolved by `Service`.
  - `frontend/src/routes/accounts.$accountId.sharing.tsx` (new),
    `frontend/src/components/AccountLabel.tsx` (shared badge + owner name),
    `AccountCard.tsx`, `accounts.index.tsx`, `accounts.$accountId.index.tsx`,
    `accounts.$accountId.edit.tsx`, `entries.index.tsx`,
    `entries.$entryId.edit.tsx`, `entries.new.tsx`, `reports.tsx`,
    `home.tsx`/`index.tsx` (whichever file backs `/home`) — permission-gated
    affordances and widened account fetches.
  - New i18n keys (`en.json` first, then `de.json`).
- **API contract**: `openapi/openapi.yaml` gains `AccountShare`, the
  `/api/accounts/{id}/shares` paths, `permission`/`owner_name`/`shared` on
  `Account`, `created_by`/`created_by_name` (replacing `owner_id`) on
  `Entry` — regenerate `backend/openapi.yaml` and
  `frontend/src/api/schema.d.ts` in the same change.
- **Spec**: deltas on `accounts`, `account-entries`, `web-client-accounts`,
  `web-client-home`, `web-client-entries`, `web-client-reports`; new specs
  for `account-sharing` and `web-client-account-sharing`.

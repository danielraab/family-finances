## Why

Categories today are strictly private (`categories.owner_id`), and
`internal/entry`'s category validation checks the *entry's creator*, not
the account owner — so on a shared account (per `account-sharing`), each
collaborator already categorizes using their own private category tree.
That means when Alice shares an account with Bob, Bob's entries carry
category ids from *his* tree, and Alice's ledger can't resolve a label for
them — they aren't in her `GET /api/categories` response at all. Family
members sharing an account end up with disconnected, inconsistent
categorization instead of one shared vocabulary.

This change adds category sharing — mirroring `account-sharing`'s shape,
but with two permission tiers instead of four, and no shareable "owner"
tier: a category's real owner (creator) is always the only one who can
edit its metadata, disable/enable, reorder, reparent, or delete it.
Sharing only ever grants visibility and/or usability of the category
itself, never editing rights over it, and it is completely orthogonal to
entry write permission (still governed solely by `account.Permission`,
unchanged by this capability).

## What Changes

- New `category_shares` table: one row per (category, user, permission).
  Permission ∈ `view | append`. `view` lets the recipient filter/see by
  the category (it resolves in entry-list/report category filters, and
  its name resolves wherever an entry using it is displayed) but it is
  never offered as a pickable option when categorizing an entry. `append`
  additionally makes it pickable on new/edited entries.
- **No cascade.** Sharing a category shares only that node — its children,
  if any, stay private to the real owner unless separately shared. A
  shared category is always presented as a flat, top-level category in
  the recipient's own category views, regardless of where it sits in the
  real owner's tree.
- `GET /api/categories` returns the union of the caller's own full tree
  (nested, exactly as today) and every category shared with them (flat,
  each carrying the new `permission`/`shared`/`owner_name` fields —
  mirroring `Account`'s fields of the same names exactly).
- `internal/category`'s `Usable` (the check `internal/entry` already calls
  when validating an entry's `category_id`) widens from "owned by the
  caller" to "owned by the caller, or shared with them at `append`, and
  not disabled." This is the one change that makes shared categories
  usable for entries with **zero changes to `internal/entry` itself** — it
  already calls `Usable(ctx, callerID, categoryID)` treating `callerID` as
  "the person doing the categorizing."
- New `/api/categories/{id}/shares` endpoints: list (any permission tier,
  including `view`), invite by email (real-owner only — there is no
  shareable owner tier to also permit this), change a permission, revoke,
  and self-leave — structurally identical to `account-sharing`'s endpoints
  of the same shapes.
- Sharing by email follows the exact same synchronous, revealing pattern
  as account sharing: a matched email creates/updates the share and sends
  a notification email; an unmatched one reports back `matched: false`
  plus whether the instance currently allows sending an application
  invite.
- Frontend: a Share action on a category the caller owns opens a new
  `/categories/{id}/sharing` page (read-only for a recipient, full
  management for the real owner). `/categories` gains a "Shared with me"
  section beneath the caller's own tree, listing every category shared
  with them flat, each showing the owner's name (mirroring `AccountLabel`'s
  shared badge). The entry form's category picker additionally offers
  every `append`-tier shared category as a flat, top-level option; the
  entry-list/report category filter additionally offers every `view`+
  shared category. Wherever a category is displayed by name, a shared
  badge + owner name appears whenever the category isn't the viewer's own
  — the same treatment `AccountLabel` already gives accounts.

## Capabilities

### Added Capabilities

- `category-sharing`: the `category_shares` model, the two permission
  tiers and what each grants, the share-management endpoints, the
  email-invite flow, and revocation/self-leave semantics.
- `web-client-category-sharing`: the `/categories/{id}/sharing` page and
  the "Shared with me" section on `/categories`.

### Modified Capabilities

- `entry-categories`: `Usable` widens to include `append`-tier shares;
  `entry_count` on a shared category (as seen by a non-owner viewer)
  counts only that viewer's own entries referencing it, consistent with
  the existing "the caller's own non-deleted entries" rule now also
  applying to a category the caller doesn't own; `GET /api/categories`
  returns owned-or-shared, each row's `permission`/`shared`/`owner_name`.
- `web-client-categories`: the "Shared with me" section; a Share action on
  the caller's own category rows.
- `web-client-entries`: the category picker on `/entries/new` and
  `/entries/{id}/edit` includes `append`-tier shared categories, flat, as
  extra top-level options; the entry-list/report category filter includes
  `view`+ shared categories; a category shown on an entry (ledger row,
  filter results) carries the shared badge + owner name when applicable.

## Impact

- **Code**:
  - `backend/internal/storage/postgres/migrations/00NN_category_sharing.sql`
    — new `category_shares` table, mirroring `account_shares`'s shape
    with a two-value `permission` check constraint.
  - `backend/internal/category/`: new `CategoryShare`/`Permission`/
    `ShareResult` types; `Category` gains `Permission`/`Shared`/
    `OwnerName`; `Store`/`Service` gain share CRUD/list/revoke/leave,
    mirroring `internal/account`'s versions; `Get`/`List`/`Usable` widen
    to owned-or-shared; a narrow `UserLookup` interface (satisfied
    structurally by `*auth.Service`, same as `account.UserLookup`) and a
    `Mailer` interface for the notification email; new handler routes
    under `/api/categories/{id}/shares`.
  - `backend/internal/mailer`: `SendCategoryShare`, mirroring
    `SendAccountShare`.
  - `backend/main.go`: wire `category.WithUserLookup`/`WithMailer`/
    `WithBaseURL`, the same way `account` is wired today.
  - `frontend/src/routes/categories.$categoryId.sharing.tsx` (new),
    `frontend/src/components/CategoryLabel.tsx` (shared badge + owner
    name), `categories.tsx` (Shared-with-me section, Share action),
    `entries.new.tsx`/`entries.$entryId.edit.tsx` (category picker
    widening), `entries.index.tsx`/`reports.tsx` (filter widening).
  - New i18n keys (`en.json` first, then `de.json`).
- **API contract**: `openapi/openapi.yaml` gains `CategoryPermission`,
  `CategoryShare`, `CategoryShareInvite`, `CategoryShareInviteResult`,
  `CategorySharePermissionUpdate` schemas and the `/api/categories/{id}/
  shares` paths, and `permission`/`shared`/`owner_name` on `Category` —
  regenerate `backend/openapi.yaml` and `frontend/src/api/schema.d.ts` in
  the same change.
- **Spec**: new `category-sharing` and `web-client-category-sharing`
  specs; deltas on `entry-categories`, `web-client-categories`,
  `web-client-entries`.

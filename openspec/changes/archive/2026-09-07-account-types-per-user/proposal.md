## Why

`account_types` is currently the one lookup in this domain that isn't
owner-scoped: accounts, categories, entries, and tags are all private to
the user who owns them, but account types are a single flat, admin-managed,
instance-wide table (`account-types-settings-tab`, merged days ago). That
mismatch surfaces as a real i18n problem: an admin authors a type's title in
whatever language they're using, and every other user on the instance —
including ones who aren't admin and can't fix it — sees that literal
string, with no per-language storage or resolution to fall back on.
Explored in `openspec/changes/account-types-per-user`'s discovery thread:
rather than adding a translation mechanism to a global lookup, the simpler
fix is to remove the reason a global lookup exists at all — account types
become private to their owner, exactly like categories already are, so the
translation mismatch can't arise (nobody but the owner ever reads their own
types). Categories also gain something account types already had and
categories never did: a starter set of rows on day one, instead of an empty
tree.

## What Changes

- **`account_types` becomes per-user**: gains a required `owner_id`; every
  list/get/create/update/disable/enable/delete is scoped to
  `owner_id = <authenticated caller>`, mirroring `internal/account`'s own
  ownership scoping. The `is_admin` gate is removed entirely — any
  authenticated user fully manages their own set of types, no admin
  involvement. A type belonging to a different owner behaves as if it
  doesn't exist (`ErrNotFound`, matching how `internal/category` already
  treats cross-owner access), including as an account's `type_id`.
- **Delete stays a hard delete, unchanged in shape**: still rejected
  (`409`, `ErrTypeInUse`) while any non-deleted account references it —
  the same guard as today, now naturally scoped to the owner's own
  accounts. `disabled` also stays exactly as it is: a reversible flag that
  blocks new (re)assignment without touching anything already using it.
- **Both account types and categories are seeded with a starter set for
  every user** — new users at signup, and existing users once, in this
  change's migration. Categories have never been seeded before; account
  types drop their old (now meaningless) global rows entirely rather than
  trying to preserve them through the ownership switch — there's no
  production data to protect, so the migration reseeds every user with the
  same starter set instead of reconciling old shared rows into per-owner
  copies. See `design.md` for the seeding mechanism and the exact starter
  lists.
- **Settings tab**: the existing **Account Types** tab
  (`settings.account-types.tsx`) stops being admin-only — it becomes
  available to any authenticated user, showing (and only ever showing)
  their own types, alongside Common and My Invitations rather than next to
  Users.
- `openapi/openapi.yaml`'s account-type paths drop their "(admin only)"
  wording and now-unreachable `403` responses; no schema field changes.
  Regenerate `backend/openapi.yaml` and `frontend/src/api/schema.d.ts` in
  the same change.

## Capabilities

### Modified Capabilities

- `accounts`: the account-type requirements change from "admin-managed,
  instance-global" to "owner-scoped, self-managed," and gain a new
  requirement that a fresh user starts with a seeded set of default types.
- `entry-categories`: gains a new requirement that a fresh user starts with
  a seeded set of default categories (the tree itself, ownership, disable,
  soft delete, and ordering are unchanged from `category-management`).
- `web-client-settings`: the Account Types tab is no longer admin-gated —
  it moves alongside Common / My Invitations in the tab nav, open to every
  authenticated visitor.

## Impact

- **Code**:
  - New migration: `account_types` gains `owner_id`; its old instance-wide
    `UNIQUE(title)` constraint is dropped (matching categories' precedent
    of dropping naming uniqueness rather than reworking it per-owner); its
    existing rows are cleared and every user (existing and future) is
    seeded with the same starter set; every existing account's `type_id` is
    repointed at its owner's seeded "Checking" type (the exact prior
    global type isn't preserved through the switch — see `design.md`).
  - A second migration seeds every *existing* user who currently has zero
    categories with the same starter category set `category-management`
    never populated.
  - `backend/internal/account/`: `Type` gains `OwnerID` (`json:"-"`); every
    `Store`/`Service` method for types gains an `ownerID` parameter; the
    now-dead title-uniqueness handling in
    `internal/storage/postgres/account.go` is removed; a new
    `SeedDefaultTypes(ctx, ownerID) error` on `Store`/`Service`.
  - `backend/internal/category/`: a parallel `SeedDefaults(ctx, ownerID)
    error` on `Store`/`Service`.
  - `backend/internal/account/handler.go`: `requireAdmin` removed from
    every account-type route; each now reads `ownerID` from
    `auth.UserFromContext`, same as the account routes beside them.
  - `backend/internal/auth`: a new narrow hook interface (mirroring the
    existing `LanguageLookup` pattern) invoked once a new user is created,
    so `main.go` can wire `account.Service` and `category.Service` in
    without `internal/auth` importing either.
  - `frontend/src/routes/settings.account-types.tsx`: drop the
    `is_admin` redirect guard.
  - `frontend/src/routes/settings.tsx`: move the Account Types tab out of
    the admin-only group.
- **API contract**: `openapi/openapi.yaml`'s account-type endpoint
  summaries/descriptions drop "(admin only)"; their now-unreachable `403`
  responses are removed. No schema field changes (`owner_id` stays
  internal, like `Account.owner_id` already is). Regenerate
  `backend/openapi.yaml` and `frontend/src/api/schema.d.ts`.
- **Spec**: deltas on `accounts`, `entry-categories`, `web-client-settings`.

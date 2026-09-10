## Context

Every ownership check in the backend today is a single equality:
`accounts.owner_id = caller` or (via `internal/entry`'s `AccountLookup`
interface) "the account's owner equals the caller." `internal/entry`'s
`Owner(ctx, accountID) (ownerID, currency, disabled, error)` is the one
choke point every entry operation goes through to decide whether the caller
may touch an account. Sharing means that choke point has to answer a richer
question — not "is this mine" but "what may I do here" — without breaking
the layering rule that domain packages don't import each other
(`account`/`entry` interact only through the `AccountLookup` interface
`entry` already declares).

Carried-over constraints, per the backend/frontend `AGENTS.md` files and the
`account_types`/`categories`/`tags` precedents:

- No web framework/ORM. Package-per-noun: sharing is a sub-noun of
  `account` (like `account_types`), not a new top-level package.
- Forward-only SQL migrations, one new file.
- Domain packages return sentinel errors; `httpapi/respond.go` maps them to
  status codes in one place.
- A resource belonging to someone else behaves as "not found," not `403`,
  wherever the existing per-owner endpoints already establish that
  convention (accounts, account types, categories, tags). Sharing extends
  who counts as "belongs to," it doesn't change that convention.

## Goals / Non-Goals

**Goals:**

- Four permission tiers (`view`, `append`, `entry_admin`, `owner`), each a
  strict superset of the one before it, enforced uniformly across every
  account and entry endpoint.
- A shared `owner`-tier grant is indistinguishable in capability from the
  real owner, except it never becomes `accounts.owner_id` — the real owner
  is always knowable and always shown.
- `created_by` (renamed from `owner_id`) makes "who logged this" a first-
  class, always-correct fact, independent of who currently has access.
- Revoking access is complete and immediate: no read, write, or historical
  visibility survives it for the revoked user, on that account.
- Sharing by email works without a user directory: you type an email, you
  don't browse a list of registered users.

**Non-Goals:**

- Category or tag sharing. An entry's `category_id`/`tag_id` continue to be
  validated against its *creator's* own lists — unchanged mechanism,
  different meaning now that the creator isn't always the account owner.
- Reassigning `account_types.type_id` on a shared account to anything but
  the real owner's own types, or letting a shared `owner`-tier user browse
  the real owner's type list at all. The account form's type field is
  read-only for anyone but the real owner in this change (see Risks).
- Transferring real ownership (`accounts.owner_id` never changes hands).
- Any change to how `internal/auth`'s own invite mechanism works — sharing
  calls into it read-only (an eligibility check) and, separately, the
  inviter uses the existing, unmodified `POST /api/auth/invites`.
- A "pending share" that auto-activates once an invited email signs up.
  Sharing and inviting stay two separate, manually-sequenced actions.

## Decisions

### Decision: `account_shares` is a new table, not a column on `accounts`

```sql
CREATE TABLE account_shares (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  uuid NOT NULL REFERENCES accounts(id),
    user_id     uuid NOT NULL REFERENCES users(id),
    permission  text NOT NULL CHECK (permission IN ('view','append','entry_admin','owner')),
    granted_by  uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, user_id)
);
CREATE INDEX ON account_shares (user_id);
```

One row per (account, user); `permission` is the current effective grant
(a `PATCH` overwrites it in place — no history table, matching how every
other mutable flag in this codebase, e.g. `categories.disabled`, is a
single current value, not a log). No `deleted_at` — a revoke or self-leave
is a hard `DELETE`, not a soft delete: unlike an account, a category, or an
invite, a share carries no data of its own worth keeping after it ends
(the entries it enabled stay on the account regardless, attributed by
`created_by`). `accounts.owner_id` is untouched — it's still "the real
owner," queried directly, never joined through this table.

### Decision: `entries.owner_id` → `created_by`, and it now means "who created this row," literally

```sql
ALTER TABLE entries RENAME COLUMN owner_id TO created_by;
```

Today `owner_id` is set to "the account's owner at creation" (per
`account-entries`'s current spec) — which was always just the caller,
since only an owner could create an entry. Once `append`-tier sharing
exists, "the account's owner" and "who created this" diverge, and the
existing column already holds the value that matters: the creator. Renaming
it (rather than adding a second column) keeps one fact in one place, and —
this is the load-bearing part — lets the **existing** "owner_id = caller"
scoping mechanism the store layer already had keep working unmodified as
the append-tier's "edit only what I created" rule, instead of needing new
code for it. `category_id`/`tag_id` validation ("must belong to the same
owner as the entry") is untouched by the rename — it already meant, and
continues to mean, the entry's creator.

### Decision: permission resolution replaces `AccountLookup.Owner`, not extends it

```go
// internal/entry's AccountLookup interface — *account.Service satisfies it.
type AccountLookup interface {
    VisibleIDs(ctx context.Context, callerID string) ([]string, error)
    Access(ctx context.Context, accountID, callerID string) (Access, error)
}

type Access struct {
    Currency   string
    Disabled   bool
    Permission Permission // "" when callerID has no access at all
}
```

`Owner(ctx, accountID) (ownerID, currency, disabled, error)` is removed
entirely rather than kept alongside a new method — every one of its four
call sites (`checkAccount`, `Balance`, `Sum`'s per-currency resolution,
`FlowSummary`'s and `BalanceSeries`' per-account currency resolution) was
doing an ownership-equality check that must become a tier check, so there
is no caller left that wants the old shape. `Permission` is one of the four
tier constants or `""` (no access at all — real owner and every
`account_shares` row for that user checked in one place inside
`account.Service.Access`, so `entry` never needs to know shares exist).

Call-site mapping, all inside `internal/entry/service.go`:

| call site | old check | new check |
|---|---|---|
| `checkAccount` (create; move-to on update) | `accOwner == callerID` | `Access.Permission >= append` |
| `Get`/`Update`/`Delete` (see below) | entry's stored owner via `store.Get(ctx, ownerID, id)` | `Access.Permission >= view` (read), then a second, entry-specific check for write (see next decision) |
| `Balance` | `owner == callerID` | `Access.Permission >= view` |
| `Sum`/`FlowSummary`/`BalanceSeries` currency resolution | n/a (already scoped via `VisibleIDs`) | unchanged call shape, `Access` used only for `.Currency` |

`VisibleIDs`'s signature is unchanged — only its implementation widens
(real-owned ∪ shared-with, any tier) — since every List/Sum/FlowSummary/
BalanceSeries call site already routes exclusively through it plus the
caller-supplied `AccountIDs` intersection (`resolveFilter`); no new
entry-side code is needed to make those four endpoints "just work" for
shared accounts once `VisibleIDs` includes them.

### Decision: entry read/write authorization moves from the Store layer into the Service layer

Today `entry.Store`'s `Get`/`Update`/`SoftDelete`/`List`/`Sum`/
`FlowSummary`/`Balance` all take an `ownerID` parameter and filter by it —
that parameter meant "the account owner," which was always the caller. It
can't mean that anymore: a `view`-tier user must be able to *read* an entry
they didn't create, while an `append`-tier user must be blocked from
*writing* one they didn't create — two different scopes for the same row,
which a single `WHERE owner_id = $1`-style filter can't express.

- `Store.Get(ctx, id) (Entry, error)` drops `ownerID` — fetches by id
  (excluding soft-deleted) with no caller-awareness at all; authorization
  moves entirely to `Service`.
- `Store.List`/`Sum`/`FlowSummary`/`Balance` drop `ownerID` too — they were
  never the primary scoping mechanism (that's `f.AccountIDs`, already
  narrowed to the caller's visible accounts by `Service.resolveFilter`
  before it reaches the store); dropping the redundant parameter removes
  the one thing that was silently also requiring `created_by = caller`,
  which is exactly the behavior that has to go away for `view`/
  `entry_admin` tiers to see entries they didn't create.
- `Store.Update`/`SoftDelete` drop `ownerID` for the same reason — the
  service has already read and authorized the entry via `Get` before
  either is called.
- `Service.Get(ctx, callerID, id)`: `store.Get(id)`, then
  `accounts.Access(ctx, entry.AccountID, callerID).Permission >= view` or
  `ErrNotFound` — an entry on an account the caller has no access to reads
  exactly like a nonexistent one, matching the account/category/tag
  convention.
- `Service.Update`/`Delete(ctx, callerID, id, ...)`: `store.Get(id)` (404 if
  missing), resolve `Access`, require `Permission >= append`; if
  `Permission < entry_admin` (i.e., exactly `append`), additionally require
  `entry.CreatedBy == callerID`, else `ErrForbidden`-shaped sentinel → a new
  `403` (not `404` — the caller *can* see the entry, listed a moment ago;
  what they can't do is edit someone else's). `entry_admin`/`owner` skip
  that second check entirely.
- `Service.Create`: unchanged shape, `checkAccount` now requiring
  `Permission >= append`; the created row's `created_by` is always
  `callerID`, never anything account-owner-derived.

### Decision: an entry response carries its creator's identity, not just an id

```go
type Entry struct {
    ...
    CreatedBy     string // user id
    CreatedByName string // display name, falling back to email — resolved server-side
}
```

The web client needs to show "logged by Dana" on any entry a shared-account
viewer didn't create themselves, across cursor-paginated, filtered,
sorted lists — resolving that client-side would mean joining every page
against the account's share list, which the client doesn't otherwise fetch
on `/entries` or `/reports` at all. Resolving it once, server-side,
alongside every other per-row field the store already assembles, is the
same shape `internal/tag`'s `entry_count` already uses (a value folded into
the existing row, not a second round-trip).

### Decision: sharing by email is a synchronous, revealing lookup — deliberately, unlike the magic-link/invite anti-enumeration pattern

`POST /api/accounts/{id}/shares` (owner-tier only) takes `email` +
`permission`. Unlike `POST /api/auth/email/start`'s "always `200`, never
say whether an account exists," this endpoint tells the caller directly
whether the email matched, because the caller here is not an anonymous
prober — they already hold `owner`-tier permission on a real account, an
authenticated, authorized action, not a public one. Matching your framing
directly: the inviter is told synchronously, and — since the useful next
step is "go invite them first" — the response also says whether inviting is
currently possible, mirroring the same `AUTH_SIGNUP_ENABLED`/
`AUTH_INVITE_ENABLED` logic `POST /api/auth/invites` already gates on:

```go
type ShareResult struct {
    Matched     bool
    Share       *AccountShare // non-nil only when Matched
    InviteAllowed bool        // only meaningful when !Matched
}
```

`internal/account` needs two narrow things from `internal/auth` for this,
wired the same optional-dependency way `entry.WithTimezoneLookup` and
`auth.WithLanguageLookup` already are (`main.go` injects a concrete
`*auth.Service` behind each interface; neither domain package imports the
other):

```go
// account.UserLookup — satisfied structurally by *auth.Service.
type UserLookup interface {
    ByEmail(ctx context.Context, email string) (userID, displayName string, err error) // ErrNotFound if none
    InvitingEnabled(ctx context.Context) (bool, error) // wraps the existing signup/invite-enabled logic
}
```

An email matching the caller themselves, or the account's real owner, is
rejected (`ErrInvalidValue`) — you can't share an account with its own
owner or with yourself.

### Decision: sharing an already-shared email updates the permission, not a duplicate row

`POST .../shares` on an email that already has a share for this account
overwrites `permission` (and `updated_at`) on the existing row rather than
returning a conflict — this keeps the share form a single "grant this
person this level of access" action regardless of whether they already had
some other level, rather than requiring the UI to branch between "invite"
and "change permission" flows for what's the same intent either way. The
dedicated `PATCH .../shares/{userId}` endpoint (see below) exists for the
read-only-list page's inline permission editor, which already knows the
target is an existing share and shouldn't re-run the email-lookup path.

### Decision: revoke and self-leave are the same store operation, gated differently

```go
func (s *Service) RevokeShare(ctx, actorID, accountID, targetUserID string) error {
    // owner-tier actor (real owner or owner-tier share) required, unless
    // actorID == targetUserID (self-leave — any tier may remove their own row)
}
```

The real owner has no `account_shares` row (they're `accounts.owner_id`,
not a share), so `targetUserID == account.OwnerID` is rejected
(`ErrInvalidValue`) for both revoke and self-leave — there is nothing to
remove, and "leaving your own account" isn't a coherent action. Deleting
the row is immediate and unconditional: no grace period, no soft delete
(see the `account_shares` decision above for why). Because every entry/
account read in this change re-resolves `Access` per request rather than
caching a permission on a session, revocation takes effect on the target's
very next request — nothing needs to be actively torn down.

### Decision: revoked users lose visibility into entries they created, not just edit rights

This is the one place the rename's "reuse the existing owner_id-scoping
mechanism" decision could mislead: `created_by = caller` is used *only* to
decide whether an `append`-tier user may edit/delete a specific entry they
can already see. It is never used to decide whether an entry is visible at
all — that's `Access.Permission != ""` alone. So once a share is revoked,
`Access` resolves to no permission, `VisibleIDs` no longer includes the
account, and every one of that user's own historically-created entries on
it disappears from their view along with everything else — exactly the
"behaves as if it doesn't exist" convention already established for a
non-owned account, extended to a formerly-shared one. The entries
themselves are untouched: `created_by` still names them, and every
remaining permission holder still sees them, unchanged.

### Decision: `type_id` stays real-owner-only; the account edit form's type field is read-only for a shared `owner`

`account.Service.Update`'s `resolveAssignableType` validates a `type_id`
against `account_types.owner_id` — always the real owner's, since types are
still a flat per-owner lookup this change doesn't touch. A shared `owner`
editing the account therefore can't resolve a *different* type even if
they wanted to (they have no visibility into the real owner's
`account_types` at all — `GET /api/account-types` is still scoped to the
caller). Extending that visibility, or reusing the real owner's types
in-place, is exactly the shape of problem category/tag sharing will need to
solve generally — so this change makes the pragmatic, narrow call: the
account form's type field renders read-only whenever the editor isn't the
real owner, and the backend's existing "effective type must resolve to a
live type" check continues to apply against the real owner's types either
way (an update that doesn't touch `type_id` never re-validates it, same as
today).

### Decision: `/accounts/{id}/sharing` is a new route, visible to any permission tier, read-only below `owner`

Mirrors the `web-client-settings` precedent of a route that's open to every
authenticated visitor but changes what it offers based on authorization
(the Users tab redirects on `!is_admin`; this page instead just hides the
mutation controls, since "every permission holder can see who else has
access" is an explicit requirement, not a gate). Real owner's name/email is
always shown as the first row, unremovable and unchangeable, distinct from
the `account_shares` rows beneath it. The caller's own row (if any) always
shows a Leave action, even in otherwise-read-only mode.

## Risks / Trade-offs

- **Read authorization now costs a query per entry operation that used to
  be a `WHERE` clause.** `Service.Get`/`Update`/`Delete` each do a
  `store.Get` then an `accounts.Access` call rather than one filtered
  query — an extra round-trip per single-entry operation. Accepted: this
  backend has no caching layer anywhere else either, and the alternative
  (a combined store-level join against `account_shares`) would leak
  sharing's existence into `internal/entry`'s `Store` interface, which the
  layering rule (`entry` doesn't import `account`) doesn't allow.
- **Shared `owner`-tier can't reassign the account's type.** Documented
  above as a deliberate scope cut, not an oversight — flag if immediate
  full parity (including type reassignment) turns out to matter before
  category/tag sharing lands.
- **No audit trail on `account_shares`.** A `PATCH` overwrites `permission`
  in place, a `DELETE` removes the row outright — "who had what access
  when" isn't reconstructable after the fact, matching the same accepted
  gap `users.disabled`/`invites.revoked_at` already have (no actor/history
  columns). Out of scope here.
- **A revoked user's own past entries become invisible to them but not to
  anyone else** — intentional per your answer, but worth restating plainly:
  this is a real-access change, not a display filter, so a revoked user
  who later gets re-shared the account will see their old entries reappear
  unchanged.

## Migration Plan

1. `backend/internal/storage/postgres/migrations/0020_account_sharing.sql`
   — create `account_shares`; `ALTER TABLE entries RENAME COLUMN owner_id
   TO created_by`.
2. `internal/account`: `AccountShare` type + validation; `Store` gains
   `CreateOrUpdateShare`, `ListShares` (including the real owner as a
   synthetic first row, or the handler composes it — implementation detail
   for tasks.md), `UpdateSharePermission`, `DeleteShare`; `Access(ctx,
   accountID, callerID)` replaces `Owner`; `List`/`VisibleIDs` widen to
   owned-or-shared. New `UserLookup` interface + `WithUserLookup` option,
   wired from `main.go` to `*auth.Service`. New `Mailer`-shaped interface
   (or reuse `internal/mailer`'s existing one) for the share-notification
   email.
3. `internal/auth`: add `ByEmail` and `InvitingEnabled` to `Service` (or a
   narrow adapter over existing logic) so `account.UserLookup` has
   something to be wired to.
4. `internal/entry`: rename `owner_id` → `created_by` throughout
   (`entry.go`, `service.go`, `store.go`, `storage/memory`,
   `storage/postgres`); `AccountLookup.Owner` → `Access`; `Store` method
   signatures drop their `ownerID` parameters per the decision above;
   `Service.Update`/`Delete` gain the append-tier-owns-check.
5. `internal/account/handler.go`: `GET`/`POST /api/accounts/{id}/shares`,
   `PATCH`/`DELETE /api/accounts/{id}/shares/{userId}`.
6. `openapi/openapi.yaml`: `AccountShare` schema, the four new paths,
   `Account` gains `permission`/`owner_name`/`shared`, `Entry`'s `owner_id`
   becomes `created_by` + `created_by_name`; `go generate ./...` /
   `pnpm generate:api`.
7. Frontend: new `/accounts/{id}/sharing` route; `AccountLabel` shared
   badge + owner name; permission-gated affordances across
   `accounts.$accountId.index.tsx`, `accounts.$accountId.edit.tsx`,
   `AccountCard.tsx`, `entries.index.tsx`, `entries.new.tsx`,
   `entries.$entryId.edit.tsx`, `reports.tsx`; widened account fetches
   wherever "every account I own" is fetched today. New i18n keys.
8. Manual verification: owner shares an account at each tier with a second
   test user and confirms that user's view/create/edit/delete boundaries;
   an unmatched-email share surfaces the invite nudge and, after the
   invited person accepts and signs up, a second share attempt matches;
   revoking mid-session removes all access on the very next request,
   including to the revoked user's own past entries; a shared `owner`
   disables/soft-deletes the account and manages shares exactly like the
   real owner, but cannot change `type_id`; self-leave works for a
   non-owner tier and is unavailable for the real owner.

Rollback: revert the commit. The migration adds a new table and renames a
column — renaming back (`created_by` → `owner_id`) is the compensating
migration if a rollback is ever needed after this ships; until then, a
plain revert of the not-yet-deployed commit needs no migration at all.

## Open Questions

- **Should the account form's type-field read-only restriction (shared
  `owner` can't reassign `type_id`) be revisited before or alongside
  category/tag sharing**, once that groundwork exists? Not blocking this
  change — flagged in Risks.
- **Does a shared `owner`-tier grant ever need to itself invite a *new*
  `owner`-tier co-owner**, or should elevating someone to `owner` be
  reserved for the real owner alone? Your answer ("a shared owner has the
  same rights as the real owner") reads as yes, no restriction, and that's
  what this design implements — flagging only because it means any single
  `owner`-tier grantee can revoke *any other* `owner`-tier grantee,
  including ones they didn't create themselves.

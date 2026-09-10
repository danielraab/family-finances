## Context

`internal/category`'s `Store`/`Service` are entirely `owner_id`-scoped
today: `Get(ctx, ownerID, id)`, `List(ctx, ownerID)`, `Exists(ctx, ownerID,
id)`, `Subtree(ctx, ownerID, id)` all take the owner as an explicit
parameter and every query filters by it. `internal/entry`'s
`CategoryLookup` interface (`*category.Service` satisfies it structurally)
calls two of these — `Usable(ctx, ownerID, id) (bool, error)` and
`Subtree(ctx, ownerID, id) ([]string, error)` — always passing `callerID`
(the entry's creator, not necessarily the account owner) as `ownerID`.
Sharing has to widen what "usable by this caller" means without breaking
the layering rule that domain packages don't import each other
(`category`/`entry` interact only through `CategoryLookup`, mirroring how
`account`/`entry` interact only through `AccountLookup`).

This closely follows `account-sharing`'s already-shipped shape (see
`openspec/specs/account-sharing/spec.md` and the (still-unarchived)
`openspec/changes/account-sharing/design.md`), reusing its patterns
directly: `UserLookup`/`Mailer` interfaces satisfied by `*auth.Service`/
`internal/mailer`, the matched/unmatched synchronous email-invite
response, and the revoke-vs-self-leave split. The one structural
difference: categories have **two** shareable tiers, not four, and no
tier ever grants share-recipient the ability to edit the category itself
— that stays with the real owner (`categories.owner_id`) exclusively, with
no shared-owner-parity option the way accounts have one.

Carried-over constraints, per `backend/AGENTS.md`'s Categories section and
the account-sharing precedent:

- No web framework/ORM. Package-per-noun: sharing is a sub-noun of
  `category`, not a new top-level package.
- Forward-only SQL migrations, one new file.
- Domain packages return sentinel errors; `httpapi/respond.go` maps them
  to status codes in one place.
- A resource belonging to someone else, and one the caller has no
  permission on at all, both behave as "not found" (`404`), not `403` —
  the same convention `account`/`category`/`tag` already establish.
  Sharing extends who counts as having access; it doesn't change that
  convention.

## Goals / Non-Goals

**Goals:**

- Two permission tiers (`view`, `append`), `append` a strict superset of
  `view`, enforced uniformly on `GET /api/categories`, the entry-category
  validation path, and the entry-list/report category filter.
- A category's real owner is always the sole party who can edit its
  metadata (name/icon/color/parent), disable/enable, reorder, or delete
  it — sharing is purely additive visibility/usability for everyone else,
  never delegated administration.
- `internal/entry` needs **zero code changes** — `CategoryLookup.Usable`
  already takes the calling user, not an owner; only its meaning inside
  `category.Service` changes.
- Sharing by email works the same way account sharing already does: no
  user directory, synchronous matched/unmatched reporting.
- No cascade: sharing one category never implicitly shares its
  descendants or ancestors.

**Non-Goals:**

- A shareable "owner" tier for categories, or any way for a share
  recipient to edit a category's own metadata, tree position, or
  lifecycle. If that's ever wanted, it's a distinct future change — this
  one keeps the real owner as the sole administrator, deliberately
  simpler than accounts' four-tier model.
- Cascading a share to a category's children, either at share time or
  dynamically. A shared category's children remain private to the real
  owner unless separately, individually shared.
- Any change to `internal/entry`'s authorization model. Editing/deleting
  an entry stays governed exclusively by `account.Permission` (`append`
  edits only entries the same user created; `entry_admin`/`owner` edit
  any). A category's own permission tier never grants or restricts entry
  write access — it only decides whether the category can be seen or
  picked at all.
- Reparenting a category into or out of another user's tree through a
  share. `Exists` (parent validation) and the reparent-cycle-detection
  `Subtree` stay strictly scoped to the caller's own tree, unaffected by
  this change.

## Decisions

### Decision: `category_shares` is a new table, mirroring `account_shares`

```sql
CREATE TABLE category_shares (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id uuid NOT NULL REFERENCES categories(id),
    user_id     uuid NOT NULL REFERENCES users(id),
    permission  text NOT NULL CHECK (permission IN ('view','append')),
    granted_by  uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (category_id, user_id)
);
CREATE INDEX ON category_shares (user_id);
```

One row per (category, user); `permission` is the current effective grant
(a `PATCH` overwrites it in place, same as `account_shares`). No
`deleted_at` — revoke/self-leave is a hard `DELETE`, matching
`account_shares`'s reasoning: a share carries no data of its own worth
keeping once it ends (entries that used the category while it was shared
keep their `category_id` regardless — see the entry-visibility decision
below). `categories.owner_id` is untouched.

### Decision: `Usable` becomes "usable by this caller," not "owned by this caller"

```go
// category.Service
func (s *Service) Usable(ctx context.Context, callerID, id string) (bool, error) {
    c, err := s.store.GetForCaller(ctx, callerID, id) // owned-or-shared, annotated
    if errors.Is(err, ErrNotFound) {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    if c.Disabled {
        return false, nil
    }
    return c.Permission.AtLeast(PermissionAppend), nil
}
```

This is the load-bearing change: `internal/entry.Service.Create`/`Update`
already call `s.categories.Usable(ctx, callerID, *categoryID)`, having
always treated the first argument as "the person doing the
categorizing" (see `entry/service.go`'s doc comment: "confirms any
category/tags belong to callerID — the entry's creator"). Widening what
`Usable` checks internally — real ownership *or* an `append`-tier share,
and not disabled — makes shared categories selectable on entries with no
change anywhere in `internal/entry`.

### Decision: `Get`/`List` return owned-or-shared; the tree-structural helpers (`Exists`, `Subtree`-for-cycles) stay owner-only

`category.Store` needs a second read path distinct from today's strict
`Get(ctx, ownerID, id)`/`List(ctx, ownerID)`:

```go
// Existing — unchanged in meaning, still used for parent validation
// (Service.Create/Update) and cycle detection (Service.Update's reparent
// check). Both operations only ever act within the caller's own tree —
// sharing never lets you reparent into or out of someone else's.
Exists(ctx context.Context, ownerID, id string) (bool, error)
Subtree(ctx context.Context, ownerID, id string) ([]string, error)

// New — resolves id as seen by callerID: real ownership, else a matching
// category_shares row, else Permission == "" (no access at all — not
// itself an error, mirroring account.Store.Get's convention).
GetForCaller(ctx context.Context, callerID, id string) (Category, error)

// List widens: every category callerID owns (full tree, Permission
// "owner") union every category shared with them (each a single row,
// Permission "view"/"append", Shared: true, OwnerName set).
List(ctx context.Context, callerID string) ([]Category, error)
```

`entry.Service`'s *other* caller of `Subtree` — resolving a category
filter to "this category or its descendants" for `List`/`Sum`'s
`CategoryMode: subtree` default — needs the *caller-scoped* read, not the
owner-only one, since a caller should be able to filter by a category
shared with them. Because there's no cascade, this is simple: a shared
category's `Subtree` (as resolved for filtering) is just `[id]` itself —
resolving successfully where today it would silently fail (a category
outside the caller's own tree doesn't `Exists`, so today's `Subtree`
calls never reach a shared id at all). `category.Service` gains a second,
caller-scoped method for this:

```go
// FilterSubtree resolves id and its descendants within the categories
// visible to callerID (owned, full recursion; shared, just the id
// itself — there is no cascade to resolve into). Used only by
// internal/entry's category-filter resolution — never for reparent-cycle
// detection, which stays strictly own-tree via Subtree above.
func (s *Service) FilterSubtree(ctx context.Context, callerID, id string) ([]string, error)
```

`entry.CategoryLookup.Subtree` is re-pointed at `FilterSubtree` instead of
`Subtree` — a one-line wiring change in `main.go`, no interface or
`internal/entry` code change (the interface's method name in `entry.go`
can stay `Subtree`; only which `category.Service` method backs it moves).

### Decision: a shared category always renders flat — no server-side `parent_id` rewriting needed

A shared category keeps its real, stored `parent_id` in every API
response — `GetForCaller`/`List` never null it out. Since a share never
cascades, the parent it points at is (almost always) not itself visible
to the recipient, so it's simply absent from their `GET /api/categories`
response. `frontend/src/lib/categoryTree.ts`'s existing
`buildCategoryTree`/`flattenCategoryTree` already bucket every category
by `parent_id` and treat anything whose parent isn't present in the list
as a root (`byParent.get(key) ?? []` walks starting from key `""`, and a
`parent_id` matching no known category simply never gets visited as
anyone's child) — so a shared category is *already* promoted to
top-level for free, with no special-casing needed on either side. (The
rare edge case — the recipient happens to independently own or be shared
a category with the same id as the real parent — can't occur: ids are
globally unique, so a `parent_id` only ever resolves to that exact
category, which by definition isn't shared unless it's the one node
someone chose to share.)

### Decision: `Usable`'s widening only affects *newly assigned* categories, exactly like the existing not-disabled rule

`entry.Service.Create`/`Update` already treat `Usable` as a check that
applies only when a category is being newly set — an update that leaves
`category_id` untouched never re-validates it (see `entry-categories`'s
existing "disabling does not affect existing entries" rule). This means a
revoked category share behaves exactly like a disabled category already
does: an entry that was categorized under it while the share was active
keeps that `category_id` and keeps resolving/displaying it, but the
category can no longer be *newly* selected by the (former) recipient.
Nothing new needs to be built for this — it falls out of the existing
Create/Update validation shape unchanged.

### Decision: sharing by email, invite/manage authorization, and revoke/self-leave — copied from account-sharing verbatim, minus the shareable-owner cases

```go
// category.go — mirrors account.go's Permission/AccountShare/ShareResult
type Permission string

const (
    PermissionView   Permission = "view"
    PermissionAppend Permission = "append"
)

var permissionRank = map[Permission]int{PermissionView: 1, PermissionAppend: 2}

func (p Permission) AtLeast(other Permission) bool { /* same shape as account.Permission.AtLeast */ }

type CategoryShare struct {
    CategoryID     string     `json:"-"`
    UserID         string     `json:"user_id"`
    Name           string     `json:"name"`
    Email          string     `json:"email"`
    Permission     Permission `json:"permission"`
    GrantedBy      string     `json:"granted_by"`
    GrantedByName  string     `json:"granted_by_name"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}

type ShareResult struct {
    Matched       bool
    Share         *CategoryShare
    InviteAllowed bool
}
```

`Service.InviteShare`/`UpdateSharePermission`/`RevokeShare` mirror
`account.Service`'s methods exactly, with one simplification: every
manage-a-share action (invite, change permission, revoke) requires the
caller to be the category's **real owner** — there is no shared-owner
tier to also admit, so the check is a plain `category.OwnerID ==
callerID` rather than `Access.Permission == PermissionOwner`. Self-leave
(`RevokeShare` with `targetUserID == callerID`) is unchanged from
account-sharing's shape: any tier may remove their own row, no
owner-only gate. `ListShares` requires only `Permission != ""` (view or
append), same as accounts. `UserLookup`/`Mailer` are new, narrow
interfaces on `category` structurally satisfied by `*auth.Service`/
`internal/mailer`'s existing shapes — `main.go` wires
`category.WithUserLookup(authSvc)`, `category.WithMailer(...)`,
`category.WithBaseURL(...)`, the same way `account` is wired.

Sharing with your own email, or an email that resolves to the category's
own real owner (i.e., yourself, since only the real owner can invite), is
rejected the same way account-sharing rejects the equivalent
self/real-owner cases.

### Decision: `entry_count` on a shared category counts only the viewer's own entries

`entry-categories`'s existing rule — "the number of the *caller's* own
non-deleted entries whose `category_id` is that category" — already reads
correctly once `List`/`GetForCaller` are caller-scoped: for a category
the caller owns, this is unchanged (all their own entries); for a shared
one, it naturally becomes "how many of *my* entries use this category I
don't own," which is the number a recipient actually cares about (the
real owner's own count, on their own view of the same category, is a
separate, larger number — no aggregation across users). No new query
shape is needed beyond parameterizing the existing correlated subquery by
`callerID` instead of implicitly by `owner_id` in every case.

## Risks / Trade-offs

- **A recipient's `GET /api/categories` now does one more query shape**
  (owned tree ∪ shared rows) instead of a single `owner_id = $1` scan —
  accepted, matching the same shape `account.Store.List`'s
  owned-∪-shared `UNION` already costs.
- **No audit trail on `category_shares`**, matching `account_shares`'s
  same accepted gap (a `PATCH` overwrites in place, a `DELETE` removes
  the row outright).
- **A category shared with someone, then later reparented by its real
  owner, keeps behaving identically for the recipient** — reparenting
  only affects the *real owner's* tree structure; since the recipient
  never saw the old parent either, there's nothing for them to notice.
  Worth stating plainly since it's a case account-sharing has no
  analogue for (accounts aren't tree-structured).
- **Two independent per-domain sharing systems (accounts, categories)
  with near-identical code shapes but no shared abstraction.** Accepted,
  consistent with this codebase's existing pattern of small,
  independent per-noun packages (`account`, `category`, `tag`) rather
  than a generic "shareable resource" framework — extracting one now,
  for two instances, would be premature.

## Migration Plan

1. `backend/internal/storage/postgres/migrations/00NN_category_sharing.sql`
   — create `category_shares` (two-value `permission` check, unlike
   `account_shares`'s four).
2. `internal/category`: `Permission`/`CategoryShare`/`ShareResult` types;
   `Category` gains `Permission`/`Shared`/`OwnerName`; `Store` gains
   `GetForCaller`, widened `List`, `CreateOrUpdateShare`, `ListShares`,
   `ShareByUser`, `UpdateSharePermission`, `DeleteShare`; `Exists`/
   `Subtree` (cycle-detection) unchanged.
3. `internal/category/service.go`: `Usable` widens per the decision
   above; new `FilterSubtree`; new `ListShares`/`InviteShare`/
   `UpdateSharePermission`/`RevokeShare`, real-owner-gated (no shared
   owner tier); new `UserLookup`/`Mailer` interfaces + `Option`s.
4. `internal/mailer`: `SendCategoryShare`.
5. `internal/category/handler.go`: `GET`/`POST /api/categories/{id}/
   shares`, `PATCH`/`DELETE /api/categories/{id}/shares/{userId}`;
   `Category` JSON gains `permission`/`shared`/`owner_name`.
6. `main.go`: wire `category.WithUserLookup(authSvc)`,
   `category.WithMailer(...)`, `category.WithBaseURL(...)`; re-point
   `entry.NewService`'s `CategoryLookup.Subtree` wiring at
   `category.Service.FilterSubtree`.
7. `openapi/openapi.yaml`: `CategoryPermission`, `CategoryShare`,
   `CategoryShareInvite`, `CategoryShareInviteResult`,
   `CategorySharePermissionUpdate` (mirroring the `Account*` schemas of
   the same shapes), `Category` gains `permission`/`shared`/
   `owner_name`, four new paths; `go generate ./...` /
   `pnpm generate:api`.
8. Frontend: new `/categories/{id}/sharing` route (mirrors
   `accounts.$accountId.sharing.tsx`, two-option permission selector);
   `categories.tsx` gains a "Shared with me" section + a Share action on
   owned rows; `CategoryLabel.tsx` gains the shared badge + owner name
   (mirroring `AccountLabel.tsx`); entry form category picker includes
   `append`-shared categories flat; entry-list/report category filter
   includes `view`+-shared categories. New i18n keys.
9. Manual verification: real owner shares a category at each tier with a
   second test user; recipient can filter by a `view`-shared category but
   not pick it on a new entry; recipient can pick an `append`-shared
   category on a new entry, and it renders flat (no parent) in their own
   `/categories` and entry-form picker; disabling or revoking the share
   stops new selection without touching entries already categorized under
   it; an unmatched-email share surfaces the invite nudge exactly like
   accounts; self-leave works for either tier; a non-owner attempting to
   invite/patch/revoke another user's access gets `403`.

Rollback: revert the commit; the migration only adds a new table, so a
plain revert needs no compensating migration before this ships.

## Open Questions

- **Should `view`-tier sharing exist at all as a separate action from
  `append`,** given it grants only filter/display visibility and no
  entry-picking? Kept per your explicit decision (two real, distinct
  tiers) — flagging only because it's a smaller behavioral surface than
  `view` on accounts (which also gates reading entries/balance, a much
  more visible capability).
- **Does a category ever need to be un-shared automatically when its real
  owner disables or deletes it?** Not addressed here — disabling a
  category already blocks new use for everyone (owner and recipients
  alike, since `Usable` checks `!Disabled` regardless of tier), and
  deleting one is already blocked while any entry references it,
  independent of who created that entry. No special-casing appears
  needed, but flagging since account-sharing has no direct analogue
  (deleting a shared account isn't blocked by its shares).

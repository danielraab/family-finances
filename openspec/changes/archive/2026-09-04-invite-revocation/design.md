## Context

`invites` (`0002_auth.sql`) has no cancellation concept at all today — once
created, a row only ever transitions via `ConsumeInvite` (accepted) or by
aging past `expires_at`. `GET /api/auth/invites` is the only read path and is
admin-only (`user-administration`); the inviter themselves has no view of
what they've sent unless they're also an admin. This change adds that view
plus the ability to cancel an invitation, for both admins and the person who
sent it.

Carried-over constraints, per `add-settings-page`'s precedent (the last
change to touch this exact area) and the backend/frontend `AGENTS.md` files:

- No web framework/ORM; `internal/auth` already owns `invites` as one of its
  nouns — this is more operations on the same `Store`, not a new package.
- Forward-only SQL migrations; `users.disabled`/`users.deleted_at`
  (`0004_user_administration.sql`) is the direct precedent for adding
  lifecycle columns to an existing table rather than a new one.
- Settings tabs are client-side route guards backed by server-side
  authorization; a tab that needs no special permission (unlike Users) is
  simpler — it just needs "authenticated," which every `/settings` route
  already requires.

## Goals / Non-Goals

**Goals:**

- An admin or an invitation's own creator can cancel it, and doing so
  reliably blocks acceptance from then on.
- Every authenticated user can see the invitations they've personally sent,
  not just admins.
- Revoking is safe to call more than once (idempotent) and safe to call on
  an invite in any state.
- An admin can permanently hide a revoked invitation from view without
  losing the underlying audit row.
- Both invitation lists (admin Users tab, new My Invitations tab) tell the
  visitor plainly when there's nothing to show.

**Non-Goals:**

- Un-revoking or un-deleting — both are one-way, matching the existing
  disable/(soft-)delete precedent for users (no undo affordance in this
  change either).
- Revoking or deleting on behalf of the *invitee* (the person the invite was
  sent to) — only the inviter and admins act on an invitation; the invitee's
  only interaction with it is the acceptance link.
- Distinguishing "expired" from "pending" in the status display — the
  existing Users tab invitations list doesn't make this distinction today
  and this change doesn't change that; it only adds a fourth status
  (revoked) alongside the existing pending/accepted rendering.
- Any change to who may *create* an invite (`POST /api/auth/invites` stays
  open to any authenticated user, unchanged).

## Decisions

### Decision: `revoked_at` and `deleted_at` as two nullable columns, not a status enum

```sql
ALTER TABLE invites
  ADD COLUMN revoked_at timestamptz,
  ADD COLUMN deleted_at timestamptz;
```

Mirrors `users.disabled`/`users.deleted_at` (`0004_user_administration.sql`)
in spirit, adapted to invites: `accepted_at` and `revoked_at` are already
timestamp-shaped state (an invite has no "disabled" toggle, only one-way
transitions), so a second timestamp column is more consistent here than
introducing a boolean alongside them. No `revoked_by`/`deleted_by` actor
column, matching the same precedent (`users.disabled` and `users.deleted_at`
don't track an actor either).

An invite's displayed status is derived, not stored: `revoked_at != NULL` →
revoked; else `accepted_at != NULL` → accepted; else `expires_at < now()` →
expired; else pending. `deleted_at != NULL` rows are excluded from every
listing query entirely (never reach the derivation).

### Decision: revoke is inviter-or-admin; delete is admin-only

Revoking is the natural undo for the person who made the mistake — requiring
an admin for every revoke would make a typo in the invite form an
admin-support ticket. Soft-delete is different: it's list hygiene ("stop
showing me old cancelled invites"), not correcting a mistake, and is scoped
to admins only, matching every other destructive/administrative action on
this page (disable, enable, soft-delete a user).

The ownership check needs the invite's `invited_by` before it can decide, so
`Service.RevokeInvite` reads the invite first (`Store.InviteByID`, a new
method — nothing existing does a single-invite lookup by id), then either
proceeds or returns a new `ErrForbidden`-shaped sentinel if the caller is
neither the inviter nor an admin. This is a read then a conditional write,
not a single atomic `UPDATE ... WHERE (...)`, because the two failure modes
("no such invite" vs "not yours to revoke") need to produce different
responses (`404` vs `403`) and a single filtered `UPDATE` can't distinguish
them without a preceding read anyway.

### Decision: revoke is idempotent by construction — `revoked_at = COALESCE(revoked_at, now())`

```sql
UPDATE invites
   SET revoked_at = COALESCE(revoked_at, $2)
 WHERE id = $1 AND deleted_at IS NULL
RETURNING ...
```

A second (or third) call to revoke the same invite doesn't just avoid
erroring — it leaves `revoked_at` at its original value rather than bumping
it to the new `now()`, so repeated calls are indistinguishable in their
effect on the stored row, not merely in their HTTP status. `Service`/
`Handler` return `200` with the (unchanged) invite every time, never a
`409` for "already revoked."

### Decision: every invite is revocable, in any state, and stays visible afterward

Revoking an already-accepted or already-expired invite is a legitimate
no-op-in-effect action (the account already exists, or the link already
stopped working) rather than an error — the alternative (rejecting revoke on
non-pending invites) would require the client to first know an invite's
derived status before deciding whether to offer the button, for no real
benefit, since a revoked-but-already-accepted invite is harmless: `revoked_at`
being set doesn't retroactively undo the account it created. `ConsumeInvite`
gains `AND revoked_at IS NULL` in its atomic acceptance query so a *pending*
invite that gets revoked can no longer be accepted — that's the one case
where revoking changes an outcome.

Revoked invites are never hidden from a listing (`ListInvites`,
`ListInvitesByInviter`) — only `deleted_at` hides a row. This keeps "did I
revoke this?" answerable from the same list a moment later, and matches your
instruction directly ("every invite can be revoked and are still shown in
the list").

### Decision: soft-delete requires `revoked_at IS NOT NULL`

```go
func (s *Service) SoftDeleteInvite(ctx, actor, id) error {
    inv, err := s.store.InviteByID(ctx, id)
    ...
    if inv.RevokedAt == nil {
        return ErrInviteNotRevoked // -> 409
    }
    return s.store.SoftDeleteInvite(ctx, id, s.now())
}
```

Matches your framing ("soft-delete a revocation") literally: this is cleanup
for a revoked entry, not a general-purpose delete for any invite regardless
of status. `DELETE /api/auth/invites/{id}` on a pending/accepted/expired,
not-yet-revoked invite is rejected (`409`) rather than silently revoking it
first — a delete should not have a side effect the caller didn't ask for.

### Decision: `GET /api/auth/invites/mine` is a new read path, not a filter param on the existing admin endpoint

`GET /api/auth/invites` stays admin-only and unfiltered (unchanged
authorization, unchanged shape). A *different* endpoint for "my own"
avoids overloading one route with two different authorization rules
depending on a query parameter, and mirrors how `ListInvites` and the new
`ListInvitesByInviter` are already two separate `Store` methods with
different `WHERE` clauses (one all-invites, one `WHERE invited_by = $1`)
rather than one method with an optional filter.

### Decision: "My Invitations" is its own tab, not folded into Common

The Common tab is preferences (language/timezone/currency) — a different
kind of content (a list with actions) belongs in its own tab rather than
appended below the preference controls. Route: `frontend/src/routes/
settings.invitations.tsx` → `/settings/invitations`, alongside the existing
`settings.index.tsx` (Common) and `settings.users.tsx` (Users, admin-only).
Tab order: Common, My Invitations, Users (admin-only, unchanged position at
the end) — personal scope before administrative scope.

The admin Users tab's invitations section and the new My Invitations tab
render materially the same list shape (email, inviter — omitted on My
Invitations since it's always "you", status, dates, Revoke button); sharing
a small presentational component between the two routes is an
implementation detail for `tasks.md`, not a spec concern.

## Risks / Trade-offs

- **A revoked-but-already-accepted invite is confusing to leave revocable.**
  Accepted at all really means "already used"; revoking it afterward does
  nothing observable except record `revoked_at`. Accepted per the "every
  invite can be revoked" instruction — the UI should still make an accepted
  invite's Revoke action visibly inert-in-effect (e.g., no separate warning,
  since nothing bad happens), but this is a documented trade-off, not a bug.
- **No actor tracking on revoke/delete.** Consistent with the existing
  `users` precedent, but means "who revoked this" isn't recoverable from the
  `invites` row itself if that ever matters later (e.g., audit logging) —
  out of scope here.

## Migration Plan

1. `backend/internal/storage/postgres/migrations/0005_invite_revocation.sql`
   — add `revoked_at`, `deleted_at` to `invites`.
2. `internal/auth.Store`: add `InviteByID`, `ListInvitesByInviter`,
   `RevokeInvite`, `SoftDeleteInvite`; implement in `storage/memory` and
   `storage/postgres`; existing `ListInvites`/`ActiveInviteForEmail` gain a
   `deleted_at IS NULL` filter; `ConsumeInvite` gains `revoked_at IS NULL`.
3. `internal/auth.Service`: `RevokeInvite(ctx, actor, id)`,
   `SoftDeleteInvite(ctx, actor, id)`, `ListMyInvites(ctx, actor)`; new
   sentinel errors for "not yours to revoke" (`403`) and "not revoked yet"
   (`409`).
4. `internal/auth.Handler`: `POST /api/auth/invites/{id}/revoke`,
   `DELETE /api/auth/invites/{id}`, `GET /api/auth/invites/mine`.
5. `openapi/openapi.yaml`: `revoked_at` on `Invite`/`InviteInfo`, the three
   new paths; `go generate ./...` (backend), `pnpm generate:api` (frontend).
6. Frontend: `settings.invitations.tsx` (new tab + route), tab nav entry in
   `settings.tsx`, Revoke action + empty state in `settings.users.tsx`, new
   i18n keys (English first).
7. Manual verification: a non-admin invites someone, sees it on My
   Invitations, revokes it, confirms the acceptance link now fails; an admin
   revokes someone else's invite from the Users tab; an admin soft-deletes a
   revoked invite and confirms it disappears from both lists; a non-admin,
   non-inviter attempt to revoke someone else's invite is rejected (`403`);
   calling revoke twice is confirmed idempotent (same `revoked_at`, no
   error).

Rollback: revert the commit; the migration only adds nullable columns, so no
compensating migration is needed for a safe rollback.

## Open Questions

None blocking — the forks raised during discovery (My Invitations' scope,
whether soft-delete requires a prior revoke) were resolved in conversation
before this design was written.

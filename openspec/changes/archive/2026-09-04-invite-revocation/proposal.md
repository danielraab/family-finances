## Why

Today an invitation, once sent, cannot be undone: `POST /api/auth/invites` is
open to any authenticated user, but there is no way to cancel a mistyped
address, an invite sent to the wrong person, or one nobody needs anymore
short of letting it sit until `AUTH_INVITE_TTL` expires. Worse, only an admin
can even *see* an invitation after creating it (`GET /api/auth/invites` is
admin-only) — the person who actually sent it has no view of what they
invited unless they happen to be an admin too. The Users tab's invitations
list also gives no feedback at all when there's nothing to show.

## What Changes

- `invites` gains two columns: `revoked_at` (set once, idempotently) and
  `deleted_at` (soft delete, admin cleanup of a revoked entry).
- New endpoint `POST /api/auth/invites/{id}/revoke` — callable by an admin or
  by the invite's own inviter, on an invite in any state (pending, accepted,
  or expired). Idempotent: revoking an already-revoked invite is a no-op that
  still returns the invite. A revoked invite's acceptance link stops working.
  Revoked invites are **not** removed from any listing — they keep showing up
  with a revoked status.
- New endpoint `DELETE /api/auth/invites/{id}` — admin-only soft delete,
  valid only on an already-revoked invite (rejected otherwise). A
  soft-deleted invite disappears from every listing.
- New endpoint `GET /api/auth/invites/mine` — any authenticated user, scoped
  to invitations they personally created (`invited_by = self`), excluding
  soft-deleted ones.
- New **My Invitations** settings tab, visible to every authenticated user
  (not just admins), listing the invitations they've sent with a Revoke
  action and empty-state text when there are none.
- The existing admin **Users** tab's invitations list gains a Revoke action
  per invitation and the same empty-state text.
- Fixes the "inventations"/"invokation" typos from the originating
  conversation — the shipped terms are **invitation(s)** and
  **revoke/revocation** throughout code, API, and copy.

## Capabilities

### Modified Capabilities

- `authentication`: the `Invites` requirement gains revocation as a rejection
  reason for acceptance (alongside expiry and prior consumption).
- `user-administration`: invitation listing is no longer admin-exclusive (a
  self-scoped listing is added); new requirements for revoking and
  soft-deleting an invitation, with their authorization and idempotency
  rules.
- `web-client-settings`: a new admin-independent **My Invitations** tab; the
  admin Users tab's invitations section gains revoke and an empty state.

## Impact

- **Code**:
  - `backend/internal/storage/postgres/migrations/0005_invite_revocation.sql`
    — `ALTER TABLE invites ADD COLUMN revoked_at timestamptz, ADD COLUMN
    deleted_at timestamptz`.
  - `backend/internal/auth/`: `Store` gains `InviteByID`,
    `ListInvitesByInviter`, `RevokeInvite`, `SoftDeleteInvite`; `Service`
    gains the matching use-case methods with the inviter-or-admin
    authorization check; `Handler` gains the three new routes;
    `ConsumeInvite`'s atomic acceptance query excludes a revoked invite.
  - `frontend/src/routes/settings.invitations.tsx` (new, `/settings/invitations`,
    "My Invitations" tab) and `frontend/src/routes/settings.tsx` (tab nav
    entry) and `frontend/src/routes/settings.users.tsx` (revoke action, empty
    state).
  - New i18n keys in `frontend/src/i18n/locales/{en,de}.json`.
- **API contract**: `openapi/openapi.yaml` gains `revoked_at` on
  `Invite`/`InviteInfo`, and paths `GET /api/auth/invites/mine`,
  `POST /api/auth/invites/{id}/revoke`, `DELETE /api/auth/invites/{id}` —
  regenerate `backend/openapi.yaml` and `frontend/src/api/schema.d.ts` in the
  same change.
- **Spec**: deltas on `authentication`, `user-administration`,
  `web-client-settings`.

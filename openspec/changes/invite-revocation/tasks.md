## 1. Backend: schema

- [ ] 1.1 Add migration `backend/internal/storage/postgres/migrations/0005_invite_revocation.sql`:
  `ALTER TABLE invites ADD COLUMN revoked_at timestamptz, ADD COLUMN
  deleted_at timestamptz`.

## 2. Backend: `internal/auth` store

- [ ] 2.1 Add `InviteByID(ctx, id) (Invite, error)` (`ErrNotFound` for a
  missing or soft-deleted row).
- [ ] 2.2 Add `ListInvitesByInviter(ctx, inviterID) ([]InviteInfo, error)` —
  same shape as `ListInvites`, filtered to `invited_by = $1`, excluding
  soft-deleted.
- [ ] 2.3 Add `RevokeInvite(ctx, id, now) (Invite, error)` —
  `revoked_at = COALESCE(revoked_at, $now)`, `WHERE id = $1 AND deleted_at IS
  NULL`, idempotent (unchanged `revoked_at` on a repeat call).
- [ ] 2.4 Add `SoftDeleteInvite(ctx, id, now) error` — `SET deleted_at = now()
  WHERE id = $1 AND revoked_at IS NOT NULL AND deleted_at IS NULL`; the
  service layer distinguishes "not found," "not yet revoked," and
  "already deleted" by reading first (see 3.1).
- [ ] 2.5 `ListInvites` and `ActiveInviteForEmail` gain `deleted_at IS NULL`;
  `ConsumeInvite`'s atomic UPDATE gains `AND revoked_at IS NULL`.
- [ ] 2.6 Implement all of the above in `internal/storage/memory` and
  `internal/storage/postgres`.

## 3. Backend: `internal/auth` service + handler

- [ ] 3.1 `Service.RevokeInvite(ctx, actorID, id) (Invite, error)`: read via
  `InviteByID` (`ErrNotFound` → `404`), authorize (`actorID == invite.InvitedBy
  || actor.IsAdmin`, else a new `ErrInviteRevokeForbidden` → `403`), then
  `store.RevokeInvite`.
- [ ] 3.2 `Service.SoftDeleteInvite(ctx, actorID) error`: admin-only (`403`
  via the existing admin guard pattern); read via `InviteByID` (`404` if
  missing/already deleted); `409` (new `ErrInviteNotRevoked`) if
  `RevokedAt == nil`; else `store.SoftDeleteInvite`.
- [ ] 3.3 `Service.ListMyInvites(ctx, actorID) ([]InviteInfo, error)` —
  thin wrapper over `store.ListInvitesByInviter`.
- [ ] 3.4 Add sentinel errors `ErrInviteRevokeForbidden`, `ErrInviteNotRevoked`
  to `store.go`'s sentinel list; map them in `httpapi/respond.go` (`403`,
  `409`).
- [ ] 3.5 `Handler`: `POST /api/auth/invites/{id}/revoke` (any authenticated
  user — authorization happens in the service, not a blanket admin guard),
  `DELETE /api/auth/invites/{id}` (admin-only guard, same shape as
  `deleteUser`), `GET /api/auth/invites/mine` (any authenticated user).
- [ ] 3.6 Unit + handler tests: revoke by the inviter, revoke by an admin who
  isn't the inviter, revoke rejected for a third party (`403`), revoke on a
  missing/deleted invite (`404`), revoke called twice is idempotent (same
  `revoked_at`, both `200`), revoke on an accepted/expired invite still
  succeeds and the invite remains listed; delete requires `revoked_at` set
  (`409` otherwise), delete removes the row from both `GET /api/auth/invites`
  and `GET /api/auth/invites/mine`; `GET /api/auth/invites/mine` returns only
  the caller's own invites; accepting a revoked invite's link fails.

## 4. API contract

- [ ] 4.1 `openapi/openapi.yaml`: add `revoked_at` (nullable) to `Invite` and
  `InviteInfo`; add paths `GET /api/auth/invites/mine`,
  `POST /api/auth/invites/{id}/revoke` (`200` → `Invite`, `401`, `403`,
  `404`), `DELETE /api/auth/invites/{id}` (`204`, `401`, `403`, `404`,
  `409`).
- [ ] 4.2 `cd backend && go generate ./...` to sync `backend/openapi.yaml`.
- [ ] 4.3 `cd frontend && pnpm generate:api` to regenerate `src/api/schema.d.ts`.

## 5. Frontend: My Invitations tab

- [ ] 5.1 `frontend/src/routes/settings.invitations.tsx` — new route,
  `/settings/invitations`, any authenticated user; fetch
  `GET /api/auth/invites/mine` on mount; render the list (email, status,
  dates) with a Revoke action per pending/non-revoked row; empty-state text
  when the list is empty.
- [ ] 5.2 Add "My Invitations" to the tab nav in `frontend/src/routes/settings.tsx`,
  between Common and the admin-only Users tab, visible to every authenticated
  user (no admin gate).
- [ ] 5.3 Revoke action calls `POST /api/auth/invites/{id}/revoke`; on
  success, update that row's status in place (no refetch needed).

## 6. Frontend: Users tab

- [ ] 6.1 `frontend/src/routes/settings.users.tsx`: add empty-state text to
  the invitations list when it's empty.
- [ ] 6.2 Add a Revoke action per invitation row (calls the same
  `POST /api/auth/invites/{id}/revoke`), and a status label covering
  pending/accepted/revoked.
- [ ] 6.3 Both Revoke actions (Users tab and My Invitations tab) go through
  the existing `@headlessui/react` `Dialog` confirmation pattern already used
  for disable/enable/delete.

## 7. i18n

- [ ] 7.1 Add new keys to `frontend/src/i18n/locales/en.json` first, then
  `de.json`: tab label ("My Invitations"), empty-state text
  (`settings.users.noInvites` / equivalent for the new tab), `revoke` action
  label, `statusRevoked`, confirm-dialog copy for revoke.
- [ ] 7.2 Double-check no "inventations"/"invokation" typos slipped into any
  new key, label, or comment — the shipped terms are invitation(s) and
  revoke/revocation.

## 8. Verify

- [ ] 8.1 `cd backend && gofmt -l . && go vet ./... && go test ./...`
  (including `internal/storage/postgres` integration tests if a local
  Postgres is reachable).
- [ ] 8.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 8.3 Manual pass per the Migration Plan's verification steps in
  `design.md`.
- [ ] 8.4 Update `backend/AGENTS.md` (new store/service/handler methods,
  the `invites` schema additions) and `frontend/AGENTS.md` (the new My
  Invitations tab and route) if their existing descriptions of these areas
  would otherwise go stale.

## 9. Spec sync

- [ ] 9.1 Apply this change's `specs/authentication`,
  `specs/user-administration`, `specs/web-client-settings` deltas onto
  `openspec/specs/*/spec.md` (the `openspec` CLI is unavailable in this
  environment, as for prior changes — apply by hand).

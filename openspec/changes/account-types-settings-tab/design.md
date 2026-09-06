## Context

`internal/account` (from `add-accounts-entries`) owns both `accounts` and
the `account_types` lookup they reference, in the repo's four-file shape.
Today `Type` is just `{ID, Name, CreatedAt}`; writes are admin-gated
(`requireAdmin` in `handler.go`), reads are open to any authenticated user.
`Account.Create`/`Update` already call `store.TypeExists` to reject a
`type_id` that doesn't exist. `DeleteType` hard-deletes, `409 ErrTypeInUse`
if a non-deleted account still references it — there is no `deleted_at` on
`account_types` and this change doesn't add one; delete stays hard,
blocked-when-in-use, unchanged.

This change extends that lookup (title/description/disabled) and its
assignment rules, and adds the first admin management UI for it — today
`account_types` has no settings-tab surface at all, only the read-only
dropdown in `AccountForm.tsx`.

## Goals / Non-Goals

**Goals:**

- An admin can create/edit an account type with a title and a description,
  disable it (reversibly) to stop new assignments without touching existing
  accounts, and delete it outright once nothing references it.
- An account whose current type becomes disabled keeps working normally
  until its owner next edits it; that edit must swap in a live type before
  anything else about it can change.
- Admin-only **Account Types** settings tab, consistent with the existing
  Users tab's list/create/confirm-dialog conventions.

**Non-Goals:**

- Soft-delete / `deleted_at` on `account_types` — delete stays hard,
  rejected (`409`) when in use, same as today. Disable is the only
  reversible off-ramp this change adds.
- Reordering, categorizing, or any hierarchy for account types — they stay
  a flat list, same as today.
- Any change to `internal/entry` or `internal/category` — this change
  touches only `internal/account`'s account-type slice and its settings UI.
- Bulk actions (multi-disable, bulk delete) on the new tab.
- Forcing every account with a disabled type to be fixed up proactively
  (e.g. a banner, a background job) — the rule bites lazily, on that
  account's next edit, not before.

## Decisions

### Decision: rename `name` → `title`, add optional `description`

Matches `Account`'s own `Title`/`Description` pair and the ask's vocabulary.
This is a breaking field rename in the JSON contract
(`AccountType.name` → `AccountType.title`); acceptable here the same way
earlier changes have renamed fields/requirements in place (e.g.
`invite-revocation`'s requirement rename) — there's no API versioning story
in this codebase and no external consumers beyond this repo's own frontend,
regenerated in the same change.

`description` is optional, following the same pattern as `Account.Description`
(empty string omitted from JSON via `omitempty`, no separate "explicitly
clear" tri-state needed — unlike `Account.ClosingDate`, a type's description
has no meaningful "unset vs. empty" distinction to preserve).

### Decision: `disabled` blocks new assignment, not existing ones — same shape as `Account.Disabled`

`account_types` gains `disabled boolean NOT NULL DEFAULT false`, toggled via
admin-only `POST /api/account-types/{id}/disable` / `/enable` — literally
the same pattern `Account.Disable`/`Enable` already established (a
reversible flag, a dedicated pair of endpoints, no effect on existing
data). `ListTypes` keeps returning every type regardless of `disabled` —
the admin tab needs to show disabled types to manage them, and any code
that resolves an account's type for display (accounts list, entries) needs
its title/description even after the type is disabled.

### Decision: the reselection rule lives in `account.Service`, not just the frontend

The stated rule — "an account on a disabled type must have its type
reselected on its very next edit, whatever else the edit changes" — is a
real invariant, not just UX polish, so it's enforced in
`account.Service.Update`, not left to the frontend to volunteer:

```
Update(ctx, ownerID, id, upd):
    current := store.Get(...)
    effectiveTypeID := upd.TypeID if upd.TypeID != nil else current.TypeID
    type := resolveType(effectiveTypeID)   // ErrNotFound if missing
    if type.Disabled:
        return ErrTypeDisabled            // 422 — reject the WHOLE update
    ...proceed with the rest of validateUpdate + store.Update
```

Consequences of doing it this way:

- Editing *any* field (say, just `financial_institute`) on an account whose
  type is currently disabled is rejected until the caller also supplies a
  `type_id` for a live type — there's no way to "sneak past" the rule by
  avoiding the `type_id` field.
- Once an update succeeds with a non-disabled `type_id`, the account is
  clean again; later edits follow the ordinary rule (only a `type_id` that
  is itself disabled gets rejected, same as `Create`).
- `Create` uses the same `resolveType` check, simpler since there's no
  "current" to fall back on: the submitted `type_id` must resolve to an
  existing, non-disabled type.
- A new sentinel, `ErrTypeDisabled`, distinct from the existing
  `ErrInvalidValue` (missing/malformed field, `400`) and `ErrNotFound`
  (no such type, `404` via the existing `TypeExists`-style check) — mapped
  to `422`, the same status `entry.ErrAccountDisabled` already uses for
  "well-formed request, rejected by a business rule."
- `store.TypeExists(ctx, id) (bool, error)` is replaced by
  `store.GetType(ctx, id) (Type, error)` (`ErrNotFound` if missing) — the
  service needs the type's `Disabled` flag, not just its existence, so a
  richer lookup replaces the boolean one rather than living alongside it.

### Decision: pre-emptive UI filtering, reactive delete error — no new "in use" field on the list endpoint

`AccountForm.tsx`'s type dropdown filters `GET /api/account-types` to
non-disabled types for a *new* account. For editing an existing account,
the dropdown starts from the same filtered list; if the account's current
`type_id` isn't in it (i.e. it's disabled), the current value renders as an
extra, non-selectable option so the form doesn't look like it silently lost
the account's type, but the field validates as invalid until a different,
live type is chosen — reusing the same required-field validation
(`validate()` in `AccountForm.tsx`) that already blocks submission on an
empty `title`.

The settings tab's Delete action, by contrast, stays reactive: it calls
`DELETE /api/account-types/{id}` and surfaces the `409` inline on failure,
matching how the Users tab already handles its own confirm-then-error flow
for disable/enable/delete. No `account_count`/`in_use` field is added to
`GET /api/account-types` — the existing hard-delete-if-unused behavior
already tells the caller definitively via the response status, and every
other admin list in this codebase (users, invites) works the same reactive
way.

## Migration Plan

1. `backend/internal/storage/postgres/migrations/0013_account_type_fields.sql`:
   `ALTER TABLE account_types RENAME COLUMN name TO title`, `ADD COLUMN
   description text`, `ADD COLUMN disabled boolean NOT NULL DEFAULT false`.
   Forward-only, no data loss — existing rows' `name` values become their
   `title`, `description` starts `NULL`, `disabled` starts `false`.
2. `internal/account`: update `account.go` (`Type` struct), `store.go`
   (`Store` interface: `GetType` replaces `TypeExists`, add
   `SetTypeDisabled`; new sentinel `ErrTypeDisabled`), `service.go`
   (`resolveType` helper used by `Create`/`Update`/the new
   `DisableType`/`EnableType`), `handler.go` (request/response bodies gain
   `description`, `disabled`; two new routes).
3. `internal/storage/memory` and `internal/storage/postgres`: implement
   `GetType`, `SetTypeDisabled`; update `CreateType`/`UpdateType` for the
   new fields.
4. `openapi/openapi.yaml`, then `go generate ./...` and
   `pnpm generate:api`.
5. Frontend: new settings tab + `AccountForm.tsx` dropdown changes + i18n.

## Verification

- Backend: unit + handler tests per the assignment rule (`Create`/`Update`
  reject a disabled `type_id`; `Update` with an untouched-but-now-disabled
  `type_id` is rejected; a subsequent `Update` with a live `type_id`
  succeeds and un-sticks the account; `DeleteType` unchanged — still `409`
  when in use, `204` otherwise); `internal/storage/postgres` integration
  tests for the migration and the new store methods.
- Frontend: `pnpm lint && pnpm exec tsc && pnpm build`; manual pass —
  create/edit/disable/enable/delete a type as an admin, confirm a non-admin
  never sees the tab (including a direct `/settings/account-types` link),
  confirm the account form excludes disabled types for a new account and
  forces reselection when editing an account on a disabled one.

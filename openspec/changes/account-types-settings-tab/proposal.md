## Why

`internal/account` already ships an `account_types` lookup (from
`add-accounts-entries`, not yet archived/synced into main specs): a flat,
admin-writable, everyone-readable table with just `id`/`name`/`created_at`,
hard-deleted (`409`) when a non-deleted account still references it. There
is no admin UI for it at all — the only place a type is visible today is the
account create/edit dropdown. Admins have no way to add a type with any
explanation attached, retire one that turned out to be a mistake without
either deleting it (blocked once it's in use) or leaving it forever
selectable, or rename/describe what a type is for.

## What Changes

- `account_types` gains `title` (renamed from `name`) and an optional
  `description`.
- `account_types` gains a reversible `disabled` flag
  (`POST /api/account-types/{id}/disable` / `/enable`, admin-only, mirroring
  `Account.Disable`/`Enable`). A disabled type stays exactly as-is for every
  account already assigned to it — visible, editable, usable for new
  entries. It only changes what *new type assignments* accept:
  - `POST /api/accounts` MAY NOT set `type_id` to a disabled type.
  - `PATCH /api/accounts/{id}` MAY NOT leave (or set) `type_id` pointing at
    a disabled type — if the account's current type is disabled, that PATCH
    must include a `type_id` for a different, non-disabled type, whatever
    else it changes. Once such a PATCH succeeds, the account is "clean"
    again and later edits aren't required to touch `type_id`.
- Deleting an account type is unchanged: hard `DELETE`, rejected `409` if
  any non-deleted account still references it — disabling is the new
  reversible off-ramp; delete stays the one-way, only-when-truly-unused
  cleanup action.
- New admin-only **Account Types** settings tab: list every type (title,
  description, Active/Disabled), create, edit (title/description),
  disable/enable, delete — cloned from the existing Users tab's
  list-plus-confirm-dialog pattern. The account create/edit form's type
  dropdown only offers non-disabled types for a new selection; if the
  account being edited currently holds a disabled type, that type still
  renders (so the form doesn't look like it lost data) but can't be
  resubmitted as-is.

## Capabilities

### Modified Capabilities

- `accounts` (backend, not yet synced into main specs — still pending in
  `add-accounts-entries`): the account-type requirements gain `title`/
  `description`, `disabled`, and the reselection-on-edit rule described
  above.
- `web-client-settings`: a new admin-only **Account Types** tab, alongside
  the existing Common / My Invitations / Users tabs.

## Impact

- **Code**:
  - `backend/internal/storage/postgres/migrations/0013_account_type_fields.sql`
    — rename `account_types.name` to `title`, add `description text`, add
    `disabled boolean NOT NULL DEFAULT false`.
  - `backend/internal/account/`: `Type` gains `Title` (renamed from `Name`),
    `Description`, `Disabled`; `Store` gains `GetType`,
    `SetTypeDisabled`; `CreateType`/`UpdateType` take title+description;
    `Service.Create`/`Update` reject a disabled type on any (re)assignment,
    including the current-type-is-disabled-and-untouched case on `Update`;
    new sentinel `ErrTypeDisabled` (`422`).
  - `backend/internal/httpapi/account.go`: register `ErrTypeDisabled` →
    `422`; new routes `POST /api/account-types/{id}/disable` and `/enable`.
  - `frontend/src/routes/settings.account-types.tsx` (new tab route),
    `frontend/src/routes/settings.tsx` (tab nav entry, admin-only),
    `frontend/src/components/AccountForm.tsx` (dropdown filtering +
    disabled-current-type handling).
  - New i18n keys in `frontend/src/i18n/locales/{en,de}.json`.
- **API contract**: `openapi/openapi.yaml` — `AccountType` gains
  `title` (renamed from `name`), `description`, `disabled`;
  `AccountTypeWrite` gains `description`; new paths
  `POST /api/account-types/{id}/disable` / `/enable`. Regenerate
  `backend/openapi.yaml` and `frontend/src/api/schema.d.ts` in the same
  change.
- **Spec**: deltas on `accounts` and `web-client-settings`.

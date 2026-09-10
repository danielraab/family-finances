## Context

`internal/account` models an account's type as a per-user `account_types`
lookup table: `id, owner_id, title, description, disabled, created_at`,
referenced by `accounts.type_id` (uuid, `NOT NULL`, FK). Around it sit six
HTTP endpoints (`GET/POST /api/account-types`,
`PATCH/DELETE /api/account-types/{id}`, `.../disable`, `.../enable`), an
`AccountType`/`AccountTypeWrite` schema pair, a `Type` domain struct,
seven `Store` methods (`ListTypes`, `GetType`, `CreateType`, `UpdateType`,
`SetTypeDisabled`, `DeleteType`, `SeedDefaultTypes`), a `SeedDefaults`
new-user hook wired through `auth`, a `DefaultTypeTitles` constant, a
`resolveAssignableType` service helper, two sentinels (`ErrTypeInUse`,
`ErrTypeDisabled`), a full Settings management tab
(`routes/settings.account-types.tsx`), and a `type_id` carve-out in the
sharing model (a shared `owner`-tier caller may edit every account field
*except* `type_id`, enforced in `Service.Update`, the handler, `AccountForm`'s
`typeLocked` prop, and three spec scenarios).

For this app a type is a label like "Checking". The lookup's disable/rename/
delete-protection machinery is not worth its surface area. Migrations run
forward-only on boot (`internal/storage/postgres/migrate.go`), numbered
`0001`..`0020`; the latest is `0020_account_sharing.sql`. Per `0015`'s note
there is no production data whose meaning must be preserved, only referential
validity.

## Goals / Non-Goals

**Goals:**

- Replace `accounts.type_id` (uuid FK) with `accounts.type` (`text`,
  `NOT NULL`), backfilled from `account_types.title`; drop the
  `account_types` table.
- `type` is required, stored trimmed of surrounding whitespace, otherwise
  verbatim (no case-fold, no canonical list).
- Repurpose `GET /api/account-types` to return `string[]` — the caller's
  distinct in-use `type` values — for client autocomplete.
- Delete the account-type CRUD API, the `Type` struct, the seven `Store`
  methods, the seeding hook, `resolveAssignableType`, and the two
  type-specific sentinels.
- Drop the sharing `type_id` carve-out: `type` becomes an ordinary
  `owner`-tier-editable field.
- Delete the Settings "Account Types" tab; rework `AccountForm`'s type
  control into a required text input with a suggestion list.
- Keep `openapi/openapi.yaml` the source of truth and regenerate both
  derived artifacts.

**Non-Goals:**

- No data migration tooling for the (nonexistent) production data beyond
  the in-migration backfill.
- No server-side vocabulary, validation against a list, normalization
  beyond trimming, or length cap on `type` (a defensive upper bound is an
  open question, below).
- No translation of the default type labels — they stay English, matching
  today's seeding behavior.
- No change to how `type` is displayed on account surfaces (it already
  renders as a plain string; only the lookup of a title by id goes away).

## Decisions

### 1. One forward-only migration `0021` does column add + backfill + drops

`0021_account_type_text.sql`:

```sql
ALTER TABLE accounts ADD COLUMN type text;
UPDATE accounts a SET type = t.title
  FROM account_types t WHERE t.id = a.type_id;
ALTER TABLE accounts ALTER COLUMN type SET NOT NULL;
ALTER TABLE accounts DROP COLUMN type_id;
DROP TABLE account_types;
```

Every `accounts` row has a non-null `type_id` FK today, so the backfill is
total and `SET NOT NULL` cannot fail. Column is dropped before the table so
no `CASCADE` is needed. Forward-only, consistent with every prior migration
— no down migration.

*Alternative considered:* keep `account_types` as a soft-deprecated table.
Rejected — leaving a dead table and its Go code half-wired is worse than a
clean drop, and there is no data to preserve.

### 2. `GET /api/account-types` returns `string[]`, computed from `accounts`

The path is kept (it is literally "the account types") but the response
becomes `["Checking","Savings",...]`: `SELECT DISTINCT type FROM accounts
WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY lower(type)`. A new
`Store.ListInUseTypes(ctx, ownerID) ([]string, error)` backs it;
`Service.ListInUseTypes` just forwards. Handler stays on the existing
`GET /api/account-types` registration.

*Alternative considered:* a fresh path `GET /api/accounts/types`. Rejected —
Go's `ServeMux` handles `GET /api/accounts/types` vs `GET /api/accounts/{id}`
fine (static segment wins), but reusing the freed, already-documented path
keeps the client diff and the OpenAPI diff smaller. The semantics are close
enough ("the set of account types in play") that the rename would be noise.

*Scope note:* the endpoint reads only the caller's own accounts, not shared
ones — suggestions are for classifying *your* accounts, and a shared
account's type is set by whoever edits it from whatever they type.

### 3. `type` validation: trim, then require non-empty — nothing else

In `validateNew`/`validateUpdate`: `strings.TrimSpace(in.Type) == "" →
ErrInvalidValue`. The service (or handler) applies the trim to the stored
value so DB and API agree. No case handling, no dedupe-on-write (two
accounts may both be `"checking"` and `"Checking"` — the suggestion list
will show both, which is acceptable and self-correcting as the user picks
one).

*Open question below:* whether to also bound length (e.g. 100 chars) purely
as a DB-sanity guard. Current lean: skip it, matching "only trim the text".

### 4. Sharing carve-out is deleted, not relaxed

`Service.Update` drops the `if upd.TypeID != nil && current.OwnerID !=
callerID { return ErrForbidden }` branch. `type` is then covered by the
existing `AtLeast(PermissionOwner)` check like every other field. Frontend
`AccountForm` loses the `typeLocked` prop entirely and the
"assigned separately, conditionally" body-construction dance collapses to a
plain field. `accounts.$accountId.edit.tsx` stops passing `typeLocked`.
This reverts the corresponding parts of the shipped `account-sharing`
change; the spec deltas in this change record that.

### 5. Default labels live in the frontend

`DefaultTypeTitles` (Go) is deleted. A frontend constant — e.g.
`DEFAULT_ACCOUNT_TYPES` in `AccountForm.tsx` or a small `lib/` module —
holds `["Checking","Savings","Cash","Credit Card","Loan","Investment"]`.
The form's `<datalist>` (or existing chip-suggestion pattern, mirroring
`financial_institute`) is the union of that constant and the
`GET /api/account-types` response, deduped by exact match.

*Alternative considered:* a backend endpoint serving the defaults too.
Rejected — they are static UI seed text, never enforced, and belong with
the client that renders them.

### 6. `auth` new-user hook wiring loses one entry

`main.go` builds `account.Service` and passes `svc.SeedDefaults` into
`auth.WithNewUserHooks(...)`. That argument is removed. `Service.SeedDefaults`
and `Store.SeedDefaultTypes` are deleted. No other hook is affected
(categories keep theirs). `auth`'s `NewUserHook` interface is unchanged.

## Risks / Trade-offs

- **[Type-name drift: "Checking" vs "checking" vs "Chequing"]** → The
  suggestion list (defaults + in-use values) nudges toward consistency;
  this is a single-family app where the same person creates most accounts.
  Accepted as a deliberate trade for removing the lookup.
- **[Losing rename-propagation: renaming a type no longer updates every
  account at once]** → Was a real feature of the lookup. Mitigation: a user
  edits each account's `type` directly (few accounts per family), or a
  future bulk action if it's ever missed. Out of scope here.
- **[The shipped `account-sharing` change had `type_id`-carve-out tasks and
  scenarios]** → This change's spec deltas explicitly MODIFY the
  `account-sharing` "Four permission tiers" requirement and REMOVE the
  `accounts` "`type_id` reassignable only by the real owner" requirement,
  so the archived history stays coherent. Implementation must also strip
  the frontend `typeLocked` path so no dead prop remains.
- **[CI `contract` job fails on any drift between `openapi/openapi.yaml`
  and the two generated files]** → Regenerate `backend/openapi.yaml`
  (`cd backend && go generate ./...`) and `frontend/src/api/schema.d.ts`
  (`cd frontend && pnpm generate:api`) in the same commit.
- **[i18n coverage CI]** → Removing `settings.accountTypes.*` and trimming
  `accounts.form.type*` must be done across every locale file or the
  `i18n-coverage` job fails.
- **[Old rows whose `type_id` pointed at a since-deleted... nothing]** →
  `type_id` is `NOT NULL` with an FK; no dangling ids possible, backfill
  covers 100% of rows.

## Migration Plan

1. Land migration `0021` + all backend code in one change; it runs
   forward-only on the next boot / deploy. No coordinated data step.
2. Ship backend and frontend together (single Docker image) so the API
   shape change (`type_id` → `type`, endpoint response type) and the
   client land atomically.
3. Rollback: revert the release image. Because `0021` is destructive
   (drops `type_id` and `account_types`), a true rollback after it has run
   requires restoring from a DB backup — call this out in the PR. Given
   "no production data to preserve," acceptable.

## Open Questions

- **Length bound on `type`?** Lean: none (only trim), matching the stated
  decision. If a defensive cap is wanted, `≤ 100` chars in
  `validateNew`/`validateUpdate` and a `varchar`-free `text` column
  (Postgres `text` needs no length) is the cheapest guard.
- **Suggestion UI: `<datalist>` vs the existing focus-chip pattern used for
  `financial_institute`?** The chip pattern is already in `AccountForm` and
  spec'd; reusing it is consistent. `<datalist>` is less code. Defer to
  implementation, but the spec is written to allow either.
- **Should `GET /api/account-types` also fold in types from accounts shared
  *to* the caller?** Current decision: no (owned accounts only). Revisit if
  users report the autocomplete feels empty on heavily-shared setups.

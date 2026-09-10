## Context

`display_name` is already present in the data model (`users.display_name`,
nullable `text`, migration `0002_auth.sql`), the domain types (`auth.User`,
`auth.AdminUser`, `auth.InviteInviter`), the OpenAPI `User` schema, the
generated frontend types, and the rendering UI (`SidebarUser`, the initials
`Avatar`, and the `web-client-auth` monogram spec). It has **no write path**:
`resolveIdentity`'s create branch builds `NewUser{Email: in.email}` with no
name, and `internal/oidcauth` does not parse a `name` claim. Every account's
`display_name` is therefore `NULL` today, and the UI already degrades to the
email address.

This change adds the write path in two forms — a self-service `PATCH` and an
OIDC re-sync — following patterns the repo already uses: `internal/settings`
for a partial-update preferences endpoint, `NormalizeEmail`/`ValidateEmail` for
input hygiene helpers, and the four-file domain-package shape.

## Goals / Non-Goals

**Goals:**

- A signed-in user can set and clear their own full name from the settings
  page, with the sidebar reflecting it immediately.
- OIDC users get a name automatically, kept in sync with the provider.
- Zero new database migration; zero new external dependency.
- The OpenAPI document stays the source of truth; both generated artifacts are
  regenerated in the same change.

**Non-Goals:**

- No admin endpoint to edit another user's name (the `user-administration`
  listing already shows `display_name`; editing it is out of scope).
- No separate `/settings/profile` route — the field lives on the renamed
  index tab.
- No avatar image upload, no additional profile fields (bio, pronouns, …).
- No attempt to preserve a self-set name across an OIDC sign-in — the provider
  wins, by decision.
- No client-side enforcement of the character rule as the source of truth; the
  backend validates and the client may mirror it only for instant feedback.

## Decisions

### D1: The endpoint is `PATCH /api/auth/me` in `internal/auth`

`display_name` lives on the `users` table, which `internal/auth` owns.
`internal/settings` deliberately imports `internal/auth` only for
`UserFromContext` and never touches `auth.Store` or a driver, and `user_settings`
is a different table — folding a name field into `PUT /api/settings` would
break that boundary. `PATCH` (not `PUT`) because the body is a partial update of
the user resource; `auth` has no prior verb here to be consistent with, and
`PATCH` is the honest semantic.

The response body is the same `meResponse` shape `GET /api/auth/me` returns
(`User` with the raw `language` folded in), so the frontend can replace its
whole `useAuth` user object with the result.

*Alternative considered:* a dedicated `/api/auth/profile` resource — rejected as
ceremony for one field; `me` is already "the current user".

### D2: Validation — trim, then `^[\p{L} .'-]{0,150}$`

A `NormalizeDisplayName` helper trims (mirroring `NormalizeEmail`). A
`ValidateDisplayName` helper checks the trimmed value against the regex; Go's
`regexp` (RE2) supports `\p{L}`. Empty is valid and maps to a cleared name.
Anything else returns a new `ErrInvalidDisplayName` sentinel, added to
`auth.Sentinels` and mapped in `httpapi/auth.go` to `400 Bad Request` — the
same status `auth.ErrInvalidEmail` and `settings.ErrInvalidValue` use, so the
error table stays uniform.

*Alternative considered:* `422 Unprocessable Entity` — rejected for consistency
with the existing auth/settings validation errors, which all use `400`.

### D3: Storage — one nullable-setting method

`auth.Store` gains:

```go
SetUserDisplayName(ctx context.Context, userID string, name *string) (User, error)
```

`name == nil` clears (writes `NULL`); a non-nil pointer writes that exact
(already-trimmed) string. PostgreSQL: `UPDATE users SET display_name = $2
WHERE id = $1 AND deleted_at IS NULL RETURNING <userCols>`, passing `nil` for
the clear case so pgx binds `NULL`; `ErrNotFound` when no row. The in-memory
store mirrors it. `userCols` already `COALESCE(display_name, '')`, so reads are
unaffected.

*Alternative considered:* a general `UpdateUser(patch)` — rejected as
over-general for a single field; add breadth when a second editable field
appears.

### D4: OIDC re-sync happens in the service, after identity resolution

`internal/oidcauth` gains a `Name string` field on its claims struct, populated
from the ID token's `name` claim. `Service.CompleteOIDC`, after it has the
resolved `user` and before issuing the session, computes
`SanitizeDisplayName(claims.Name)` — trim, delete every rune outside
`[\p{L} .'-]`, cap at 150 — and, when the result is non-empty and differs from
the stored value, calls `SetUserDisplayName(ctx, user.ID, &sanitized)` and uses
the returned user. An empty claim or empty sanitised result is a no-op. This
runs for all resolution outcomes (create, sign-in, attach) because it keys off
`kind == oidc`, not the link action.

*Alternatives considered:*
- *Validate-and-skip on failure* — rejected: real provider names routinely
  contain commas, parentheses, digits (`Jane Doe III`), so "skip on any
  invalid char" would leave most OIDC users nameless.
- *Store the raw claim unmodified* — rejected: it would let arbitrary provider
  text (including markup) into a field the client renders; sanitising to the
  same character class the self-service field enforces keeps one safe alphabet.
- *Seed only on first sign-in (`ActionCreate`)* — rejected by the explicit
  decision that the provider is authoritative on every sign-in.

### D5: Frontend — rename the tab, add a blur-saved field, lift the user into context

- `settings.tsx`: the tab entry's label key becomes `settings.tabs.profile`.
- `en.json` / `de.json`: rename `settings.tabs.common` → `settings.tabs.profile`
  and the `settings.common.*` field namespace → `settings.profile.*`; add
  `settings.profile.name` (label) and `settings.profile.nameError` (revert
  message). All references in `settings.index.tsx` and `settings.tsx` updated.
- `settings.index.tsx`: a `Name` field at the top of the existing list. It
  keeps a local draft, saves on blur when the draft differs from the saved
  value, calls `api.PATCH("/api/auth/me", { body: { display_name } })`, and on
  failure reverts the draft and shows the error — the same shape as the
  existing `default_currency` field. Language/timezone/currency/decimals are
  untouched and keep calling `PUT /api/settings`.
- `AuthProvider.tsx`: `AuthContextValue` gains `setUser(user: User): void` (or
  `refreshUser()`); today only `logout` mutates the context. `settings.index`
  calls it with the `PATCH` response so `SidebarUser` re-renders with the new
  name and monogram. No new `GET /api/auth/me` round-trip.

### D6: API contract regeneration

Add the `patch` operation to `/api/auth/me` in `openapi/openapi.yaml` with a
`ProfileUpdate` request schema (`{ display_name: string, maxLength: 150 }`) and
a `200` referencing the existing `User` schema plus a `400`. Then
`cd backend && go generate ./...` (syncs `backend/openapi.yaml`) and
`cd frontend && pnpm generate:api` (regenerates `schema.d.ts`), committed
together. The CI `contract` job then covers it with no extra wiring.

## Risks / Trade-offs

- **A linked email+OIDC user edits their name, then it silently reverts on next
  OIDC sign-in** → Documented as explicit behaviour in the `authentication`
  spec (scenario "Provider name claim overrides a self-set name"); the frontend
  copy for the field can note that SSO manages the name for SSO users. Not a
  bug.
- **`SanitizeDisplayName` mangles comma-formatted provider names** (`Doe, Jane`
  → `Doe Jane`) → Accepted by decision (strip over skip); the result is still a
  reasonable display string. Collapsing doubled spaces left by stripped runs is
  a cheap nicety worth doing in the helper.
- **Every OIDC sign-in now issues a write** when the name changed → Guarded by a
  value comparison so an unchanged name is a no-op; the cost is one `UPDATE` on
  genuine change, negligible against the sign-in flow's existing DB work.
- **i18n key rename touches every `settings.common.*` reference** → Mechanical;
  the `i18n-coverage` CI job catches a missed key in either locale.
- **`\p{L}` in the browser mirror needs the `u` flag** → Only affects the
  optional client-side hint; the backend remains authoritative, so a mismatch
  degrades to a server `400` with revert, which is already handled.
- **Rollback**: revert the commit. No migration ran; `display_name` values
  written while live remain valid data the read path already tolerated.

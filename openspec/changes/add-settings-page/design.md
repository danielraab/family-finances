## Context

Nothing in the app today is gated on being signed in — every route renders
for anonymous and authenticated visitors alike (`/login` only redirects
*away from itself* when already authenticated). `is_admin` exists on `users`
but, per the `authentication` spec, "gates nothing yet." There is no listing
endpoint for users or invites — `internal/cli`'s `admin grant|revoke|list` is
the only way to see or change admin status today, and invites can only be
*created* (`POST /api/auth/invites`, open to any authenticated user), never
listed or revoked.

`frontend/AGENTS.md` already anticipates this: "A per-user, account-level
language setting is a planned follow-up once a settings surface exists — it
will take priority over browser detection, not replace it." This change is
that follow-up, plus the admin user-management surface the task also asked
for.

Constraints carried over from the rest of the codebase:

- No web framework/ORM on the backend; package-per-noun with a `Store`
  interface the domain package declares and `storage/postgres` implements.
- No `os.Getenv` outside `internal/config`; no HTTP concerns in domain code.
- Frontend is a static SPA — no server-side gating, only client-side route
  guards backed by the (already-enforced) backend authorization.
- Forward-only SQL migrations are the established way schema evolves in this
  project (`backend-persistence`); a fixed, developer-defined set of settings
  is exactly the case a typed table (not a generic key/value store) is built
  for.

## Goals / Non-Goals

**Goals:**

- A signed-in visitor can reach `/settings`, set language/timezone/default
  currency, and see the change apply immediately (language: to the running
  app; timezone/currency: captured for future features to consume).
- An admin can see every user and every invitation, invite a new user from
  the UI, and disable or soft-delete a user, with the target losing access
  immediately.
- `is_admin` becomes a real authorization boundary for the first time.

**Non-Goals:**

- Consuming the stored timezone/default-currency anywhere else in the app
  (formatting dates, converting amounts) — there is no accounts/transactions
  feature yet for them to affect. This change only captures and persists the
  preference.
- Un-deleting a soft-deleted user, revoking/resending an invitation, or any
  server-side protection against an admin locking themselves out (disabling
  or deleting the last remaining admin) — explicitly accepted, see Risks.
- A canonical ISO-4217 currency list or IANA timezone list shipped/validated
  server-side beyond shape checking — see Decisions.
- Changing who may *create* an invite (`POST /api/auth/invites` stays open to
  any authenticated user, per the current `authentication` spec); this change
  only adds the admin-only Users tab as a (for now, only) UI surface for it.

## Decisions

### Decision: `user_settings` — one row per user, all-nullable columns, no DB defaults

```sql
CREATE TABLE user_settings (
    user_id          uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    language         text CHECK (language IN ('en', 'de')),
    timezone         text,
    default_currency char(3),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
```

Typed columns, not a generic `settings(key, user_id, value)` table: the set
of settings is small, fixed, and developer-defined (not user- or
plugin-defined at runtime), so the EAV flexibility of a generic table buys
nothing here, while costing DB-level type/value safety (`CHECK` per column),
one-row-per-user reads, and joinable admin queries. Adding a fourth setting
later is an ordinary migration (`ADD COLUMN`), which this project already
treats as the normal way schema changes happen.

No row is created at signup. Every column is nullable and carries no SQL
`DEFAULT`; "no row" and "row present but a column is `NULL`" are handled
identically — both resolve to the hardcoded Go-side default. A write is
`INSERT ... ON CONFLICT (user_id) DO UPDATE`, touching only the column(s)
being changed, so nothing needs to hook the account-creation transaction
(which already does bootstrap-admin work under an advisory lock).

Resolution — and the hardcoded defaults themselves — lives once, in
`internal/settings`'s service layer:

```go
const (
    DefaultLanguage        = "en"
    DefaultTimezone        = "UTC"
    DefaultDefaultCurrency = "EUR"
)

type Settings struct {
    Language        string
    Timezone        string
    DefaultCurrency string
}
```

`GET /api/settings` returns this resolved, always-populated struct — both for
the Common tab and for any future backend consumer that just wants "this
user's timezone" without re-implementing the null-coalescing.

### Decision: language precedence needs the *raw* preference, not the resolved one — carried on `/api/auth/me`, not `/api/settings`

The resolved `GET /api/settings` response can't distinguish "the user
explicitly chose English" from "no preference set, defaulted to English" —
once the default is substituted, both look identical. That distinction is
exactly what the browser-detection fallback needs: an authenticated visitor
with *no* language preference must still get German from
`i18next-browser-languagedetector` if their browser says so; only an
*explicit* preference should override it.

Rather than adding a "was this explicit?" flag to `/api/settings`, the raw
(nullable) `language` preference rides on the `User` record returned by
`GET /api/auth/me` — a request the client already makes once, on mount, via
`AuthProvider`. This keeps `/api/settings` purely "resolved settings for the
form and for future consumers" and keeps the i18n bootstrap path to the
single request it already had:

```
AuthProvider mounts
  → GET /api/auth/me
      → 200, user.language = "de"   → i18n.changeLanguage("de") — explicit wins
      → 200, user.language = null   → leave browser-detected language as-is
      → 401 / anonymous             → browser detection only, as today
```

Accepted trade-off: `i18next-browser-languagedetector` resolves
synchronously on load (before first paint), while `/api/auth/me` is async —
so an authenticated visitor whose browser and account language disagree can
see a brief flash of the browser-detected language before the account
preference applies. Fixing that would mean blocking first paint on a network
round trip, which this project's static-SPA/no-flash-of-wrong-*theme*
precedent solved with a synchronous pre-paint script — not available here
since the preference lives server-side, not in `localStorage`. Accepted as-is;
flagged in Risks.

### Decision: timezone and currency validation is shape-only, not a canonical list

`timezone` is validated with `time.LoadLocation` (Go stdlib, backed by the
system/embedded tzdata) — rejects garbage, accepts any real IANA zone, no
list to maintain. `default_currency` is validated as three uppercase ASCII
letters (the ISO-4217 shape) via a regexp, not checked against an actual
currency list — this app has no accounts/transactions feature yet to give a
"real" currency list meaning, and inventing one now would be validating
against a table nothing else reads. Revisit once a feature actually consumes
`default_currency`.

The frontend's timezone `<select>` is populated from the browser's
`Intl.supportedValuesOf("timeZone")` (broadly supported in evergreen
browsers) rather than a shipped static list, so there's nothing to keep in
sync with tzdata updates. Default currency is a plain text input,
client-side validated to the same three-letter shape as the backend.

### Decision: disable vs. (soft) delete — both revoke sessions immediately, delete is one-way in this UI

Both actions immediately delete the target's `sessions` rows (not just flip a
flag and let the sliding-expiry lookup catch it later) — the task was
explicit that this must be immediate, and a targeted `DELETE FROM sessions
WHERE user_id = $1` is a two-line addition next to the existing session
store. The distinction between them:

- **Disable** (`disabled = true`) — reversible via **Enable** in the same UI.
  Blocks sign-in and (belt-and-suspenders, alongside the immediate session
  revocation) is also re-checked by the auth middleware on every request, in
  case a session row somehow outlives the revocation.
- **(Soft) delete** (`deleted_at = now()`) — no "undelete" affordance in this
  change (Non-Goal). Also blocks sign-in and is also re-checked by the
  middleware, identically to disabled.

A soft-deleted user is excluded from `GET /api/auth/users`'s default listing
(deleting reads as "remove from view," not "annotate as removed"); the row
itself is retained for future referential integrity (e.g., "invited by" on
existing invites keeps resolving).

No guard prevents an admin from disabling or deleting themselves, or the
last remaining admin — explicit, per the task. The frontend still requires
an explicit confirmation step before either action (ordinary
destructive-action hygiene, not a substitute for a server-side guard that
was deliberately not built), with copy that calls out self-targeting
specifically ("You're about to disable your own account — you'll be signed
out immediately").

### Decision: admin endpoints live in `internal/auth`, not a new package

`users`, `sessions`, and `invites` are already `internal/auth`'s nouns; the
new admin endpoints (`GET /api/auth/users`, `GET /api/auth/invites`,
disable/enable/delete) are additional operations over the same Store, not a
new domain. A new `internal/admin` package would just be a thin HTTP layer
re-importing `auth`'s types for no benefit, and would cut against "package
per noun, not per layer."

Authorization is a simple `if !user.IsAdmin { WriteError(w, ErrForbidden)
}` guard at the top of each new handler — the first real use of `is_admin`
as an authorization check anywhere in the backend.

### Decision: settings save immediately per-field, no separate Save button

Matches the existing `ThemeSwitch` interaction (pick a value, it applies) and
avoids dirty-state tracking across three independent fields for no real
benefit — there's no multi-field validation that depends on the fields being
submitted together.

## Risks / Trade-offs

- **Admin self-lockout is possible by design.** An admin can disable or
  delete themselves, including as the last admin, with no server-side
  guardrail — recovery in that case is out-of-band (direct DB access or,
  once `internal/cli`'s `admin grant` is pointed at a re-created account,
  restarting from a new bootstrap). Accepted per explicit instruction; the
  frontend confirmation copy is the only mitigation.
- **Brief language flash for authenticated visitors.** See the precedence
  decision above — the account-language override can't apply before first
  paint the way the theme system's pre-paint script does, because it depends
  on an async request. Accepted; not solved by this change.
- **No currency/timezone list to validate against.** Shape-only validation
  means `default_currency = "ZZZ"` is accepted even though it's not a real
  currency. Acceptable now since nothing consumes the value yet; revisit
  when an accounts/transactions feature gives it real meaning.
- **Revoking sessions on disable/delete is a full-table-scan-free but
  unindexed-by-purpose delete** (`sessions_user_id_idx` already exists, per
  `0002_auth.sql`, so this is actually cheap — noted only because it's new
  write traffic on a path that previously only ever inserted/extended).

## Migration Plan

1. `backend/internal/storage/postgres/migrations/0003_user_settings.sql` —
   create `user_settings`.
2. `backend/internal/storage/postgres/migrations/0004_user_administration.sql`
   — `ALTER TABLE users ADD COLUMN disabled boolean NOT NULL DEFAULT false,
   ADD COLUMN deleted_at timestamptz`.
3. `backend/internal/settings/` (four-file shape) +
   `storage/{memory,postgres}` implementations.
4. Extend `internal/auth`: disabled/deleted checks in the middleware and both
   sign-in callbacks; new admin handlers; `Store` gains the listing/lifecycle
   methods.
5. `openapi/openapi.yaml` additions, then `go generate ./...` (backend) and
   `pnpm generate:api` (frontend) to sync the generated copies.
6. Frontend: `/settings` route + tabs, `SidebarUser` menu item, i18n wiring,
   new translation keys (English first, per `web-client-i18n`).
7. Manual verification: sign in as a non-admin (no Users tab, `/settings/users`
   redirects away), sign in as an admin (full Users tab), disable a user in
   another session and confirm their next request is `401`, set a language
   preference and reload to confirm it beats browser detection.

Rollback: revert the commit; the two migrations are additive
(new table, new nullable/defaulted columns) so a rollback needs no
compensating migration — a later `DROP TABLE user_settings` /
`ALTER TABLE users DROP COLUMN` would only be needed to fully undo the schema,
not to make the revert safe.

## Open Questions

None blocking — the three open threads from discovery (i18n precedence order,
settings-table shape, admin lifecycle semantics) were resolved in
conversation before this design was written; the shape-only currency/timezone
validation and the no-self-lockout-guard behavior are documented above as
accepted, not open.

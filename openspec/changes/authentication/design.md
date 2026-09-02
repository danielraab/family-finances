## Context

Family Finances has no users. This change adds the first domain package,
`internal/auth/`, and with it the first real `Store` implementation on the
Postgres layer that `add-postgres-persistence` establishes (this change depends
on that one).

Requirements from the product owner, settled during exploration:

- Two sign-in methods — email magic link over SMTP, and one OAuth/OIDC provider
  — both configured from environment variables.
- One person, one account: sign in with the provider first and add magic-link
  later, or the reverse, resolving to the *same* `user`.
- A native Android client is coming, so the HTTP API must authenticate a mobile
  app, not only a same-origin browser.
- Registration is policy-controlled by env: signup on/off, an allowed
  email-domain list, invites (any authenticated user), and a zero-users
  bootstrap that mints the first admin.
- `is_admin` is only a flag for now. `internal/cli` is introduced now with
  `admin grant|revoke|list`.

Backend constraints still apply: `main.go` is wiring only; `os.Getenv` only in
`internal/config`; domain packages import no `httpapi`, no `storage`, no
driver, and expose an `http.Handler` mounted under `/api/<noun>/`; sentinel
errors map to status codes in `internal/httpapi/respond.go`.

"Standard library only" is already retired by `add-postgres-persistence`. This
change adds two more vetted dependencies, justified below.

## Goals / Non-Goals

**Goals:**

- `internal/auth/` in the four-file shape, owning users, identities, sessions,
  magic-link tokens, invites, and OIDC login state.
- Opaque session tokens, hashed at rest, carried by an `HttpOnly` cookie for
  browsers and as `Authorization: Bearer` for everything else — one `sessions`
  table, one middleware.
- Magic-link sign-in over SMTP with no account enumeration.
- Single-provider OIDC (authorization code + PKCE + `nonce`), `id_token`
  verified with `coreos/go-oidc`.
- Account linking by verified email, plus explicit link while authenticated.
- Env-driven registration policy, domain allow-list, invites, bootstrap admin.
- `internal/cli` with `admin grant|revoke|list`, dispatched from `main.go`.
- Auth middleware in `internal/httpapi`; `/api/auth/me` and `/api/auth/logout`.

**Non-Goals:**

- More than one OIDC provider. The route/config shape leaves room to add later.
- Any `is_admin`-gated behavior, roles, or permissions.
- Ledger, sharing, households/families, or any product resource.
- A frontend login page or route gating — a later change consumes this API.
- Native mobile OIDC (`POST id_token` from an Android SDK), device/session
  management UI, "log out everywhere", refresh tokens, password login, WebAuthn,
  rate limiting beyond what enumeration-resistance requires.
- Email deliverability concerns (SPF/DKIM/DMARC) — operator's responsibility.

## Decisions

### D1: One `user`, many `identity` rows, linked by verified email

`users` holds the account; `identities` holds each way to sign in
(`kind ∈ {email, oidc}`). Linking key is a **verified** email:

| Situation | Outcome |
| --- | --- |
| Magic link, unknown address | create `user` + `email` identity |
| Magic link, known address | sign in as that user |
| OIDC, `email_verified`, address matches a user | attach `oidc` identity to that user |
| OIDC, `email_verified`, no match | create `user` + `oidc` identity |
| OIDC, email not verified / absent, no session | do **not** auto-merge; create separate user or require explicit link |
| Authenticated, completing any flow for an unattached identity | attach to the current user |

Magic link is definitionally proof of the address. OIDC auto-link requires the
provider's `email_verified: true`; otherwise the safe move is a distinct
account or an explicit link performed while already signed in.

*Alternatives considered:* a single `users` row carrying nullable
`oidc_sub`/`email` columns — collapses as soon as a person has two OIDC
providers or multiple emails, and makes "add a method" an `UPDATE` with
awkward uniqueness. Separate identity rows is the standard shape.

### D2: Opaque session tokens, not JWT

32 bytes from `crypto/rand`, base64url. Store `sha256(token)` in `sessions`;
look up by hashing the presented value and comparing with
`subtle.ConstantTimeCompare`. Sliding expiry (`AUTH_SESSION_TTL`, bumped when a
request lands on a session past half its life) under a hard cap
(`AUTH_SESSION_MAX_TTL`).

Opaque + server-stored gives instant revocation (logout = `DELETE`), a natural
"sessions" list later, and no signing-key management. JWTs solve stateless
verification at scale — irrelevant for an app that already hits Postgres on
every request, and their revocation story would need a denylist that reintroduces
the state we'd have skipped.

*Alternatives considered:* JWT access + opaque refresh (more moving parts for
no benefit here); cookie-only server session (no clean bearer story for
Android).

### D3: Cookie for browsers, bearer for everyone else, one table

The sign-in callbacks branch on the client:

- Browser (`Accept: text/html`, or no explicit JSON): `302` to a safe in-app
  path + `Set-Cookie: ff_session=<token>; HttpOnly; SameSite=Lax; Path=/`
  (`Secure` unless `AUTH_COOKIE_SECURE=false` for local http).
- JSON client (`Accept: application/json` or an explicit `client=api` marker):
  `200 {session_token, user}`, no cookie.

The middleware reads `Authorization: Bearer` first, then the cookie. `SameSite=Lax`
plus a same-origin static frontend covers CSRF for state-changing `fetch`
calls (a cross-site `fetch` won't send the cookie); `Lax` still allows the
top-level GET navigations that the magic-link and OIDC redirects rely on.

The Android magic-link path later: the callback URL is registered as an App
Link so the OS routes it to the app, which calls the callback with
`Accept: application/json` and stores the returned token in the Keystore. The
OIDC redirect for mobile later becomes a `familyfinances://` deep link carrying
the token. Neither is built now, but the JSON branch is the seam that makes
them cheap.

### D4: Magic link — always `200`, send only when permitted

`POST /api/auth/email/start` returns `200` unconditionally. An email is
dispatched only if: the address belongs to an existing user, OR
`AUTH_SIGNUP_ENABLED` and the domain passes `AUTH_ALLOWED_EMAIL_DOMAINS`, OR an
unexpired invite exists. Token: 32 random bytes, stored hashed in
`magic_link_tokens`, TTL `AUTH_MAGIC_LINK_TTL` (default 15m), consumed
atomically (`UPDATE ... SET consumed_at = now() WHERE token_hash = $1 AND
consumed_at IS NULL RETURNING ...`).

### D5: SMTP via `net/smtp` first

A `Mailer` interface is declared in `internal/auth`; `internal/mailer`
implements it over `net/smtp` (STARTTLS on 587, PLAIN auth, a hand-built
`text/plain` + `text/html` MIME body). `SMTP_TLS ∈ {starttls, implicit, none}`.
`wneessen/go-mail` is adopted **only if** an operator's provider needs implicit
TLS on 465, XOAUTH2, or the MIME hand-assembly becomes unmaintainable — the
interface keeps that swap contained. Starting stdlib honors "add deps only when
necessary".

*Alternatives considered:* `wneessen/go-mail` up front (nicer API, 465 support
— deferred as not yet necessary); an HTTP email API (Postmark/SES) — an
external service dependency the product owner didn't ask for; the spec says
SMTP.

### D6: OIDC — `coreos/go-oidc/v3` + `golang.org/x/oauth2`, one provider

`internal/oidcauth` wraps the two libraries behind an `OIDCClient` interface
declared in `internal/auth`. At startup, `oidc.NewProvider(ctx, OIDC_ISSUER)`
runs discovery; `oauth2.Config` is built from `OIDC_CLIENT_ID` /
`OIDC_CLIENT_SECRET` / discovered endpoints / `OIDC_SCOPES` (default
`openid email profile`) / `AUTH_BASE_URL + /api/auth/oidc/callback`.

`GET /api/auth/oidc/start`: generate `state`, `nonce`, PKCE verifier; persist
in `oidc_login_state` (short TTL); `302` to the provider's auth URL with
`code_challenge` (`S256`).
`GET /api/auth/oidc/callback`: reject unknown/expired `state`; exchange code
with the stored verifier; `provider.Verifier(&oidc.Config{ClientID: ...}).
Verify(ctx, rawIDToken)` (signature via auto-cached JWKS, `iss`, `aud`, `exp`);
check `nonce`; read `sub`, `email`, `email_verified`; resolve/create identity
per D1; establish session per D3.

Hand-rolling JWKS + RS256/ES256 verification is exactly the security-critical
code `go-oidc` exists to own; with the stdlib constraint gone there's no reason
to. `x/oauth2` is a `golang.org/x` module. Single provider only — the
`oidc_login_state` row can carry a `provider` column now so multi-provider is a
later additive change, not a migration.

*Alternatives considered:* the `/userinfo`-over-TLS trick to avoid `id_token`
verification (was only attractive under stdlib-only); raw hand-rolled JWT
verification (unjustifiable risk); a heavier all-in-one auth framework
(violates "no framework").

### D7: Registration policy resolution order

On a sign-in that would create an account, evaluate in order:

1. `users` empty → **bootstrap**: allow, set `is_admin = true`, skip 2–4.
2. Valid unexpired invite for the address → allow, `is_admin = false`, skip
   3–4 (invites bypass the domain list).
3. `AUTH_SIGNUP_ENABLED=false` → deny.
4. `AUTH_ALLOWED_EMAIL_DOMAINS` non-empty and domain not listed → deny.
5. Otherwise allow, `is_admin = false`.

Invite-enabled logic: `signup_enabled || AUTH_INVITE_ENABLED`. So invites are
always on while signup is on; `AUTH_INVITE_ENABLED=false` only bites once signup
is off; both off = fully closed. The domain list is checked only here (account
creation), never on returning-user sign-in.

### D8: Invites in their own table

`invites(id, email, invited_by, token_hash, created_at, expires_at,
accepted_at, accepted_user_id)`. `POST /api/auth/invites` (auth required, any
user) creates a row + sends an email with a link to
`GET /api/auth/invites/accept?token=…`. Acceptance verifies+consumes the token,
creates the account bypassing signup-disabled and the domain list, then
establishes a session per D3. Own table (not a `magic_link_tokens` row with a
`purpose`) so "list/revoke pending invites" and "who invited whom" are
first-class later — the product owner asked for a dedicated table.

### D9: `internal/cli` now; `main.go` dispatches subcommands

`main.go` grows from one special-case (`os.Args[1] == "healthcheck"`) to a
small dispatch: `healthcheck` stays in `healthcheck.go`; `admin` →
`cli.Admin(ctx, args, store)`. `internal/cli` builds a config + Postgres pool +
the auth store and calls service methods (`SetAdmin(email, bool)`,
`ListAdmins()`). The `backend-package-architecture` delta broadens "wiring
only" to name subcommand dispatch and pins command logic to `internal/cli`.
Keeping `healthcheck` where it is preserves the "root package = main.go +
healthcheck.go + embed.go" invariant.

### D10: Auth middleware in `internal/httpapi`, storage-free

`internal/httpapi` may resolve a user but must not import `internal/storage` or
a driver. It takes an `Authenticator` interface (`Authenticate(ctx, token)
(User, error)`) in `Deps`, satisfied by `auth.Service`. The middleware pulls
the token (bearer, then cookie), calls `Authenticate`, and stores the user on
the request context; `requireAuth` wraps routes that need it (`/api/auth/me`,
`/api/auth/logout`, `POST /api/auth/invites`). Sign-in routes stay open. No
product route is protected — none exist yet.

### D11: Schema

New migration `0002_auth.sql` (after `add-postgres-persistence`'s `0001`):

```
users(id uuid pk default gen_random_uuid(), email citext not null unique,
      display_name text, is_admin boolean not null default false,
      created_at timestamptz not null default now())

identities(id uuid pk default gen_random_uuid(),
      user_id uuid not null references users(id) on delete cascade,
      kind text not null check (kind in ('email','oidc')),
      email citext, email_verified boolean not null default false,
      provider text, subject text,
      created_at timestamptz not null default now(),
      unique (kind, email),            -- email identities
      unique (provider, subject))      -- oidc identities

sessions(id uuid pk default gen_random_uuid(),
      user_id uuid not null references users(id) on delete cascade,
      token_hash bytea not null unique,
      client text not null check (client in ('web','api')),
      user_agent text, ip inet,
      created_at timestamptz not null default now(),
      last_seen_at timestamptz not null default now(),
      expires_at timestamptz not null)

magic_link_tokens(token_hash bytea primary key, email citext not null,
      expires_at timestamptz not null, consumed_at timestamptz)

invites(id uuid pk default gen_random_uuid(), email citext not null,
      invited_by uuid not null references users(id),
      token_hash bytea not null unique,
      created_at timestamptz not null default now(),
      expires_at timestamptz not null,
      accepted_at timestamptz,
      accepted_user_id uuid references users(id))

oidc_login_state(state text primary key, nonce text not null,
      pkce_verifier text not null, provider text not null,
      return_to text,
      created_at timestamptz not null default now(),
      expires_at timestamptz not null)
```

`citext` is enabled by `0001` (from `add-postgres-persistence`). `uuid` PKs via
`gen_random_uuid()` (built into PG 13+). Partial-unique variants of the
`identities` constraints (`WHERE kind = 'email'` / `WHERE kind = 'oidc'`) are
an implementation choice for the task phase.

## Risks / Trade-offs

- **Hand-built SMTP/MIME in `internal/mailer`** → kept behind the `Mailer`
  interface; swap to `wneessen/go-mail` is a contained change if a provider
  needs 465/XOAUTH2. Covered by a test using a local SMTP capture.
- **`SameSite=Lax` as the primary CSRF control** → adequate for a same-origin
  static SPA; the magic-link/OIDC callbacks are GET top-level navigations which
  `Lax` permits. If a future cross-site embedding need appears, add a
  double-submit token then.
- **Auto-linking by verified email trusts the provider's `email_verified`** →
  we only auto-link on `true`; unverified/absent falls back to a separate
  account or an explicit authenticated link. A provider lying about
  `email_verified` is outside this threat model (you chose to trust it as an
  IdP).
- **No rate limiting on `/api/auth/email/start` beyond enumeration-resistance**
  → a determined actor can trigger many emails to valid addresses. Acceptable
  for a family instance; `golang.org/x/time/rate` keyed by IP+address is a
  small follow-up if abused.
- **Bootstrap mode (`users` empty ⇒ signup forced open)** → a race between
  first-boot and the first sign-in could in principle create two "first"
  accounts; the `users`-empty check runs inside the account-creation
  transaction and only the row that finds the table empty gets `is_admin`.
  Operationally, close signup or invite-only immediately after first sign-in.
- **Two new dependencies (`go-oidc`, `x/oauth2`)** → both are the de-facto
  standard, widely used, and `x/oauth2` is a `golang.org/x` module; the
  alternative is hand-rolled token verification, which is worse.
- **`internal/httpapi` now depends on an `Authenticator`** → interface only, no
  storage/driver import; the `backend-package-architecture` delta records this.

## Migration Plan

1. Land after `add-postgres-persistence` is applied (pool + migration runner
   exist).
2. `0002_auth.sql` migration; auth `Store` interface in `internal/auth/store.go`
   + Postgres implementation in `internal/storage/postgres/`.
3. `internal/config` fields for all `AUTH_*`, `SMTP_*`, `OIDC_*` vars +
   `.env.example`.
4. `internal/auth` domain + service; `internal/mailer`; `internal/oidcauth`.
5. `internal/httpapi` auth middleware + `Authenticator` in `Deps`.
6. `internal/cli` + `main.go` `admin` dispatch and auth wiring.
7. Docs: `backend/AGENTS.md`, `backend/README.md`, `openspec/config.yaml`.

**Rollback:** no product feature depends on auth yet; `git revert` the change.
The `0002_auth.sql` tables can be dropped (forward-only runner means a
`0003_drop_auth.sql` if a deployed environment already applied `0002`).

## Open Questions

_None blocking._ Deferred by explicit scope: native mobile OIDC endpoint,
session-management endpoints / "log out everywhere", rate limiting, and any
`is_admin`-gated behavior — each a later change.

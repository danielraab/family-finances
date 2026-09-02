## Why

The backend has no concept of a user. Every endpoint is anonymous, and the
frontend cannot show anything personal. Family Finances needs sign-in before any
ledger, sharing, or budget feature can exist.

Two sign-in methods are required — **email magic link** (SMTP) and
**OAuth/OIDC** — and a person must be able to use either against the *same*
account: sign in with the identity provider first and add magic-link later, or
the reverse. A native Android client is planned, so the same HTTP API must
authenticate a mobile app, not just a same-origin browser.

This change depends on `add-postgres-persistence` for the database.

## What Changes

- Add `internal/auth/` — the first domain package, following the four-file
  shape (`auth.go` / `service.go` / `store.go` / `handler.go`). It owns users,
  identities, sessions, magic-link tokens, invites, and OIDC login state, and
  exposes an `http.Handler` mounted at `/api/auth/`.
- **Session transport: opaque bearer token.** 256-bit random token from
  `crypto/rand`, stored only as a SHA-256 hash. Delivered to browsers as an
  `HttpOnly; Secure; SameSite=Lax` cookie and to other clients as the token in
  the JSON response body for use as `Authorization: Bearer`. One `sessions`
  table; the auth middleware accepts either transport. No JWT.
- **Magic-link sign-in.** `POST /api/auth/email/start` always returns `200`
  (no account enumeration) and sends a link only when the address is permitted.
  The link hits `GET /api/auth/email/callback`, which for a browser sets the
  cookie and redirects, and for an API client returns `{ session_token, user }`.
  Mail is sent over SMTP configured entirely from environment variables, via
  `net/smtp` (a vetted mail library is adopted only if implicit-TLS/XOAUTH2/MIME
  needs force it).
- **OIDC sign-in (one provider).** `GET /api/auth/oidc/start` →
  authorization-code flow with `state`, `nonce`, and PKCE →
  `GET /api/auth/oidc/callback` verifies the `id_token` with
  `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2` and establishes a
  session. Provider issuer and client credentials come from environment
  variables. Multi-provider support is deliberately out of scope; the seam is
  left for an additive change.
- **Account linking by verified email.** Magic link inherently proves the
  address. OIDC links to an existing user only when the provider asserts
  `email_verified`. An authenticated user can also explicitly add a second
  identity from the API.
- **Registration policy, all env-driven.** Signup can be disabled
  (`AUTH_SIGNUP_ENABLED`); an allowed-email-domain list
  (`AUTH_ALLOWED_EMAIL_DOMAINS`, empty = any) gates *account creation* only,
  never login of an existing user. Invites can be disabled
  (`AUTH_INVITE_ENABLED`) but are always on while signup is enabled; any
  authenticated user may invite; invited addresses bypass the domain list.
  When `users` is empty the system is in bootstrap mode: signup is forced open
  and the first account becomes an admin.
- **Invites** get their own table and an email-delivered acceptance link.
- **Admin.** `users.is_admin` boolean only — no admin-gated behavior yet
  (that arrives with future models/frontend). A new `internal/cli` package
  provides `admin grant <email>`, `admin revoke <email>`, and `admin list`,
  dispatched from `main.go` alongside `healthcheck`.
- **Auth middleware** in `internal/httpapi`: resolves bearer-or-cookie to a
  user in the request context; `/api/auth/me` and `/api/auth/logout` require it.
  No route is otherwise protected yet (there are no product routes).
- New dependencies: `github.com/coreos/go-oidc/v3`, `golang.org/x/oauth2`.
- Docs: `backend/AGENTS.md` (new `internal/auth/`, `internal/cli/`,
  `internal/mailer/`, `internal/oidcauth/`; auth env vars), `backend/.env.example`,
  `backend/README.md`, `openspec/config.yaml`.

## Capabilities

### New Capabilities

- `authentication`: user accounts with multiple linkable sign-in identities;
  opaque-token sessions carried by cookie or bearer; magic-link sign-in over
  SMTP; single-provider OIDC sign-in; env-driven registration, domain
  allow-list, invites, and bootstrap-admin policy; the `internal/cli` admin
  commands; and the `/api/auth/*` HTTP surface plus the auth middleware.

### Modified Capabilities

- `backend-package-architecture`: `main.go`'s wiring responsibilities gain a
  second CLI subcommand path (`admin …` → `internal/cli`) beside the existing
  `healthcheck` short-circuit, and `internal/httpapi` gains an authentication
  middleware that reads a request-scoped user — recorded so the "wiring only"
  and "routing owned by httpapi" requirements stay accurate.

## Impact

- New: `backend/internal/auth/`, `backend/internal/cli/`,
  `backend/internal/mailer/`, `backend/internal/oidcauth/`,
  `backend/internal/storage/postgres/` auth `Store` implementation +
  migrations, `backend/internal/httpapi/middleware.go` auth middleware.
- `backend/main.go` — `admin` subcommand dispatch; build mailer + OIDC client +
  auth store/service/handler; mount `/api/auth/`; inject the middleware.
- `backend/internal/config/` — session, SMTP, OIDC, and registration-policy
  fields.
- `backend/go.mod` / `go.sum` — `coreos/go-oidc/v3`, `golang.org/x/oauth2`
  (and transitively `golang.org/x/crypto`, `golang.org/x/oauth2` deps).
- New tables: `users`, `identities`, `sessions`, `magic_link_tokens`,
  `invites`, `oidc_login_state`.
- `backend/.env.example`, `backend/AGENTS.md`, `backend/README.md`,
  `openspec/config.yaml`.
- Frontend: unchanged in this proposal (a login page and gating consume this
  API in a later change).

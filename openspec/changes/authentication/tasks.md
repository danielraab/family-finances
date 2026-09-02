## 1. Dependencies

- [ ] 1.1 Add `github.com/coreos/go-oidc/v3` and `golang.org/x/oauth2` to
  `backend/go.mod`; `go mod tidy`; confirm no cgo and
  `CGO_ENABLED=0 go build .` still works.

## 2. Config

- [ ] 2.1 Add fields to `config.Config` for `AUTH_BASE_URL`,
  `AUTH_SESSION_TTL` (default `720h`), `AUTH_SESSION_MAX_TTL` (default `2160h`),
  `AUTH_COOKIE_SECURE` (default true), `AUTH_SIGNUP_ENABLED` (default true),
  `AUTH_ALLOWED_EMAIL_DOMAINS` (`[]string`, empty = any), `AUTH_INVITE_ENABLED`
  (default true), `AUTH_INVITE_TTL` (default `168h`), `AUTH_MAGIC_LINK_TTL`
  (default `15m`).
- [ ] 2.2 Add SMTP fields: `SMTP_HOST`, `SMTP_PORT` (default `587`),
  `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS` (default
  `starttls`; one of `starttls|implicit|none`).
- [ ] 2.3 Add OIDC fields: `OIDC_ISSUER`, `OIDC_CLIENT_ID`,
  `OIDC_CLIENT_SECRET`, `OIDC_SCOPES` (default `openid,email,profile`).
- [ ] 2.4 Parse durations, bools, and the comma-separated domain/scope lists in
  `internal/config`; keep all `os.Getenv` calls in this package.
- [ ] 2.5 Extend `config_test.go`: defaults when unset, parsing when set,
  domain-list splitting, invalid `SMTP_TLS` rejected.
- [ ] 2.6 Add every new var to `backend/.env.example` with comments.

## 3. Schema

- [ ] 3.1 Add `internal/storage/postgres/migrations/0002_auth.sql` creating
  `users`, `identities`, `sessions`, `magic_link_tokens`, `invites`,
  `oidc_login_state` per design §D11 (uuid PKs via `gen_random_uuid()`,
  `citext` emails, checks, FKs, unique constraints).
- [ ] 3.2 Add a migration-runner test asserting `0002` applies on top of `0001`
  and is idempotent.

## 4. internal/auth — domain

- [ ] 4.1 `internal/auth/auth.go`: `User`, `Identity` (kind email|oidc),
  `Session`, `Invite` types; email normalization (trim + lower); the
  linking-decision helper implementing the design §D1 table; validation.
- [ ] 4.2 `internal/auth/store.go`: `Store` interface (users, identities,
  sessions, magic-link tokens, invites, oidc login state) and sentinel errors
  (`ErrNotFound`, `ErrSignupDisabled`, `ErrDomainNotAllowed`,
  `ErrTokenInvalid`, `ErrTokenExpired`, `ErrTokenConsumed`, `ErrInviteInvalid`,
  `ErrIdentityConflict`).
- [ ] 4.3 Declare the side-effect interfaces the service needs: `Mailer`
  (`SendMagicLink`, `SendInvite`) and `OIDCClient` (`AuthCodeURL`,
  `Exchange`, `VerifyIDToken`).
- [ ] 4.4 `internal/auth/service.go`: `StartEmailLogin`, `CompleteEmailLogin`,
  `StartOIDC`, `CompleteOIDC`, `Authenticate(token)`, `Logout(token)`,
  `CreateInvite`, `AcceptInvite`, `LinkIdentity(userID, …)`, `SetAdmin`,
  `ListAdmins`.
- [ ] 4.5 Implement the registration resolution order (design §D7): bootstrap
  when `users` empty (+ `is_admin=true`), invite bypass, signup toggle, domain
  allow-list — checked only on account creation.
- [ ] 4.6 Implement session issue/verify: 32 bytes `crypto/rand`, base64url,
  store `sha256` only, constant-time lookup, sliding expiry bumped past
  half-life, absolute `AUTH_SESSION_MAX_TTL` cap.
- [ ] 4.7 Unit tests with `storage/memory` (see task 6): linking table cases,
  bootstrap admin, signup disabled, domain list, invite bypass, magic-link
  single-use + expiry, session sliding/cap, enumeration (start always ok).

## 5. internal/auth — HTTP handler

- [ ] 5.1 `internal/auth/handler.go`: `http.Handler` mounting
  `POST /api/auth/email/start`, `GET /api/auth/email/callback`,
  `GET /api/auth/oidc/start`, `GET /api/auth/oidc/callback`,
  `GET /api/auth/me`, `POST /api/auth/logout`, `POST /api/auth/invites`,
  `GET /api/auth/invites/accept`.
- [ ] 5.2 Client branching helper: browser → `302` + `Set-Cookie ff_session`
  (`HttpOnly; SameSite=Lax; Path=/`, `Secure` per config); JSON client
  (`Accept: application/json` or `client=api`) → `200 {session_token, user}`.
- [ ] 5.3 `email/start` always `200`; validate/parse body; delegate to service.
- [ ] 5.4 `logout` deletes the current session and, for a browser, clears the
  cookie; requires auth.
- [ ] 5.5 `invites` POST requires auth; returns `201` with the created invite
  (no token in the body).
- [ ] 5.6 Validate any post-login redirect target to a same-origin relative
  path; otherwise use a safe default (open-redirect guard).
- [ ] 5.7 Register auth sentinel errors → status codes in
  `internal/httpapi/respond.go` (`ErrSignupDisabled`/`ErrDomainNotAllowed` →
  `403`, token errors → `400`/`410`, `ErrIdentityConflict` → `409`).
- [ ] 5.8 Handler tests with `httptest` + `storage/memory` + stub `Mailer` /
  `OIDCClient`: each route's success and failure, cookie vs bearer branch,
  `401` paths.

## 6. storage/memory + storage/postgres

- [ ] 6.1 Implement the `auth.Store` interface in `internal/storage/memory` for
  domain/handler tests.
- [ ] 6.2 Implement `auth.Store` in `internal/storage/postgres` using
  `pgxpool` — parameterized queries, `RETURNING`, atomic token consume
  (`UPDATE … WHERE consumed_at IS NULL RETURNING …`), account-creation in one
  transaction with the `users`-empty bootstrap check.
- [ ] 6.3 Postgres `Store` integration tests (skip without `DATABASE_URL`):
  create/link identities, uniqueness conflicts, session lifecycle, invite
  accept, bootstrap-admin race check.

## 7. internal/mailer

- [ ] 7.1 `internal/mailer`: implement `auth.Mailer` over `net/smtp` —
  STARTTLS/implicit/none per `SMTP_TLS`, PLAIN auth, `text/plain` + `text/html`
  MIME body; build magic-link and invite URLs from `AUTH_BASE_URL`.
- [ ] 7.2 Test against a local in-process SMTP capture: correct `From`/`To`,
  subject, both MIME parts, link contains the token.

## 8. internal/oidcauth

- [ ] 8.1 `internal/oidcauth`: implement `auth.OIDCClient` with
  `coreos/go-oidc/v3` + `golang.org/x/oauth2` — discovery at construction from
  `OIDC_ISSUER`, `oauth2.Config` from client creds + `AUTH_BASE_URL` callback +
  scopes.
- [ ] 8.2 `StartOIDC` support: generate `state`/`nonce`/PKCE `S256`, build the
  auth-code URL with `code_challenge`.
- [ ] 8.3 `CompleteOIDC` support: exchange with the stored verifier, verify the
  `id_token` (JWKS signature, `iss`, `aud`, `exp`), check `nonce`, return
  `sub` / `email` / `email_verified`.
- [ ] 8.4 Tests with a stub OIDC provider (local HTTP server exposing discovery
  + JWKS + token endpoints): happy path, bad `state`, failed `id_token`
  verification, `email_verified=false` does not auto-link.

## 9. httpapi auth middleware

- [ ] 9.1 Add an `Authenticator` interface
  (`Authenticate(ctx, token) (auth.User, error)` — via a local alias to avoid
  importing storage) to `httpapi.Deps`.
- [ ] 9.2 `internal/httpapi/middleware.go`: middleware resolving the token
  (bearer header, then `ff_session` cookie), calling `Authenticate`, and
  placing the user on the request context; a `requireAuth` wrapper returning
  `401` when absent.
- [ ] 9.3 Confirm `internal/httpapi` imports no `internal/storage/...` and no
  driver (grep in a test or CI check).
- [ ] 9.4 Middleware tests: valid bearer, valid cookie, header-wins-over-cookie,
  missing/invalid → `401`, sign-in routes reachable without auth.

## 10. internal/cli + main.go

- [ ] 10.1 `internal/cli/cli.go`: `Admin(ctx, args) int` handling
  `grant <email>`, `revoke <email>`, `list`; builds config + pool + auth store,
  calls `SetAdmin` / `ListAdmins`; clear error + non-zero exit on unknown
  email; prints admin emails for `list`.
- [ ] 10.2 `backend/main.go`: dispatch `os.Args[1] == "admin"` →
  `os.Exit(cli.Admin(...))`, beside the existing `healthcheck` short-circuit.
- [ ] 10.3 `backend/main.go` wiring: construct `mailer`, `oidcauth`, auth
  `Store` (Postgres), `auth.Service`, `auth` handler; mount it via
  `httpapi.Deps`; pass the `Authenticator`.
- [ ] 10.4 `internal/cli` tests (skip without `DATABASE_URL`): grant then list,
  revoke, unknown-email error path.

## 11. Docs

- [ ] 11.1 `backend/AGENTS.md`: add `internal/auth/` (first domain package,
  four-file shape), `internal/cli/`, `internal/mailer/`, `internal/oidcauth/`
  to the layout; document the `admin` subcommand; list the new env vars;
  describe the opaque-token cookie/bearer session model.
- [ ] 11.2 `backend/README.md`: auth setup (SMTP + OIDC env), first-run
  bootstrap admin, `admin` CLI usage.
- [ ] 11.3 `openspec/config.yaml` context: note authentication (magic link +
  single OIDC provider, opaque sessions).

## 12. Verify

- [ ] 12.1 `gofmt -l backend/` clean; `go vet ./...` clean.
- [ ] 12.2 `go test ./...` passes with and without `DATABASE_URL`.
- [ ] 12.3 `grep -rn "os.Getenv" backend/` shows only `internal/config`.
- [ ] 12.4 Manual: fresh DB → first magic-link sign-in yields an admin;
  `/api/auth/me` works by cookie and by bearer; `admin list` shows the first
  user; second sign-in from a new address is non-admin; `AUTH_SIGNUP_ENABLED=false`
  blocks a third unless invited.
- [ ] 12.5 `openspec validate authentication` passes.

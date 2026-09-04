## MODIFIED Requirements

### Requirement: Admin flag and CLI administration

`users` SHALL carry an `is_admin` boolean. `is_admin` gates the
`user-administration` capability's endpoints (listing users and invitations,
disabling/enabling, and soft-deleting a user); no other behavior is gated on
it.

A new `internal/cli` package SHALL provide `admin grant <email>`,
`admin revoke <email>`, and `admin list`, dispatched from `main.go` as a
subcommand alongside the existing `healthcheck` probe. `grant` and `revoke`
SHALL operate on an existing user and report a clear error for an unknown
email. `list` SHALL print the email of every user with `is_admin = true`.

#### Scenario: Grant promotes an existing user

- **WHEN** `admin grant existing@example.com` is run
- **THEN** that user's `is_admin` becomes true and the command exits `0`

#### Scenario: Grant on unknown email errors

- **WHEN** `admin grant nobody@example.com` is run and no such user exists
- **THEN** the command prints an error and exits non-zero, creating nothing

#### Scenario: List shows admins

- **WHEN** `admin list` is run
- **THEN** it prints the email of each user whose `is_admin` is true

#### Scenario: Revoke demotes

- **WHEN** `admin revoke admin@example.com` is run for an admin user
- **THEN** that user's `is_admin` becomes false

### Requirement: Authentication middleware and session endpoints

`internal/httpapi` SHALL provide authentication middleware that resolves a
session from the `Authorization: Bearer` header or the `ff_session` cookie and
places the authenticated user on the request context, without itself importing
`internal/auth`'s storage or a driver. Routes that require authentication SHALL
respond `401` when no valid session is presented. The middleware SHALL also
respond `401` when the session's user is disabled (`disabled = true`) or
soft-deleted (`deleted_at` is not `NULL`), even if the session row itself has
not yet been removed.

`GET /api/auth/me` SHALL return the authenticated user as JSON, or `401`.
`POST /api/auth/logout` SHALL revoke the current session (delete its row) and,
for a browser, clear the `ff_session` cookie; it SHALL require authentication.
Unauthenticated requests to `/api/auth/*` sign-in routes SHALL remain
accessible.

#### Scenario: me returns the current user

- **WHEN** `GET /api/auth/me` is called with a valid session
- **THEN** the response is `200` with the user as JSON

#### Scenario: me without a session is unauthorized

- **WHEN** `GET /api/auth/me` is called with no session
- **THEN** the response is `401`

#### Scenario: logout revokes the session

- **WHEN** `POST /api/auth/logout` is called with a valid session and the same
  token is used again afterwards
- **THEN** the logout succeeds and the subsequent request is `401`

#### Scenario: Disabled or deleted user's session is rejected

- **WHEN** a request carries a session token whose user has `disabled = true`
  or a non-`NULL` `deleted_at`
- **THEN** the response is `401`, regardless of whether the session row
  itself still exists

### Requirement: Magic-link sign-in over SMTP

`POST /api/auth/email/start` SHALL accept an email address and SHALL always
respond `200` regardless of whether an account exists or an email is sent, to
prevent account enumeration. An email SHALL be sent only when the address is
permitted: it belongs to an existing, non-disabled, non-deleted user, OR
signup is enabled and the address passes the domain allow-list, OR the
address has an unexpired invite.

When an email is sent it SHALL contain a link to `GET /api/auth/email/callback`
carrying a single-use token of at least 256 bits of entropy, stored hashed,
expiring after `AUTH_MAGIC_LINK_TTL`. The callback SHALL verify and atomically
consume the token, establish the session, and respond per the browser or JSON
client rules above. A consumed or expired token SHALL be rejected. A token
whose address now belongs to a disabled or soft-deleted user SHALL also be
rejected, without establishing a session.

Mail SHALL be sent over SMTP configured entirely from environment variables
(`SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`,
`SMTP_TLS`). The implementation MAY use the standard library; a third-party
mail library MAY be introduced only if implicit-TLS, XOAUTH2, or MIME
requirements make the standard library insufficient.

#### Scenario: Start never reveals account existence

- **WHEN** `POST /api/auth/email/start` is called for an address with no
  account and no invite while signup is disabled
- **THEN** the response is `200` and no email is sent

#### Scenario: Permitted address receives a working link

- **WHEN** `POST /api/auth/email/start` is called for a permitted address
- **THEN** an email is sent whose link, when followed, signs the person in

#### Scenario: Token is single-use

- **WHEN** a magic-link token is used a second time
- **THEN** the second attempt is rejected and no session is created

#### Scenario: Expired token is rejected

- **WHEN** a magic-link callback is invoked after `AUTH_MAGIC_LINK_TTL` has
  elapsed
- **THEN** it is rejected and no session is created

#### Scenario: Disabled or deleted account cannot sign in

- **WHEN** a magic-link callback is completed for a token whose address
  belongs to a disabled or soft-deleted user
- **THEN** the callback is rejected and no session is created

### Requirement: OIDC sign-in with a single configured provider

The backend SHALL support exactly one OIDC provider, configured by
`OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_SCOPES`
(default `openid email profile`), and `OIDC_LABEL` (default `Single sign-on`, a
human-facing button label), discovered via the issuer's
`.well-known/openid-configuration`. An OIDC client SHALL be constructed only when
both `OIDC_ISSUER` and `OIDC_CLIENT_ID` are set; with either unset, OIDC sign-in
SHALL be treated as not configured and `GET /api/auth/config` SHALL report
`oidc: null`.

`GET /api/auth/oidc/start` SHALL begin the authorization-code flow with a
random `state`, a random `nonce`, and PKCE (`S256`), persisting them server-side
with a short TTL. `GET /api/auth/oidc/callback` SHALL reject a missing or
unknown `state`, exchange the code using the stored PKCE verifier, and verify
the returned `id_token` (signature via the provider JWKS, `iss`, `aud`, `exp`,
and `nonce`) using `github.com/coreos/go-oidc/v3` with `golang.org/x/oauth2`.
On success, and only when the resolved account is neither disabled nor
soft-deleted, it SHALL establish the session per the browser or JSON client
rules.

Support for more than one provider is out of scope; the route path and
configuration MAY be shaped to allow adding providers later without breaking
changes.

#### Scenario: Successful provider sign-in

- **WHEN** a person completes the provider login and returns to the callback
  with a valid code and matching `state`
- **THEN** the `id_token` is verified, an identity of kind `oidc` is resolved
  or created, and a session is established

#### Scenario: State mismatch is rejected

- **WHEN** the callback is invoked with a `state` that was not issued or has
  expired
- **THEN** the request is rejected and no session is created

#### Scenario: id_token failing verification is rejected

- **WHEN** the returned `id_token` fails signature, `iss`, `aud`, `exp`, or
  `nonce` verification
- **THEN** the callback fails and no session is created

#### Scenario: Issuer without a client id is treated as unconfigured

- **WHEN** the backend starts with `OIDC_ISSUER` set but `OIDC_CLIENT_ID` empty
- **THEN** no OIDC client is constructed, the `/api/auth/oidc/*` routes behave as
  not configured, and `GET /api/auth/config` returns `oidc: null`

#### Scenario: Disabled or deleted account cannot sign in via OIDC

- **WHEN** an OIDC callback resolves to an account that is disabled or
  soft-deleted
- **THEN** the callback is rejected and no session is created

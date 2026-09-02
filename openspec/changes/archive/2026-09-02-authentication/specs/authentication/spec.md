## ADDED Requirements

### Requirement: User accounts with linkable sign-in identities

The backend SHALL model a `user` account separately from the means used to sign
in to it. Each user SHALL have one or more `identity` records; every identity
SHALL be of kind `email` or `oidc` and SHALL belong to exactly one user. An
`email` identity SHALL carry the verified email address. An `oidc` identity
SHALL carry the provider issuer and the provider subject (`sub`) and SHALL be
unique on that pair. An `email` identity SHALL be unique on its address.

A user's `email` address SHALL be normalized (trimmed, lower-cased) before
storage and comparison.

#### Scenario: New person signs in and an account is created

- **WHEN** a person completes a sign-in flow for an email or provider subject
  that matches no existing identity, and account creation is permitted
- **THEN** a new `user` is created with exactly one `identity` recording that
  sign-in method

#### Scenario: Second method links to the same account

- **WHEN** a person who already has an account completes a second sign-in flow
  whose verified email matches their existing account
- **THEN** a new `identity` of the second kind is attached to that same `user`
  and no second account is created

#### Scenario: Explicit link while authenticated

- **WHEN** an authenticated user completes a sign-in flow for an identity not
  yet attached to any user
- **THEN** that identity is attached to the authenticated user's account

#### Scenario: Identity uniqueness is enforced

- **WHEN** a sign-in would attach an `oidc` identity whose `(issuer, subject)`
  already belongs to another user, or an `email` identity whose address already
  belongs to another user
- **THEN** the operation does not move or duplicate the identity; the person is
  signed in as the owning user

### Requirement: Sessions are opaque bearer tokens

A successful sign-in SHALL create a `session` row and issue a session token
that is at least 256 bits of `crypto/rand` entropy, encoded url-safe. The token
SHALL be stored only as its SHA-256 hash; the plaintext SHALL never be
persisted or logged. Session lookup SHALL hash the presented token and compare
in constant time. Tokens SHALL NOT be JWTs.

Each session SHALL have a sliding expiry (`AUTH_SESSION_TTL`, extended on use)
bounded by an absolute maximum age (`AUTH_SESSION_MAX_TTL`) after which it
SHALL be rejected regardless of activity.

#### Scenario: Token is unguessable and stored hashed

- **WHEN** a session is created
- **THEN** the database stores a SHA-256 hash, not the token
- **AND** the token has at least 256 bits of entropy

#### Scenario: Sliding expiry extends an active session

- **WHEN** an authenticated request arrives on a session older than half its
  TTL but within `AUTH_SESSION_MAX_TTL`
- **THEN** the session's expiry is pushed forward and the request succeeds

#### Scenario: Absolute cap ends a long-lived session

- **WHEN** a request arrives on a session whose age exceeds
  `AUTH_SESSION_MAX_TTL`
- **THEN** the session is rejected as expired even if it was used recently

### Requirement: Browser sessions are carried by a cookie

For a browser client, the sign-in callbacks SHALL set the session token in a
cookie named `ff_session` with `HttpOnly`, `SameSite=Lax`, `Path=/`, and
`Secure` when `AUTH_COOKIE_SECURE` is true, and SHALL respond with a redirect
to a safe in-app location. Sign-out SHALL clear this cookie.

#### Scenario: Callback sets the cookie and redirects

- **WHEN** a browser completes a magic-link or OIDC callback successfully
- **THEN** the response is a redirect and sets `ff_session` as
  `HttpOnly; SameSite=Lax; Path=/` (and `Secure` unless disabled by config)

#### Scenario: Cookie is never readable by script

- **WHEN** the `ff_session` cookie is set
- **THEN** it carries the `HttpOnly` attribute

### Requirement: API and mobile clients are carried by a bearer token

When a sign-in callback is invoked by a non-browser client (indicated by an
`Accept: application/json` request or an explicit client marker), it SHALL
return the session token and the user record in a JSON body instead of setting
a cookie. The authentication middleware SHALL accept a session token supplied
as `Authorization: Bearer <token>`, and SHALL prefer that header over the
cookie when both are present. The same `sessions` table backs both transports.

#### Scenario: JSON client receives the token in the body

- **WHEN** a client completes a sign-in callback with `Accept: application/json`
- **THEN** the response is `200` with a JSON body containing `session_token`
  and the authenticated `user`, and sets no cookie

#### Scenario: Bearer token authenticates a request

- **WHEN** a request carries `Authorization: Bearer <valid session token>`
- **THEN** it is authenticated as the session's user

#### Scenario: Header wins over cookie

- **WHEN** a request carries both a valid `Authorization: Bearer` token and a
  different valid `ff_session` cookie
- **THEN** the request is authenticated as the bearer token's session

### Requirement: Magic-link sign-in over SMTP

`POST /api/auth/email/start` SHALL accept an email address and SHALL always
respond `200` regardless of whether an account exists or an email is sent, to
prevent account enumeration. An email SHALL be sent only when the address is
permitted: it belongs to an existing user, OR signup is enabled and the address
passes the domain allow-list, OR the address has an unexpired invite.

When an email is sent it SHALL contain a link to `GET /api/auth/email/callback`
carrying a single-use token of at least 256 bits of entropy, stored hashed,
expiring after `AUTH_MAGIC_LINK_TTL`. The callback SHALL verify and atomically
consume the token, establish the session, and respond per the browser or JSON
client rules above. A consumed or expired token SHALL be rejected.

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

### Requirement: OIDC sign-in with a single configured provider

The backend SHALL support exactly one OIDC provider, configured by
`OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, and `OIDC_SCOPES`
(default `openid email profile`), discovered via the issuer's
`.well-known/openid-configuration`.

`GET /api/auth/oidc/start` SHALL begin the authorization-code flow with a
random `state`, a random `nonce`, and PKCE (`S256`), persisting them server-side
with a short TTL. `GET /api/auth/oidc/callback` SHALL reject a missing or
unknown `state`, exchange the code using the stored PKCE verifier, and verify
the returned `id_token` (signature via the provider JWKS, `iss`, `aud`, `exp`,
and `nonce`) using `github.com/coreos/go-oidc/v3` with `golang.org/x/oauth2`.
On success it SHALL establish the session per the browser or JSON client rules.

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

### Requirement: Account linking requires a verified email

An `oidc` sign-in SHALL be linked to an existing user by matching email only
when the provider asserts `email_verified` true for that address. When the
provider does not assert a verified email and the address matches an existing
user, the flows SHALL NOT silently merge accounts; the person MAY instead link
explicitly while authenticated. A magic-link sign-in SHALL always be treated as
proof of the address.

#### Scenario: Verified provider email links to existing account

- **WHEN** an OIDC sign-in returns `email_verified: true` for an address that
  already belongs to a user
- **THEN** the `oidc` identity is attached to that existing user

#### Scenario: Unverified provider email does not merge

- **WHEN** an OIDC sign-in returns `email_verified: false` (or omits it) for an
  address that already belongs to a user, and no one is authenticated
- **THEN** the `oidc` identity is not attached to that user by the automatic
  flow

#### Scenario: Magic link proves the address

- **WHEN** a person completes a magic-link sign-in for an address that already
  belongs to a user
- **THEN** they are signed in as that user

### Requirement: Signup toggle and first-user bootstrap

Account creation SHALL be gated by `AUTH_SIGNUP_ENABLED`. When it is false, no
new account SHALL be created except through invite acceptance. As an exception,
when the `users` table is empty the system SHALL behave as if signup were
enabled (bootstrap mode), and the first account created SHALL have
`is_admin = true`.

#### Scenario: Signup disabled blocks new accounts

- **WHEN** `AUTH_SIGNUP_ENABLED=false`, at least one user exists, and a person
  with no account and no invite attempts to sign in
- **THEN** no account is created

#### Scenario: First account bootstraps an admin

- **WHEN** the `users` table is empty and a person completes any sign-in flow
- **THEN** an account is created despite the signup setting, and its
  `is_admin` is true

#### Scenario: Bootstrap applies only to the first account

- **WHEN** a second account is created while `AUTH_SIGNUP_ENABLED=false` via an
  invite
- **THEN** its `is_admin` is false

### Requirement: Email-domain allow-list gates account creation only

`AUTH_ALLOWED_EMAIL_DOMAINS` SHALL be a comma-separated list of domains. When
non-empty, a new account SHALL be created only if the person's email domain is
in the list (case-insensitive). When empty, any domain SHALL be allowed. The
allow-list SHALL be checked only at account creation and SHALL NOT affect
sign-in of an already-existing user, nor invite acceptance.

#### Scenario: Disallowed domain cannot create an account

- **WHEN** `AUTH_ALLOWED_EMAIL_DOMAINS=example.com` and a person with
  `someone@other.org` and no invite attempts first-time sign-in
- **THEN** no account is created

#### Scenario: Existing user with now-disallowed domain still signs in

- **WHEN** a user whose email domain is not in a later-configured allow-list
  signs in
- **THEN** the sign-in succeeds

#### Scenario: Empty list allows any domain

- **WHEN** `AUTH_ALLOWED_EMAIL_DOMAINS` is unset and signup is enabled
- **THEN** a person at any domain can create an account

### Requirement: Invites

Any authenticated user SHALL be able to create an invite for an email address
via `POST /api/auth/invites`. Invites SHALL be stored in their own table with
the inviting user, the target email, a hashed single-use token, and an expiry
(`AUTH_INVITE_TTL`). An invite email SHALL be sent over SMTP with an acceptance
link.

Inviting SHALL be enabled whenever `AUTH_SIGNUP_ENABLED` is true. When signup is
false, inviting SHALL be governed by `AUTH_INVITE_ENABLED`; when both are false,
no new accounts can be created at all. Accepting a valid invite SHALL create
the account even when signup is disabled, and SHALL bypass the email-domain
allow-list. The invite token SHALL be single-use and rejected once consumed or
expired.

#### Scenario: Authenticated user invites someone

- **WHEN** an authenticated user posts a valid email to `/api/auth/invites`
  and inviting is enabled
- **THEN** an invite row is created and an acceptance email is sent

#### Scenario: Invite acceptance creates an account despite disabled signup

- **WHEN** `AUTH_SIGNUP_ENABLED=false` and a person follows a valid invite link
- **THEN** an account is created for that address

#### Scenario: Invite bypasses the domain allow-list

- **WHEN** an invited address is outside `AUTH_ALLOWED_EMAIL_DOMAINS` and the
  invite is accepted
- **THEN** the account is created

#### Scenario: Inviting can be disabled only when signup is disabled

- **WHEN** `AUTH_SIGNUP_ENABLED=true` and `AUTH_INVITE_ENABLED=false`
- **THEN** inviting is still enabled

#### Scenario: Fully closed instance

- **WHEN** `AUTH_SIGNUP_ENABLED=false` and `AUTH_INVITE_ENABLED=false`
- **THEN** `POST /api/auth/invites` does not send an invite and no new account
  can be created

#### Scenario: Invite token is single-use

- **WHEN** an invite acceptance link is followed a second time
- **THEN** it is rejected

### Requirement: Admin flag and CLI administration

`users` SHALL carry an `is_admin` boolean. This change SHALL NOT introduce any
behavior gated on `is_admin` beyond its being set by bootstrap and by the CLI;
admin-gated features arrive later.

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
respond `401` when no valid session is presented.

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

### Requirement: Authentication configuration comes from the environment

All authentication configuration SHALL be fields on `config.Config` populated in
`internal/config` from environment variables and documented in
`backend/.env.example`: `AUTH_BASE_URL`, `AUTH_SESSION_TTL`,
`AUTH_SESSION_MAX_TTL`, `AUTH_COOKIE_SECURE`, `AUTH_SIGNUP_ENABLED`,
`AUTH_ALLOWED_EMAIL_DOMAINS`, `AUTH_INVITE_ENABLED`, `AUTH_INVITE_TTL`,
`AUTH_MAGIC_LINK_TTL`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`,
`SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`, `OIDC_ISSUER`, `OIDC_CLIENT_ID`,
`OIDC_CLIENT_SECRET`, `OIDC_SCOPES`. No code outside `internal/config` SHALL
read these via `os.Getenv`.

`AUTH_BASE_URL` SHALL be used to build magic-link URLs and the OIDC
`redirect_uri`. Redirect targets derived from request input SHALL be validated
to be same-origin relative paths to prevent open redirects.

#### Scenario: Config is loaded once in internal/config

- **WHEN** the backend starts
- **THEN** every auth setting is read by `config.Load()` and passed as a value;
  `grep` for `os.Getenv` outside `internal/config` finds nothing

#### Scenario: Open-redirect attempt is neutralized

- **WHEN** a sign-in flow is given a post-login redirect target pointing at an
  external origin
- **THEN** the person is redirected to a safe in-app default instead

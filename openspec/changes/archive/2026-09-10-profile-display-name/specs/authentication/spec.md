## ADDED Requirements

### Requirement: A user can set their own display name

The backend SHALL expose `PATCH /api/auth/me`, requiring authentication, with a
JSON body `{ "display_name": <string> }`. It SHALL apply the value to the
authenticated user's `display_name` and respond `200` with the updated user in
the same shape as `GET /api/auth/me` (including the raw, unresolved
`language`). An unauthenticated request SHALL receive `401`.

Before storing, the backend SHALL trim leading and trailing whitespace from the
submitted value. The trimmed value SHALL be accepted only when it matches
`^[\p{L} .'-]{0,150}$` — at most 150 characters, each a Unicode letter, a
space, `.`, `'`, or `-`. A trimmed value that is the empty string SHALL be
accepted and SHALL clear the name (persisted as SQL `NULL`). Any other value
SHALL be rejected with `400` and the `auth.ErrInvalidDisplayName` sentinel, and
SHALL NOT change the stored name.

The persistence contract SHALL gain a single method to set (or clear, via a nil
value) a user's display name and return the updated user; both the in-memory
and PostgreSQL stores SHALL implement it.

#### Scenario: Setting a valid name

- **WHEN** an authenticated user sends `PATCH /api/auth/me` with
  `{ "display_name": "Jane O'Brien-Doe" }`
- **THEN** the response is `200`, its `display_name` is `"Jane O'Brien-Doe"`,
  and a subsequent `GET /api/auth/me` returns the same value

#### Scenario: Surrounding whitespace is trimmed

- **WHEN** an authenticated user sends `{ "display_name": "  Jane Doe  " }`
- **THEN** the stored and returned `display_name` is `"Jane Doe"`

#### Scenario: An empty value clears the name

- **WHEN** an authenticated user who has a display name sends
  `{ "display_name": "" }` (or a value that is only whitespace)
- **THEN** the response is `200` with no `display_name`, and the column is
  persisted as `NULL`

#### Scenario: A disallowed character is rejected

- **WHEN** an authenticated user sends `{ "display_name": "Jane <b>Doe</b>" }`
  or any value containing a character outside `[\p{L} .'-]`
- **THEN** the response is `400` with the `ErrInvalidDisplayName` sentinel and
  the stored name is unchanged

#### Scenario: An over-long value is rejected

- **WHEN** an authenticated user sends a `display_name` whose trimmed length
  exceeds 150 characters
- **THEN** the response is `400` and the stored name is unchanged

#### Scenario: Unauthenticated request is rejected

- **WHEN** `PATCH /api/auth/me` is called with no valid session
- **THEN** the response is `401` and no user is modified

## MODIFIED Requirements

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

On every successful OIDC sign-in, the backend SHALL read the ID token's `name`
claim and re-synchronise the resolved user's `display_name` from it: the claim
value SHALL be trimmed, characters outside `[\p{L} .'-]` SHALL be stripped, the
result SHALL be truncated to 150 characters, and — when that result is
non-empty — stored as the user's `display_name`, replacing any previous value
(including one the user set themselves via `PATCH /api/auth/me`). When the
`name` claim is absent or its sanitised result is empty, the user's existing
`display_name` SHALL be left unchanged.

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

#### Scenario: Provider name claim populates the display name

- **WHEN** an OIDC sign-in succeeds and the ID token carries `name` "Jane Doe"
- **THEN** the resolved user's `display_name` is set to "Jane Doe"

#### Scenario: Provider name claim is sanitised before storing

- **WHEN** an OIDC sign-in succeeds and the `name` claim is "Doe, Jane (Dr.)"
- **THEN** characters outside `[\p{L} .'-]` are stripped and the stored
  `display_name` is "Doe Jane Dr."

#### Scenario: Provider name claim overrides a self-set name

- **WHEN** a user with a linked `oidc` identity has set their `display_name`
  via `PATCH /api/auth/me`, then signs in again through the provider, whose
  `name` claim differs
- **THEN** the stored `display_name` is replaced by the sanitised claim value

#### Scenario: Missing name claim leaves the display name untouched

- **WHEN** an OIDC sign-in succeeds and the ID token has no `name` claim (or it
  sanitises to empty)
- **THEN** the user's existing `display_name` is unchanged

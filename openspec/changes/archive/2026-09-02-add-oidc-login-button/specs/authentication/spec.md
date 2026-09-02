## ADDED Requirements

### Requirement: Unauthenticated OIDC sign-in availability endpoint

The backend SHALL expose `GET /api/auth/config`, requiring no authentication,
that reports the sign-in affordances the web client should present. The response
SHALL be JSON with an `oidc` field that is either `null` (no OIDC sign-in
available) or an object with `label` (string, the configured `OIDC_LABEL`) and
`start_path` (string, `/api/auth/oidc/start`). The `oidc` object SHALL be
non-null only when an OIDC client is configured — that is, when both
`OIDC_ISSUER` and `OIDC_CLIENT_ID` are set and discovery succeeded at startup.

The endpoint SHALL NOT expose the issuer URL, client id, client secret, or any
other provider configuration beyond the label and the start path.

#### Scenario: OIDC configured

- **WHEN** `GET /api/auth/config` is requested and the backend started with a
  working OIDC provider and `OIDC_LABEL="Continue with Google"`
- **THEN** the response is `200` with
  `{ "oidc": { "label": "Continue with Google", "start_path": "/api/auth/oidc/start" } }`

#### Scenario: OIDC not configured

- **WHEN** `GET /api/auth/config` is requested and `OIDC_ISSUER` is unset
- **THEN** the response is `200` with `{ "oidc": null }`

#### Scenario: Available without a session

- **WHEN** `GET /api/auth/config` is requested with no session cookie or bearer
  token
- **THEN** the response is `200` (it is never `401`)

#### Scenario: No provider secrets are leaked

- **WHEN** the `GET /api/auth/config` response is inspected
- **THEN** it contains no issuer URL, client id, or client secret

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

#### Scenario: Issuer without a client id is treated as unconfigured

- **WHEN** the backend starts with `OIDC_ISSUER` set but `OIDC_CLIENT_ID` empty
- **THEN** no OIDC client is constructed, the `/api/auth/oidc/*` routes behave as
  not configured, and `GET /api/auth/config` returns `oidc: null`

### Requirement: Authentication configuration comes from the environment

All authentication configuration SHALL be fields on `config.Config` populated in
`internal/config` from environment variables and documented in
`backend/.env.example`: `AUTH_BASE_URL`, `AUTH_SESSION_TTL`,
`AUTH_SESSION_MAX_TTL`, `AUTH_COOKIE_SECURE`, `AUTH_SIGNUP_ENABLED`,
`AUTH_ALLOWED_EMAIL_DOMAINS`, `AUTH_INVITE_ENABLED`, `AUTH_INVITE_TTL`,
`AUTH_MAGIC_LINK_TTL`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`,
`SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`, `OIDC_ISSUER`, `OIDC_CLIENT_ID`,
`OIDC_CLIENT_SECRET`, `OIDC_SCOPES`, `OIDC_LABEL`. No code outside
`internal/config` SHALL read these via `os.Getenv`.

`AUTH_BASE_URL` SHALL be used to build magic-link URLs and the OIDC
`redirect_uri`. Redirect targets derived from request input SHALL be validated
to be same-origin relative paths to prevent open redirects.

#### Scenario: Config is loaded once in internal/config

- **WHEN** the backend starts
- **THEN** every auth setting is read by `config.Load()` and passed as a value;
  `grep` for `os.Getenv` outside `internal/config` finds nothing

#### Scenario: OIDC label defaults when unset

- **WHEN** `OIDC_LABEL` is not set in the environment
- **THEN** `config.Load()` returns an OIDC label of `Single sign-on`

#### Scenario: Open-redirect attempt is neutralized

- **WHEN** a sign-in flow is given a post-login redirect target pointing at an
  external origin
- **THEN** the person is redirected to a safe in-app default instead

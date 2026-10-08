# Spec Delta

## MODIFIED Requirements

### Requirement: Authentication configuration comes from the environment

All authentication configuration SHALL be fields on `config.Config` populated in
`internal/config` from environment variables and documented in
`backend/.env.example`: `AUTH_BASE_URL`, `AUTH_SESSION_TTL`,
`AUTH_SESSION_MAX_TTL`, `AUTH_COOKIE_SECURE`, `AUTH_SIGNUP_ENABLED`,
`AUTH_ALLOWED_EMAIL_DOMAINS`, `AUTH_INVITE_ENABLED`, `AUTH_INVITE_TTL`,
`AUTH_MAGIC_LINK_TTL`, `AUTH_TRUSTED_PROXIES`, `AUTH_CLEANUP_INTERVAL`,
`AUTH_PASSKEY_REAUTH_WINDOW`,
`RATE_LIMIT_IP_ENABLED`, `RATE_LIMIT_IP_REQUESTS`, `RATE_LIMIT_IP_WINDOW`,
`RATE_LIMIT_EMAIL_REQUESTS`, `RATE_LIMIT_EMAIL_WINDOW`, `SMTP_HOST`,
`SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`,
`OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_SCOPES`,
`OIDC_LABEL`. No code outside `internal/config` SHALL read these via
`os.Getenv`.

`AUTH_BASE_URL` SHALL be used to build magic-link URLs and the OIDC
`redirect_uri`, and its host and origin SHALL be the WebAuthn relying-party ID
and the only accepted passkey origin. Redirect targets derived from request input SHALL be validated
to be same-origin relative paths to prevent open redirects.

`AUTH_TRUSTED_PROXIES` SHALL be a comma-separated list of CIDRs (a bare IP
SHALL be accepted as a single-host CIDR); an unparsable entry SHALL make
configuration loading fail.

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

#### Scenario: Trusted proxies are parsed

- **WHEN** `AUTH_TRUSTED_PROXIES="10.0.0.0/8, 192.168.1.5"`
- **THEN** `config.Load()` returns two prefixes, `10.0.0.0/8` and
  `192.168.1.5/32`

#### Scenario: Malformed trusted proxy is rejected

- **WHEN** `AUTH_TRUSTED_PROXIES="10.0.0.0/33"`
- **THEN** configuration loading fails and the backend does not start

#### Scenario: Passkey re-authentication window defaults

- **WHEN** `AUTH_PASSKEY_REAUTH_WINDOW` is unset
- **THEN** `config.Load()` returns a window of 5 minutes, and a value that is
  not a positive duration makes loading fail

#### Scenario: Relying party follows the base URL

- **WHEN** `AUTH_BASE_URL=https://money.example.com`
- **THEN** passkeys are created for relying-party ID `money.example.com` and
  only the origin `https://money.example.com` is accepted

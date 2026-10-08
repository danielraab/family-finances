# Spec Delta

## MODIFIED Requirements

### Requirement: Per-IP throttling of unauthenticated sign-in endpoints

When `RATE_LIMIT_IP_ENABLED` is true (the default), the backend SHALL allow at
most `RATE_LIMIT_IP_REQUESTS` (default 20) requests per client IP within
`RATE_LIMIT_IP_WINDOW` (default `1m`), counted together across
`POST /api/auth/email/start`, `GET /api/auth/email/callback`,
`GET /api/auth/oidc/start`, `GET /api/auth/oidc/callback`,
`GET /api/auth/invites/accept`, `POST /api/auth/passkeys/login/start` and
`POST /api/auth/passkeys/login/finish`. Excess requests SHALL get `429` with a
`Retry-After` header and SHALL perform no side effect.

#### Scenario: Within the limit

- **WHEN** a client IP sends fewer than `RATE_LIMIT_IP_REQUESTS` requests to the
  throttled endpoints within the window
- **THEN** every request is processed normally

#### Scenario: Over the limit

- **WHEN** a client IP exceeds `RATE_LIMIT_IP_REQUESTS` within the window
- **THEN** further requests receive `429` with a `Retry-After` header in
  seconds, and no mail is sent and no session is created

#### Scenario: Budget recovers after the window

- **WHEN** a throttled client IP waits until its window has passed
- **THEN** its next request is processed normally

#### Scenario: Other clients are unaffected

- **WHEN** one client IP is throttled
- **THEN** requests from a different client IP are processed normally

#### Scenario: Limiter disabled

- **WHEN** `RATE_LIMIT_IP_ENABLED=false`
- **THEN** no request is ever rejected with `429` by the per-IP limiter

#### Scenario: Authenticated API routes are not throttled

- **WHEN** a client calls an authenticated endpoint such as `GET /api/auth/me`
  or `GET /api/accounts` many times
- **THEN** the per-IP limiter does not reject those requests

#### Scenario: Passkey sign-in shares the budget

- **WHEN** a client IP has used its whole budget on
  `POST /api/auth/passkeys/login/start`
- **THEN** its next `POST /api/auth/email/start` also receives `429`

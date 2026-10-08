# Spec Delta

## Purpose

Protects the unauthenticated sign-in endpoints and users' inboxes from abuse by
throttling requests per client IP (correctly resolved behind trusted reverse
proxies) and magic-link mails per recipient address.

## ADDED Requirements

### Requirement: Client IP is resolved through trusted proxies only

The backend SHALL determine a request's client IP from the direct peer address.
Only when that peer is inside a CIDR listed in `AUTH_TRUSTED_PROXIES` SHALL it
consult `X-Forwarded-For`, walking it right to left and taking the first
address not inside a trusted CIDR. With the list empty, forwarding headers
SHALL be ignored. The resolved IP SHALL be used for rate limiting and recorded
on new sessions.

#### Scenario: No trusted proxies configured

- **WHEN** `AUTH_TRUSTED_PROXIES` is empty and a request from peer `203.0.113.7`
  carries `X-Forwarded-For: 198.51.100.1`
- **THEN** the client IP is `203.0.113.7`

#### Scenario: Request through a trusted proxy

- **WHEN** `AUTH_TRUSTED_PROXIES=10.0.0.0/8` and a request from peer `10.0.0.2`
  carries `X-Forwarded-For: 198.51.100.1`
- **THEN** the client IP is `198.51.100.1`

#### Scenario: Spoofed left-most entry is ignored

- **WHEN** `AUTH_TRUSTED_PROXIES=10.0.0.0/8` and a request from peer `10.0.0.2`
  carries `X-Forwarded-For: 1.2.3.4, 198.51.100.1`
- **THEN** the client IP is `198.51.100.1`, not the client-supplied `1.2.3.4`

#### Scenario: Untrusted peer cannot forge its address

- **WHEN** `AUTH_TRUSTED_PROXIES=10.0.0.0/8` and a request from peer
  `203.0.113.7` carries `X-Forwarded-For: 198.51.100.1`
- **THEN** the client IP is `203.0.113.7`

#### Scenario: Header made only of trusted hops

- **WHEN** every `X-Forwarded-For` entry is inside a trusted CIDR
- **THEN** the client IP is the left-most entry

#### Scenario: Session records the resolved IP

- **WHEN** a sign-in through a trusted proxy creates a session
- **THEN** the session's recorded IP is the resolved client IP, not the proxy's

### Requirement: Per-IP throttling of unauthenticated sign-in endpoints

When `RATE_LIMIT_IP_ENABLED` is true (default), the backend SHALL allow at most
`RATE_LIMIT_IP_REQUESTS` (default 20) requests per client IP within
`RATE_LIMIT_IP_WINDOW` (default `1m`), counted together across the
unauthenticated sign-in endpoints: email start and callback, OIDC start and
callback, invite acceptance, and passkey login start and finish. Excess
requests SHALL get `429` with `Retry-After` and SHALL have no side effect.

#### Scenario: Throttled endpoint set

- **WHEN** the per-IP limiter is enabled
- **THEN** it applies to exactly `POST /api/auth/email/start`,
  `GET /api/auth/email/callback`, `GET /api/auth/oidc/start`,
  `GET /api/auth/oidc/callback`, `GET /api/auth/invites/accept`,
  `POST /api/auth/passkeys/login/start` and
  `POST /api/auth/passkeys/login/finish`

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

#### Scenario: Passkey sign-in shares the budget

- **WHEN** a client IP has used its whole budget on
  `POST /api/auth/passkeys/login/start`
- **THEN** its next `POST /api/auth/email/start` also receives `429`, and a
  throttled `login/start` stores no challenge

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

### Requirement: Per-recipient throttling of magic-link mails

The backend SHALL send at most `RATE_LIMIT_EMAIL_REQUESTS` (default 5)
magic-link mails per normalized recipient address within
`RATE_LIMIT_EMAIL_WINDOW` (default `15m`). Once the budget is spent,
`POST /api/auth/email/start` for that address SHALL still respond `200`
exactly as for any other address, and SHALL send no mail and create no token.
This limit SHALL apply regardless of `RATE_LIMIT_IP_ENABLED`.

#### Scenario: Recipient budget exhausted

- **WHEN** `POST /api/auth/email/start` is called a sixth time within 15 minutes
  for `person@example.com` with default settings
- **THEN** the response is `200` and no mail is sent

#### Scenario: Budget is per normalized address

- **WHEN** requests alternate between `Person@Example.com` and
  `person@example.com`
- **THEN** they count against the same budget

#### Scenario: Unpermitted addresses do not consume budget

- **WHEN** `POST /api/auth/email/start` is called for an address that would not
  be sent a mail anyway
- **THEN** the response is `200` and that address's budget is unchanged

#### Scenario: Throttled response is indistinguishable

- **WHEN** a recipient's budget is exhausted
- **THEN** the response status and body are identical to those for an address
  that was sent a mail

### Requirement: Limits are configurable with defaults

`RATE_LIMIT_IP_ENABLED`, `RATE_LIMIT_IP_REQUESTS`, `RATE_LIMIT_IP_WINDOW`,
`RATE_LIMIT_EMAIL_REQUESTS` and `RATE_LIMIT_EMAIL_WINDOW` SHALL be optional;
when unset the documented defaults apply. A count that is not a positive
integer, or a window that is not a positive Go duration, SHALL make the backend
refuse to start with a clear error.

#### Scenario: Defaults apply when unset

- **WHEN** none of the rate-limit variables are set
- **THEN** per-IP limiting is on at 20 requests per minute and per-recipient
  limiting allows 5 mails per 15 minutes

#### Scenario: Invalid value is rejected at startup

- **WHEN** `RATE_LIMIT_IP_REQUESTS=0` or `RATE_LIMIT_EMAIL_WINDOW=soon`
- **THEN** configuration loading fails and the backend does not start

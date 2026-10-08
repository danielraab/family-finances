# Proposal

## Why

The unauthenticated sign-in endpoints have no throttling: anyone can call
`POST /api/auth/email/start` in a loop, which floods a victim's inbox with
magic-link mails and lets an attacker hammer token and callback endpoints.
Short-lived auth rows (expired sessions, consumed/expired magic-link tokens,
abandoned OIDC login state) are also never deleted and accumulate forever.
The passkey login added by `add-passkey-login`, which lands first, adds
another unauthenticated ceremony with its own short-lived challenge rows, so
it is covered here too.

## What Changes

- Resolve the **client IP** in one place, honouring `X-Forwarded-For` only when
  the direct peer is inside a configured trusted-proxy CIDR list
  (`AUTH_TRUSTED_PROXIES`). Sessions record this resolved IP instead of the
  proxy's address.
- Add an **in-memory per-IP rate limiter** in front of the unauthenticated
  sign-in endpoints, including the passkey login endpoints; over-limit requests get `429 Too Many Requests` with
  `Retry-After`. It can be switched off with `RATE_LIMIT_IP_ENABLED=false`.
- Add an **in-memory per-recipient limiter** for magic-link mails: once an
  address hits its budget, `POST /api/auth/email/start` still answers `200`
  but sends no mail (no account enumeration).
- All limits are configurable through optional env variables with defaults.
- Add a **periodic cleanup job** that deletes expired sessions, expired or
  consumed magic-link tokens, expired OIDC login state, and expired passkey
  ceremony challenges, and evicts idle
  limiter buckets. Interval configurable (`AUTH_CLEANUP_INTERVAL`).
- Web client: the login view shows a "too many attempts, try again later"
  message on a `429`, for both the email and the passkey sign-in.

## Capabilities

### New Capabilities

- `auth-rate-limiting`: client-IP resolution behind trusted reverse proxies,
  per-IP throttling of unauthenticated sign-in endpoints, and per-recipient
  throttling of magic-link mails.
- `auth-cleanup`: the periodic job that removes expired and consumed
  short-lived authentication records.

### Modified Capabilities

- `authentication`: the environment-configuration requirement gains the new
  rate-limit, trusted-proxy and cleanup variables.
- `web-client-auth`: the login view handles a `429` response.

## Impact

- **Backend**: new `internal/clientip` (trusted-proxy IP resolution) and
  `internal/ratelimit` (sliding-window limiter) packages; `internal/auth` handler wiring for the limiters and the resolved
  IP; `auth.Store` gains delete-expired methods (memory + postgres); a cleanup
  goroutine started from `main.go` and stopped on shutdown; `internal/config`
  and `backend/.env.example` gain the new variables.
- **API**: `429` responses (with `Retry-After`) documented on the affected
  `/api/auth/*` operations in `openapi/openapi.yaml`; regenerate
  `backend/openapi.yaml` and `frontend/src/api/schema.d.ts`.
- **Frontend**: login view error state and i18n strings (`en`, `de`).
- **Dependencies**: none — standard library only.
- **Operations**: limits are per process; running several replicas multiplies
  the effective limit. Deployments behind a reverse proxy must set
  `AUTH_TRUSTED_PROXIES` for per-IP limiting to see real client addresses.

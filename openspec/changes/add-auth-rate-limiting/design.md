# Design

## Context

- `internal/auth/handler.go` derives the session IP from `r.RemoteAddr` only
  (`clientIP`), so behind a reverse proxy every session records the proxy.
- No endpoint is throttled. `POST /api/auth/email/start` always answers `200`
  (anti-enumeration) and sends a mail whenever the address is permitted.
- Nothing ever deletes `sessions`, `magic_link_tokens` or `oidc_login_state`
  rows except as a side effect of use: an expired session is deleted only when
  it is presented again (`Service.Authenticate`); consumed tokens stay forever.
- `internal/auth` must not import `internal/httpapi`; it receives
  `RenderError` through `HandlerOptions`. Sentinels map to HTTP statuses via
  `registerErrStatus` in `httpapi/auth.go`.
- `add-passkey-login` has landed: `POST /api/auth/passkeys/login/{start,finish}`
  exist unthrottled, and `webauthn_challenges` is purged only inline on
  challenge creation (`auth.Store.DeleteExpiredWebAuthnChallenges` exists).
- The reference topology is one app container plus Postgres. The user
  accepted per-process (in-memory) limiting for now.

## Goals / Non-Goals

**Goals:**

- One trustworthy client-IP function used by both rate limiting and session
  recording.
- Exact "at most N per window" semantics with a cheap, dependency-free limiter.
- A cleanup job that is idempotent and that the passkey change can extend with
  its own challenge table.

**Non-Goals:**

- Distributed or persistent rate limiting (Postgres/Redis). Limits reset on
  restart and are per replica.
- Throttling authenticated endpoints, or per-user throttling.
- `Forwarded` (RFC 7239), `X-Real-IP` or CDN-specific headers. Only
  `X-Forwarded-For` is honoured.
- Account lockout or CAPTCHA.

## Decisions

### D1. New `internal/clientip` package for IP resolution

`clientip.Resolver{Trusted []netip.Prefix}` with
`Resolve(r *http.Request) netip.Addr`: parse the `RemoteAddr` host; if it is not
in `Trusted`, return it. Otherwise split `X-Forwarded-For` on commas, walk right
to left, and return the first parseable address not in `Trusted`. If every hop
is trusted, return the left-most one; if the header is missing or unparsable,
return the peer. IPv4-mapped IPv6 addresses are unmapped before comparison.

- The package is separate so `internal/auth` (session IP, per-IP keys) and any
  future `httpapi` middleware can share it without an import cycle.
- *Alternative — a boolean `TRUST_PROXY_HEADERS`*: rejected. Any client can
  then forge its address and get a fresh bucket for every request.
- *Alternative — take the left-most `X-Forwarded-For` entry*: rejected for the
  same reason; only right-to-left past trusted hops is safe.

### D2. New `internal/ratelimit` package with a sliding-window log

`ratelimit.Limiter` is created with `(limit int, window time.Duration,
now func() time.Time)` and holds `map[string]*entry` under a `sync.Mutex`. Each
entry stores up to `limit` timestamps in a small ring buffer.

- `Allow(key) (ok bool, retryAfter time.Duration)`: drop timestamps older than
  `now-window`. If fewer than `limit` remain, record `now` and return true.
  Otherwise return false and `oldest+window-now`; a rejected call records
  nothing.
- `Evict(now)`: delete entries whose newest timestamp is older than
  `now-window`.

The log gives exact "≤ N in any window" semantics, matching the spec. With the
defaults (20 and 5) it stores at most 20 timestamps per key, which is trivial.
Rejected alternatives:

- *Fixed window*: allows a 2×N burst across a window boundary.
- *Token bucket via `golang.org/x/time/rate`*: a new dependency, and its
  semantics differ from "N per window".

Two limiter instances are built in `main.go`: `ipLimiter` and `mailLimiter`.
When `RATE_LIMIT_IP_ENABLED=false`, `ipLimiter` is `nil` and the throttle
wrapper is a no-op.

### D3. Wiring into `internal/auth`

`auth` declares a minimal interface and takes the pieces through options:

```go
type Limiter interface {
    Allow(key string) (bool, time.Duration)
}

HandlerOptions{ ..., ClientIP func(*http.Request) string, IPLimiter Limiter }
auth.WithMailLimiter(Limiter) // service option
```

- **Per-IP:** the handler wraps the seven unauthenticated routes (the five
  magic-link/OIDC/invite routes plus both passkey login routes) in
  `h.throttle(next)`. When `IPLimiter` is set and `Allow("ip:"+ip)` fails, it
  sets `Retry-After` (seconds, rounded up, minimum 1) and renders the new
  sentinel `auth.ErrRateLimited`, which `httpapi/auth.go` registers (via `registerErrStatus`) as
  `429`. This happens before the body is read, so a rejected request has no
  side effect. All seven routes share one budget per IP, keyed by IP only. A
  per-route budget would let an attacker get 5× the attempts.
- **Per-recipient:** inside `Service.StartEmailLogin`, after `emailPermitted`
  returns true and before the token is created, call
  `mailLimiter.Allow("mail:"+normalizedEmail)`. On refusal, log at `Info`
  (address hashed, not in plain text) and return `nil`, so the handler answers
  the same `200`. Unpermitted addresses never reach the limiter, so they cannot
  be used to probe or to exhaust a real user's budget.
- `sessionContext` uses `opts.ClientIP(r)`. The existing `clientIP` helper is
  removed, and tests default to a `RemoteAddr` resolver.

### D4. Cleanup job

- `auth.Service.Cleanup(ctx) error` runs the deletes and calls each registered
  `Evict(now)`. It joins the errors and keeps going, so one failing delete
  does not skip the others.
- New `auth.Store` methods, implemented in the memory and Postgres stores:
  - `DeleteExpiredSessions(ctx, now, createdBefore)`, with
    `createdBefore = now - SessionMaxTTL`
  - `DeleteStaleMagicLinkTokens(ctx, now)`, for rows that are consumed or past
    `expires_at`
  - `DeleteExpiredOIDCState(ctx, now)`
  - the existing `DeleteExpiredWebAuthnChallenges(ctx, now)` from
    `add-passkey-login`; the inline purge on challenge creation is then
    removed
- Each method is a single `DELETE … WHERE …`, so passes are idempotent and safe
  to run from several replicas without a lock. An advisory lock would only
  avoid duplicate work and is not needed at this scale.
- The runner is a small `runCleanup(ctx, svc, interval)` goroutine in
  `main.go`. It runs once immediately, then on a `time.Ticker`, using the same
  `signal.NotifyContext` context as the HTTP server, so it stops on shutdown.
  Each pass logs completion at `Info` and errors at `Error`.
- Indexes: the tables are small (a family-sized install), so no new index is
  added. If that changes, `sessions(expires_at)` is the first candidate.

### D5. Configuration

New fields in `config.Config`:

- `Auth.TrustedProxies []netip.Prefix`
- `Auth.CleanupInterval`
- `RateLimit{IPEnabled, IPRequests, IPWindow, EmailRequests, EmailWindow}`

They are parsed with the existing `durationEnv` and `boolEnv` helpers plus a
new positive-int helper and a CIDR-list helper. All are optional, and invalid
values fail `Load()`. Every variable is documented in `backend/.env.example`
and the root `README` if it lists env variables.

### D6. API contract

A shared `TooManyRequests` response (error body plus a `Retry-After` header) is
added to `openapi/openapi.yaml` and referenced from the seven throttled
operations. Both generated copies are regenerated in the same change.

## Risks / Trade-offs

- **[Per-process limits]** Several replicas multiply the limit, and a restart
  resets it. → This is documented in `.env.example`. A shared store can replace
  the limiter later behind the same `Limiter` interface.
- **[Misconfigured `AUTH_TRUSTED_PROXIES`]** If it is too broad (e.g.
  `0.0.0.0/0`), clients can spoof their IP; if empty behind a proxy, every
  client shares the proxy's bucket and gets throttled together. → Document
  both failure modes and log the configured prefixes at startup.
- **[Shared NAT]** A household behind one IP shares a budget. → The default of
  20/min is generous for human sign-ins, it is configurable, and limiting can
  be switched off.
- **[429 on browser navigations]** `email/callback`, `oidc/*` and
  `invites/accept` are full-page navigations, so a throttled visitor sees the
  same raw error rendering as other callback failures today. → This is
  acceptable at this limit level; a friendlier error page is out of scope.
- **[Mail limiter blocks a legitimate user]** Someone requesting six links in
  15 minutes silently gets no sixth mail. → The UI already says "check your
  inbox". The default is generous, and the budget recovers on its own.

## Migration Plan

- No schema migration. Deploying turns per-IP limiting on by default.
  Operators behind a reverse proxy should set `AUTH_TRUSTED_PROXIES` at the
  same time; otherwise all clients share the proxy's bucket.
- Rollback: set `RATE_LIMIT_IP_ENABLED=false` to remove per-IP throttling at
  once, or revert the release. The cleanup job only deletes rows that can no
  longer be used, so nothing needs restoring.

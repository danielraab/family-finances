# Tasks

## 1. Configuration

- [ ] 1.1 Add `Auth.TrustedProxies` (CIDR list; bare IPs become /32 or /128), `Auth.CleanupInterval` (default `15m`) and `RateLimit{IPEnabled=true, IPRequests=20, IPWindow=1m, EmailRequests=5, EmailWindow=15m}` to `internal/config`, with a positive-int helper and a CIDR-list helper; verify with `config_test.go` cases for defaults, valid overrides, and rejection of `0`, negative values, `0s`, garbage durations and `10.0.0.0/33`
- [ ] 1.2 Document every new variable with its default in `backend/.env.example`, including the per-process caveat and the "set `AUTH_TRUSTED_PROXIES` behind a reverse proxy" note; verify the file lists all six `RATE_LIMIT_*`/`AUTH_*` additions

## 2. Client IP resolution

- [ ] 2.1 Create `internal/clientip` with `Resolver.Resolve(*http.Request) netip.Addr` per design D1 (peer check, right-to-left `X-Forwarded-For` walk, all-trusted → left-most, unmapping IPv4-in-IPv6); verify with table tests covering every scenario in the `auth-rate-limiting` spec's client-IP requirement plus malformed headers and a `RemoteAddr` without a port
- [ ] 2.2 Pass a `ClientIP func(*http.Request) string` through `auth.HandlerOptions`, use it in `sessionContext`, delete the old `clientIP` helper, and wire the resolver in `main.go`; verify with a handler test that a session created through a trusted proxy records the forwarded IP

## 3. Rate limiter

- [ ] 3.1 Create `internal/ratelimit` with the sliding-window-log `Limiter` (`Allow`, `Evict`, injectable clock) per design D2; verify with unit tests for allow-up-to-N, reject-N+1, correct `retryAfter`, recovery after the window, key isolation, rejected calls not recording, `Evict` removing only idle keys, and a `-race` concurrent-use test
- [ ] 3.2 Add the `auth.Limiter` interface and the `auth.ErrRateLimited` sentinel, and register it as `429` in `httpapi/auth.go`; verify the existing sentinel-mapping test covers it

## 4. Per-IP throttling

- [ ] 4.1 Add `IPLimiter` to `auth.HandlerOptions` and wrap `POST /api/auth/email/start`, `GET /api/auth/email/callback`, `GET /api/auth/oidc/start`, `GET /api/auth/oidc/callback` and `GET /api/auth/invites/accept` in a shared-budget throttle that sets `Retry-After` and renders `ErrRateLimited` before reading the body; verify with handler tests: N requests pass, request N+1 is `429` with `Retry-After` and sends no mail, budget is shared across routes, other IPs are unaffected, and `/api/auth/me` is never throttled
- [ ] 4.2 Build the IP limiter in `main.go` only when `RATE_LIMIT_IP_ENABLED` is true; verify with a test that a nil limiter never returns `429`

## 5. Per-recipient throttling

- [ ] 5.1 Add an `auth.WithMailLimiter` service option and apply it in `StartEmailLogin` after `emailPermitted` and before token creation, keyed on the normalized address, logging a hashed address on refusal; verify with service tests: the 6th permitted request in the window sends no mail and creates no token but returns nil, `Person@Example.com` and `person@example.com` share a budget, unpermitted addresses don't consume budget, and it applies with the IP limiter disabled
- [ ] 5.2 Verify with a handler test that a throttled recipient gets a response byte-identical to a normal `200`

## 6. Cleanup job

- [ ] 6.1 Add `DeleteExpiredSessions(ctx, now, createdBefore)`, `DeleteStaleMagicLinkTokens(ctx, now)` and `DeleteExpiredOIDCState(ctx, now)` to `auth.Store`, implemented in `storage/memory` and `storage/postgres`; verify with store tests (memory, and Postgres integration tests) that expired, over-age and consumed rows are removed while valid sessions/tokens, invites, users and identities are kept
- [ ] 6.2 Add `auth.Service.Cleanup(ctx)`, which runs all deletes and evicts registered limiters, joining errors without short-circuiting; verify with a service test using a fake store where one delete fails and the others still run
- [ ] 6.3 Start `runCleanup(ctx, svc, interval)` from `main.go` on the shutdown context: run once at startup, then on a ticker, log errors, and stop on cancel; verify with a test that injects a short interval and a cancellable context and observes passes, error tolerance, and termination

## 7. API contract and docs

- [ ] 7.1 Add a reusable `TooManyRequests` response with a `Retry-After` header to `openapi/openapi.yaml` and reference it from the five throttled operations; regenerate with `cd backend && go generate ./...` and `cd frontend && pnpm generate:api`, and verify the spec lint and the contract drift check pass
- [ ] 7.2 Update `backend/AGENTS.md` (package layout: `clientip`, `ratelimit`; the cleanup job in `main.go`) and verify the layout block matches the tree

## 8. Web client

- [ ] 8.1 Handle `429` from `POST /api/auth/email/start` on `/login` with a dedicated "too many attempts" message (keep the form; separate from the generic error), adding `login.rateLimited` to `en.json` and `de.json`; verify with a component/route test or manual run, plus `pnpm lint` and the i18n-coverage check

## 9. Integration check

- [ ] 9.1 Run `cd backend && go vet ./... && go test -race ./...` (with Postgres for the integration tests) and `cd frontend && pnpm lint && pnpm build`; manually confirm through `compose.yaml` that repeated sign-in requests hit `429` and that a cleanup pass logs at startup

# Tasks

## 1. Prerequisites and setup

- [x] 1.1 Add `github.com/go-webauthn/webauthn` with `cd backend && go get` and `go mod tidy`, and record the justified dependency in `backend/AGENTS.md`; verify `go build ./...` passes and the openapi guard test still passes
- [x] 1.2 Add `Auth.PasskeyReauthWindow` (`AUTH_PASSKEY_REAUTH_WINDOW`, default `5m`, positive duration) to `internal/config`, and derive the RP ID and origin from `AUTH_BASE_URL`; document both in `backend/.env.example`, including the "changing the host invalidates passkeys" and "browser origin must equal `AUTH_BASE_URL`" notes; verify with `config_test.go` cases for the default, an override, and an invalid value

## 2. Schema and persistence

- [x] 2.1 Add the migration from design D2 (`webauthn_credentials`, `webauthn_challenges`, `sessions.passkey_credential_id` with `ON DELETE CASCADE`, indexes); verify `migrate_test.go` applies it cleanly on a fresh database
- [x] 2.2 Add a `Passkey` domain type, `Session.PasskeyCredentialID`, and `auth.Store` methods: `CreatePasskey`, `PasskeyByCredentialID`, `ListPasskeysByUser`, `UpdatePasskeyUsage`, `DeletePasskey(userID, id)` (returns `ErrNotFound` for another user's passkey), `CreateWebAuthnChallenge`, `ConsumeWebAuthnChallenge`, `DeleteExpiredWebAuthnChallenges`; let `CreateSession` persist the passkey id
- [x] 2.3 Implement 2.2 in `storage/postgres` and verify with integration tests: duplicate `credential_id` is rejected, deleting a passkey removes only its sessions, challenge consume is single-use, and deleting a session cascades its pending registration challenge
- [x] 2.4 Implement 2.2 in `storage/memory`, including the explicit session cascade, and verify with the same behaviours in memory store tests

## 3. Session on the request context

- [x] 3.1 Change `Authenticator.Authenticate` to return `(User, Session, error)`, add `auth.WithSession` / `SessionFromContext`, store both in `authResolve`, and update every fake; verify `go test ./...` passes and a new test shows the session on the context for both bearer and cookie
- [x] 3.2 Extend `GET` / `PATCH /api/auth/me` with the `session` object (`created_at`, `client`, `passkey_registration_until`); verify with handler tests for a web session (`created_at + window`) and a bearer session (`null`)

## 4. Registration

- [x] 4.1 Add the `auth.WebAuthn` interface over go-webauthn, wired in `main.go`, and a fake for tests; verify the service compiles against the fake
- [x] 4.2 Implement `requireFreshWebSession` plus the sentinels `ErrReauthRequired`, `ErrPasskeyWebOnly`, `ErrCeremonyInvalid`, `ErrPasskeyInvalid`, `ErrPasskeyAuthFailed`, `ErrInvalidPasskeyName`, registered in `httpapi/auth.go`; verify the sentinel status mapping test
- [x] 4.3 Implement register start and finish (resident key, UV required, excludeCredentials, ceremony bound to the session, name trim/default/≤100, freshness re-checked on finish); verify with service and handler tests for every scenario under "Registration requires a fresh web session" and "Registration ceremony" in the `passkeys` spec

## 5. Sign-in

- [x] 5.1 Implement login start and finish (discoverable, user-handle check, counter policy, disabled/deleted check, update usage, issue a web session bound to the passkey, set the cookie, replace an existing cookie session); verify with tests for success, unknown credential, bad signature or user handle, counter regression, disabled user, single-use ceremony, and "no user or identity created"
- [x] 5.2 Purge expired challenges inside `CreateWebAuthnChallenge` (design D7); verify with a store test that creating a challenge removes expired ones and keeps unexpired ones
- [x] 5.3 Verify with a test that a passkey-created session can register another passkey within the window

## 6. Listing and deletion

- [x] 6.1 Implement `GET /api/auth/passkeys` (own passkeys only, `current` flag, no key material); verify with handler tests for ownership, the `current` flag, and the response shape
- [x] 6.2 Implement `DELETE /api/auth/passkeys/{id}` (any session and age; 404 for others' or unknown ids; clears the cookie when it is the current session's passkey); verify with tests for the S1/S2/S3 scenario, current-passkey sign-out, a 404 on someone else's passkey, and that a deleted passkey can no longer sign in

## 7. API contract

- [x] 7.1 Document the six passkey operations, the `Passkey` and ceremony schemas, the `session` object on the `me` responses, and the `401`/`403` responses in `openapi/openapi.yaml`; regenerate `backend/openapi.yaml` and `frontend/src/api/schema.d.ts`; verify the spec lint, the contract drift check, and the `openapicheck` response validation in handler tests all pass

## 8. Web client

- [x] 8.1 Add `src/lib/webauthn.ts` (native JSON helpers with a base64url fallback, cancel detection); verify with unit tests for the conversion and cancel mapping, or a typed manual check, plus `pnpm lint`
- [x] 8.2 Add "Sign in with a passkey" to `/login` (feature-detected, above OIDC/email, silent cancel, failure message, auth refresh plus navigate to `/`), with `en`/`de` strings; verify manually against the embedded build, plus `pnpm lint` and i18n coverage
- [x] 8.3 Add the `/settings/passkeys` tab (list, current marker, empty state, fresh-sign-in notice, Add passkey with optional name, live stale switch, 403 handling, in-place passkey re-auth or sign-out fallback, confirmed removal with a current-passkey warning and redirect), with `settings.passkeys.*` strings in `en`/`de`; verify manually against the embedded build, plus `pnpm lint`, `pnpm build` and i18n coverage

## 9. Integration check

- [x] 9.1 Run `cd backend && go vet ./... && go test -race ./...` with Postgres, and `cd frontend && pnpm lint && pnpm build`; then, through `compose.yaml` with `AUTH_BASE_URL` matching the browser origin, manually walk through: magic-link sign-in → add passkey → sign out → passkey sign-in → wait past 5 minutes and confirm Add is blocked → remove the current passkey and confirm sign-out
- [x] 9.2 Add a short "Passkeys" section to the README (how they work, origin requirement, local-dev limitation); verify it matches the `.env.example` wording

# Design

## Context

- **Prerequisite:** `add-auth-rate-limiting` provides `internal/clientip`, the
  shared per-IP `auth.Limiter` throttle, and `auth.Service.Cleanup`. This
  change extends all three and must be implemented after it.
- **Sessions:** `sessions.created_at` is written once in `issueSessionToken`.
  The sliding expiry only moves `last_seen_at` and `expires_at`, so
  "session age" is already trustworthy without a schema change.
- **Short-lived ceremony state** already has a pattern: `oidc_login_state` is
  consumed atomically (`DELETE … RETURNING`).
- **Auth context:** `httpapi.Authenticator.Authenticate` returns only `User`,
  and only the user is put on the request context. Passkey registration needs
  the session (its age, its client, and its passkey) as well.
- **Error mapping:** `internal/auth` declares sentinels, and
  `internal/httpapi/auth.go` maps them to statuses via `registerErrStatus`.
  The error body is `{error, request_id}`.

## Goals / Non-Goals

**Goals:**

- Passkeys as an additional way to sign in to an existing account, with
  discoverable credentials (no username entry).
- A stolen cookie must not be able to plant a passkey: registration only on a
  web session younger than the re-authentication window.
- Deleting a passkey ends exactly the sessions it created, atomically.
- No new frontend dependency.

**Non-Goals:**

- Passkey-only accounts and signup via passkey. A passkey always attaches to
  an existing user.
- Conditional UI / autofill (`mediation: "conditional"`), attestation
  verification, authenticator allow-lists (AAGUID policy), renaming passkeys.
- Extra allowed origins (e.g. the Vite dev port) or related-origin requests.
- Passkey ceremonies for bearer/API clients.
- Showing passkeys to admins or managing other users' passkeys.

## Decisions

### D1. Use `github.com/go-webauthn/webauthn`

WebAuthn verification needs:

- CBOR decoding of `attestationObject` and `authenticatorData`;
- COSE key parsing for ES256, RS256 and EdDSA;
- RP-ID-hash, flag and origin checks.

That is roughly 500+ lines of security-critical parsing. `go-webauthn` is the
maintained, widely used Go implementation. Its transitive dependencies are
`fxamacker/cbor`, `go-webauthn/x`, `golang-jwt/jwt` (used for the FIDO MDS,
which this change doesn't use) and `google/uuid`. It is used only from
`internal/auth`, behind a small `auth.WebAuthn` interface, so tests can fake
it and the library stays swappable.

- *Alternative — hand-rolled verifier with `attestation: "none"` only:*
  rejected, because owning CBOR and COSE edge cases isn't worth the risk.
- *Alternative — `duo-labs/webauthn`:* rejected; it is the archived
  predecessor of `go-webauthn`.

The library is configured with:

- `RPID` = host of `AUTH_BASE_URL`
- `RPOrigins` = `[origin of AUTH_BASE_URL]`
- `RPDisplayName` = `"Family Finances"`
- attestation preference `none`
- `ResidentKey: required`, `UserVerification: required`

`http://localhost:<port>` is a valid secure context, so passkeys work locally
when the browser uses the same origin as `AUTH_BASE_URL` (the embedded build
or `go run`). They do not work through the Vite dev port.

### D2. Schema (one migration)

```sql
CREATE TABLE webauthn_credentials (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id    bytea NOT NULL UNIQUE,
    public_key       bytea NOT NULL,          -- COSE
    sign_count       bigint NOT NULL DEFAULT 0,
    transports       text[] NOT NULL DEFAULT '{}',
    aaguid           bytea,
    backup_eligible  boolean NOT NULL,
    backup_state     boolean NOT NULL,
    name             text NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    last_used_at     timestamptz
);
CREATE INDEX webauthn_credentials_user_id_idx ON webauthn_credentials (user_id);

ALTER TABLE sessions
    ADD COLUMN passkey_credential_id uuid
        REFERENCES webauthn_credentials(id) ON DELETE CASCADE;
CREATE INDEX sessions_passkey_credential_id_idx ON sessions (passkey_credential_id);

CREATE TABLE webauthn_challenges (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),   -- the ceremony id
    kind        text NOT NULL CHECK (kind IN ('registration', 'login')),
    session_id  uuid REFERENCES sessions(id) ON DELETE CASCADE, -- registration only
    data        jsonb NOT NULL,          -- go-webauthn SessionData
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL
);
```

- **Separate table, not `identities.kind = 'passkey'`:** passkeys share no
  columns with identities and never take part in email linking
  (`ResolveLink`). Keeping them apart leaves the identity-linking table
  untouched.
- **`ON DELETE CASCADE` on `sessions.passkey_credential_id`:** deleting a
  passkey revokes its sessions in the same statement, which satisfies the
  spec's "only that passkey's sessions" requirement atomically. The memory
  store must imitate this explicitly; a shared store contract test covers it.
- **`challenges.session_id` cascade:** logging out also discards a pending
  registration.
- **Ceremony id:** a random UUID returned to the client and echoed on finish.
  The challenge itself lives in `data`. Consumption is a single
  `DELETE … WHERE id=$1 AND kind=$2 RETURNING …`, followed by checks for
  expiry and (for registration) `session_id` = the caller's session. Challenge
  TTL is 5 minutes, a constant longer than the 120 s client timeout.

### D3. Session on the request context

- `Authenticator.Authenticate` returns `(User, Session, error)`.
- `authResolve` stores both values; `auth.WithSession` and
  `auth.SessionFromContext` are added to `httpctx.go`.
- `auth.Session` gains `PasskeyCredentialID string`.

This is the smallest change that lets handlers read the session's
`CreatedAt`, `Client` and passkey without a second lookup. Test fakes of
`Authenticator` are updated.

- *Alternative — re-hash the cookie inside passkey handlers and look the
  session up again:* rejected, because it duplicates `authResolve`'s
  bearer/cookie precedence logic.

### D4. Freshness rule in one place

`Service.requireFreshWebSession(sess)` enforces the rule. It returns
`ErrReauthRequired` (403) when `now - sess.CreatedAt > ReauthWindow`, and
`ErrPasskeyWebOnly` (403) when `sess.Client != web`. It is called at both
register start and register finish.

`me` computes `passkey_registration_until = CreatedAt + ReauthWindow` (web)
or `null` (API). `ReauthWindow` comes from `AUTH_PASSKEY_REAUTH_WINDOW`
(default `5m`).

### D5. Endpoints

All endpoints are JSON and live in the auth handler. The `options` value is
the WebAuthn JSON form, so the browser can use `parse*OptionsFromJSON`.

| Route | Auth | Notes |
|---|---|---|
| `POST /api/auth/passkeys/register/start` | fresh web session | → `{ceremony_id, options}` |
| `POST /api/auth/passkeys/register/finish` | fresh web session | `{ceremony_id, name?, credential}` → `201 Passkey` |
| `GET /api/auth/passkeys` | any session | → `Passkey[]` |
| `DELETE /api/auth/passkeys/{id}` | any session | → `204`; clears the cookie if it was the current session's passkey |
| `POST /api/auth/passkeys/login/start` | none, IP-throttled | → `{ceremony_id, options}` |
| `POST /api/auth/passkeys/login/finish` | none, IP-throttled | `{ceremony_id, credential}` → `200 {user}` + `Set-Cookie` |

- **Login finish** always issues a **web** session with
  `passkey_credential_id` set, regardless of `Accept`. The ceremony is
  browser-only, and this keeps a passkey from minting bearer tokens. If the
  request already carried a valid session cookie, that old session is deleted
  after the new one is issued (the in-place "Sign in again" replaces rather
  than stacks sessions).
- **Login verification:**
  - Use `ValidateDiscoverableLogin`, with a handler that loads the credential
    by `credential_id` and checks that `user_handle` equals its `user_id`.
  - Counter policy: reject when both counters are non-zero and the new one is
    ≤ the stored one; synced passkeys report 0 and are allowed.
  - On success, update `sign_count`, `backup_state` and `last_used_at`.
  - The user must not be disabled or deleted (`ErrAccountDisabled`).
- **New sentinels:**
  - `ErrReauthRequired` (403)
  - `ErrPasskeyWebOnly` (403)
  - `ErrCeremonyInvalid` (400): unknown, expired, consumed, or wrong session
  - `ErrPasskeyInvalid` (400): registration verification failed
  - `ErrPasskeyAuthFailed` (401): any login verification failure, with one
    generic message, so an attacker can't tell an unknown credential from a
    bad signature
  - `ErrInvalidPasskeyName` (400): longer than 100 characters after trimming
- **Errors at the boundary:** registration rejects a duplicate `credential_id`
  with `ErrPasskeyInvalid`. `excludeCredentials` stops the browser
  re-registering one of the user's own authenticators.

### D6. Frontend

- **`src/lib/webauthn.ts`** uses
  `PublicKeyCredential.parseCreationOptionsFromJSON` /
  `parseRequestOptionsFromJSON` and `credential.toJSON()` when available. It
  falls back to a ~40-line base64url conversion for browsers without them
  (Safari < 18.4). `NotAllowedError` / `AbortError` are mapped to a
  "cancelled" result the UI treats as silent.
- **No `@simplewebauthn/browser`:** the native API plus a tiny fallback
  covers the two calls needed.
- **Login:** the passkey button sits above the OIDC/email controls. On success
  the client calls the existing `AuthProvider` refresh and navigates to `/`.
- **Settings:**
  - A new route `settings.passkeys.tsx`, plus a tab entry in `settings.tsx`.
  - A timer re-renders at `passkey_registration_until`.
  - "Sign in again" runs the passkey login ceremony in place when the list is
    non-empty, then refreshes `me`. Otherwise it calls logout and navigates
    to `/login`.
  - Removal uses the existing confirmation modal. When the removed passkey is
    `current`, the client clears its auth state and goes to `/login`.

### D7. Rate limiting and cleanup hooks

- Both login routes are wrapped in the shared `h.throttle` from
  `add-auth-rate-limiting`. Registration and management routes are
  authenticated and stay unthrottled.
- `Service.Cleanup` gains `DeleteExpiredWebAuthnChallenges(ctx, now)`.

### D8. Spec sequencing

The `authentication` configuration requirement and the `auth-rate-limiting`
per-IP requirement are MODIFIED here. Their text already includes the
additions from `add-auth-rate-limiting`, so archive that change first, then
this one.

## Risks / Trade-offs

- **[Attacker with a stolen cookie deletes the victim's passkeys or signs out
  the victim's passkey sessions]** → Deletion grants the attacker no lasting
  access. The victim recovers with a magic link and re-registers, and deletion
  without re-authentication was explicitly requested.
- **[Attacker who briefly controls a fresh session (e.g. an intercepted magic
  link) registers passkey B; the victim later deletes only passkey A]** →
  Under option C, B's sessions survive. However, B is visible in the list with
  its creation date and can be removed. This trade-off was accepted.
- **[Dependency weight and supply chain]** → The dependency is used only in
  `internal/auth` behind an interface, pinned in `go.sum`, and added to
  `backend/AGENTS.md`'s justified dependency list.
- **[`AUTH_BASE_URL` changes host]** → Existing passkeys are bound to the old
  RP ID and stop working. Users fall back to a magic link and re-register.
  This is documented in `.env.example`.
- **[Lost device with a synced passkey]** → The user deletes the passkey from
  any other session, which immediately ends the sessions signed in with it.
- **[Clock-based freshness]** → Server-side clock only; the client timer is
  cosmetic, and the server re-checks on finish.

## Migration Plan

- One forward migration (`00NN_passkeys.sql`, the next free number at
  implementation time). It adds tables and a nullable column, so it is
  backward compatible with a rollback of the binary: the old binary ignores
  the new column and tables.
- No backfill: existing sessions have `passkey_credential_id = NULL`, and
  nobody has passkeys yet.
- Rollback: deploy the previous image. Registered passkeys stay in the
  database unused until the feature returns.

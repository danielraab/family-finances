# Proposal

## Why

Signing in today means waiting for a magic-link mail or going through the OIDC
provider. Passkeys give a phishing-resistant one-tap sign-in with no mail round
trip. The main new risk is that a stolen session cookie could register an
attacker's passkey and keep access after the session ends. The design closes
that by allowing registration only on a freshly created web session.

This change lands first. `add-auth-rate-limiting` follows it and adds per-IP
throttling of the passkey sign-in endpoints and periodic cleanup of their
challenges.

## What Changes

- A signed-in user can **register one or more passkeys** (WebAuthn
  discoverable credentials, user verification required) and give each a name.
- **Registration requires re-authentication:** it is allowed only on a web
  (cookie) session created within the last `AUTH_PASSKEY_REAUTH_WINDOW`
  (optional, default `5m`), checked on both start and finish. Bearer/API
  sessions can never register a passkey.
- **"Sign in with a passkey"** on `/login` uses discoverable credentials, so
  there is no email field. It only signs in to the account the passkey belongs
  to: it never creates an account and never links by email. Disabled or
  deleted accounts are rejected.
- A user can **list and delete** their passkeys at any time, from any session,
  with no freshness requirement. **Deleting a passkey ends every session that
  passkey created**, and only those. Deleting the passkey behind the current
  session signs the user out.
- `GET /api/auth/me` reports the current session's creation time and until
  when it may register a passkey, so the UI can explain the fresh-login rule.
- New Settings **Passkeys** tab: the list, an "Add passkey" control (enabled
  only while the session is fresh), a notice that adding a passkey needs a
  sign-in from the last 5 minutes, a "sign in again" path, and deletion with
  confirmation.
- Expired passkey ceremony challenges are deleted whenever a new ceremony
  starts, so anonymous `login/start` calls cannot grow the table without
  bound before the cleanup job exists.
- The relying-party ID and allowed origin come from `AUTH_BASE_URL`. A dev
  setup where the browser origin differs from `AUTH_BASE_URL` (such as the
  Vite port) is not supported for passkeys.

## Capabilities

### New Capabilities

- `passkeys`: passkey registration (with the re-authentication rule), passkey
  sign-in, listing and deletion, and the binding of sessions to the passkey
  that created them.

### Modified Capabilities

- `authentication`: the environment-configuration requirement gains
  `AUTH_PASSKEY_REAUTH_WINDOW`.
- `web-client-auth`: `/login` offers "Sign in with a passkey".
- `web-client-settings`: a new Passkeys tab.

## Impact

- **Backend:**
  - New dependency `github.com/go-webauthn/webauthn`. CBOR, COSE and
    attestation verification is not something to hand-roll; justified in
    design.md.
  - New migration: a `webauthn_credentials` table, a `webauthn_challenges`
    table, and `sessions.passkey_credential_id` with
    `ON DELETE CASCADE`.
  - `auth.Store` and both store implementations gain passkey and challenge
    methods.
  - The authenticated session (not just the user) is placed on the request
    context. This changes the `httpapi.Authenticator` signature.
  - New `/api/auth/passkeys/*` routes.
  - Config and `.env.example` gain `AUTH_PASSKEY_REAUTH_WINDOW`.
- **API:** six new operations plus the extended `GET /api/auth/me` response in
  `openapi/openapi.yaml`. Both generated copies are regenerated.
- **Frontend:** a small WebAuthn helper (native browser API, no new
  dependency), the login button, the Passkeys settings tab, and `en`/`de`
  strings.
- **Docs:** `backend/AGENTS.md` dependency list, and a README note on passkeys
  requiring the browser origin to equal `AUTH_BASE_URL`.

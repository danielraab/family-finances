-- 0032_passkeys: WebAuthn passkeys as an additional way to sign in to an
-- existing user, the short-lived ceremony challenges, and the binding of a
-- session to the passkey that created it. See openspec change
-- `add-passkey-login` §D2.

-- One row per registered passkey. Never linked by email: a passkey only ever
-- signs in to the user it was registered on.
CREATE TABLE webauthn_credentials (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id   bytea NOT NULL UNIQUE,
    public_key      bytea NOT NULL,
    sign_count      bigint NOT NULL DEFAULT 0,
    transports      text[] NOT NULL DEFAULT '{}',
    aaguid          bytea,
    backup_eligible boolean NOT NULL,
    backup_state    boolean NOT NULL,
    name            text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_used_at    timestamptz
);
CREATE INDEX webauthn_credentials_user_id_idx ON webauthn_credentials (user_id);

-- NULL for sessions created by a magic link, OIDC, or an invite. Deleting a
-- passkey revokes exactly the sessions it created, in the same statement.
ALTER TABLE sessions
    ADD COLUMN passkey_credential_id uuid
        REFERENCES webauthn_credentials(id) ON DELETE CASCADE;
CREATE INDEX sessions_passkey_credential_id_idx ON sessions (passkey_credential_id);

-- Per-ceremony state (the library's SessionData), consumed atomically by
-- DELETE … RETURNING. A registration challenge belongs to the session that
-- started it, so logging out discards it.
CREATE TABLE webauthn_challenges (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind       text NOT NULL CHECK (kind IN ('registration', 'login')),
    session_id uuid REFERENCES sessions(id) ON DELETE CASCADE,
    data       jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX webauthn_challenges_expires_at_idx ON webauthn_challenges (expires_at);

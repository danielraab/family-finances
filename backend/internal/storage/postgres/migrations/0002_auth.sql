-- 0002_auth: the authentication schema — user accounts, linkable sign-in
-- identities, opaque-token sessions, single-use magic-link and invite tokens,
-- and short-lived OIDC login state. See openspec change `authentication` §D11.

CREATE TABLE users (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email        citext NOT NULL UNIQUE,
    display_name text,
    is_admin     boolean NOT NULL DEFAULT false,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- Each row is one way to sign in to a user. `email` identities carry the
-- verified address; `oidc` identities carry (provider issuer, subject).
CREATE TABLE identities (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind           text NOT NULL CHECK (kind IN ('email', 'oidc')),
    email          citext,
    email_verified boolean NOT NULL DEFAULT false,
    provider       text,
    subject        text,
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- One email identity per address; one oidc identity per (provider, subject).
CREATE UNIQUE INDEX identities_email_key
    ON identities (email) WHERE kind = 'email';
CREATE UNIQUE INDEX identities_provider_subject_key
    ON identities (provider, subject) WHERE kind = 'oidc';
CREATE INDEX identities_user_id_idx ON identities (user_id);

-- Opaque session tokens, stored only as their SHA-256 hash.
CREATE TABLE sessions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   bytea NOT NULL UNIQUE,
    client       text NOT NULL CHECK (client IN ('web', 'api')),
    user_agent   text,
    ip           inet,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- Single-use magic-link tokens, stored hashed, consumed atomically.
CREATE TABLE magic_link_tokens (
    token_hash  bytea PRIMARY KEY,
    email       citext NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz
);

CREATE TABLE invites (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email            citext NOT NULL,
    invited_by       uuid NOT NULL REFERENCES users(id),
    token_hash       bytea NOT NULL UNIQUE,
    created_at       timestamptz NOT NULL DEFAULT now(),
    expires_at       timestamptz NOT NULL,
    accepted_at      timestamptz,
    accepted_user_id uuid REFERENCES users(id)
);
CREATE INDEX invites_email_idx ON invites (email);

-- Short-lived per-attempt OIDC state: CSRF `state`, replay `nonce`, and the
-- PKCE verifier. `provider` is carried now so multi-provider is later additive.
CREATE TABLE oidc_login_state (
    state         text PRIMARY KEY,
    nonce         text NOT NULL,
    pkce_verifier text NOT NULL,
    provider      text NOT NULL,
    return_to     text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL
);

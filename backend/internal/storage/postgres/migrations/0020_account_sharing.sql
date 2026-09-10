-- 0020_account_sharing: accounts can be shared with other users at one of
-- four permission tiers (view, append, entry_admin, owner). account_shares
-- is additive to accounts.owner_id, which keeps meaning "real owner" and is
-- never referenced by a row in this table. entries.owner_id is renamed to
-- created_by: it already meant "who logged this" in practice (only an
-- owner could create an entry); sharing makes that literal, since any
-- permitted user can now create one. See openspec change `account-sharing`.

CREATE TABLE account_shares (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  uuid NOT NULL REFERENCES accounts(id),
    user_id     uuid NOT NULL REFERENCES users(id),
    permission  text NOT NULL CHECK (permission IN ('view', 'append', 'entry_admin', 'owner')),
    granted_by  uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, user_id)
);

CREATE INDEX account_shares_user_id_idx ON account_shares (user_id);

ALTER TABLE entries RENAME COLUMN owner_id TO created_by;

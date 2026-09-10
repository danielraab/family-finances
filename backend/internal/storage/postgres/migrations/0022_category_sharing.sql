-- 0022_category_sharing: categories can be shared with other users at one
-- of two permission tiers (view, append). category_shares is additive to
-- categories.owner_id, which keeps meaning "real owner" and is never
-- referenced by a row in this table — unlike accounts, there is no
-- shareable "owner" tier for categories; only the real owner ever manages
-- a category's own metadata, lifecycle, or shares. See openspec change
-- `category-sharing`.

CREATE TABLE category_shares (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id uuid NOT NULL REFERENCES categories(id),
    user_id     uuid NOT NULL REFERENCES users(id),
    permission  text NOT NULL CHECK (permission IN ('view', 'append')),
    granted_by  uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (category_id, user_id)
);

CREATE INDEX category_shares_user_id_idx ON category_shares (user_id);

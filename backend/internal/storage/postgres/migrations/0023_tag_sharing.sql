-- 0023_tag_sharing: tags can be shared with other users at one of two
-- permission tiers (view, append), structurally identical to
-- category_shares (0022_category_sharing.sql). tag_shares is additive to
-- tags.owner_id, which keeps meaning "real owner" and is never referenced
-- by a row in this table — only the real owner ever manages a tag's own
-- name, lifecycle, or shares. See openspec change `add-tag-sharing`.

CREATE TABLE tag_shares (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tag_id     uuid NOT NULL REFERENCES tags(id),
    user_id    uuid NOT NULL REFERENCES users(id),
    permission text NOT NULL CHECK (permission IN ('view', 'append')),
    granted_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tag_id, user_id)
);

CREATE INDEX tag_shares_user_id_idx ON tag_shares (user_id);

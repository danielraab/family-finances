-- 0009_tags: per-user tags, private to their owner, attachable to entries.
-- See openspec change `add-accounts-entries`.

CREATE TABLE tags (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   uuid NOT NULL REFERENCES users(id),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_id, name)
);

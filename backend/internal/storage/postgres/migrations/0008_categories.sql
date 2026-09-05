-- 0008_categories: a global, tree-structured, admin-managed category
-- lookup that every transaction entry references. See openspec change
-- `add-accounts-entries`.

CREATE TABLE categories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id  uuid REFERENCES categories(id),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (parent_id, name)
);

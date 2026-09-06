-- 0014_category_ownership: categories move from a single global,
-- admin-managed tree to a per-user tree each owner fully self-serves. See
-- openspec change `category-management`.
--
-- Greenfield — no production data to preserve, so this drops any existing
-- rows rather than backfilling an owner_id for them.

DELETE FROM categories;

ALTER TABLE categories DROP CONSTRAINT categories_parent_id_name_key;

ALTER TABLE categories
    ADD COLUMN owner_id   uuid NOT NULL REFERENCES users(id),
    ADD COLUMN sort_order integer NOT NULL DEFAULT 0,
    ADD COLUMN disabled   boolean NOT NULL DEFAULT false,
    ADD COLUMN deleted_at timestamptz;

CREATE INDEX ON categories (owner_id, parent_id);

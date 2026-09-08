-- 0017_tag_disabled: a reversible flag blocking a tag from being newly
-- attached to an entry, without affecting entries that already carry it.
-- See openspec change `add-tag-management`.

ALTER TABLE tags ADD COLUMN disabled boolean NOT NULL DEFAULT false;

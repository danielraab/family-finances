-- 0021_account_type_text: an account's type collapses from a per-user
-- account_types lookup (id, owner_id, title, description, disabled) to a
-- plain required text label on the account itself. Every accounts row has a
-- NOT NULL type_id FK today, so the backfill from account_types.title is
-- total and SET NOT NULL cannot fail. Forward-only, like every migration
-- here; this drop is destructive (a rollback past it needs a DB restore).
-- See openspec change `account-type-text-column`.

ALTER TABLE accounts ADD COLUMN type text;

UPDATE accounts a
SET type = t.title
FROM account_types t
WHERE t.id = a.type_id;

ALTER TABLE accounts ALTER COLUMN type SET NOT NULL;

ALTER TABLE accounts DROP COLUMN type_id;

DROP TABLE account_types;

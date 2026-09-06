-- 0013_account_type_fields: account_types gains a title (renamed from name),
-- an optional description, and a reversible disabled flag. See openspec
-- change `account-types-settings-tab`.

ALTER TABLE account_types RENAME COLUMN name TO title;
ALTER TABLE account_types ADD COLUMN description text;
ALTER TABLE account_types ADD COLUMN disabled boolean NOT NULL DEFAULT false;

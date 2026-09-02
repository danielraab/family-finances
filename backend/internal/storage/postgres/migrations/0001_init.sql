-- 0001_init: baseline. Enable the citext extension so later migrations can use
-- case-insensitive text columns (e.g. email addresses). No product tables yet;
-- the first domain package adds its own migration.
CREATE EXTENSION IF NOT EXISTS citext;

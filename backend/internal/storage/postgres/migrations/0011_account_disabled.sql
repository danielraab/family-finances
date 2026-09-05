-- 0011_account_disabled: a reversible flag that blocks creating new entries
-- against an account without hiding it or affecting its existing entries —
-- distinct from closing_date (informational) and deleted_at (soft delete).
-- See openspec change `add-accounts-entries`.

ALTER TABLE accounts
    ADD COLUMN disabled boolean NOT NULL DEFAULT false;

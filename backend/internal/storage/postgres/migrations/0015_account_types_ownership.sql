-- 0015_account_types_ownership: account_types moves from a flat,
-- admin-managed, instance-wide lookup to a per-user one, mirroring
-- categories (0014_category_ownership.sql). No production data to
-- preserve, so existing rows are cleared and every user is reseeded with
-- the same starter set used for new signups going forward, rather than
-- cloning old shared rows per owner. See openspec change
-- account-types-per-user.

DELETE FROM account_types;

ALTER TABLE account_types ADD COLUMN owner_id uuid REFERENCES users(id);

-- account_types_name_key predates the 0013 title rename (RENAME COLUMN
-- does not rename the constraint) — instance-wide uniqueness has no place
-- once titles are per-owner, matching categories' precedent of dropping
-- naming uniqueness rather than reworking it per-owner.
ALTER TABLE account_types DROP CONSTRAINT account_types_name_key;

INSERT INTO account_types (owner_id, title)
SELECT u.id, v.title
FROM users u
CROSS JOIN (VALUES ('Checking'), ('Savings'), ('Cash'),
                    ('Credit Card'), ('Loan'), ('Investment')) AS v(title);

-- Every pre-existing account is repointed at its own owner's seeded
-- "Checking" row — not an attempt to preserve what its old (now deleted)
-- global type used to mean, just a mechanical way to leave accounts.type_id
-- valid.
UPDATE accounts a
SET type_id = t.id
FROM account_types t
WHERE t.owner_id = a.owner_id AND t.title = 'Checking';

ALTER TABLE account_types ALTER COLUMN owner_id SET NOT NULL;
CREATE INDEX ON account_types (owner_id);

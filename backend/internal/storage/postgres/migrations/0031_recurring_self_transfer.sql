-- 0031_recurring_self_transfer: a kind on recurring transaction templates,
-- transaction (what every existing template is) or self_transfer, the latter
-- carrying a to_account_id the way an entry of that kind already does.
-- recurring_transaction_legs below exposes a self_transfer template as two
-- oppositely-signed rows, one per account, for listing and summing — a direct
-- copy of entry_legs (migration 0030). category_id becomes optional for
-- self_transfer, mirroring entries_check. See openspec change
-- `add-recurring-self-transfers`.

ALTER TABLE recurring_transactions ADD COLUMN kind text NOT NULL DEFAULT 'transaction';
ALTER TABLE recurring_transactions ADD CONSTRAINT recurring_transactions_kind_check
    CHECK (kind IN ('transaction', 'self_transfer'));

ALTER TABLE recurring_transactions ADD COLUMN to_account_id uuid REFERENCES accounts(id);
ALTER TABLE recurring_transactions ADD CONSTRAINT recurring_transactions_to_account_matches_kind
    CHECK ((kind = 'self_transfer') = (to_account_id IS NOT NULL));
ALTER TABLE recurring_transactions ADD CONSTRAINT recurring_transactions_to_account_differs
    CHECK (to_account_id IS NULL OR to_account_id != account_id);

-- category_id is required for a transaction and optional for a self_transfer,
-- exactly as entries_check already states it for entries.
ALTER TABLE recurring_transactions ALTER COLUMN category_id DROP NOT NULL;
ALTER TABLE recurring_transactions ADD CONSTRAINT recurring_transactions_category_matches_kind CHECK (
    (kind = 'transaction' AND category_id IS NOT NULL)
    OR kind = 'self_transfer'
);

CREATE INDEX recurring_transactions_to_account_idx
    ON recurring_transactions (to_account_id) WHERE to_account_id IS NOT NULL;

-- Every pre-existing row is a transaction; the default has served its purpose
-- and is dropped so kind is always written explicitly from here on, like every
-- other required column on this table.
ALTER TABLE recurring_transactions ALTER COLUMN kind DROP DEFAULT;

-- recurring_transaction_legs presents every template as it is seen from each
-- account it touches: the stored row as-is, plus, only for a self_transfer, a
-- second row for to_account_id with account_id/to_account_id swapped and
-- amount negated. Filtering account_id = ANY(...) against this view is what
-- makes a self-transfer template list and sum once per account in scope, or
-- twice when both are — see design.md. `native` distinguishes a row's own leg
-- from its flipped counterpart, both for the caller-visible flag and for
-- ordering the two together.
CREATE VIEW recurring_transaction_legs AS
SELECT id, created_by, account_id, to_account_id, kind, title, description, category_id,
       counterparty, location, amount, interval_unit, interval_count, starts_on, ends_on,
       deleted_at, created_at, updated_at, true AS native
FROM recurring_transactions
UNION ALL
SELECT id, created_by, to_account_id AS account_id, account_id AS to_account_id, kind, title,
       description, category_id, counterparty, location, -amount AS amount, interval_unit,
       interval_count, starts_on, ends_on, deleted_at, created_at, updated_at, false AS native
FROM recurring_transactions
WHERE kind = 'self_transfer';

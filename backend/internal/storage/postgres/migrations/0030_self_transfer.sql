-- 0030_self_transfer: a third entry kind, self_transfer, moving amount from
-- account_id (the sending account, unchanged meaning) to a new to_account_id
-- (the receiving account) — one row represents both legs; entry_legs below
-- exposes it as two oppositely-signed rows, one per account, for listing/
-- summing. category_id stays optional for self_transfer, same as
-- balance_adjustment. See openspec change `add-self-transfer`.

ALTER TABLE entries DROP CONSTRAINT entries_kind_check;
ALTER TABLE entries ADD CONSTRAINT entries_kind_check
    CHECK (kind IN ('transaction', 'balance_adjustment', 'self_transfer'));

ALTER TABLE entries DROP CONSTRAINT entries_check;
ALTER TABLE entries ADD CONSTRAINT entries_check CHECK (
    (kind = 'transaction' AND category_id IS NOT NULL)
    OR (kind IN ('balance_adjustment', 'self_transfer'))
);

ALTER TABLE entries ADD COLUMN to_account_id uuid REFERENCES accounts(id);
ALTER TABLE entries ADD CONSTRAINT entries_to_account_matches_kind
    CHECK ((kind = 'self_transfer') = (to_account_id IS NOT NULL));
ALTER TABLE entries ADD CONSTRAINT entries_to_account_differs
    CHECK (to_account_id IS NULL OR to_account_id != account_id);

CREATE INDEX entries_to_account_idx ON entries (to_account_id) WHERE to_account_id IS NOT NULL;

-- entry_legs presents every entry as it is seen from each account it
-- touches: the stored row as-is (account_id, amount unchanged), plus, only
-- for a self_transfer, a second row for to_account_id with account_id/
-- to_account_id swapped and amount negated. Filtering account_id = ANY(...)
-- against this view is what makes a self-transfer list/sum once per account
-- in scope, or twice when both of its accounts are in scope at once — see
-- design.md's "GET /api/entries becomes... a UNION ALL of two projections"
-- decision. `native` orders a row's own leg before its flipped counterpart
-- when both land on the same page, for stable (not semantically required)
-- ordering only.
CREATE VIEW entry_legs AS
SELECT id, created_by, account_id, to_account_id, kind, amount, balance_reading,
       booking_timestamp, title, description, category_id, counterparty, location,
       recurring_transaction_id, deleted_at, created_at, updated_at, true AS native
FROM entries
UNION ALL
SELECT id, created_by, to_account_id AS account_id, account_id AS to_account_id, kind, -amount AS amount,
       balance_reading, booking_timestamp, title, description, category_id, counterparty, location,
       recurring_transaction_id, deleted_at, created_at, updated_at, false AS native
FROM entries
WHERE kind = 'self_transfer';

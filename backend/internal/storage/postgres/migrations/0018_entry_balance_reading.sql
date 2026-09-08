-- 0018_entry_balance_reading: entries.balance_reading holds a
-- balance_adjustment's absolute reading (previously stored in amount);
-- amount becomes a uniform signed delta for both kinds, so Balance() can
-- be a single SUM with no kind-branching. See openspec change
-- add-account-flow-chart's design.md for the recompute algorithm and its
-- worked example.

ALTER TABLE entries ADD COLUMN balance_reading bigint;

-- Pass 1: copy every balance_adjustment's current (absolute) amount into
-- balance_reading before amount is touched by pass 2.
UPDATE entries SET balance_reading = amount WHERE kind = 'balance_adjustment';

-- Pass 2: recompute amount for every live (non-deleted) balance_adjustment
-- as its delta from the balance immediately before it, reproducing the
-- pre-migration Balance() formula ("latest non-deleted balance_adjustment
-- at or before a point, or 0, plus every non-deleted transaction strictly
-- after it") exactly. A deleted adjustment is skipped by the window (so a
-- later live adjustment's "previous" correctly looks past it, matching
-- Balance()'s own "latest non-deleted" rule) and keeps its old, now-stale
-- amount — harmless, since a deleted row never contributes to Balance().
WITH adj AS (
    SELECT id, account_id, booking_timestamp, balance_reading,
           LAG(id) OVER w AS prev_id,
           LAG(booking_timestamp) OVER w AS prev_ts,
           LAG(balance_reading) OVER w AS prev_balance
    FROM entries
    WHERE kind = 'balance_adjustment' AND deleted_at IS NULL
    WINDOW w AS (PARTITION BY account_id ORDER BY booking_timestamp, id)
),
txn_between AS (
    SELECT a.id, COALESCE(SUM(t.amount), 0) AS total
    FROM adj a
    LEFT JOIN entries t
        ON t.account_id = a.account_id
       AND t.kind = 'transaction'
       AND t.deleted_at IS NULL
       AND (t.booking_timestamp, t.id) > (COALESCE(a.prev_ts, '-infinity'::timestamptz), COALESCE(a.prev_id, 0))
       AND (t.booking_timestamp, t.id) <= (a.booking_timestamp, a.id)
    GROUP BY a.id
)
UPDATE entries e
SET amount = a.balance_reading - COALESCE(a.prev_balance, 0) - tb.total
FROM adj a
JOIN txn_between tb ON tb.id = a.id
WHERE e.id = a.id;

-- Added last so the two passes above are never fought by it mid-backfill.
ALTER TABLE entries ADD CONSTRAINT entries_balance_reading_matches_kind
    CHECK ((kind = 'balance_adjustment') = (balance_reading IS NOT NULL));

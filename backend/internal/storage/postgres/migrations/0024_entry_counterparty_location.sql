-- 0024_entry_counterparty_location: two nullable, backend-uninterpreted
-- text columns on entries. counterparty is the other party in a
-- transaction (who was paid, or who paid), suggested to the caller from
-- their own history (see GET /api/entries/counterparties) and folded into
-- the existing free-text q search. location holds either a typed address
-- or a JSON-encoded {"lat":…,"lng":…} coordinate string produced by
-- device GPS or a dropped map pin — the backend never parses it, the web
-- client decides which it's looking at. Both are accepted only when
-- kind = 'transaction', enforced in application code (like category_id's
-- kind-gating), not by a CHECK constraint, since either being set on a
-- balance_adjustment is simply rejected outright rather than storable in
-- an inconsistent state. See openspec change
-- `add-entry-counterparty-location`.

ALTER TABLE entries ADD COLUMN counterparty text;
ALTER TABLE entries ADD COLUMN location text;

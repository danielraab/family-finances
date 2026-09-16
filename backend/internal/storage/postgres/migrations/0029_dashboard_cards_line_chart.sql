-- 0029_dashboard_cards_line_chart: widen dashboard_cards' type check
-- constraint to allow line_chart (a running-balance line chart card),
-- alongside the existing account_stat/query_stat/entry_list/bar_chart —
-- see openspec change `line-chart-overlay-and-card`.

ALTER TABLE dashboard_cards DROP CONSTRAINT dashboard_cards_type_check;
ALTER TABLE dashboard_cards ADD CONSTRAINT dashboard_cards_type_check
    CHECK (type IN ('account_stat', 'query_stat', 'entry_list', 'bar_chart', 'line_chart'));

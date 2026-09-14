-- 0027_dashboard_cards: each user's own, ordered set of /home dashboard
-- cards. A card's type is fixed at creation (account_stat, query_stat,
-- entry_list, bar_chart); config holds its inline filter/target as a
-- plain JSON object, meaning depending on type — validated by
-- internal/dashboard, not by the database. There is no sharing: a
-- dashboard is always private to its owner, so owner_id is the only
-- scoping this table needs. sort_order is meaningful only among a given
-- owner_id's own cards. See openspec change `add-customizable-dashboard`.

CREATE TABLE dashboard_cards (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   uuid NOT NULL REFERENCES users(id),
    type       text NOT NULL CHECK (type IN ('account_stat', 'query_stat', 'entry_list', 'bar_chart')),
    config     jsonb NOT NULL DEFAULT '{}'::jsonb,
    sort_order integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX dashboard_cards_owner_idx ON dashboard_cards (owner_id, sort_order);

package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"at.draab/familyfinances/internal/dashboard"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardStore is the PostgreSQL implementation of dashboard.Store.
type DashboardStore struct {
	pool *pgxpool.Pool
}

// NewDashboardStore returns a DashboardStore over pool.
func NewDashboardStore(pool *pgxpool.Pool) *DashboardStore { return &DashboardStore{pool: pool} }

const dashboardCardCols = `id::text, type, config, sort_order`

func scanDashboardCard(row pgx.Row, ownerID string) (dashboard.Card, error) {
	var c dashboard.Card
	var typ string
	var raw []byte
	err := row.Scan(&c.ID, &typ, &raw, &c.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return dashboard.Card{}, dashboard.ErrNotFound
	}
	if err != nil {
		return dashboard.Card{}, err
	}
	c.Type = dashboard.CardType(typ)
	if err := json.Unmarshal(raw, &c.Config); err != nil {
		return dashboard.Card{}, err
	}
	c.OwnerID = ownerID
	return c, nil
}

func (s *DashboardStore) List(ctx context.Context, ownerID string) ([]dashboard.Card, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+dashboardCardCols+` FROM dashboard_cards WHERE owner_id = $1 ORDER BY sort_order, id`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dashboard.Card
	for rows.Next() {
		c, err := scanDashboardCard(rows, ownerID)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *DashboardStore) Get(ctx context.Context, ownerID, id string) (dashboard.Card, error) {
	return scanDashboardCard(s.pool.QueryRow(ctx,
		`SELECT `+dashboardCardCols+` FROM dashboard_cards WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	), ownerID)
}

func (s *DashboardStore) Create(ctx context.Context, ownerID string, in dashboard.New) (dashboard.Card, error) {
	raw, err := json.Marshal(in.Config)
	if err != nil {
		return dashboard.Card{}, err
	}
	return scanDashboardCard(s.pool.QueryRow(ctx, `
		INSERT INTO dashboard_cards (owner_id, type, config, sort_order)
		VALUES ($1, $2, $3::jsonb, (
			SELECT COALESCE(MAX(sort_order) + 1, 0) FROM dashboard_cards WHERE owner_id = $1
		))
		RETURNING `+dashboardCardCols,
		ownerID, string(in.Type), raw,
	), ownerID)
}

func (s *DashboardStore) Update(ctx context.Context, ownerID, id string, upd dashboard.Update) (dashboard.Card, error) {
	raw, err := json.Marshal(upd.Config)
	if err != nil {
		return dashboard.Card{}, err
	}
	return scanDashboardCard(s.pool.QueryRow(ctx, `
		UPDATE dashboard_cards SET config = $3::jsonb, updated_at = now()
		WHERE id = $1 AND owner_id = $2
		RETURNING `+dashboardCardCols,
		id, ownerID, raw,
	), ownerID)
}

func (s *DashboardStore) Delete(ctx context.Context, ownerID, id string) error {
	cmdTag, err := s.pool.Exec(ctx,
		`DELETE FROM dashboard_cards WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return dashboard.ErrNotFound
	}
	return nil
}

// move swaps id's sort_order with its adjacent card in ownerID's own list
// — the previous one (up=true) or the next one (up=false). A no-op (the
// row unchanged, no error) if there is no such neighbor. Mirrors
// internal/storage/postgres/category.go's move, minus the sibling-group
// (parent_id) scoping a dashboard card has no equivalent of.
func (s *DashboardStore) move(ctx context.Context, ownerID, id string, up bool) (dashboard.Card, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dashboard.Card{}, err
	}
	defer tx.Rollback(ctx)

	current, err := scanDashboardCard(tx.QueryRow(ctx,
		`SELECT `+dashboardCardCols+` FROM dashboard_cards WHERE id = $1 AND owner_id = $2 FOR UPDATE`,
		id, ownerID,
	), ownerID)
	if err != nil {
		return dashboard.Card{}, err
	}

	cmp, order := "<", "DESC"
	if !up {
		cmp, order = ">", "ASC"
	}
	var neighborID string
	var neighborSortOrder int
	err = tx.QueryRow(ctx, `
		SELECT id::text, sort_order FROM dashboard_cards
		WHERE owner_id = $1 AND id::text != $2 AND (sort_order, id::text) `+cmp+` ($3, $2)
		ORDER BY sort_order `+order+`, id::text `+order+`
		LIMIT 1
		FOR UPDATE`,
		ownerID, id, current.SortOrder,
	).Scan(&neighborID, &neighborSortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return current, nil
	}
	if err != nil {
		return dashboard.Card{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE dashboard_cards SET sort_order = $2 WHERE id = $1`, neighborID, current.SortOrder); err != nil {
		return dashboard.Card{}, err
	}
	updated, err := scanDashboardCard(tx.QueryRow(ctx,
		`UPDATE dashboard_cards SET sort_order = $2 WHERE id = $1 RETURNING `+dashboardCardCols,
		id, neighborSortOrder,
	), ownerID)
	if err != nil {
		return dashboard.Card{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return dashboard.Card{}, err
	}
	return updated, nil
}

func (s *DashboardStore) MoveUp(ctx context.Context, ownerID, id string) (dashboard.Card, error) {
	return s.move(ctx, ownerID, id, true)
}

func (s *DashboardStore) MoveDown(ctx context.Context, ownerID, id string) (dashboard.Card, error) {
	return s.move(ctx, ownerID, id, false)
}

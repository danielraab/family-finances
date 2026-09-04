package postgres

import (
	"context"
	"errors"

	"at.draab/familyfinances/internal/settings"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingsStore is the PostgreSQL implementation of settings.Store.
type SettingsStore struct {
	pool *pgxpool.Pool
}

// NewSettingsStore returns a SettingsStore over pool.
func NewSettingsStore(pool *pgxpool.Pool) *SettingsStore { return &SettingsStore{pool: pool} }

func (s *SettingsStore) Get(ctx context.Context, userID string) (settings.Row, error) {
	var row settings.Row
	err := s.pool.QueryRow(ctx,
		`SELECT language, timezone, default_currency FROM user_settings WHERE user_id = $1`,
		userID,
	).Scan(&row.Language, &row.Timezone, &row.DefaultCurrency)
	if errors.Is(err, pgx.ErrNoRows) {
		return settings.Row{}, nil
	}
	if err != nil {
		return settings.Row{}, err
	}
	return row, nil
}

// Upsert inserts or merges — COALESCE keeps any column not present in upd at
// its current value (or NULL, on first insert), so a partial update never
// clobbers the other two fields.
func (s *SettingsStore) Upsert(ctx context.Context, userID string, upd settings.Update) (settings.Row, error) {
	var row settings.Row
	err := s.pool.QueryRow(ctx, `
		INSERT INTO user_settings (user_id, language, timezone, default_currency)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			language         = COALESCE($2, user_settings.language),
			timezone         = COALESCE($3, user_settings.timezone),
			default_currency = COALESCE($4, user_settings.default_currency),
			updated_at       = now()
		RETURNING language, timezone, default_currency`,
		userID, upd.Language, upd.Timezone, upd.DefaultCurrency,
	).Scan(&row.Language, &row.Timezone, &row.DefaultCurrency)
	if err != nil {
		return settings.Row{}, err
	}
	return row, nil
}

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
		`SELECT language, timezone, default_currency, displayed_decimal_places, week_start, recurring_preview_horizon FROM user_settings WHERE user_id = $1`,
		userID,
	).Scan(&row.Language, &row.Timezone, &row.DefaultCurrency, &row.DisplayedDecimalPlaces, &row.WeekStart, &row.RecurringPreviewHorizon)
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
// clobbers the other fields.
func (s *SettingsStore) Upsert(ctx context.Context, userID string, upd settings.Update) (settings.Row, error) {
	var row settings.Row
	err := s.pool.QueryRow(ctx, `
		INSERT INTO user_settings (user_id, language, timezone, default_currency, displayed_decimal_places, week_start, recurring_preview_horizon)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			language                   = COALESCE($2, user_settings.language),
			timezone                   = COALESCE($3, user_settings.timezone),
			default_currency           = COALESCE($4, user_settings.default_currency),
			displayed_decimal_places   = COALESCE($5, user_settings.displayed_decimal_places),
			week_start                 = COALESCE($6, user_settings.week_start),
			recurring_preview_horizon  = COALESCE($7, user_settings.recurring_preview_horizon),
			updated_at                 = now()
		RETURNING language, timezone, default_currency, displayed_decimal_places, week_start, recurring_preview_horizon`,
		userID, upd.Language, upd.Timezone, upd.DefaultCurrency, upd.DisplayedDecimalPlaces, upd.WeekStart, upd.RecurringPreviewHorizon,
	).Scan(&row.Language, &row.Timezone, &row.DefaultCurrency, &row.DisplayedDecimalPlaces, &row.WeekStart, &row.RecurringPreviewHorizon)
	if err != nil {
		return settings.Row{}, err
	}
	return row, nil
}

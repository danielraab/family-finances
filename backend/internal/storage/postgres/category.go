package postgres

import (
	"context"
	"errors"

	"at.draab/familyfinances/internal/category"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CategoryStore is the PostgreSQL implementation of category.Store.
type CategoryStore struct {
	pool *pgxpool.Pool
}

// NewCategoryStore returns a CategoryStore over pool.
func NewCategoryStore(pool *pgxpool.Pool) *CategoryStore { return &CategoryStore{pool: pool} }

const categoryCols = `id::text, parent_id::text, name, created_at`

func scanCategory(row pgx.Row) (category.Category, error) {
	var c category.Category
	err := row.Scan(&c.ID, &c.ParentID, &c.Name, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return category.Category{}, category.ErrNotFound
	}
	return c, err
}

func (s *CategoryStore) List(ctx context.Context) ([]category.Category, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+categoryCols+` FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []category.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *CategoryStore) Get(ctx context.Context, id string) (category.Category, error) {
	return scanCategory(s.pool.QueryRow(ctx, `SELECT `+categoryCols+` FROM categories WHERE id = $1`, id))
}

func (s *CategoryStore) Create(ctx context.Context, in category.New) (category.Category, error) {
	c, err := scanCategory(s.pool.QueryRow(ctx,
		`INSERT INTO categories (parent_id, name) VALUES ($1, $2) RETURNING `+categoryCols,
		in.ParentID, in.Name,
	))
	if isForeignKeyViolation(err) || isUniqueViolation(err) {
		return category.Category{}, category.ErrInvalidValue
	}
	return c, err
}

func (s *CategoryStore) Update(ctx context.Context, id string, upd category.Update) (category.Category, error) {
	var parentSet bool
	var parentID *string
	if upd.ParentID.Set {
		parentSet = true
		parentID = upd.ParentID.Value
	}
	c, err := scanCategory(s.pool.QueryRow(ctx, `
		UPDATE categories SET
			name      = COALESCE($2, name),
			parent_id = CASE WHEN $3 THEN $4 ELSE parent_id END
		WHERE id = $1
		RETURNING `+categoryCols,
		id, upd.Name, parentSet, parentID,
	))
	if isForeignKeyViolation(err) || isUniqueViolation(err) {
		return category.Category{}, category.ErrInvalidValue
	}
	return c, err
}

func (s *CategoryStore) Delete(ctx context.Context, id string) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return category.ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM categories
		WHERE id = $1
			AND NOT EXISTS (SELECT 1 FROM categories WHERE parent_id = $1)
			AND NOT EXISTS (SELECT 1 FROM entries WHERE category_id = $1 AND deleted_at IS NULL)`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return category.ErrInUse
	}
	return nil
}

func (s *CategoryStore) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

// Subtree returns id and every descendant id via a recursive CTE.
func (s *CategoryStore) Subtree(ctx context.Context, id string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT id FROM categories WHERE id = $1
			UNION ALL
			SELECT c.id FROM categories c
			JOIN subtree s ON c.parent_id = s.id
		)
		SELECT id::text FROM subtree`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var cid string
		if err := rows.Scan(&cid); err != nil {
			return nil, err
		}
		out = append(out, cid)
	}
	return out, rows.Err()
}

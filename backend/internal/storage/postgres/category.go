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

const categoryCols = `id::text, parent_id::text, name, sort_order, disabled, created_at`

func scanCategory(row pgx.Row) (category.Category, error) {
	var c category.Category
	err := row.Scan(&c.ID, &c.ParentID, &c.Name, &c.SortOrder, &c.Disabled, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return category.Category{}, category.ErrNotFound
	}
	return c, err
}

func (s *CategoryStore) List(ctx context.Context, ownerID string) ([]category.Category, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+categoryCols+` FROM categories WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY sort_order, id`,
		ownerID,
	)
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

func (s *CategoryStore) Get(ctx context.Context, ownerID, id string) (category.Category, error) {
	return scanCategory(s.pool.QueryRow(ctx,
		`SELECT `+categoryCols+` FROM categories WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`,
		id, ownerID,
	))
}

func (s *CategoryStore) Create(ctx context.Context, ownerID string, in category.New) (category.Category, error) {
	c, err := scanCategory(s.pool.QueryRow(ctx, `
		INSERT INTO categories (owner_id, parent_id, name, sort_order)
		VALUES ($1, $2, $3, (
			SELECT COALESCE(MAX(sort_order) + 1, 0) FROM categories
			WHERE owner_id = $1 AND parent_id IS NOT DISTINCT FROM $2 AND deleted_at IS NULL
		))
		RETURNING `+categoryCols,
		ownerID, in.ParentID, in.Name,
	))
	if isForeignKeyViolation(err) {
		return category.Category{}, category.ErrInvalidValue
	}
	return c, err
}

func (s *CategoryStore) Update(ctx context.Context, ownerID, id string, upd category.Update) (category.Category, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return category.Category{}, err
	}
	defer tx.Rollback(ctx)

	current, err := scanCategory(tx.QueryRow(ctx,
		`SELECT `+categoryCols+` FROM categories WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL FOR UPDATE`,
		id, ownerID,
	))
	if err != nil {
		return category.Category{}, err
	}

	newParent := current.ParentID
	reparented := false
	if upd.ParentID.Set {
		newParent = upd.ParentID.Value
		reparented = !equalStringPtr(current.ParentID, newParent)
	}

	sortOrder := current.SortOrder
	if reparented {
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(sort_order) + 1, 0) FROM categories
			WHERE owner_id = $1 AND parent_id IS NOT DISTINCT FROM $2 AND deleted_at IS NULL AND id::text != $3`,
			ownerID, newParent, id,
		).Scan(&sortOrder); err != nil {
			return category.Category{}, err
		}
	}

	c, err := scanCategory(tx.QueryRow(ctx, `
		UPDATE categories SET
			name       = COALESCE($3, name),
			parent_id  = CASE WHEN $4 THEN $5 ELSE parent_id END,
			sort_order = $6
		WHERE id = $1 AND owner_id = $2
		RETURNING `+categoryCols,
		id, ownerID, upd.Name, upd.ParentID.Set, newParent, sortOrder,
	))
	if isForeignKeyViolation(err) {
		return category.Category{}, category.ErrInvalidValue
	}
	if err != nil {
		return category.Category{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return category.Category{}, err
	}
	return c, nil
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func (s *CategoryStore) Delete(ctx context.Context, ownerID, id string) error {
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL)`,
		id, ownerID,
	).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return category.ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE categories SET deleted_at = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM categories WHERE parent_id = $1 AND deleted_at IS NULL)
			AND NOT EXISTS (SELECT 1 FROM entries WHERE category_id = $1 AND deleted_at IS NULL)`,
		id, ownerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return category.ErrInUse
	}
	return nil
}

func (s *CategoryStore) SetDisabled(ctx context.Context, ownerID, id string, disabled bool) (category.Category, error) {
	return scanCategory(s.pool.QueryRow(ctx, `
		UPDATE categories SET disabled = $3
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
		RETURNING `+categoryCols,
		id, ownerID, disabled,
	))
}

// move swaps id's sort_order with its adjacent sibling — the previous one
// (up=true) or the next one (up=false). A no-op (the row unchanged, no
// error) if there is no such sibling.
func (s *CategoryStore) move(ctx context.Context, ownerID, id string, up bool) (category.Category, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return category.Category{}, err
	}
	defer tx.Rollback(ctx)

	current, err := scanCategory(tx.QueryRow(ctx,
		`SELECT `+categoryCols+` FROM categories WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL FOR UPDATE`,
		id, ownerID,
	))
	if err != nil {
		return category.Category{}, err
	}

	cmp, order := "<", "DESC"
	if !up {
		cmp, order = ">", "ASC"
	}
	var sibID string
	var sibSortOrder int
	err = tx.QueryRow(ctx, `
		SELECT id::text, sort_order FROM categories
		WHERE owner_id = $1 AND parent_id IS NOT DISTINCT FROM $2 AND deleted_at IS NULL AND id::text != $3
			AND (sort_order, id::text) `+cmp+` ($4, $3)
		ORDER BY sort_order `+order+`, id::text `+order+`
		LIMIT 1
		FOR UPDATE`,
		ownerID, current.ParentID, id, current.SortOrder,
	).Scan(&sibID, &sibSortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return current, nil
	}
	if err != nil {
		return category.Category{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE categories SET sort_order = $2 WHERE id = $1`, sibID, current.SortOrder); err != nil {
		return category.Category{}, err
	}
	updated, err := scanCategory(tx.QueryRow(ctx,
		`UPDATE categories SET sort_order = $2 WHERE id = $1 RETURNING `+categoryCols,
		id, sibSortOrder,
	))
	if err != nil {
		return category.Category{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return category.Category{}, err
	}
	return updated, nil
}

func (s *CategoryStore) MoveUp(ctx context.Context, ownerID, id string) (category.Category, error) {
	return s.move(ctx, ownerID, id, true)
}

func (s *CategoryStore) MoveDown(ctx context.Context, ownerID, id string) (category.Category, error) {
	return s.move(ctx, ownerID, id, false)
}

func (s *CategoryStore) Exists(ctx context.Context, ownerID, id string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL)`,
		id, ownerID,
	).Scan(&exists)
	return exists, err
}

func (s *CategoryStore) SeedDefaults(ctx context.Context, ownerID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO categories (owner_id, name, sort_order)
		SELECT $1, name, ord - 1
		FROM unnest($2::text[]) WITH ORDINALITY AS t(name, ord)`,
		ownerID, category.DefaultNames,
	)
	return err
}

// Subtree returns id and every descendant id, scoped to ownerID's own
// tree, via a recursive CTE.
func (s *CategoryStore) Subtree(ctx context.Context, ownerID, id string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT id FROM categories WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
			UNION ALL
			SELECT c.id FROM categories c
			JOIN subtree s ON c.parent_id = s.id
			WHERE c.owner_id = $2 AND c.deleted_at IS NULL
		)
		SELECT id::text FROM subtree`,
		id, ownerID,
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

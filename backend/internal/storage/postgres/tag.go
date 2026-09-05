package postgres

import (
	"context"
	"errors"

	"at.draab/familyfinances/internal/tag"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TagStore is the PostgreSQL implementation of tag.Store.
type TagStore struct {
	pool *pgxpool.Pool
}

// NewTagStore returns a TagStore over pool.
func NewTagStore(pool *pgxpool.Pool) *TagStore { return &TagStore{pool: pool} }

const tagCols = `id::text, name, created_at`

func scanTag(row pgx.Row, ownerID string) (tag.Tag, error) {
	var t tag.Tag
	err := row.Scan(&t.ID, &t.Name, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return tag.Tag{}, tag.ErrNotFound
	}
	t.OwnerID = ownerID
	return t, err
}

func (s *TagStore) List(ctx context.Context, ownerID string) ([]tag.Tag, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+tagCols+` FROM tags WHERE owner_id = $1 ORDER BY name`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tag.Tag
	for rows.Next() {
		t, err := scanTag(rows, ownerID)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *TagStore) Get(ctx context.Context, ownerID, id string) (tag.Tag, error) {
	return scanTag(s.pool.QueryRow(ctx,
		`SELECT `+tagCols+` FROM tags WHERE id = $1 AND owner_id = $2`, id, ownerID,
	), ownerID)
}

func (s *TagStore) ByName(ctx context.Context, ownerID, name string) (tag.Tag, error) {
	return scanTag(s.pool.QueryRow(ctx,
		`SELECT `+tagCols+` FROM tags WHERE owner_id = $1 AND name = $2`, ownerID, name,
	), ownerID)
}

func (s *TagStore) Create(ctx context.Context, ownerID, name string) (tag.Tag, error) {
	t, err := scanTag(s.pool.QueryRow(ctx,
		`INSERT INTO tags (owner_id, name) VALUES ($1, $2) RETURNING `+tagCols,
		ownerID, name,
	), ownerID)
	if isUniqueViolation(err) {
		return tag.Tag{}, tag.ErrDuplicateName
	}
	return t, err
}

func (s *TagStore) Update(ctx context.Context, ownerID, id, name string) (tag.Tag, error) {
	t, err := scanTag(s.pool.QueryRow(ctx,
		`UPDATE tags SET name = $3 WHERE id = $1 AND owner_id = $2 RETURNING `+tagCols,
		id, ownerID, name,
	), ownerID)
	if isUniqueViolation(err) {
		return tag.Tag{}, tag.ErrDuplicateName
	}
	return t, err
}

func (s *TagStore) Delete(ctx context.Context, ownerID, id string) error {
	cmdTag, err := s.pool.Exec(ctx, `DELETE FROM tags WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return tag.ErrNotFound
	}
	return nil
}

func (s *TagStore) OwnedBy(ctx context.Context, ownerID string, tagIDs []string) (bool, error) {
	if len(tagIDs) == 0 {
		return true, nil
	}
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM tags WHERE owner_id = $1 AND id = ANY($2::uuid[])`,
		ownerID, tagIDs,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == len(uniqueStrings(tagIDs)), nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

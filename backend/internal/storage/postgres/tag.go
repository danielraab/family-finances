package postgres

import (
	"context"
	"errors"
	"fmt"

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

// tagCols is the plain column list, parameterized by callerParam — the
// 1-based positional parameter index bound to whichever user's own entries
// entry_count should reflect. For the strictly-owner-scoped methods below
// (Create/Update/SetDisabled/Get), that's always the ownerID param, since
// caller and owner are the same there; GetForCaller/List bind their own
// callerID instead, who may be a share recipient rather than the real
// owner. Mirrors internal/storage/postgres/category.go's categoryCols.
func tagCols(callerParam int) string {
	return fmt.Sprintf(`id::text, name, disabled, created_at, (
		SELECT count(*) FROM entry_tags et
			JOIN entries e ON e.id = et.entry_id AND e.deleted_at IS NULL
		WHERE et.tag_id = tags.id AND e.created_by = $%[1]d
	)`, callerParam)
}

// tagViewCols is tagCols plus three caller-dependent columns — permission,
// shared, owner_name — used by GetForCaller/List, mirroring
// internal/storage/postgres/category.go's categoryViewCols.
func tagViewCols(callerParam int) string {
	return tagCols(callerParam) + fmt.Sprintf(`,
	CASE WHEN owner_id = $%[1]d THEN 'owner' ELSE COALESCE(
		(SELECT permission FROM tag_shares WHERE tag_id = tags.id AND user_id = $%[1]d), ''
	) END,
	(owner_id != $%[1]d),
	COALESCE((SELECT COALESCE(display_name, email) FROM users WHERE users.id = tags.owner_id), '')`, callerParam)
}

func scanTag(row pgx.Row, ownerID string) (tag.Tag, error) {
	var t tag.Tag
	err := row.Scan(&t.ID, &t.Name, &t.Disabled, &t.CreatedAt, &t.EntryCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return tag.Tag{}, tag.ErrNotFound
	}
	t.OwnerID = ownerID
	return t, err
}

func scanTagView(row pgx.Row) (tag.Tag, error) {
	var t tag.Tag
	var permission string
	err := row.Scan(&t.ID, &t.Name, &t.Disabled, &t.CreatedAt, &t.EntryCount,
		&permission, &t.Shared, &t.OwnerName)
	if errors.Is(err, pgx.ErrNoRows) {
		return tag.Tag{}, tag.ErrNotFound
	}
	if err != nil {
		return tag.Tag{}, err
	}
	t.Permission = tag.Permission(permission)
	return t, nil
}

func (s *TagStore) List(ctx context.Context, callerID string) ([]tag.Tag, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+tagViewCols(1)+` FROM tags
		 WHERE owner_id = $1 OR EXISTS (
		     SELECT 1 FROM tag_shares WHERE tag_id = tags.id AND user_id = $1
		 )
		 ORDER BY name`,
		callerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tag.Tag
	for rows.Next() {
		t, err := scanTagView(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *TagStore) Get(ctx context.Context, ownerID, id string) (tag.Tag, error) {
	return scanTag(s.pool.QueryRow(ctx,
		`SELECT `+tagCols(2)+` FROM tags WHERE id = $1 AND owner_id = $2`, id, ownerID,
	), ownerID)
}

func (s *TagStore) GetForCaller(ctx context.Context, callerID, id string) (tag.Tag, error) {
	return scanTagView(s.pool.QueryRow(ctx,
		`SELECT `+tagViewCols(2)+` FROM tags WHERE id = $1`,
		id, callerID,
	))
}

func (s *TagStore) ByName(ctx context.Context, ownerID, name string) (tag.Tag, error) {
	return scanTag(s.pool.QueryRow(ctx,
		`SELECT `+tagCols(1)+` FROM tags WHERE owner_id = $1 AND name = $2`, ownerID, name,
	), ownerID)
}

func (s *TagStore) Create(ctx context.Context, ownerID, name string) (tag.Tag, error) {
	t, err := scanTag(s.pool.QueryRow(ctx,
		`INSERT INTO tags (owner_id, name) VALUES ($1, $2) RETURNING `+tagCols(1),
		ownerID, name,
	), ownerID)
	if isUniqueViolation(err) {
		return tag.Tag{}, tag.ErrDuplicateName
	}
	return t, err
}

func (s *TagStore) Update(ctx context.Context, ownerID, id, name string) (tag.Tag, error) {
	t, err := scanTag(s.pool.QueryRow(ctx,
		`UPDATE tags SET name = $3 WHERE id = $1 AND owner_id = $2 RETURNING `+tagCols(2),
		id, ownerID, name,
	), ownerID)
	if isUniqueViolation(err) {
		return tag.Tag{}, tag.ErrDuplicateName
	}
	return t, err
}

func (s *TagStore) Delete(ctx context.Context, ownerID, id string) error {
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1 AND owner_id = $2)`,
		id, ownerID,
	).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return tag.ErrNotFound
	}
	cmdTag, err := s.pool.Exec(ctx, `
		DELETE FROM tags WHERE id = $1 AND owner_id = $2
			AND NOT EXISTS (SELECT 1 FROM tag_shares WHERE tag_id = $1)`,
		id, ownerID,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return tag.ErrInUse
	}
	return nil
}

func (s *TagStore) SetDisabled(ctx context.Context, ownerID, id string, disabled bool) (tag.Tag, error) {
	return scanTag(s.pool.QueryRow(ctx,
		`UPDATE tags SET disabled = $3 WHERE id = $1 AND owner_id = $2 RETURNING `+tagCols(2),
		id, ownerID, disabled,
	), ownerID)
}

func (s *TagStore) OwnedBy(ctx context.Context, callerID string, tagIDs []string) (bool, error) {
	if len(tagIDs) == 0 {
		return true, nil
	}
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT count(DISTINCT tags.id) FROM tags
		WHERE tags.id = ANY($2::uuid[])
			AND (tags.owner_id = $1 OR EXISTS (
				SELECT 1 FROM tag_shares WHERE tag_shares.tag_id = tags.id AND tag_shares.user_id = $1
			))`,
		callerID, tagIDs,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == len(uniqueStrings(tagIDs)), nil
}

func (s *TagStore) Usable(ctx context.Context, callerID string, tagIDs []string) (bool, error) {
	if len(tagIDs) == 0 {
		return true, nil
	}
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT count(DISTINCT tags.id) FROM tags
		WHERE tags.id = ANY($2::uuid[]) AND tags.disabled = false
			AND (tags.owner_id = $1 OR EXISTS (
				SELECT 1 FROM tag_shares
				WHERE tag_shares.tag_id = tags.id AND tag_shares.user_id = $1 AND tag_shares.permission = 'append'
			))`,
		callerID, tagIDs,
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

// --- sharing -------------------------------------------------------------

const tagShareCols = `tag_shares.user_id::text, COALESCE(u.display_name, u.email), u.email,
	permission, granted_by::text, COALESCE(gu.display_name, gu.email), tag_shares.created_at, tag_shares.updated_at`

const tagShareJoins = `FROM tag_shares
	JOIN users u ON u.id = tag_shares.user_id
	JOIN users gu ON gu.id = tag_shares.granted_by`

func scanTagShare(row pgx.Row, tagID string) (tag.TagShare, error) {
	var sh tag.TagShare
	err := row.Scan(&sh.UserID, &sh.Name, &sh.Email, &sh.Permission, &sh.GrantedBy, &sh.GrantedByName, &sh.CreatedAt, &sh.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return tag.TagShare{}, tag.ErrNotFound
	}
	sh.TagID = tagID
	return sh, err
}

func (s *TagStore) CreateOrUpdateShare(ctx context.Context, tagID, userID string, permission tag.Permission, grantedBy string) (tag.TagShare, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tag_shares (tag_id, user_id, permission, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tag_id, user_id)
		DO UPDATE SET permission = $3, granted_by = $4, updated_at = now()`,
		tagID, userID, permission, grantedBy,
	)
	if isForeignKeyViolation(err) {
		return tag.TagShare{}, tag.ErrInvalidValue
	}
	if err != nil {
		return tag.TagShare{}, err
	}
	return scanTagShare(s.pool.QueryRow(ctx,
		`SELECT `+tagShareCols+` `+tagShareJoins+` WHERE tag_shares.tag_id = $1 AND tag_shares.user_id = $2`,
		tagID, userID,
	), tagID)
}

func (s *TagStore) ListShares(ctx context.Context, tagID string) ([]tag.TagShare, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+tagShareCols+` `+tagShareJoins+` WHERE tag_shares.tag_id = $1 ORDER BY tag_shares.created_at`,
		tagID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tag.TagShare
	for rows.Next() {
		sh, err := scanTagShare(rows, tagID)
		if err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

func (s *TagStore) ShareByUser(ctx context.Context, tagID, userID string) (tag.TagShare, error) {
	return scanTagShare(s.pool.QueryRow(ctx,
		`SELECT `+tagShareCols+` `+tagShareJoins+` WHERE tag_shares.tag_id = $1 AND tag_shares.user_id = $2`,
		tagID, userID,
	), tagID)
}

func (s *TagStore) UpdateSharePermission(ctx context.Context, tagID, userID string, permission tag.Permission) (tag.TagShare, error) {
	cmdTag, err := s.pool.Exec(ctx,
		`UPDATE tag_shares SET permission = $3, updated_at = now() WHERE tag_id = $1 AND user_id = $2`,
		tagID, userID, permission,
	)
	if err != nil {
		return tag.TagShare{}, err
	}
	if cmdTag.RowsAffected() == 0 {
		return tag.TagShare{}, tag.ErrNotFound
	}
	return s.ShareByUser(ctx, tagID, userID)
}

func (s *TagStore) DeleteShare(ctx context.Context, tagID, userID string) error {
	cmdTag, err := s.pool.Exec(ctx,
		`DELETE FROM tag_shares WHERE tag_id = $1 AND user_id = $2`,
		tagID, userID,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return tag.ErrNotFound
	}
	return nil
}

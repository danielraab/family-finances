package postgres

import (
	"context"
	"errors"
	"time"

	"at.draab/familyfinances/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthStore is the PostgreSQL implementation of auth.Store, backed by a pgx
// pool. Queries are parameterized; token consumption is a single atomic
// UPDATE … RETURNING; account creation runs in one transaction that takes an
// advisory lock so the zero-users bootstrap decision is race-free.
type AuthStore struct {
	pool *pgxpool.Pool
}

// NewAuthStore returns an AuthStore over pool.
func NewAuthStore(pool *pgxpool.Pool) *AuthStore { return &AuthStore{pool: pool} }

// bootstrapLockKey serializes account creation so exactly one transaction can
// observe an empty users table.
const bootstrapLockKey = 0x66665f626f6f74 // "ff_boot"

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// --- users ---------------------------------------------------------------

func (a *AuthStore) UserCount(ctx context.Context) (int, error) {
	var n int
	err := a.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

const userCols = `id::text, email, COALESCE(display_name, ''), is_admin, created_at, disabled, deleted_at`

func scanUser(row pgx.Row) (auth.User, error) {
	var u auth.User
	if err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.IsAdmin, &u.CreatedAt, &u.Disabled, &u.DeletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrNotFound
		}
		return auth.User{}, err
	}
	return u, nil
}

func (a *AuthStore) UserByID(ctx context.Context, id string) (auth.User, error) {
	return scanUser(a.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

func (a *AuthStore) UserByEmail(ctx context.Context, email string) (auth.User, error) {
	return scanUser(a.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email = $1`, auth.NormalizeEmail(email)))
}

func (a *AuthStore) SetUserAdmin(ctx context.Context, email string, isAdmin bool) error {
	tag, err := a.pool.Exec(ctx, `UPDATE users SET is_admin = $2 WHERE email = $1`, auth.NormalizeEmail(email), isAdmin)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrNotFound
	}
	return nil
}

func (a *AuthStore) ListAdminEmails(ctx context.Context) ([]string, error) {
	rows, err := a.pool.Query(ctx, `SELECT email FROM users WHERE is_admin ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// --- admin: users --------------------------------------------------------

func (a *AuthStore) ListUsers(ctx context.Context) ([]auth.User, error) {
	rows, err := a.pool.Query(ctx, `SELECT `+userCols+` FROM users WHERE deleted_at IS NULL ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []auth.User
	for rows.Next() {
		var u auth.User
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.IsAdmin, &u.CreatedAt, &u.Disabled, &u.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (a *AuthStore) SetUserDisabled(ctx context.Context, id string, disabled bool) error {
	tag, err := a.pool.Exec(ctx, `UPDATE users SET disabled = $2 WHERE id = $1`, id, disabled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrNotFound
	}
	return nil
}

func (a *AuthStore) SoftDeleteUser(ctx context.Context, id string, now time.Time) error {
	tag, err := a.pool.Exec(ctx, `UPDATE users SET deleted_at = $2 WHERE id = $1`, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrNotFound
	}
	return nil
}

func (a *AuthStore) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	_, err := a.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

// --- invite listing -------------------------------------------------------

const inviteInfoCols = `i.id::text, i.email, u.id::text, u.email, COALESCE(u.display_name, ''),
	       i.created_at, i.expires_at, i.accepted_at, i.revoked_at`

func scanInviteInfo(rows pgx.Rows) (auth.InviteInfo, error) {
	var inv auth.InviteInfo
	err := rows.Scan(&inv.ID, &inv.Email, &inv.InvitedBy.ID, &inv.InvitedBy.Email, &inv.InvitedBy.DisplayName,
		&inv.CreatedAt, &inv.ExpiresAt, &inv.AcceptedAt, &inv.RevokedAt)
	return inv, err
}

func (a *AuthStore) ListInvites(ctx context.Context) ([]auth.InviteInfo, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT `+inviteInfoCols+`
		FROM invites i
		JOIN users u ON u.id = i.invited_by
		WHERE i.deleted_at IS NULL
		ORDER BY i.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []auth.InviteInfo{} // never nil: an empty list must render as [], not JSON null
	for rows.Next() {
		inv, err := scanInviteInfo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (a *AuthStore) ListInvitesByInviter(ctx context.Context, inviterID string) ([]auth.InviteInfo, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT `+inviteInfoCols+`
		FROM invites i
		JOIN users u ON u.id = i.invited_by
		WHERE i.deleted_at IS NULL AND i.invited_by = $1
		ORDER BY i.created_at DESC`, inviterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []auth.InviteInfo{} // never nil: an empty personal list is common, unlike the admin listing
	for rows.Next() {
		inv, err := scanInviteInfo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// --- identities --------------------------------------------------------

const identityCols = `id::text, user_id::text, kind, COALESCE(email, ''), email_verified, COALESCE(provider, ''), COALESCE(subject, ''), created_at`

func scanIdentity(row pgx.Row) (auth.Identity, error) {
	var id auth.Identity
	var kind string
	if err := row.Scan(&id.ID, &id.UserID, &kind, &id.Email, &id.EmailVerified, &id.Provider, &id.Subject, &id.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.Identity{}, auth.ErrNotFound
		}
		return auth.Identity{}, err
	}
	id.Kind = auth.IdentityKind(kind)
	return id, nil
}

func (a *AuthStore) IdentityByEmail(ctx context.Context, email string) (auth.Identity, error) {
	return scanIdentity(a.pool.QueryRow(ctx,
		`SELECT `+identityCols+` FROM identities WHERE kind = 'email' AND email = $1`,
		auth.NormalizeEmail(email)))
}

func (a *AuthStore) IdentityByProviderSubject(ctx context.Context, provider, subject string) (auth.Identity, error) {
	return scanIdentity(a.pool.QueryRow(ctx,
		`SELECT `+identityCols+` FROM identities WHERE kind = 'oidc' AND provider = $1 AND subject = $2`,
		provider, subject))
}

const insertIdentitySQL = `
	INSERT INTO identities (user_id, kind, email, email_verified, provider, subject)
	VALUES ($1, $2, NULLIF($3, '')::citext, $4, NULLIF($5, ''), NULLIF($6, ''))
	RETURNING ` + identityCols

func (a *AuthStore) AddIdentity(ctx context.Context, in auth.Identity) (auth.Identity, error) {
	if in.Kind == auth.IdentityEmail {
		in.Email = auth.NormalizeEmail(in.Email)
	}
	id, err := scanIdentity(a.pool.QueryRow(ctx, insertIdentitySQL,
		in.UserID, string(in.Kind), in.Email, in.EmailVerified, in.Provider, in.Subject))
	switch {
	case isUniqueViolation(err):
		return auth.Identity{}, auth.ErrIdentityConflict
	case isForeignKeyViolation(err):
		return auth.Identity{}, auth.ErrNotFound
	case err != nil:
		return auth.Identity{}, err
	}
	return id, nil
}

func (a *AuthStore) CreateUserWithIdentity(ctx context.Context, u auth.NewUser, id auth.Identity) (auth.User, auth.Identity, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return auth.User{}, auth.Identity{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(bootstrapLockKey)); err != nil {
		return auth.User{}, auth.Identity{}, err
	}

	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n); err != nil {
		return auth.User{}, auth.Identity{}, err
	}
	isAdmin := n == 0

	user, err := scanUser(tx.QueryRow(ctx,
		`INSERT INTO users (email, display_name, is_admin)
		 VALUES ($1, NULLIF($2, ''), $3)
		 RETURNING `+userCols,
		auth.NormalizeEmail(u.Email), u.DisplayName, isAdmin))
	if isUniqueViolation(err) {
		return auth.User{}, auth.Identity{}, auth.ErrIdentityConflict
	}
	if err != nil {
		return auth.User{}, auth.Identity{}, err
	}

	if id.Kind == auth.IdentityEmail {
		id.Email = auth.NormalizeEmail(id.Email)
	}
	identity, err := scanIdentity(tx.QueryRow(ctx, insertIdentitySQL,
		user.ID, string(id.Kind), id.Email, id.EmailVerified, id.Provider, id.Subject))
	if isUniqueViolation(err) {
		return auth.User{}, auth.Identity{}, auth.ErrIdentityConflict
	}
	if err != nil {
		return auth.User{}, auth.Identity{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return auth.User{}, auth.Identity{}, err
	}
	return user, identity, nil
}

// --- sessions --------------------------------------------------------

func (a *AuthStore) CreateSession(ctx context.Context, s auth.Session, tokenHash []byte) (auth.Session, error) {
	err := a.pool.QueryRow(ctx,
		`INSERT INTO sessions (user_id, token_hash, client, user_agent, ip, created_at, last_seen_at, expires_at)
		 VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, '')::inet, $6, $7, $8)
		 RETURNING id::text`,
		s.UserID, tokenHash, string(clientOrAPI(s.Client)), s.UserAgent, s.IP,
		s.CreatedAt, s.LastSeenAt, s.ExpiresAt,
	).Scan(&s.ID)
	if err != nil {
		return auth.Session{}, err
	}
	return s, nil
}

func clientOrAPI(c auth.SessionClient) auth.SessionClient {
	if c == auth.ClientWeb || c == auth.ClientAPI {
		return c
	}
	return auth.ClientAPI
}

func (a *AuthStore) SessionByTokenHash(ctx context.Context, tokenHash []byte) (auth.Session, error) {
	var s auth.Session
	var client string
	err := a.pool.QueryRow(ctx,
		`SELECT id::text, user_id::text, client, COALESCE(user_agent, ''), COALESCE(host(ip), ''),
		        created_at, last_seen_at, expires_at
		 FROM sessions WHERE token_hash = $1`, tokenHash,
	).Scan(&s.ID, &s.UserID, &client, &s.UserAgent, &s.IP, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Session{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.Session{}, err
	}
	s.Client = auth.SessionClient(client)
	return s, nil
}

func (a *AuthStore) TouchSession(ctx context.Context, id string, lastSeen, expires time.Time) error {
	_, err := a.pool.Exec(ctx,
		`UPDATE sessions SET last_seen_at = $2, expires_at = $3 WHERE id = $1`,
		id, lastSeen, expires)
	return err
}

func (a *AuthStore) DeleteSessionByTokenHash(ctx context.Context, tokenHash []byte) error {
	tag, err := a.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrNotFound
	}
	return nil
}

// --- magic-link tokens -------------------------------------------

func (a *AuthStore) CreateMagicLinkToken(ctx context.Context, tokenHash []byte, email string, expiresAt time.Time) error {
	_, err := a.pool.Exec(ctx,
		`INSERT INTO magic_link_tokens (token_hash, email, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, auth.NormalizeEmail(email), expiresAt)
	return err
}

func (a *AuthStore) ConsumeMagicLinkToken(ctx context.Context, tokenHash []byte, now time.Time) (string, error) {
	var email string
	err := a.pool.QueryRow(ctx,
		`UPDATE magic_link_tokens SET consumed_at = $2
		 WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > $2
		 RETURNING email`, tokenHash, now).Scan(&email)
	if err == nil {
		return email, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return "", a.classifyToken(ctx, `magic_link_tokens`, `consumed_at`, tokenHash, now, auth.ErrTokenInvalid)
}

// classifyToken turns a failed atomic consume into the right sentinel by
// inspecting the row: absent → notFound; the consumed column set → consumed;
// past expiry → expired.
func (a *AuthStore) classifyToken(ctx context.Context, table, consumedCol string, tokenHash []byte, now time.Time, notFound error) error {
	var consumed, expired bool
	err := a.pool.QueryRow(ctx,
		`SELECT `+consumedCol+` IS NOT NULL, expires_at <= $2 FROM `+table+` WHERE token_hash = $1`,
		tokenHash, now).Scan(&consumed, &expired)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	if err != nil {
		return err
	}
	switch {
	case consumed:
		return auth.ErrTokenConsumed
	case expired:
		return auth.ErrTokenExpired
	default:
		return notFound
	}
}

// --- invites -------------------------------------------------

const inviteCols = `id::text, email, invited_by::text, created_at, expires_at, accepted_at, revoked_at`

func scanInvite(row pgx.Row) (auth.Invite, error) {
	var inv auth.Invite
	if err := row.Scan(&inv.ID, &inv.Email, &inv.InvitedBy, &inv.CreatedAt, &inv.ExpiresAt, &inv.AcceptedAt, &inv.RevokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.Invite{}, auth.ErrNotFound
		}
		return auth.Invite{}, err
	}
	return inv, nil
}

func (a *AuthStore) CreateInvite(ctx context.Context, in auth.Invite, tokenHash []byte) (auth.Invite, error) {
	return scanInvite(a.pool.QueryRow(ctx,
		`INSERT INTO invites (email, invited_by, token_hash, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+inviteCols,
		auth.NormalizeEmail(in.Email), in.InvitedBy, tokenHash, in.CreatedAt, in.ExpiresAt))
}

func (a *AuthStore) ActiveInviteForEmail(ctx context.Context, email string, now time.Time) (auth.Invite, error) {
	return scanInvite(a.pool.QueryRow(ctx,
		`SELECT `+inviteCols+` FROM invites
		 WHERE email = $1 AND accepted_at IS NULL AND revoked_at IS NULL
		   AND deleted_at IS NULL AND expires_at > $2
		 ORDER BY created_at DESC LIMIT 1`,
		auth.NormalizeEmail(email), now))
}

func (a *AuthStore) ConsumeInvite(ctx context.Context, tokenHash []byte, now time.Time) (auth.Invite, error) {
	inv, err := scanInvite(a.pool.QueryRow(ctx,
		`UPDATE invites SET accepted_at = $2
		 WHERE token_hash = $1 AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > $2
		 RETURNING `+inviteCols, tokenHash, now))
	if err == nil {
		return inv, nil
	}
	if !errors.Is(err, auth.ErrNotFound) {
		return auth.Invite{}, err
	}
	return auth.Invite{}, a.classifyToken(ctx, `invites`, `accepted_at`, tokenHash, now, auth.ErrInviteInvalid)
}

func (a *AuthStore) MarkInviteAcceptedBy(ctx context.Context, inviteID, userID string, now time.Time) error {
	tag, err := a.pool.Exec(ctx,
		`UPDATE invites SET accepted_at = $3, accepted_user_id = $2 WHERE id = $1`,
		inviteID, userID, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrNotFound
	}
	return nil
}

func (a *AuthStore) InviteByID(ctx context.Context, id string) (auth.Invite, error) {
	return scanInvite(a.pool.QueryRow(ctx,
		`SELECT `+inviteCols+` FROM invites WHERE id = $1 AND deleted_at IS NULL`, id))
}

// RevokeInvite is idempotent: COALESCE leaves an already-set revoked_at
// untouched on a repeat call.
func (a *AuthStore) RevokeInvite(ctx context.Context, id string, now time.Time) (auth.Invite, error) {
	return scanInvite(a.pool.QueryRow(ctx,
		`UPDATE invites SET revoked_at = COALESCE(revoked_at, $2)
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING `+inviteCols, id, now))
}

func (a *AuthStore) SoftDeleteInvite(ctx context.Context, id string, now time.Time) error {
	tag, err := a.pool.Exec(ctx,
		`UPDATE invites SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return auth.ErrNotFound
	}
	return nil
}

// --- oidc login state ------------------------------------

func (a *AuthStore) CreateOIDCState(ctx context.Context, st auth.OIDCState) error {
	_, err := a.pool.Exec(ctx,
		`INSERT INTO oidc_login_state (state, nonce, pkce_verifier, provider, return_to, expires_at)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)`,
		st.State, st.Nonce, st.PKCEVerifier, st.Provider, st.ReturnTo, st.ExpiresAt)
	return err
}

func (a *AuthStore) ConsumeOIDCState(ctx context.Context, state string, now time.Time) (auth.OIDCState, error) {
	st := auth.OIDCState{State: state}
	err := a.pool.QueryRow(ctx,
		`DELETE FROM oidc_login_state WHERE state = $1
		 RETURNING nonce, pkce_verifier, provider, COALESCE(return_to, ''), expires_at`,
		state).Scan(&st.Nonce, &st.PKCEVerifier, &st.Provider, &st.ReturnTo, &st.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.OIDCState{}, auth.ErrTokenInvalid
	}
	if err != nil {
		return auth.OIDCState{}, err
	}
	if now.After(st.ExpiresAt) {
		return auth.OIDCState{}, auth.ErrTokenExpired
	}
	return st, nil
}

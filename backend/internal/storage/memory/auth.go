package memory

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"at.draab/familyfinances/internal/auth"
)

// AuthStore is the in-memory implementation of auth.Store: the default for
// domain and handler tests and for local runs without a database. It is
// safe for concurrent use.
type AuthStore struct {
	mu sync.Mutex

	users      map[string]auth.User
	identities map[string]auth.Identity
	sessions   map[string]sessionRow
	magic      map[string]magicRow
	invites    map[string]inviteRow
	oidc       map[string]auth.OIDCState

	seq int
}

type sessionRow struct {
	s    auth.Session
	hash []byte
}

type magicRow struct {
	email      string
	expiresAt  time.Time
	consumedAt *time.Time
}

type inviteRow struct {
	inv  auth.Invite
	hash []byte
}

// NewAuthStore returns an empty AuthStore.
func NewAuthStore() *AuthStore {
	return &AuthStore{
		users:      map[string]auth.User{},
		identities: map[string]auth.Identity{},
		sessions:   map[string]sessionRow{},
		magic:      map[string]magicRow{},
		invites:    map[string]inviteRow{},
		oidc:       map[string]auth.OIDCState{},
	}
}

func (a *AuthStore) nextID(prefix string) string {
	a.seq++
	return prefix + strconv.Itoa(a.seq)
}

// --- users ---------------------------------------------------------------

func (a *AuthStore) UserCount(context.Context) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.users), nil
}

func (a *AuthStore) UserByID(_ context.Context, id string) (auth.User, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if u, ok := a.users[id]; ok {
		return u, nil
	}
	return auth.User{}, auth.ErrNotFound
}

func (a *AuthStore) UserByEmail(_ context.Context, email string) (auth.User, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	email = auth.NormalizeEmail(email)
	for _, u := range a.users {
		if auth.NormalizeEmail(u.Email) == email {
			return u, nil
		}
	}
	return auth.User{}, auth.ErrNotFound
}

func (a *AuthStore) SetUserAdmin(_ context.Context, email string, isAdmin bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	email = auth.NormalizeEmail(email)
	for id, u := range a.users {
		if auth.NormalizeEmail(u.Email) == email {
			u.IsAdmin = isAdmin
			a.users[id] = u
			return nil
		}
	}
	return auth.ErrNotFound
}

func (a *AuthStore) ListAdminEmails(context.Context) ([]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []string
	for _, u := range a.users {
		if u.IsAdmin {
			out = append(out, u.Email)
		}
	}
	sort.Strings(out)
	return out, nil
}

// --- admin: users --------------------------------------------------------

func (a *AuthStore) ListUsers(context.Context) ([]auth.User, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []auth.User
	for _, u := range a.users {
		if u.DeletedAt == nil {
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (a *AuthStore) SetUserDisabled(_ context.Context, id string, disabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	u, ok := a.users[id]
	if !ok {
		return auth.ErrNotFound
	}
	u.Disabled = disabled
	a.users[id] = u
	return nil
}

func (a *AuthStore) SoftDeleteUser(_ context.Context, id string, now time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	u, ok := a.users[id]
	if !ok {
		return auth.ErrNotFound
	}
	t := now
	u.DeletedAt = &t
	a.users[id] = u
	return nil
}

func (a *AuthStore) DeleteSessionsByUserID(_ context.Context, userID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, row := range a.sessions {
		if row.s.UserID == userID {
			delete(a.sessions, id)
		}
	}
	return nil
}

// --- admin: invites -------------------------------------------------------

func (a *AuthStore) ListInvites(context.Context) ([]auth.InviteInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []auth.InviteInfo
	for _, row := range a.invites {
		inviter := a.users[row.inv.InvitedBy]
		out = append(out, auth.InviteInfo{
			ID:    row.inv.ID,
			Email: row.inv.Email,
			InvitedBy: auth.InviteInviter{
				ID:          inviter.ID,
				Email:       inviter.Email,
				DisplayName: inviter.DisplayName,
			},
			CreatedAt:  row.inv.CreatedAt,
			ExpiresAt:  row.inv.ExpiresAt,
			AcceptedAt: row.inv.AcceptedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// --- identities --------------------------------------------------------

func (a *AuthStore) IdentityByEmail(_ context.Context, email string) (auth.Identity, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	email = auth.NormalizeEmail(email)
	for _, id := range a.identities {
		if id.Kind == auth.IdentityEmail && auth.NormalizeEmail(id.Email) == email {
			return id, nil
		}
	}
	return auth.Identity{}, auth.ErrNotFound
}

func (a *AuthStore) IdentityByProviderSubject(_ context.Context, provider, subject string) (auth.Identity, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, id := range a.identities {
		if id.Kind == auth.IdentityOIDC && id.Provider == provider && id.Subject == subject {
			return id, nil
		}
	}
	return auth.Identity{}, auth.ErrNotFound
}

// identityConflictLocked reports whether an identity with in's unique key
// already exists. Caller holds a.mu.
func (a *AuthStore) identityConflictLocked(in auth.Identity) bool {
	for _, id := range a.identities {
		switch in.Kind {
		case auth.IdentityEmail:
			if id.Kind == auth.IdentityEmail && auth.NormalizeEmail(id.Email) == auth.NormalizeEmail(in.Email) {
				return true
			}
		case auth.IdentityOIDC:
			if id.Kind == auth.IdentityOIDC && id.Provider == in.Provider && id.Subject == in.Subject {
				return true
			}
		}
	}
	return false
}

func (a *AuthStore) AddIdentity(_ context.Context, in auth.Identity) (auth.Identity, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.users[in.UserID]; !ok {
		return auth.Identity{}, auth.ErrNotFound
	}
	if a.identityConflictLocked(in) {
		return auth.Identity{}, auth.ErrIdentityConflict
	}
	in.ID = a.nextID("id")
	in.CreatedAt = time.Now()
	if in.Kind == auth.IdentityEmail {
		in.Email = auth.NormalizeEmail(in.Email)
	}
	a.identities[in.ID] = in
	return in, nil
}

func (a *AuthStore) CreateUserWithIdentity(_ context.Context, u auth.NewUser, id auth.Identity) (auth.User, auth.Identity, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.identityConflictLocked(id) {
		return auth.User{}, auth.Identity{}, auth.ErrIdentityConflict
	}

	user := auth.User{
		ID:          a.nextID("u"),
		Email:       auth.NormalizeEmail(u.Email),
		DisplayName: u.DisplayName,
		IsAdmin:     len(a.users) == 0, // zero-users bootstrap
		CreatedAt:   time.Now(),
	}
	for _, existing := range a.users {
		if auth.NormalizeEmail(existing.Email) == user.Email {
			return auth.User{}, auth.Identity{}, auth.ErrIdentityConflict
		}
	}
	a.users[user.ID] = user

	id.ID = a.nextID("id")
	id.UserID = user.ID
	id.CreatedAt = time.Now()
	if id.Kind == auth.IdentityEmail {
		id.Email = auth.NormalizeEmail(id.Email)
	}
	a.identities[id.ID] = id

	return user, id, nil
}

// --- sessions --------------------------------------------------------

func (a *AuthStore) CreateSession(_ context.Context, s auth.Session, tokenHash []byte) (auth.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s.ID = a.nextID("s")
	hash := append([]byte(nil), tokenHash...)
	a.sessions[s.ID] = sessionRow{s: s, hash: hash}
	return s, nil
}

func (a *AuthStore) SessionByTokenHash(_ context.Context, tokenHash []byte) (auth.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, row := range a.sessions {
		if auth.ConstantTimeEqualHash(row.hash, tokenHash) {
			return row.s, nil
		}
	}
	return auth.Session{}, auth.ErrNotFound
}

func (a *AuthStore) TouchSession(_ context.Context, id string, lastSeen, expires time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	row, ok := a.sessions[id]
	if !ok {
		return auth.ErrNotFound
	}
	row.s.LastSeenAt = lastSeen
	row.s.ExpiresAt = expires
	a.sessions[id] = row
	return nil
}

func (a *AuthStore) DeleteSessionByTokenHash(_ context.Context, tokenHash []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, row := range a.sessions {
		if auth.ConstantTimeEqualHash(row.hash, tokenHash) {
			delete(a.sessions, id)
			return nil
		}
	}
	return auth.ErrNotFound
}

// --- magic-link tokens ---------------------------------------------

func (a *AuthStore) CreateMagicLinkToken(_ context.Context, tokenHash []byte, email string, expiresAt time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.magic[string(tokenHash)] = magicRow{email: auth.NormalizeEmail(email), expiresAt: expiresAt}
	return nil
}

func (a *AuthStore) ConsumeMagicLinkToken(_ context.Context, tokenHash []byte, now time.Time) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	key := string(tokenHash)
	row, ok := a.magic[key]
	if !ok {
		return "", auth.ErrTokenInvalid
	}
	if row.consumedAt != nil {
		return "", auth.ErrTokenConsumed
	}
	if now.After(row.expiresAt) {
		return "", auth.ErrTokenExpired
	}
	t := now
	row.consumedAt = &t
	a.magic[key] = row
	return row.email, nil
}

// --- invites -----------------------------------------------------

func (a *AuthStore) CreateInvite(_ context.Context, in auth.Invite, tokenHash []byte) (auth.Invite, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	in.ID = a.nextID("inv")
	in.Email = auth.NormalizeEmail(in.Email)
	a.invites[in.ID] = inviteRow{inv: in, hash: append([]byte(nil), tokenHash...)}
	return in, nil
}

func (a *AuthStore) ActiveInviteForEmail(_ context.Context, email string, now time.Time) (auth.Invite, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	email = auth.NormalizeEmail(email)
	for _, row := range a.invites {
		if auth.NormalizeEmail(row.inv.Email) == email && row.inv.AcceptedAt == nil && now.Before(row.inv.ExpiresAt) {
			return row.inv, nil
		}
	}
	return auth.Invite{}, auth.ErrNotFound
}

func (a *AuthStore) ConsumeInvite(_ context.Context, tokenHash []byte, now time.Time) (auth.Invite, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, row := range a.invites {
		if !auth.ConstantTimeEqualHash(row.hash, tokenHash) {
			continue
		}
		if row.inv.AcceptedAt != nil {
			return auth.Invite{}, auth.ErrTokenConsumed
		}
		if now.After(row.inv.ExpiresAt) {
			return auth.Invite{}, auth.ErrTokenExpired
		}
		t := now
		row.inv.AcceptedAt = &t
		a.invites[id] = row
		return row.inv, nil
	}
	return auth.Invite{}, auth.ErrInviteInvalid
}

func (a *AuthStore) MarkInviteAcceptedBy(_ context.Context, inviteID, userID string, now time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	row, ok := a.invites[inviteID]
	if !ok {
		return auth.ErrNotFound
	}
	t := now
	row.inv.AcceptedAt = &t
	row.inv.AcceptedUserID = userID
	a.invites[inviteID] = row
	return nil
}

// --- oidc login state ------------------------------------------

func (a *AuthStore) CreateOIDCState(_ context.Context, st auth.OIDCState) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.oidc[st.State] = st
	return nil
}

func (a *AuthStore) ConsumeOIDCState(_ context.Context, state string, now time.Time) (auth.OIDCState, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, ok := a.oidc[state]
	if !ok {
		return auth.OIDCState{}, auth.ErrTokenInvalid
	}
	delete(a.oidc, state)
	if now.After(st.ExpiresAt) {
		return auth.OIDCState{}, auth.ErrTokenExpired
	}
	return st, nil
}

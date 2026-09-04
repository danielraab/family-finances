package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Mailer sends the two transactional emails auth produces. internal/mailer
// implements it over SMTP.
type Mailer interface {
	// SendMagicLink emails a sign-in link to addr.
	SendMagicLink(ctx context.Context, addr, link string) error
	// SendInvite emails an acceptance link to addr, naming the inviter.
	SendInvite(ctx context.Context, addr, link, invitedByEmail string) error
}

// OIDCClient wraps the configured provider. internal/oidcauth implements it
// with github.com/coreos/go-oidc/v3 + golang.org/x/oauth2.
type OIDCClient interface {
	// AuthCodeURL builds the provider authorization URL carrying state, the
	// PKCE S256 challenge derived from verifier, and nonce.
	AuthCodeURL(state, nonce, verifier string) string
	// Exchange trades an authorization code for the raw id_token, using the
	// PKCE verifier stored at StartOIDC time.
	Exchange(ctx context.Context, code, verifier string) (rawIDToken string, err error)
	// VerifyIDToken checks the id_token's signature (provider JWKS), iss, aud,
	// exp and the nonce, and returns the identity claims.
	VerifyIDToken(ctx context.Context, rawIDToken, nonce string) (OIDCClaims, error)
}

// OIDCClaims are the fields auth reads from a verified id_token.
type OIDCClaims struct {
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
}

// LanguageLookup resolves an authenticated user's raw, unresolved language
// preference for GET /api/auth/me — nil when unset. internal/settings'
// Service satisfies this structurally; wiring it via WithLanguageLookup is
// optional (a nil lookup means the me response's language is always nil).
// auth declares this interface itself rather than importing internal/settings
// — see that package's design note on the client's i18n precedence.
type LanguageLookup interface {
	Language(ctx context.Context, userID string) (*string, error)
}

// Params is the slice of configuration the service needs, mapped from
// config.Config by package main.
type Params struct {
	BaseURL             string
	SessionTTL          time.Duration
	SessionMaxTTL       time.Duration
	SignupEnabled       bool
	AllowedEmailDomains []string
	InviteEnabled       bool
	InviteTTL           time.Duration
	MagicLinkTTL        time.Duration
	OIDCIssuer          string
	OIDCLabel           string
}

// Service is the auth use-case layer. It depends only on the Store interface
// and the two side-effect interfaces above.
type Service struct {
	store  Store
	mailer Mailer
	oidc   OIDCClient     // nil when no provider is configured
	lang   LanguageLookup // nil when not wired
	p      Params
	now    func() time.Time
}

// Option customizes a Service (used by tests).
type Option func(*Service)

// WithClock overrides the time source, for deterministic expiry tests.
func WithClock(fn func() time.Time) Option { return func(s *Service) { s.now = fn } }

// WithLanguageLookup wires the raw-language-preference source for
// GET /api/auth/me. package main passes internal/settings' Service.
func WithLanguageLookup(l LanguageLookup) Option { return func(s *Service) { s.lang = l } }

// NewService builds the auth service. mailer must be non-nil; oidc may be nil,
// in which case the OIDC routes return ErrOIDCNotConfigured.
func NewService(store Store, mailer Mailer, oidc OIDCClient, p Params, opts ...Option) *Service {
	s := &Service{store: store, mailer: mailer, oidc: oidc, p: p, now: time.Now}
	for _, o := range opts {
		o(s)
	}
	return s
}

const oidcStateTTL = 10 * time.Minute

// --- magic link -------------------------------------------------------------

// StartEmailLogin is the POST /api/auth/email/start use case. It never reports
// whether an email was sent: a malformed address or a disallowed one is a
// silent no-op, so the endpoint cannot be used to enumerate accounts. A
// non-nil error means an internal failure (store or mailer), which the handler
// still turns into 200.
func (s *Service) StartEmailLogin(ctx context.Context, rawEmail string) error {
	email := NormalizeEmail(rawEmail)
	if ValidateEmail(email) != nil {
		return nil
	}

	permitted, err := s.emailPermitted(ctx, email)
	if err != nil {
		return err
	}
	if !permitted {
		return nil
	}

	token, hash, err := newToken()
	if err != nil {
		return err
	}
	if err := s.store.CreateMagicLinkToken(ctx, hash, email, s.now().Add(s.p.MagicLinkTTL)); err != nil {
		return err
	}

	link := s.buildURL("/api/auth/email/callback", url.Values{"token": {token}})
	return s.mailer.SendMagicLink(ctx, email, link)
}

// emailPermitted reports whether a magic link may be sent to email: it belongs
// to an existing, non-disabled, non-deleted user, OR signup is enabled and
// the domain passes, OR an unexpired invite exists. An address that belongs
// to a disabled or soft-deleted user is treated as not permitted outright —
// it already has an account, just a blocked one, so the signup/invite checks
// below don't apply either.
func (s *Service) emailPermitted(ctx context.Context, email string) (bool, error) {
	if u, err := s.store.UserByEmail(ctx, email); err == nil {
		return !u.Disabled && u.DeletedAt == nil, nil
	} else if !errors.Is(err, ErrNotFound) {
		return false, err
	}

	if s.p.SignupEnabled && DomainAllowed(email, s.p.AllowedEmailDomains) {
		return true, nil
	}

	if _, err := s.store.ActiveInviteForEmail(ctx, email, s.now()); err == nil {
		return true, nil
	} else if !errors.Is(err, ErrNotFound) {
		return false, err
	}
	return false, nil
}

// CompleteEmailLogin consumes a magic-link token and establishes a session.
// currentUserID is the caller's already-authenticated user ("" if anonymous),
// which links the new email identity to that account.
func (s *Service) CompleteEmailLogin(ctx context.Context, token, currentUserID string, sc SessionContext) (User, string, error) {
	hash := hashToken(token)
	email, err := s.store.ConsumeMagicLinkToken(ctx, hash, s.now())
	if err != nil {
		return User{}, "", err
	}

	user, err := s.resolveIdentity(ctx, identityInput{
		kind:          IdentityEmail,
		email:         email,
		emailVerified: true,
		currentUserID: currentUserID,
		allowCreate:   true,
	})
	if err != nil {
		return User{}, "", err
	}
	if err := checkAccountUsable(user); err != nil {
		return User{}, "", err
	}
	return s.issueSession(ctx, user, sc)
}

// --- OIDC -----------------------------------------------------------------

// OIDCLogin reports whether OIDC sign-in is available (an OIDC client was
// constructed at startup) and, if so, its configured button label. The web
// client reads this via GET /api/auth/config to decide whether to show the
// provider button on the login page.
func (s *Service) OIDCLogin() (label string, ok bool) {
	if s.oidc == nil {
		return "", false
	}
	return s.p.OIDCLabel, true
}

// StartOIDC creates per-attempt state and returns the provider authorization
// URL to redirect the browser to. returnTo is a post-login in-app path,
// validated later by the handler.
func (s *Service) StartOIDC(ctx context.Context, returnTo string) (string, error) {
	if s.oidc == nil {
		return "", ErrOIDCNotConfigured
	}
	state, err := randString()
	if err != nil {
		return "", err
	}
	nonce, err := randString()
	if err != nil {
		return "", err
	}
	verifier, err := randString()
	if err != nil {
		return "", err
	}

	st := OIDCState{
		State:        state,
		Nonce:        nonce,
		PKCEVerifier: verifier,
		Provider:     s.p.OIDCIssuer,
		ReturnTo:     returnTo,
		ExpiresAt:    s.now().Add(oidcStateTTL),
	}
	if err := s.store.CreateOIDCState(ctx, st); err != nil {
		return "", err
	}
	return s.oidc.AuthCodeURL(state, nonce, verifier), nil
}

// CompleteOIDC finishes the authorization-code flow: it validates state,
// exchanges the code, verifies the id_token, and resolves or creates an
// identity per the linking rules. It returns the session plus the stored
// return-to path.
func (s *Service) CompleteOIDC(ctx context.Context, state, code, currentUserID string, sc SessionContext) (User, string, string, error) {
	if s.oidc == nil {
		return User{}, "", "", ErrOIDCNotConfigured
	}
	st, err := s.store.ConsumeOIDCState(ctx, state, s.now())
	if err != nil {
		return User{}, "", "", err
	}

	raw, err := s.oidc.Exchange(ctx, code, st.PKCEVerifier)
	if err != nil {
		return User{}, "", "", fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}
	claims, err := s.oidc.VerifyIDToken(ctx, raw, st.Nonce)
	if err != nil {
		return User{}, "", "", fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}
	if claims.Subject == "" {
		return User{}, "", "", ErrTokenInvalid
	}

	user, err := s.resolveIdentity(ctx, identityInput{
		kind:          IdentityOIDC,
		email:         NormalizeEmail(claims.Email),
		emailVerified: claims.EmailVerified,
		provider:      claims.Issuer,
		subject:       claims.Subject,
		currentUserID: currentUserID,
		allowCreate:   true,
	})
	if err != nil {
		return User{}, "", "", err
	}
	if err := checkAccountUsable(user); err != nil {
		return User{}, "", "", err
	}
	tok, err := s.issueSessionToken(ctx, user, sc)
	if err != nil {
		return User{}, "", "", err
	}
	return user, tok, st.ReturnTo, nil
}

// --- invites --------------------------------------------------------------

// InviteEnabled reports whether an authenticated user may currently create
// invites: always while signup is enabled, otherwise per AUTH_INVITE_ENABLED.
func (s *Service) InviteEnabled() bool {
	return s.p.SignupEnabled || s.p.InviteEnabled
}

// CreateInvite records an invite from inviterID for addr and emails an
// acceptance link.
func (s *Service) CreateInvite(ctx context.Context, inviterID, addr string) (Invite, error) {
	if !s.InviteEnabled() {
		return Invite{}, ErrSignupDisabled
	}
	email := NormalizeEmail(addr)
	if err := ValidateEmail(email); err != nil {
		return Invite{}, err
	}

	inviter, err := s.store.UserByID(ctx, inviterID)
	if err != nil {
		return Invite{}, err
	}

	token, hash, err := newToken()
	if err != nil {
		return Invite{}, err
	}
	now := s.now()
	inv, err := s.store.CreateInvite(ctx, Invite{
		Email:     email,
		InvitedBy: inviterID,
		CreatedAt: now,
		ExpiresAt: now.Add(s.p.InviteTTL),
	}, hash)
	if err != nil {
		return Invite{}, err
	}

	link := s.buildURL("/api/auth/invites/accept", url.Values{"token": {token}})
	if err := s.mailer.SendInvite(ctx, email, link, inviter.Email); err != nil {
		return Invite{}, err
	}
	return inv, nil
}

// AcceptInvite consumes an invite token and establishes a session. Account
// creation here bypasses the signup toggle and the domain allow-list.
func (s *Service) AcceptInvite(ctx context.Context, token, currentUserID string, sc SessionContext) (User, string, error) {
	hash := hashToken(token)
	inv, err := s.store.ConsumeInvite(ctx, hash, s.now())
	if err != nil {
		return User{}, "", err
	}

	user, err := s.resolveIdentity(ctx, identityInput{
		kind:          IdentityEmail,
		email:         inv.Email,
		emailVerified: true,
		currentUserID: currentUserID,
		allowCreate:   true,
		bypassPolicy:  true,
	})
	if err != nil {
		return User{}, "", err
	}
	if err := checkAccountUsable(user); err != nil {
		return User{}, "", err
	}
	if err := s.store.MarkInviteAcceptedBy(ctx, inv.ID, user.ID, s.now()); err != nil {
		return User{}, "", err
	}
	return s.issueSession(ctx, user, sc)
}

// checkAccountUsable rejects a disabled or soft-deleted account from
// completing any sign-in flow.
func checkAccountUsable(u User) error {
	if u.Disabled || u.DeletedAt != nil {
		return ErrAccountDisabled
	}
	return nil
}

// --- sessions -----------------------------------------------------------

// Authenticate resolves a presented session token to its user, applying the
// sliding-expiry bump and the absolute cap. It returns ErrNotFound for any
// missing, expired, or over-age session.
func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrNotFound
	}
	hash := hashToken(token)
	sess, err := s.store.SessionByTokenHash(ctx, hash)
	if err != nil {
		return User{}, err
	}

	now := s.now()
	if now.After(sess.ExpiresAt) || now.Sub(sess.CreatedAt) >= s.p.SessionMaxTTL {
		_ = s.store.DeleteSessionByTokenHash(ctx, hash)
		return User{}, ErrNotFound
	}

	// Sliding expiry: once past half the sliding window since last activity,
	// push the window forward, capped at the absolute maximum age.
	if now.Sub(sess.LastSeenAt) >= s.p.SessionTTL/2 {
		newExpiry := now.Add(s.p.SessionTTL)
		if cap := sess.CreatedAt.Add(s.p.SessionMaxTTL); newExpiry.After(cap) {
			newExpiry = cap
		}
		_ = s.store.TouchSession(ctx, sess.ID, now, newExpiry)
	}

	user, err := s.store.UserByID(ctx, sess.UserID)
	if err != nil {
		return User{}, err
	}
	// Belt-and-suspenders: disable/delete already revoke sessions
	// immediately, but a session row can outlive that in a race — reject it
	// here too rather than trusting the row alone.
	if user.Disabled || user.DeletedAt != nil {
		return User{}, ErrNotFound
	}
	return user, nil
}

// Logout revokes the session identified by token. It is idempotent: an unknown
// token is not an error.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	err := s.store.DeleteSessionByTokenHash(ctx, hashToken(token))
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// --- admin ------------------------------------------------------------------

// SetAdmin sets or clears the admin flag for the user with this email.
func (s *Service) SetAdmin(ctx context.Context, email string, isAdmin bool) error {
	return s.store.SetUserAdmin(ctx, NormalizeEmail(email), isAdmin)
}

// ListAdmins returns the sorted emails of all admin users.
func (s *Service) ListAdmins(ctx context.Context) ([]string, error) {
	return s.store.ListAdminEmails(ctx)
}

// UserLanguage returns userID's raw language preference via the wired
// LanguageLookup, or nil if none is wired or the lookup fails — GET
// /api/auth/me degrades to "no preference" rather than failing the request.
func (s *Service) UserLanguage(ctx context.Context, userID string) *string {
	if s.lang == nil {
		return nil
	}
	v, err := s.lang.Language(ctx, userID)
	if err != nil {
		return nil
	}
	return v
}

// --- admin: users and invitations -----------------------------------------

// ListUsers returns every non-soft-deleted user.
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.store.ListUsers(ctx)
}

// DisableUser sets disabled and immediately revokes every session belonging
// to the user. No check prevents an admin from disabling themselves, even as
// the last remaining admin — see design.md.
func (s *Service) DisableUser(ctx context.Context, id string) (User, error) {
	if err := s.store.SetUserDisabled(ctx, id, true); err != nil {
		return User{}, err
	}
	if err := s.store.DeleteSessionsByUserID(ctx, id); err != nil {
		return User{}, err
	}
	return s.store.UserByID(ctx, id)
}

// EnableUser clears disabled. It does not restore any session revoked while
// disabled — the user signs in again.
func (s *Service) EnableUser(ctx context.Context, id string) (User, error) {
	if err := s.store.SetUserDisabled(ctx, id, false); err != nil {
		return User{}, err
	}
	return s.store.UserByID(ctx, id)
}

// SoftDeleteUser sets deleted_at and immediately revokes every session
// belonging to the user. There is no undelete in this change. No check
// prevents an admin from deleting themselves, even as the last remaining
// admin — see design.md.
func (s *Service) SoftDeleteUser(ctx context.Context, id string) error {
	if err := s.store.SoftDeleteUser(ctx, id, s.now()); err != nil {
		return err
	}
	return s.store.DeleteSessionsByUserID(ctx, id)
}

// ListInvites returns every invitation regardless of status, each carrying
// the inviter's identity.
func (s *Service) ListInvites(ctx context.Context) ([]InviteInfo, error) {
	return s.store.ListInvites(ctx)
}

// ListMyInvites returns every invitation userID personally created,
// regardless of status.
func (s *Service) ListMyInvites(ctx context.Context, userID string) ([]InviteInfo, error) {
	return s.store.ListInvitesByInviter(ctx, userID)
}

// RevokeInvite marks an invitation revoked, permitted for its own inviter or
// an admin (ErrInviteRevokeForbidden otherwise). It is idempotent: revoking
// an already-revoked invite succeeds without changing its revoked_at, and it
// is permitted regardless of the invite's status (pending, accepted, or
// expired) — a revoked invite stays visible in every listing, only its
// acceptance is blocked (see Store.ConsumeInvite).
func (s *Service) RevokeInvite(ctx context.Context, actor User, id string) (InviteInfo, error) {
	inv, err := s.store.InviteByID(ctx, id)
	if err != nil {
		return InviteInfo{}, err
	}
	if inv.InvitedBy != actor.ID && !actor.IsAdmin {
		return InviteInfo{}, ErrInviteRevokeForbidden
	}
	revoked, err := s.store.RevokeInvite(ctx, id, s.now())
	if err != nil {
		return InviteInfo{}, err
	}
	return s.inviteInfo(ctx, revoked)
}

// SoftDeleteInvite hides an already-revoked invitation from every listing.
// Admin-only authorization is enforced by the caller (the handler's
// requireAdmin guard), matching SoftDeleteUser. Returns ErrInviteNotRevoked
// for an invitation whose revoked_at is not yet set.
func (s *Service) SoftDeleteInvite(ctx context.Context, id string) error {
	inv, err := s.store.InviteByID(ctx, id)
	if err != nil {
		return err
	}
	if inv.RevokedAt == nil {
		return ErrInviteNotRevoked
	}
	return s.store.SoftDeleteInvite(ctx, id, s.now())
}

// inviteInfo resolves inv's inviter identity to build the InviteInfo shape
// returned over HTTP.
func (s *Service) inviteInfo(ctx context.Context, inv Invite) (InviteInfo, error) {
	inviter, err := s.store.UserByID(ctx, inv.InvitedBy)
	if err != nil {
		return InviteInfo{}, err
	}
	return InviteInfo{
		ID:    inv.ID,
		Email: inv.Email,
		InvitedBy: InviteInviter{
			ID:          inviter.ID,
			Email:       inviter.Email,
			DisplayName: inviter.DisplayName,
		},
		CreatedAt:  inv.CreatedAt,
		ExpiresAt:  inv.ExpiresAt,
		AcceptedAt: inv.AcceptedAt,
		RevokedAt:  inv.RevokedAt,
	}, nil
}

// LinkIdentity attaches an identity to an existing, already-authenticated user
// without going through a full sign-in flow. It is the explicit-link path.
func (s *Service) LinkIdentity(ctx context.Context, userID string, id Identity) (Identity, error) {
	if _, err := s.store.UserByID(ctx, userID); err != nil {
		return Identity{}, err
	}
	id.UserID = userID
	if id.Kind == IdentityEmail {
		id.Email = NormalizeEmail(id.Email)
	}
	return s.store.AddIdentity(ctx, id)
}

// --- identity resolution --------------------------------------------------

type identityInput struct {
	kind          IdentityKind
	email         string
	emailVerified bool
	provider      string
	subject       string
	currentUserID string
	allowCreate   bool
	bypassPolicy  bool // invite acceptance: skip signup toggle + domain list
}

// resolveIdentity applies ResolveLink against the store and returns the user
// the caller should be signed in as, creating the account (subject to policy)
// when needed.
func (s *Service) resolveIdentity(ctx context.Context, in identityInput) (User, error) {
	existing, err := s.lookupIdentity(ctx, in)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return User{}, err
	}

	var emailMatchUserID string
	if in.email != "" {
		if u, e := s.store.UserByEmail(ctx, in.email); e == nil {
			emailMatchUserID = u.ID
		} else if !errors.Is(e, ErrNotFound) {
			return User{}, e
		}
	}

	action, userID := ResolveLink(LinkInput{
		Kind:                   in.kind,
		EmailVerified:          in.emailVerified,
		ExistingIdentityUserID: existing.UserID,
		EmailMatchUserID:       emailMatchUserID,
		CurrentUserID:          in.currentUserID,
	})

	switch action {
	case ActionSignIn:
		return s.store.UserByID(ctx, userID)

	case ActionAttach:
		_, err := s.store.AddIdentity(ctx, Identity{
			UserID:        userID,
			Kind:          in.kind,
			Email:         in.email,
			EmailVerified: in.emailVerified,
			Provider:      in.provider,
			Subject:       in.subject,
		})
		if err != nil && !errors.Is(err, ErrIdentityConflict) {
			return User{}, err
		}
		if errors.Is(err, ErrIdentityConflict) {
			// Raced with a concurrent link of the same identity; fall back to
			// signing in as whoever owns it now.
			owner, lookErr := s.lookupIdentity(ctx, in)
			if lookErr != nil {
				return User{}, err
			}
			return s.store.UserByID(ctx, owner.UserID)
		}
		return s.store.UserByID(ctx, userID)

	default: // ActionCreate
		if !in.allowCreate {
			return User{}, ErrNotFound
		}
		if emailMatchUserID != "" {
			// ResolveLink chose Create only because the incoming identity
			// wasn't verified enough to attach (an unverified OIDC email
			// matching an existing account) — creating would collide on
			// that account's email, which is unique per user. Surface a
			// clear, actionable error instead of a doomed insert.
			return User{}, ErrEmailInUse
		}
		if err := s.checkCreatePolicy(ctx, in); err != nil {
			return User{}, err
		}
		if in.email == "" {
			return User{}, ErrEmailRequired
		}
		user, _, err := s.store.CreateUserWithIdentity(ctx, NewUser{Email: in.email}, Identity{
			Kind:          in.kind,
			Email:         in.email,
			EmailVerified: in.emailVerified,
			Provider:      in.provider,
			Subject:       in.subject,
		})
		if errors.Is(err, ErrIdentityConflict) {
			owner, lookErr := s.lookupIdentity(ctx, in)
			if lookErr != nil {
				return User{}, err
			}
			return s.store.UserByID(ctx, owner.UserID)
		}
		if err != nil {
			return User{}, err
		}
		return user, nil
	}
}

func (s *Service) lookupIdentity(ctx context.Context, in identityInput) (Identity, error) {
	if in.kind == IdentityOIDC {
		return s.store.IdentityByProviderSubject(ctx, in.provider, in.subject)
	}
	return s.store.IdentityByEmail(ctx, in.email)
}

// checkCreatePolicy implements the design's registration resolution order
// (§D7). The bootstrap (users-empty ⇒ admin) is enforced atomically by the
// store; here we only gate on invite, signup toggle, and the domain list.
func (s *Service) checkCreatePolicy(ctx context.Context, in identityInput) error {
	if in.bypassPolicy {
		return nil
	}

	n, err := s.store.UserCount(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return nil // bootstrap: first account always allowed
	}

	if in.email != "" {
		if _, err := s.store.ActiveInviteForEmail(ctx, in.email, s.now()); err == nil {
			return nil // invite bypasses the signup toggle and the domain list
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
	}

	if !s.p.SignupEnabled {
		return ErrSignupDisabled
	}
	if !DomainAllowed(in.email, s.p.AllowedEmailDomains) {
		return ErrDomainNotAllowed
	}
	return nil
}

// --- session issuing ----------------------------------------------------

func (s *Service) issueSession(ctx context.Context, user User, sc SessionContext) (User, string, error) {
	tok, err := s.issueSessionToken(ctx, user, sc)
	if err != nil {
		return User{}, "", err
	}
	return user, tok, nil
}

func (s *Service) issueSessionToken(ctx context.Context, user User, sc SessionContext) (string, error) {
	token, hash, err := newToken()
	if err != nil {
		return "", err
	}
	now := s.now()
	client := sc.Client
	if client != ClientWeb && client != ClientAPI {
		client = ClientAPI
	}
	_, err = s.store.CreateSession(ctx, Session{
		UserID:     user.ID,
		Client:     client,
		UserAgent:  sc.UserAgent,
		IP:         sc.IP,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  now.Add(s.p.SessionTTL),
	}, hash)
	if err != nil {
		return "", err
	}
	return token, nil
}

// --- token helpers ------------------------------------------------------

// newToken returns a fresh 256-bit url-safe token and its SHA-256 hash. Only
// the hash is ever persisted.
func newToken() (token string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// ConstantTimeEqualHash compares two token hashes without leaking timing. The
// store implementations use it where they hold a candidate hash in memory.
func ConstantTimeEqualHash(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

func randString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// buildURL joins the configured base URL with a path and query.
func (s *Service) buildURL(path string, q url.Values) string {
	base := strings.TrimRight(s.p.BaseURL, "/")
	if len(q) == 0 {
		return base + path
	}
	return base + path + "?" + q.Encode()
}

// AsSentinel is exported for tests that want the mapped sentinel of a wrapped
// error.
func AsSentinel(err error) error { return asSentinel(err) }

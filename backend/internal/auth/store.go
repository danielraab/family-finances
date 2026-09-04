package auth

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors. internal/httpapi/respond.go maps these to status codes in
// one place; domain and service code never mentions net/http.
var (
	// ErrNotFound: no such user, session, or record.
	ErrNotFound = errors.New("not found")
	// ErrSignupDisabled: account creation is off and no invite applies.
	ErrSignupDisabled = errors.New("signup is disabled")
	// ErrDomainNotAllowed: the email domain is outside the allow-list.
	ErrDomainNotAllowed = errors.New("email domain is not allowed")
	// ErrTokenInvalid: a magic-link / invite / OIDC token or state does not
	// exist or is malformed.
	ErrTokenInvalid = errors.New("token is invalid")
	// ErrTokenExpired: the token existed but its TTL elapsed.
	ErrTokenExpired = errors.New("token has expired")
	// ErrTokenConsumed: the token was already used.
	ErrTokenConsumed = errors.New("token has already been used")
	// ErrInviteInvalid: no usable invite for this address / token.
	ErrInviteInvalid = errors.New("invite is invalid")
	// ErrIdentityConflict: the identity is already linked to another account.
	ErrIdentityConflict = errors.New("identity already linked to another account")
	// ErrEmailInUse: an OIDC sign-in's address already belongs to another
	// account, but the provider did not assert it as verified, so the
	// automatic flow refuses to attach or create — the address can't be
	// reused for a new account either, since it already belongs to one.
	ErrEmailInUse = errors.New("provider did not verify this email address, and it already belongs to another account")
	// ErrInvalidEmail: the address is syntactically invalid.
	ErrInvalidEmail = errors.New("invalid email address")
	// ErrOIDCNotConfigured: an OIDC route was hit but no provider is set.
	ErrOIDCNotConfigured = errors.New("oidc is not configured")
	// ErrEmailRequired: an OIDC sign-in returned no email, so no account can
	// be created.
	ErrEmailRequired = errors.New("provider returned no email address")
	// ErrAccountDisabled: the resolved account is disabled or soft-deleted;
	// no session is established.
	ErrAccountDisabled = errors.New("account is disabled")
)

// Sentinels is every error above, for the httpapi mapping and for tests.
var Sentinels = []error{
	ErrNotFound, ErrSignupDisabled, ErrDomainNotAllowed, ErrTokenInvalid,
	ErrTokenExpired, ErrTokenConsumed, ErrInviteInvalid, ErrIdentityConflict,
	ErrEmailInUse, ErrInvalidEmail, ErrOIDCNotConfigured, ErrEmailRequired,
	ErrAccountDisabled,
}

// NewUser is the input to account creation.
type NewUser struct {
	Email       string
	DisplayName string
}

// Store is the persistence contract auth declares. internal/storage/memory and
// internal/storage/postgres implement it; package main injects one.
//
// Every "…ByX" lookup returns ErrNotFound when there is no row. Token-consuming
// methods are atomic (a single UPDATE … RETURNING) so a token cannot be used
// twice even under concurrency.
type Store interface {
	// --- users ---

	// UserCount reports how many accounts exist. Used for the bootstrap check
	// outside a transaction (advisory only); CreateUserWithIdentity re-checks
	// atomically.
	UserCount(ctx context.Context) (int, error)
	UserByID(ctx context.Context, id string) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)

	// SetUserAdmin sets is_admin on the user with this email, returning
	// ErrNotFound if there is none.
	SetUserAdmin(ctx context.Context, email string, isAdmin bool) error
	// ListAdminEmails returns the email of every user with is_admin = true,
	// sorted.
	ListAdminEmails(ctx context.Context) ([]string, error)

	// --- identities ---

	IdentityByEmail(ctx context.Context, email string) (Identity, error)
	IdentityByProviderSubject(ctx context.Context, provider, subject string) (Identity, error)

	// AddIdentity attaches a new identity to an existing user. It returns
	// ErrIdentityConflict if the identity's unique key already exists.
	AddIdentity(ctx context.Context, in Identity) (Identity, error)

	// CreateUserWithIdentity creates a user and its first identity in one
	// transaction. Inside that transaction it sets is_admin = true iff the
	// users table was empty (the zero-users bootstrap), regardless of what the
	// caller's advisory UserCount said. Returns ErrIdentityConflict on a
	// unique-key clash.
	CreateUserWithIdentity(ctx context.Context, u NewUser, id Identity) (User, Identity, error)

	// --- sessions ---

	CreateSession(ctx context.Context, s Session, tokenHash []byte) (Session, error)
	// SessionByTokenHash looks up a session by the SHA-256 hash of the
	// presented token (an indexed exact match; the token plaintext is never
	// stored).
	SessionByTokenHash(ctx context.Context, tokenHash []byte) (Session, error)
	TouchSession(ctx context.Context, id string, lastSeen, expires time.Time) error
	DeleteSessionByTokenHash(ctx context.Context, tokenHash []byte) error

	// --- magic-link tokens ---

	CreateMagicLinkToken(ctx context.Context, tokenHash []byte, email string, expiresAt time.Time) error
	// ConsumeMagicLinkToken atomically marks the token consumed and returns
	// its address. ErrTokenInvalid / ErrTokenExpired / ErrTokenConsumed
	// distinguish the failure modes.
	ConsumeMagicLinkToken(ctx context.Context, tokenHash []byte, now time.Time) (email string, err error)

	// --- invites ---

	CreateInvite(ctx context.Context, in Invite, tokenHash []byte) (Invite, error)
	// ActiveInviteForEmail returns an unexpired, unaccepted invite for the
	// address, or ErrNotFound.
	ActiveInviteForEmail(ctx context.Context, email string, now time.Time) (Invite, error)
	// ConsumeInvite atomically marks the invite accepted-pending and returns
	// it; MarkInviteAcceptedBy then records which user accepted.
	ConsumeInvite(ctx context.Context, tokenHash []byte, now time.Time) (Invite, error)
	MarkInviteAcceptedBy(ctx context.Context, inviteID, userID string, now time.Time) error

	// --- oidc login state ---

	CreateOIDCState(ctx context.Context, st OIDCState) error
	// ConsumeOIDCState atomically deletes and returns the row for state,
	// yielding ErrTokenInvalid when unknown and ErrTokenExpired when stale.
	ConsumeOIDCState(ctx context.Context, state string, now time.Time) (OIDCState, error)

	// --- admin: users ---

	// ListUsers returns every non-soft-deleted user, ordered by created_at.
	ListUsers(ctx context.Context) ([]User, error)
	// SetUserDisabled sets disabled on the user with this id, returning
	// ErrNotFound if there is none.
	SetUserDisabled(ctx context.Context, id string, disabled bool) error
	// SoftDeleteUser sets deleted_at on the user with this id, returning
	// ErrNotFound if there is none.
	SoftDeleteUser(ctx context.Context, id string, now time.Time) error
	// DeleteSessionsByUserID immediately revokes every session belonging to
	// the user — used by disable/soft-delete so access ends at once, not on
	// next expiry check.
	DeleteSessionsByUserID(ctx context.Context, userID string) error

	// --- admin: invites ---

	// ListInvites returns every invite regardless of status, newest first,
	// each carrying the inviter's identity.
	ListInvites(ctx context.Context) ([]InviteInfo, error)
}

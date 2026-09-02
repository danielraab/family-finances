// Package auth is the first domain package: it owns user accounts and every
// way to sign in to them — email magic links, a single OIDC provider, opaque
// bearer/cookie sessions, invites, and the zero-users bootstrap admin.
//
// It follows the repo's four-file shape: auth.go (types, normalization, the
// linking-decision helper), store.go (the Store interface it declares and the
// sentinel errors), service.go (use-case logic plus the Mailer/OIDCClient
// side-effect interfaces), handler.go (the http.Handler mounted at
// /api/auth/). It imports no internal/httpapi, no internal/storage, and no
// database driver.
package auth

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

// IdentityKind is how a person proves who they are: an email address they
// control, or a subject asserted by the OIDC provider.
type IdentityKind string

const (
	IdentityEmail IdentityKind = "email"
	IdentityOIDC  IdentityKind = "oidc"
)

// SessionClient records how a session's token is carried.
type SessionClient string

const (
	// ClientWeb sessions are carried by the ff_session cookie.
	ClientWeb SessionClient = "web"
	// ClientAPI sessions are carried by an Authorization: Bearer header.
	ClientAPI SessionClient = "api"
)

// User is one account. A person has exactly one User no matter how many ways
// they can sign in to it.
type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name,omitempty"`
	IsAdmin     bool      `json:"is_admin"`
	CreatedAt   time.Time `json:"created_at"`
}

// Identity is one way to sign in to a User. An email identity carries the
// verified address; an oidc identity carries the provider issuer and subject.
type Identity struct {
	ID            string
	UserID        string
	Kind          IdentityKind
	Email         string
	EmailVerified bool
	Provider      string
	Subject       string
	CreatedAt     time.Time
}

// Session is a signed-in session. The token itself is never stored here — only
// its SHA-256 hash, in the store.
type Session struct {
	ID         string
	UserID     string
	Client     SessionClient
	UserAgent  string
	IP         string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

// Invite is a pending invitation created by an authenticated user.
type Invite struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	InvitedBy      string     `json:"invited_by"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	AcceptedUserID string     `json:"accepted_user_id,omitempty"`
}

// OIDCState is the short-lived per-attempt state for an OIDC authorization-code
// flow: the CSRF state, the replay nonce, and the PKCE verifier.
type OIDCState struct {
	State        string
	Nonce        string
	PKCEVerifier string
	Provider     string
	ReturnTo     string
	ExpiresAt    time.Time
}

// SessionContext carries request metadata recorded on a new session.
type SessionContext struct {
	Client    SessionClient
	UserAgent string
	IP        string
}

// NormalizeEmail trims surrounding whitespace and lower-cases an address so it
// compares and stores canonically. citext columns are case-insensitive too;
// this keeps Go-side comparisons agreeing with the database.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// EmailDomain returns the lower-cased domain part of a normalized address, or
// "" if there is no "@".
func EmailDomain(email string) string {
	at := strings.LastIndexByte(email, '@')
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[at+1:])
}

// ValidateEmail reports whether s is a syntactically valid single email
// address (already normalized by the caller).
func ValidateEmail(s string) error {
	if s == "" {
		return ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || EmailDomain(s) == "" {
		return ErrInvalidEmail
	}
	return nil
}

// DomainAllowed reports whether email's domain passes an allow-list. An empty
// list allows any domain.
func DomainAllowed(email string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	d := EmailDomain(email)
	for _, a := range allowed {
		if strings.EqualFold(strings.TrimSpace(a), d) {
			return true
		}
	}
	return false
}

// LinkAction is the outcome of ResolveLink.
type LinkAction int

const (
	// ActionSignIn: a matching identity already exists; sign in as its owner
	// (UserID). No new identity is created.
	ActionSignIn LinkAction = iota
	// ActionAttach: attach a new identity of the incoming kind to UserID.
	ActionAttach
	// ActionCreate: no existing identity and nothing to link to; create a new
	// user together with the incoming identity.
	ActionCreate
)

// LinkInput is the state ResolveLink decides from.
type LinkInput struct {
	Kind IdentityKind
	// EmailVerified is whether the incoming sign-in proves control of the
	// address. Magic-link and invite flows set this true unconditionally; OIDC
	// passes the provider's email_verified claim.
	EmailVerified bool
	// ExistingIdentityUserID owns an identity that already matches the
	// incoming one ((kind,email) for email, (provider,subject) for oidc), or
	// "" if none.
	ExistingIdentityUserID string
	// EmailMatchUserID owns the account whose email equals the incoming
	// address, or "" if none.
	EmailMatchUserID string
	// CurrentUserID is the already-authenticated user completing the flow, or
	// "" for an anonymous sign-in.
	CurrentUserID string
}

// ResolveLink implements the design's identity-linking table (§D1): a matching
// identity signs in as its owner; an authenticated user attaches the new
// identity to their own account; a magic link (always) or a verified OIDC
// email attaches to the account that already owns that address; otherwise a
// new account is created.
func ResolveLink(in LinkInput) (LinkAction, string) {
	if in.ExistingIdentityUserID != "" {
		return ActionSignIn, in.ExistingIdentityUserID
	}
	if in.CurrentUserID != "" {
		return ActionAttach, in.CurrentUserID
	}
	if in.EmailMatchUserID != "" {
		switch in.Kind {
		case IdentityEmail:
			return ActionAttach, in.EmailMatchUserID
		case IdentityOIDC:
			if in.EmailVerified {
				return ActionAttach, in.EmailMatchUserID
			}
		}
	}
	return ActionCreate, ""
}

// asSentinel unwraps to the closest known sentinel, used by tests and the
// httpapi error mapping.
func asSentinel(err error) error {
	for _, s := range Sentinels {
		if errors.Is(err, s) {
			return s
		}
	}
	return err
}

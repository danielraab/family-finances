package auth

import "context"

// CookieName is the session cookie for browser clients.
const CookieName = "ff_session"

type contextKey int

const (
	userKey contextKey = iota
	sessionKey
)

// WithUser returns a copy of ctx carrying u as the authenticated user. The
// httpapi auth middleware calls this after a successful token resolution.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFromContext returns the authenticated user attached by the middleware,
// and false when the request is anonymous.
func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey).(User)
	return u, ok
}

// WithSession returns a copy of ctx carrying the authenticated request's
// session. The httpapi auth middleware calls this alongside WithUser.
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

// SessionFromContext returns the session the request authenticated with, and
// false when the request is anonymous.
func SessionFromContext(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(sessionKey).(Session)
	return s, ok
}

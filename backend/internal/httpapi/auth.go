package httpapi

import (
	"context"
	"net/http"
	"strings"

	"at.draab/familyfinances/internal/auth"
)

// Authenticator resolves a session token to a user. auth.Service satisfies it.
// internal/httpapi depends on this interface — not on internal/auth's store or
// a database driver — so the middleware can name an authenticated user without
// importing the persistence layer.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (auth.User, error)
}

func init() {
	// The auth domain package owns these sentinels; their HTTP meaning is
	// registered here, the one place errors become status codes.
	registerErrStatus(auth.ErrNotFound, http.StatusNotFound)
	registerErrStatus(auth.ErrSignupDisabled, http.StatusForbidden)
	registerErrStatus(auth.ErrDomainNotAllowed, http.StatusForbidden)
	registerErrStatus(auth.ErrTokenInvalid, http.StatusBadRequest)
	registerErrStatus(auth.ErrTokenExpired, http.StatusGone)
	registerErrStatus(auth.ErrTokenConsumed, http.StatusGone)
	registerErrStatus(auth.ErrInviteInvalid, http.StatusBadRequest)
	registerErrStatus(auth.ErrIdentityConflict, http.StatusConflict)
	registerErrStatus(auth.ErrInvalidEmail, http.StatusBadRequest)
	registerErrStatus(auth.ErrOIDCNotConfigured, http.StatusNotFound)
	registerErrStatus(auth.ErrEmailRequired, http.StatusBadRequest)
}

// authResolve is middleware that reads a session token from the
// Authorization: Bearer header (preferred) or the ff_session cookie, resolves
// it through a, and — on success only — attaches the user to the request
// context. It never rejects: routes that require a user enforce that
// themselves (RequireAuth, or the auth handler's own guard).
func authResolve(a Authenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			if c, err := r.Cookie(auth.CookieName); err == nil {
				token = c.Value
			}
		}
		if token != "" {
			if u, err := a.Authenticate(r.Context(), token); err == nil {
				r = r.WithContext(auth.WithUser(r.Context(), u))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth wraps h so it is reached only with an authenticated user on the
// context (put there by authResolve); otherwise it writes a 401. Product
// routes added later use this; the auth handler guards its own routes.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserFromContext(r.Context()); !ok {
			writeJSON(w, r, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

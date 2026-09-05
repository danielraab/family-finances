// Package httpapi is the HTTP wiring shared by every backend endpoint:
// routing, middleware, JSON/error response helpers, and serving the embedded
// frontend. Domain packages never import it — they expose an http.Handler that
// Routes mounts under /api/<noun>/.
package httpapi

import (
	"context"
	"io/fs"
	"net/http"

	"at.draab/familyfinances/internal/config"
)

// DBPinger is the narrow view of the database the HTTP layer needs: a
// connectivity check for the health endpoint. *pgxpool.Pool satisfies it, so
// internal/httpapi never imports the driver.
type DBPinger interface {
	Ping(context.Context) error
}

// Deps are the runtime dependencies the HTTP layer needs, assembled by
// package main and passed in.
type Deps struct {
	// Static is the frontend's static export (already rooted at the site
	// root, e.g. via fs.Sub). Requests that match no /api/ route are served
	// from it.
	Static fs.FS

	// DB backs the /api/healthz connectivity probe.
	DB DBPinger

	// Auth resolves a session token to a user for the request-scoped auth
	// middleware. Nil disables the middleware (no user is ever attached).
	Auth Authenticator

	// AuthHandler is the auth domain package's http.Handler, mounted at
	// /api/auth/. Nil leaves the prefix unrouted.
	AuthHandler http.Handler

	// SettingsHandler is the settings domain package's http.Handler, mounted
	// at /api/settings. Nil leaves it unrouted.
	SettingsHandler http.Handler

	// AccountHandler is the account domain package's http.Handler, mounted
	// at /api/accounts and /api/account-types. Nil leaves them unrouted.
	AccountHandler http.Handler

	// CategoryHandler is the category domain package's http.Handler, mounted
	// at /api/categories. Nil leaves it unrouted.
	CategoryHandler http.Handler

	// TagHandler is the tag domain package's http.Handler, mounted at
	// /api/tags. Nil leaves it unrouted.
	TagHandler http.Handler

	// EntryHandler is the entry domain package's http.Handler, mounted at
	// /api/entries and (for the live balance read) GET
	// /api/accounts/{id}/balance — the one URL that lives under another
	// package's namespace, since balance is entry-owned logic. Go's
	// ServeMux resolves that specific pattern over AccountHandler's broader
	// /api/accounts/ mount by specificity, regardless of registration
	// order. Nil leaves it unrouted.
	EntryHandler http.Handler

	// OpenAPISpec is the raw bytes of the hand-written API contract
	// (openapi/openapi.yaml), served verbatim at GET /api/openapi.yaml. Nil
	// leaves that route unregistered (it 404s as any other unknown /api/ path).
	OpenAPISpec []byte
}

// Routes builds the request multiplexer: backend routes live under /api/,
// everything else falls through to the embedded static site.
func Routes(deps Deps) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", healthHandler(deps.DB))
	if deps.OpenAPISpec != nil {
		mux.HandleFunc("GET /api/openapi.yaml", openAPIHandler(deps.OpenAPISpec))
	}
	if deps.AuthHandler != nil {
		// More specific than the "/api/" fallback below, so ServeMux routes
		// every /api/auth/... path here.
		mux.Handle("/api/auth/", deps.AuthHandler)
	}
	if deps.SettingsHandler != nil {
		mux.Handle("/api/settings", deps.SettingsHandler)
	}
	if deps.AccountHandler != nil {
		mux.Handle("/api/accounts", deps.AccountHandler)
		mux.Handle("/api/accounts/", deps.AccountHandler)
		mux.Handle("/api/account-types", deps.AccountHandler)
		mux.Handle("/api/account-types/", deps.AccountHandler)
	}
	if deps.CategoryHandler != nil {
		mux.Handle("/api/categories", deps.CategoryHandler)
		mux.Handle("/api/categories/", deps.CategoryHandler)
	}
	if deps.TagHandler != nil {
		mux.Handle("/api/tags", deps.TagHandler)
		mux.Handle("/api/tags/", deps.TagHandler)
	}
	if deps.EntryHandler != nil {
		mux.Handle("/api/entries", deps.EntryHandler)
		mux.Handle("/api/entries/", deps.EntryHandler)
		// More specific than AccountHandler's "/api/accounts/" mount above,
		// so it wins for this one path regardless of registration order.
		mux.Handle("GET /api/accounts/{id}/balance", deps.EntryHandler)
	}
	// Unmatched /api/ paths get a JSON 404 — the reserved namespace never
	// falls through to the static site. More specific than "/", so it wins
	// for /api/... only.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, r, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	mux.Handle("/", staticHandler(deps.Static))
	return mux
}

// New builds the HTTP server: the route mux wrapped in the standard
// middleware chain, listening on the configured port.
func New(cfg config.Config, deps Deps) *http.Server {
	var handler http.Handler = Routes(deps)
	if deps.Auth != nil {
		handler = authResolve(deps.Auth, handler)
	}
	return &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: withMiddleware(handler),
	}
}

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
}

// Routes builds the request multiplexer: backend routes live under /api/,
// everything else falls through to the embedded static site.
func Routes(deps Deps) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", healthHandler(deps.DB))
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
	return &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: withMiddleware(Routes(deps)),
	}
}

package httpapi

import (
	"context"
	"net/http"
	"time"
)

// healthzDBTimeout bounds the database probe performed while serving
// GET /api/healthz.
const healthzDBTimeout = 2 * time.Second

// healthHandler serves GET /api/healthz. It reports 200 with body "ok" only
// when the database answers a bounded Ping, and 503 otherwise, so the Docker
// HEALTHCHECK and any orchestrator readiness probe reflect database
// reachability rather than mere process liveness. The Docker HEALTHCHECK
// relies on the plain "ok" body (see backend/healthcheck.go).
func healthHandler(db DBPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthzDBTimeout)
		defer cancel()

		if db == nil {
			http.Error(w, "no database configured", http.StatusServiceUnavailable)
			return
		}
		if err := db.Ping(ctx); err != nil {
			Logger(r.Context()).ErrorContext(r.Context(), "healthz database probe failed", "error", err)
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := w.Write([]byte("ok")); err != nil {
			Logger(r.Context()).ErrorContext(r.Context(), "write healthz response", "error", err)
		}
	}
}

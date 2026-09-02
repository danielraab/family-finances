package httpapi

import (
	"log/slog"
	"net/http"
)

// handleHealthz is the liveness probe behind GET /api/healthz. It always
// reports OK for a running process; the Docker HEALTHCHECK relies on the plain
// "ok" body (see backend/healthcheck.go).
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte("ok")); err != nil {
		slog.ErrorContext(r.Context(), "write healthz response", "error", err)
	}
}

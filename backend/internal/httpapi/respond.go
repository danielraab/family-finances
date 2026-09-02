package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// errStatus maps domain sentinel errors to HTTP status codes. Domain and
// service code never imports net/http; it returns sentinel errors and this is
// the single place they become status codes. Each domain package's store.go
// registers its sentinels here (see registerErrStatus).
var errStatus = map[error]int{}

// registerErrStatus lets a domain package associate its sentinel errors with
// HTTP status codes without this package importing it. Call from an init() in
// the wiring that mounts the handler.
func registerErrStatus(err error, status int) {
	errStatus[err] = status
}

// writeJSON encodes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.ErrorContext(r.Context(), "encode json response", "error", err)
	}
}

// writeError maps err to a status code and writes a JSON error body. Unknown
// errors become 500 and are logged; mapped sentinels use their status and a
// safe message.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	for sentinel, code := range errStatus {
		if errors.Is(err, sentinel) {
			status = code
			break
		}
	}

	if status == http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "unhandled request error", "error", err)
		writeJSON(w, r, status, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, r, status, map[string]string{"error": err.Error()})
}

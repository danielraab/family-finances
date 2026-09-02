package httpapi

import (
	"encoding/json"
	"errors"
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
		Logger(r.Context()).ErrorContext(r.Context(), "encode json response", "error", err)
	}
}

// writeError maps err to a status code and writes a JSON error body carrying
// the request id (so a caller can quote it). Unknown errors become 500 and are
// logged with the full error; mapped sentinels use their status and message.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	for sentinel, code := range errStatus {
		if errors.Is(err, sentinel) {
			status = code
			break
		}
	}

	body := map[string]string{"request_id": RequestID(r.Context())}
	if status == http.StatusInternalServerError {
		Logger(r.Context()).ErrorContext(r.Context(), "unhandled request error", "error", err)
		body["error"] = "internal server error"
	} else {
		body["error"] = err.Error()
	}

	writeJSON(w, r, status, body)
}
